package perf

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	"github.com/golang/mock/gomock"
)

func TestCreateConfigsFileNotExist(t *testing.T) {
	runner := PerfRunner{}
	errs := runner.CreateConfigs("does-not-exist-file")
	if errs == nil || len(errs) != 1 {
		t.Errorf("expected reading file error, but found errs is nil.")
	}
	if strings.Contains(errs[0].Error(), "Error reading file") {
		t.Errorf("expected error reading file but got %v.", errs)
	}
}

func TestCreateConfigsProtoParseError(t *testing.T) {
	tmpFile := dcp.CreateTmpFile("perfrunner-test-", "This is corrupted proto")
	defer os.Remove(tmpFile) // clean up
	runner := PerfRunner{}
	errs := runner.CreateConfigs(tmpFile)
	if errs == nil || len(errs) != 1 {
		t.Errorf("expected parsing proto error, but found errs is nil.")
	}
	if strings.Contains(errs[0].Error(), "Error parsing proto") {
		t.Errorf("expected error parsing proto but got %v.", errs)
	}
}

func TestCreateConfigsPartialFail(t *testing.T) {
	tmpFile := dcp.CreateTmpFile("perfrunner-test-", `
name: "dummy-perf-test"
config: {
  sourceDir: "dir-1"
  destinationBucket: "bucket-1"
}
config: {
  sourceDir: "dir-2"
  destinationBucket: "bucket-2"
}`)
	defer os.Remove(tmpFile) // clean up

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockJobService := NewMockJobService(mockCtrl)

	expectedErr := fmt.Errorf("failed creating job config")
	mockJobService.EXPECT().CreateJobConfig(gomock.Any(), "dir-1", "bucket-1").Return(nil)
	mockJobService.EXPECT().CreateJobConfig(gomock.Any(), "dir-2", "bucket-2").Return(expectedErr)

	runner := PerfRunner{
		jobService: mockJobService,
	}

	errs := runner.CreateConfigs(tmpFile)
	if errs == nil || len(errs) != 1 {
		t.Errorf("expected 1 failure but found %v.", errs)
	}
	if errs[0] != expectedErr {
		t.Errorf("expected err[0] to be: %v, but found: %v.", errs[0], expectedErr)
	}
}

func TestCreateConfigsSuccess(t *testing.T) {
	tmpFile := dcp.CreateTmpFile("perfrunner-test-", `
name: "dummy-perf-test"
config: {
  sourceDir: "dir-1"
  destinationBucket: "bucket-1"
}
config: {
  sourceDir: "dir-2"
  destinationBucket: "bucket-2"
}`)
	defer os.Remove(tmpFile) // clean up

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockJobService := NewMockJobService(mockCtrl)

	mockJobService.EXPECT().CreateJobConfig(gomock.Any(), "dir-1", "bucket-1").Return(nil)
	mockJobService.EXPECT().CreateJobConfig(gomock.Any(), "dir-2", "bucket-2").Return(nil)

	runner := PerfRunner{
		jobService: mockJobService,
	}

	errs := runner.CreateConfigs(tmpFile)
	if len(errs) != 0 {
		t.Errorf("expected success found errors: %v.", errs)
	}
}

func TestGetJobStatusJobsTerminated(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockJobService := NewMockJobService(mockCtrl)

	configs := []string{"config-1", "config-2", "config-3"}
	runner := PerfRunner{
		configIds:  configs,
		jobService: mockJobService,
	}
	expectedStatuses := []*JobRunStatus{
		&JobRunStatus{Status: dcp.JobSuccess},
		&JobRunStatus{Status: dcp.JobFailed},
		&JobRunStatus{Status: dcp.JobSuccess},
	}

	for i := range expectedStatuses {
		mockJobService.EXPECT().GetJobStatus(configs[i], defaultRunId).Return(
			expectedStatuses[i], nil)
	}
	statuses, terminated := runner.getJobsStatus()
	if !terminated {
		t.Errorf("expected all jobs to be terminated, but terminated is false.")
	}
	if !reflect.DeepEqual(expectedStatuses, statuses) {
		t.Errorf("expected jobs statuses to be: %v, but found: %v.",
			expectedStatuses, statuses)
	}
}

func TestGetJobStatusJobsNotTerminated(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockJobService := NewMockJobService(mockCtrl)

	configs := []string{"config-1", "config-2", "config-3"}
	runner := PerfRunner{
		configIds:  configs,
		jobService: mockJobService,
	}
	expectedStatuses := []*JobRunStatus{
		&JobRunStatus{Status: dcp.JobSuccess},
		&JobRunStatus{Status: dcp.JobInProgress},
		&JobRunStatus{Status: dcp.JobFailed},
	}

	for i := range expectedStatuses {
		mockJobService.EXPECT().GetJobStatus(configs[i], defaultRunId).Return(
			expectedStatuses[i], nil)
	}
	statuses, terminated := runner.getJobsStatus()
	if terminated {
		t.Errorf("expected jobs are not terminated, but terminated is true.")
	}
	if !reflect.DeepEqual(expectedStatuses, statuses) {
		t.Errorf("expected jobs statuses to be: %v, but found: %v.",
			expectedStatuses, statuses)
	}
}

