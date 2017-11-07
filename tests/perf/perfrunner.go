package perf

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	pb "github.com/GoogleCloudPlatform/cloud-ingest/tests/perf/proto"
	"github.com/golang/protobuf/proto"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	jobStatusPollingInterval = 10 * time.Second
	noProgessTimeout         = 5 * time.Minute
	maxNoProgressCount       = int(noProgessTimeout / jobStatusPollingInterval)

	defaultRunId = "jobrun"
)

// PerfResult represents the result after the perf run is complete.
type PerfResult struct {
	// Number of succeeded job runs.
	SucceededJobs int

	// Number of failed job runs.
	FailedJobs int

	// Total time taken to complete all the job runs.
	TotalTime time.Duration

	// The aggregate counters associated with all the job runs.
	Counters dcp.JobCounters
}

// PerfRunner is a struct to create job runs and monitor their statuses.
type PerfRunner struct {
	configIds  []string
	jobService JobService
	ticker     dcp.Ticker
}

// NewPerfRunner creates a new PerfRunner based on a projectId. Uses the default
// project if the projectId is empty.
func NewPerfRunner(projectId string) (*PerfRunner, error) {
	creds, err := google.FindDefaultCredentials(context.Background())
	if err != nil {
		log.Printf("Can not find default credentials, err: %v.", err)
		return nil, err
	}
	if projectId == "" {
		projectId = creds.ProjectID
	}
	return &PerfRunner{
		jobService: NewIngestService(
			projectId, oauth2.NewClient(context.Background(), creds.TokenSource)),
		ticker: dcp.NewClockTicker(jobStatusPollingInterval),
	}, nil
}

// CreateConfigs reads a LoadTestingConfiguration message from filePath and
// creates job configs based on that message.
func (p *PerfRunner) CreateConfigs(filePath string) []error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading file %v", err)
		return []error{err}
	}
	loadTestingConfig := &pb.LoadTestingConfiguration{}
	if err := proto.UnmarshalText(string(data), loadTestingConfig); err != nil {
		log.Printf("Error parsing proto with error %v", err)
		return []error{err}
	}

	var wg sync.WaitGroup
	var errs []error
	var mu sync.Mutex // Protecting the errs and configIds array.
	runTimeStamp := time.Now().UnixNano()
	for i, jobConfig := range loadTestingConfig.Config {
		wg.Add(1)
		go func(jobConfigId string, jobConfig *pb.JobConfig) {
			defer wg.Done()
			if err := p.jobService.CreateJobConfig(
				jobConfigId, jobConfig.SourceDir, jobConfig.DestinationBucket); err != nil {
				mu.Lock()
				defer mu.Unlock()
				errs = append(errs, err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			p.configIds = append(p.configIds, jobConfigId)
		}(fmt.Sprintf("%s-%d-%d", loadTestingConfig.Name, runTimeStamp, i), jobConfig)
	}
	// Wait for all the requests to be triggered.
	wg.Wait()

	return errs
}

// MonitorJobs monitors the running jobs until either all the jobs are
// terminated or no progress has occurred in any of the running jobs for a
// timeout duration. After all jobs have terminated, it returns the PerfResult
// object with performance run results.
func (p PerfRunner) MonitorJobs(ctx context.Context) (*PerfResult, error) {
	jobs := make([]*JobRunStatus, len(p.configIds))
	done := false
	noProgressCount := 0
	for !done {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context is done with error: %v", ctx.Err())
		case <-p.ticker.GetChannel():
			var newJobs []*JobRunStatus
			newJobs, done = p.getJobsStatus()

			if reflect.DeepEqual(jobs, newJobs) {
				noProgressCount++
			} else {
				noProgressCount = 0
			}

			if noProgressCount >= maxNoProgressCount {
				return nil, fmt.Errorf(
					"operation timed-out after no progress of %v.", noProgessTimeout)
			}

			jobs = newJobs
		}
	}
	return getPerfResult(jobs), nil
}

// getJobsStatus return a tuple of jobs statuses for all the jobs, and whether
// all the jobs terminated.
func (p PerfRunner) getJobsStatus() ([]*JobRunStatus, bool) {
	jobsTerminated := true
	jobs := make([]*JobRunStatus, len(p.configIds))
	for i, configId := range p.configIds {
		j, err := p.jobService.GetJobStatus(configId, defaultRunId)
		if err != nil {
			j = nil
		}
		if j == nil || !dcp.IsJobTerminated(j.Status) {
			jobsTerminated = false
		}
		jobs[i] = j
	}
	return jobs, jobsTerminated
}

// getPerfResult return PerfResult object with the performance run results.
func getPerfResult(jobs []*JobRunStatus) *PerfResult {
	result := &PerfResult{}
	startTime := time.Now().UnixNano()
	finishTime := int64(0)
	for _, j := range jobs {
		if j.Status == dcp.JobSuccess {
			result.SucceededJobs++
		} else if j.Status == dcp.JobFailed {
			result.FailedJobs++
		}
		if j.CreationTime < startTime {
			startTime = j.CreationTime
		}
		if j.FinishTime > finishTime {
			finishTime = j.FinishTime
		}
		result.Counters.ApplyDelta(&j.Counters)
	}
	result.TotalTime = time.Duration(finishTime - startTime)
	return result
}
