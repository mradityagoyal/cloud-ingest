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
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	pb "github.com/GoogleCloudPlatform/cloud-ingest/tests/perf/proto"
	"github.com/golang/protobuf/proto"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"math"
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

	// Number of in-progress job runs.
	InProgressJobs int

	// Total time take to complete all the job runs. Only propagated when all the
	// running jobs are terminated.
	TotalTime time.Duration

	// The aggregate counters associated with all the job runs.
	Counters dcp.JobCounters
}

func (r PerfResult) String() string {
	return fmt.Sprintf("\n"+
		"Succeeded Jobs:   %d\n"+
		"Failed Jobs:      %d\n"+
		"In-Progress Jobs: %d\n"+
		"Time Taken:       %v\n"+
		"Counters:         %v",
		r.SucceededJobs, r.FailedJobs, r.InProgressJobs, r.TotalTime, r.Counters)
}

// PerfRunner is a struct to create job run and monitor their statuses.
type PerfRunner struct {
	configs      []runConfig
	jobService   JobService
	ticker       helpers.Ticker
	clock        helpers.Clock
	distribution helpers.Distribution
	gcs          gcloud.GCS
	projectId    string
	newBuckets   struct {
		sync.Mutex
		val []string
	}

	// Holds the last status of the perf run.
	currStatus struct {
		sync.Mutex
		val *PerfResult
	}
}

type runConfig struct {
	id         string
	validators []Validator
}

// NewPerfRunner creates a new PerfRunner based on a projectId. Uses the default
// project if the projectId is empty.
func NewPerfRunner(projectId, apiEndpoint string, gcs gcloud.GCS) (*PerfRunner, error) {
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
			projectId, apiEndpoint, oauth2.NewClient(context.Background(), creds.TokenSource)),
		ticker:       helpers.NewClockTicker(jobStatusPollingInterval),
		clock:        helpers.NewClock(),
		distribution: helpers.NewUniformDistribution(0, math.MaxInt32, time.Now().UnixNano()),
		gcs:          gcs,
		projectId:    projectId,
	}, nil
}

func (p *PerfRunner) getRandomBucketName() string {
	now := p.clock.Now()
	return fmt.Sprintf("opi-test-bucket-%04d-%02d-%02d-%02d-%02d-%d",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), p.distribution.GetNext())
}

// CreateConfigs reads a LoadTestingConfiguration message from filePath and
// creates job configs based on that message.
func (p *PerfRunner) CreateConfigs(ctx context.Context, filePath string) []error {
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
	var mu sync.Mutex // Protecting the errs and configs array.
	runTimeStamp := time.Now().UnixNano()
	for i, jobConfig := range loadTestingConfig.Config {
		// Create temporary bucket if one doesn't exist.
		if jobConfig.DestinationBucket == "" {
			jobConfig.DestinationBucket = p.getRandomBucketName()
			err := p.gcs.CreateBucket(ctx, p.projectId, jobConfig.DestinationBucket, nil)
			if err != nil {
				errs = append(errs, err)
				break
			}
			p.newBuckets.Lock()
			p.newBuckets.val = append(p.newBuckets.val, jobConfig.DestinationBucket)
			p.newBuckets.Unlock()
		}
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
			p.configs = append(p.configs, runConfig{id: jobConfigId, validators: p.getValidators(jobConfig)})
		}(fmt.Sprintf("%s-%d-%d", loadTestingConfig.Name, runTimeStamp, i), jobConfig)
	}
	// Wait for all the requests to be triggered.
	wg.Wait()

	return errs
}

func (p *PerfRunner) getValidators(jobConfig *pb.JobConfig) []Validator {
	validators := []Validator{}
	for _, v := range jobConfig.Validators {
		switch v {
		case pb.JobConfig_METADATA_VALIDATOR:
			validators = append(validators, NewMetadataValidator(
				p.gcs, jobConfig.SourceDir, jobConfig.DestinationBucket))
		case pb.JobConfig_DEEP_COMPARISON_VALIDATOR:
			validators = append(validators, NewDeepComparisonValidator(
				p.gcs, jobConfig.SourceDir, jobConfig.DestinationBucket))
		}
	}

	return validators
}

// MonitorJobs monitors the running jobs until either all the jobs are
// terminated or no progress has occurred in any of the running jobs for a
// timeout duration. After all jobs have terminated, it returns the PerfResult
// object with performance run results.
func (p *PerfRunner) MonitorJobs(ctx context.Context) (*PerfResult, error) {
	jobs := make([]*JobRunStatus, len(p.configs))
	done := false
	noProgressCount := 0
	for !done {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context is done with error: %v", ctx.Err())
		case <-p.ticker.GetChannel():
			var newJobs []*JobRunStatus
			newJobs, done = p.getJobsStatus()

			// Update the current status
			p.currStatus.Lock()
			p.currStatus.val = getPerfResult(newJobs)
			p.currStatus.Unlock()

			if done {
				break
			}

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

	p.currStatus.Lock()
	defer p.currStatus.Unlock()
	return p.currStatus.val, nil
}

type ConfigValidationResult struct {
	ConfigId string
	Success  bool
	Results  []ValidationResult
}

// ValidateResults runs all validators for all tests.
func (p *PerfRunner) ValidateResults(ctx context.Context) []ConfigValidationResult {
	results := make([]ConfigValidationResult, 0, len(p.configs))
	for _, config := range p.configs {
		configResult := ConfigValidationResult{ConfigId: config.id, Success: true}
		for _, validator := range config.validators {
			validationResult := validator.Validate(ctx)
			configResult.Results = append(configResult.Results, validationResult)
			if validationResult.Err != nil || !validationResult.Success {
				// Stop running validators for a config as soon as we find a failure.
				configResult.Success = false
				break
			}
		}
		results = append(results, configResult)
	}

	return results
}

// CleanUp performs any post-run cleanup tasks, like deleting temporary GCS buckets.
func (p *PerfRunner) CleanUp(ctx context.Context) []error {
	errs := []error{}
	p.newBuckets.Lock()
	defer p.newBuckets.Unlock()
	for _, bucket := range p.newBuckets.val {
		// Delete all contents. Consider doing this in parallel if this is slow.
		iter := p.gcs.ListObjects(ctx, bucket, nil)
		for {
			objAttrs, err := iter.Next()
			if err == iterator.Done {
				break
			} else if err != nil {
				// Can't trust a broken iterator; bail here.
				errs = append(errs, err)
				break
			}

			err = p.gcs.DeleteObject(ctx, bucket, objAttrs.Name)
			if err != nil {
				errs = append(errs, err)
			}
		}

		// Nuke the now empty bucket.
		err := p.gcs.DeleteBucket(ctx, bucket)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// GetStatus gets the current status of the running perf.
func (p *PerfRunner) GetStatus() *PerfResult {
	p.currStatus.Lock()
	defer p.currStatus.Unlock()
	return p.currStatus.val
}

// getJobsStatus return a tuple of jobs statuses for all the jobs, and whether
// all the jobs terminated.
func (p PerfRunner) getJobsStatus() ([]*JobRunStatus, bool) {
	jobsTerminated := true
	jobs := make([]*JobRunStatus, len(p.configs))
	for i, config := range p.configs {
		j, err := p.jobService.GetJobStatus(config.id, defaultRunId)
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
		if j == nil {
			continue
		}
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
	result.InProgressJobs = len(jobs) - (result.SucceededJobs + result.FailedJobs)
	if result.InProgressJobs == 0 {
		result.TotalTime = time.Duration(finishTime - startTime)
	}
	return result
}