func TestGetJobStatusJobsErrorGettingJobStatus(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockJobService := NewMockJobService(mockCtrl)

	runner := PerfRunner{
		configIds:  []string{"config-1", "config-2"},
		jobService: mockJobService,
	}
	expectedStatuses := []*JobRunStatus{
		nil,
		&JobRunStatus{Status: dcp.JobInProgress},
	}

	mockJobService.EXPECT().GetJobStatus("config-1", defaultRunId).Return(
		expectedStatuses[0], fmt.Errorf("error getting job status"))
	mockJobService.EXPECT().GetJobStatus("config-2", defaultRunId).Return(
		expectedStatuses[1], nil)

	statuses, terminated := runner.getJobsStatus()
	if terminated {
		t.Errorf("expected jobs are not terminated, but terminated is true.")
	}
	if !reflect.DeepEqual(expectedStatuses, statuses) {
		t.Errorf("expected jobs statuses to be: %v, but found: %v.",
			expectedStatuses, statuses)
	}
}

func TestGetPerfResult(t *testing.T) {
	counters1 := dcp.JobCounters{}
	counters1.Unmarshal(`{"counter-1": 20, "counter-2": 30}`)

	counters2 := dcp.JobCounters{}
	counters2.Unmarshal(`{"counter-2": 10, "counter-3": 50}`)

	aggregateCounters := dcp.JobCounters{}
	aggregateCounters.Unmarshal(`{"counter-1": 20, "counter-2": 40, "counter-3": 50}`)

	jobStatuses := []*JobRunStatus{
		&JobRunStatus{
			Status:       dcp.JobFailed,
			CreationTime: 5,
			FinishTime:   25,
			Counters:     counters1,
		},
		&JobRunStatus{
			Status:       dcp.JobSuccess,
			CreationTime: 7,
			FinishTime:   30,
			Counters:     counters2,
		},
	}

	expectedResult := &PerfResult{
		SucceededJobs: 1,
		FailedJobs:    1,
		TotalTime:     25,
		Counters:      aggregateCounters,
	}
	result := getPerfResult(jobStatuses)
	if !reflect.DeepEqual(result, expectedResult) {
		t.Errorf("expected perf result to be : %v, but found: %v",
			expectedResult, result)
	}
}

func TestMonitorJobsTimeout(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockJobService := NewMockJobService(mockCtrl)

	mockTicker := dcp.NewMockTicker()
	runner := PerfRunner{
		configIds:  []string{"config-1", "config-2"},
		jobService: mockJobService,
		ticker:     mockTicker,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := runner.MonitorJobs(context.Background())
		if err == nil {
			t.Errorf("expected MonitorJobs fail with timeout error, but error is nil.")
		}
		if !strings.Contains(err.Error(), "operation timed-out after no progress") {
			t.Errorf("expected timeout error, but found: %v", err)
		}
	}()

	for i := 0; i < maxNoProgressCount; i++ {
		jobStatusErr := fmt.Errorf("error getting job config")
		mockJobService.EXPECT().GetJobStatus("config-1", defaultRunId).Return(
			nil, jobStatusErr)
		mockJobService.EXPECT().GetJobStatus("config-2", defaultRunId).Return(
			nil, jobStatusErr)
		mockTicker.Tick()
	}

	wg.Wait()
}

func TestMonitorJobsCancelledContext(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockJobService := NewMockJobService(mockCtrl)

	mockTicker := dcp.NewMockTicker()
	runner := PerfRunner{
		configIds:  []string{"config-1", "config-2"},
		jobService: mockJobService,
		ticker:     mockTicker,
	}

	ctx, cancelFn := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := runner.MonitorJobs(ctx)
		if err == nil {
			t.Errorf("expected MonitorJobs fail with context error, but error is nil.")
		}
		if !strings.Contains(err.Error(), "context is done with error") {
			t.Errorf("expected context done error, but found: %v", err)
		}
	}()

	mockJobService.EXPECT().GetJobStatus("config-1", defaultRunId).Return(
		&JobRunStatus{Status: dcp.JobSuccess}, nil)
	mockJobService.EXPECT().GetJobStatus("config-2", defaultRunId).Return(
		&JobRunStatus{Status: dcp.JobInProgress}, nil)
	mockTicker.Tick()

	// Cancel the context.
	cancelFn()
	wg.Wait()
}

func TestMonitorJobsSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockJobService := NewMockJobService(mockCtrl)

	counters1 := dcp.JobCounters{}
	counters1.Unmarshal(`{"counter-1": 20, "counter-2": 30}`)

	counters2 := dcp.JobCounters{}
	counters2.Unmarshal(`{"counter-2": 10, "counter-3": 50}`)

	aggregateCounters := dcp.JobCounters{}
	aggregateCounters.Unmarshal(`{"counter-1": 20, "counter-2": 40, "counter-3": 50}`)

	mockTicker := dcp.NewMockTicker()
	runner := PerfRunner{
		configIds:  []string{"config-1", "config-2"},
		jobService: mockJobService,
		ticker:     mockTicker,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err := runner.MonitorJobs(context.Background())
		if err != nil {
			t.Errorf("expected MonitorJobs to succeed, but found err: %v", err)
		}
		expectedResult := &PerfResult{
			SucceededJobs: 1,
			FailedJobs:    1,
			TotalTime:     6,
			Counters:      aggregateCounters,
		}

		if !reflect.DeepEqual(result, expectedResult) {
			t.Errorf("expected perf result to be : %v, but found: %v",
				expectedResult, result)
		}
	}()

	mockJobService.EXPECT().GetJobStatus("config-1", defaultRunId).Return(
		&JobRunStatus{
			Status:       dcp.JobSuccess,
			CreationTime: 5,
			FinishTime:   10,
			Counters:     counters1,
		}, nil)

	mockJobService.EXPECT().GetJobStatus("config-2", defaultRunId).Return(
		&JobRunStatus{
			Status:       dcp.JobFailed,
			CreationTime: 6,
			FinishTime:   11,
			Counters:     counters2,
		}, nil)
	mockTicker.Tick()

	wg.Wait()
}

func TestCreateJobsAndMonitorJobsSuccess(t *testing.T) {
	tmpFile := dcp.CreateTmpFile("perfrunner-test-", `
name: "dummy-perf-test"
config: {
  sourceDir: "dir-1"
  destinationBucket: "bucket-1"
}
config: {
  sourceDir: "dir-2"
  destinationBucket: "bucket-2"
}`)
	defer os.Remove(tmpFile) // clean up

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockJobService := NewMockJobService(mockCtrl)

	mockJobService.EXPECT().CreateJobConfig(gomock.Any(), "dir-1", "bucket-1").Return(nil)
	mockJobService.EXPECT().CreateJobConfig(gomock.Any(), "dir-2", "bucket-2").Return(nil)

	mockTicker := dcp.NewMockTicker()
	runner := PerfRunner{
		jobService: mockJobService,
		ticker:     mockTicker,
	}

	if errs := runner.CreateConfigs(tmpFile); len(errs) != 0 {
		t.Errorf("expected success found errors: %v.", errs)
	}

	counters1 := dcp.JobCounters{}
	counters1.Unmarshal(`{"counter-1": 20, "counter-2": 30}`)

	counters2 := dcp.JobCounters{}
	counters2.Unmarshal(`{"counter-2": 10, "counter-3": 50}`)

	aggregateCounters := dcp.JobCounters{}
	aggregateCounters.Unmarshal(`{"counter-1": 20, "counter-2": 40, "counter-3": 50}`)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err := runner.MonitorJobs(context.Background())
		if err != nil {
			t.Errorf("expected MonitorJobs to succeed, but found err: %v", err)
		}
		expectedResult := &PerfResult{
			SucceededJobs: 1,
			FailedJobs:    1,
			TotalTime:     6,
			Counters:      aggregateCounters,
		}

		if !reflect.DeepEqual(result, expectedResult) {
			t.Errorf("expected perf result to be : %v, but found: %v",
				expectedResult, result)
		}
	}()

	// Get through intermediate state. One job failed and one job in progress.
	mockJobService.EXPECT().GetJobStatus(runner.configIds[0], defaultRunId).Return(
		&JobRunStatus{
			Status:       dcp.JobInProgress,
			CreationTime: 5,
		}, nil)

	mockJobService.EXPECT().GetJobStatus(runner.configIds[1], defaultRunId).Return(
		&JobRunStatus{
			Status:       dcp.JobFailed,
			CreationTime: 6,
			FinishTime:   11,
			Counters:     counters2,
		}, nil)
	mockTicker.Tick()

	// Transition the in progress job to success.
	mockJobService.EXPECT().GetJobStatus(runner.configIds[0], defaultRunId).Return(
		&JobRunStatus{
			Status:       dcp.JobSuccess,
			CreationTime: 5,
			FinishTime:   10,
			Counters:     counters1,
		}, nil)

	mockJobService.EXPECT().GetJobStatus(runner.configIds[1], defaultRunId).Return(
		&JobRunStatus{
			Status:       dcp.JobFailed,
			CreationTime: 6,
			FinishTime:   11,
			Counters:     counters2,
		}, nil)
	mockTicker.Tick()

	wg.Wait()
}
