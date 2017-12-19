package perf

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/mock/gomock"
)

const (
	fakeValidatorName = "fake validation"
	testConfigId      = "job-config-id"
)

var errFail error = errors.New("some error occurred")

func TestCreateConfigsFileNotExist(t *testing.T) {
	runner := PerfRunner{}
	errs := runner.CreateConfigs(context.Background(), "does-not-exist-file")
	if errs == nil || len(errs) != 1 {
		t.Errorf("expected reading file error, but found errs is nil.")
	}
	if strings.Contains(errs[0].Error(), "Error reading file") {
		t.Errorf("expected error reading file but got %v.", errs)
	}
}

func TestCreateConfigsProtoParseError(t *testing.T) {
	tmpFile := helpers.CreateTmpFile("", "perfrunner-test-", "This is corrupted proto")
	defer os.Remove(tmpFile) // clean up
	runner := PerfRunner{}
	errs := runner.CreateConfigs(context.Background(), tmpFile)
	if errs == nil || len(errs) != 1 {
		t.Errorf("expected parsing proto error, but found errs is nil.")
	}
	if strings.Contains(errs[0].Error(), "Error parsing proto") {
		t.Errorf("expected error parsing proto but got %v.", errs)
	}
}

func TestCreateConfigsPartialFail(t *testing.T) {
	tmpFile := helpers.CreateTmpFile("", "perfrunner-test-", `
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

	errs := runner.CreateConfigs(context.Background(), tmpFile)
	if errs == nil || len(errs) != 1 {
		t.Errorf("expected 1 failure but found %v.", errs)
	} else if errs[0] != expectedErr {
		t.Errorf("expected err[0] to be: %v, but found: %v.", expectedErr, errs[0])
	}
}

func TestCreateConfigsSuccess(t *testing.T) {
	tmpFile := helpers.CreateTmpFile("", "perfrunner-test-", `
		name: "dummy-perf-test"
		config: {
		  sourceDir: "dir-1"
		  destinationBucket: "bucket-1"
		}
		config: {
		  sourceDir: "dir-2"
		  destinationBucket: "bucket-2"
		}
		config: {
		  sourceDir: "dir-3"
		}
		config: {
		  sourceDir: "dir-4"
		}`)
	defer os.Remove(tmpFile) // clean up

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockJobService := NewMockJobService(mockCtrl)

	projectId := "project-id"
	newBucket1 := "opi-test-bucket-2017-12-07-00-00-1234"
	newBucket2 := "opi-test-bucket-2017-12-07-00-01-2345"

	mockJobService.EXPECT().CreateJobConfig(gomock.Any(), "dir-1", "bucket-1").Return(nil)
	mockJobService.EXPECT().CreateJobConfig(gomock.Any(), "dir-2", "bucket-2").Return(nil)
	mockJobService.EXPECT().CreateJobConfig(gomock.Any(), "dir-3", newBucket1).Return(nil)
	mockJobService.EXPECT().CreateJobConfig(gomock.Any(), "dir-4", newBucket2).Return(nil)

	mockClock := helpers.NewMockClock(mockCtrl)
	mockClock.EXPECT().Now().Return(time.Date(2017, time.December, 7, 0, 0, 45, 456, time.UTC))
	mockClock.EXPECT().Now().Return(time.Date(2017, time.December, 7, 0, 1, 45, 456, time.UTC))

	mockDistribution := helpers.NewMockDistribution(mockCtrl)
	mockDistribution.EXPECT().GetNext().Return(1234)
	mockDistribution.EXPECT().GetNext().Return(2345)

	mockGcs := gcloud.NewMockGCS(mockCtrl)
	mockGcs.EXPECT().CreateBucket(gomock.Any(), projectId, newBucket1, nil).Return(nil)
	mockGcs.EXPECT().CreateBucket(gomock.Any(), projectId, newBucket2, nil).Return(nil)
	mockGcs.EXPECT().ListObjects(gomock.Any(), newBucket1, nil).Return(gcloud.NewObjectIterator(
		&storage.ObjectAttrs{Name: "object1"},
		&storage.ObjectAttrs{Name: "object2"},
	))
	mockGcs.EXPECT().DeleteObject(gomock.Any(), newBucket1, "object1").Return(nil)
	mockGcs.EXPECT().DeleteObject(gomock.Any(), newBucket1, "object2").Return(nil)
	mockGcs.EXPECT().DeleteBucket(gomock.Any(), newBucket1).Return(nil)
	mockGcs.EXPECT().ListObjects(gomock.Any(), newBucket2, nil).Return(gcloud.NewObjectIterator())
	mockGcs.EXPECT().DeleteBucket(gomock.Any(), newBucket2).Return(nil)

	runner := PerfRunner{
		jobService:   mockJobService,
		clock:        mockClock,
		distribution: mockDistribution,
		gcs:          mockGcs,
		projectId:    projectId,
	}

	errs := runner.CreateConfigs(context.Background(), tmpFile)
	if len(errs) != 0 {
		t.Errorf("expected config creation success found errors: %v.", errs)
	}

	errs = runner.CleanUp(context.Background())
	if len(errs) != 0 {
		t.Errorf("expected cleanup success found errors: %v.", errs)
	}
}

func TestCreateConfigValidators(t *testing.T) {
	tmpFile := helpers.CreateTmpFile("", "perfrunner-test-", `
		name: "dummy-perf-test"
		config: {
		  sourceDir: "dir-1"
		  destinationBucket: "bucket-1"
		  validators: [
			METADATA_VALIDATOR,
			DEEP_COMPARISON_VALIDATOR
		  ]
		}
		config: {
		  sourceDir: "dir-2"
		  destinationBucket: "bucket-2"
		}`)
	defer os.Remove(tmpFile) // clean up

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockJobService := NewMockJobService(mockCtrl)

	projectId := "project-id"

	mockJobService.EXPECT().CreateJobConfig(gomock.Any(), "dir-1", "bucket-1").Return(nil)
	mockJobService.EXPECT().CreateJobConfig(gomock.Any(), "dir-2", "bucket-2").Return(nil)
	mockGcs := gcloud.NewMockGCS(mockCtrl)

	runner := PerfRunner{
		jobService: mockJobService,
		gcs:        mockGcs,
		projectId:  projectId,
	}

	errs := runner.CreateConfigs(context.Background(), tmpFile)
	if len(errs) != 0 {
		t.Errorf("expected config creation success found errors: %v.", errs)
	}

	if len(runner.configs) == 2 {
		// Results are appended by separate goroutines here, so ordering is not guaranteed.
		// We make sure to set the resulting array into a consistent order, for testing.
		// The IDs used have a trailing "-0" or "-1" that are consistent.
		if runner.configs[0].id[len(runner.configs[0].id)-1:] != "0" {
			runner.configs = []runConfig{runner.configs[1], runner.configs[0]}
		}

		// 2 validators for dir-1
		expectedValidators := []Validator{
			NewMetadataValidator(mockGcs, "dir-1", "bucket-1"),
			NewDeepComparisonValidator(mockGcs, "dir-1", "bucket-1"),
		}
		if !reflect.DeepEqual(runner.configs[0].validators, expectedValidators) {
			t.Errorf("expected %T %v, got %T: %v",
				expectedValidators, expectedValidators,
				runner.configs[0].validators, runner.configs[0].validators)
		}

		// no validators for dir-2
		if len(runner.configs[1].validators) != 0 {
			t.Errorf("expected no validators, got %T: %v",
				runner.configs[1].validators, runner.configs[1].validators)
		}
	} else {
		t.Errorf("expected two configs loaded, found %v.", runner.configs)
	}
}

func TestCreateConfigPartialBucketCreationFail(t *testing.T) {
	tmpFile := helpers.CreateTmpFile("", "perfrunner-test-", `
		name: "dummy-perf-test"
		config: {
		  sourceDir: "dir-1"
		}
		config: {
		  sourceDir: "dir-2"
		}`)
	defer os.Remove(tmpFile) // clean up

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockJobService := NewMockJobService(mockCtrl)

	projectId := "project-id"
	newBucket1 := "opi-test-bucket-2017-12-07-00-00-1234"
	newBucket2 := "opi-test-bucket-2017-12-07-00-01-2345"

	mockJobService.EXPECT().CreateJobConfig(gomock.Any(), "dir-1", newBucket1).Return(nil)

	mockClock := helpers.NewMockClock(mockCtrl)
	mockClock.EXPECT().Now().Return(time.Date(2017, time.December, 7, 0, 0, 45, 456, time.UTC))
	mockClock.EXPECT().Now().Return(time.Date(2017, time.December, 7, 0, 1, 45, 456, time.UTC))

	mockDistribution := helpers.NewMockDistribution(mockCtrl)
	mockDistribution.EXPECT().GetNext().Return(1234)
	mockDistribution.EXPECT().GetNext().Return(2345)

	mockGcs := gcloud.NewMockGCS(mockCtrl)
	expectedErr := fmt.Errorf("bucket creation error")
	mockGcs.EXPECT().CreateBucket(gomock.Any(), projectId, newBucket1, nil).Return(nil)
	mockGcs.EXPECT().CreateBucket(gomock.Any(), projectId, newBucket2, nil).Return(expectedErr)
	mockGcs.EXPECT().ListObjects(gomock.Any(), newBucket1, nil).Return(gcloud.NewObjectIterator())
	mockGcs.EXPECT().DeleteBucket(gomock.Any(), newBucket1).Return(nil)

	runner := PerfRunner{
		jobService:   mockJobService,
		clock:        mockClock,
		distribution: mockDistribution,
		gcs:          mockGcs,
		projectId:    projectId,
	}

	errs := runner.CreateConfigs(context.Background(), tmpFile)
	if errs == nil || len(errs) != 1 {
		t.Errorf("expected 1 failure but found %v.", errs)
	} else if errs[0] != expectedErr {
		t.Errorf("expected err[0] to be: %v, but found: %v.", expectedErr, errs[0])
	}

	runner.CleanUp(context.Background())
}

func TestCleanupPartialFail(t *testing.T) {
	tmpFile := helpers.CreateTmpFile("", "perfrunner-test-", `
		name: "dummy-perf-test"
		config: {
		  sourceDir: "dir-1"
		}
		config: {
		  sourceDir: "dir-2"
		}`)
	defer os.Remove(tmpFile) // clean up

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockJobService := NewMockJobService(mockCtrl)

	projectId := "project-id"
	newBucket1 := "opi-test-bucket-2017-12-07-00-00-1234"
	newBucket2 := "opi-test-bucket-2017-12-07-00-01-2345"

	mockJobService.EXPECT().CreateJobConfig(gomock.Any(), "dir-1", newBucket1).Return(nil)
	mockJobService.EXPECT().CreateJobConfig(gomock.Any(), "dir-2", newBucket2).Return(nil)

	mockClock := helpers.NewMockClock(mockCtrl)
	mockClock.EXPECT().Now().Return(time.Date(2017, time.December, 7, 0, 0, 45, 456, time.UTC))
	mockClock.EXPECT().Now().Return(time.Date(2017, time.December, 7, 0, 1, 45, 456, time.UTC))

	mockDistribution := helpers.NewMockDistribution(mockCtrl)
	mockDistribution.EXPECT().GetNext().Return(1234)
	mockDistribution.EXPECT().GetNext().Return(2345)

	mockGcs := gcloud.NewMockGCS(mockCtrl)
	bucketDelError := fmt.Errorf("bucket deletion error")
	objectDelError := fmt.Errorf("object deletion error")
	iterError := fmt.Errorf("iteration failed")
	mockGcs.EXPECT().CreateBucket(gomock.Any(), projectId, newBucket1, nil).Return(nil)
	mockGcs.EXPECT().CreateBucket(gomock.Any(), projectId, newBucket2, nil).Return(nil)
	mockGcs.EXPECT().ListObjects(gomock.Any(), newBucket1, nil).Return(gcloud.NewObjectIterator(
		&storage.ObjectAttrs{Name: "object1"}, // Delete fails
		&storage.ObjectAttrs{Name: "object2"}, // Delete succeeds
	))
	mockGcs.EXPECT().DeleteObject(gomock.Any(), newBucket1, "object1").Return(objectDelError)
	mockGcs.EXPECT().DeleteObject(gomock.Any(), newBucket1, "object2").Return(nil)
	mockGcs.EXPECT().DeleteBucket(gomock.Any(), newBucket1).Return(bucketDelError)
	mockGcs.EXPECT().ListObjects(gomock.Any(), newBucket2, nil).Return(gcloud.NewObjectIterator(
		iterError, // Iterator fails, don't end up deleting anything
		&storage.ObjectAttrs{Name: "object4"},
	))
	mockGcs.EXPECT().DeleteBucket(gomock.Any(), newBucket2).Return(nil)

	runner := PerfRunner{
		jobService:   mockJobService,
		clock:        mockClock,
		distribution: mockDistribution,
		gcs:          mockGcs,
		projectId:    projectId,
	}

	runner.CreateConfigs(context.Background(), tmpFile)
	errs := runner.CleanUp(context.Background())
	expectedErrors := []error{objectDelError, bucketDelError, iterError}
	if !reflect.DeepEqual(errs, expectedErrors) {
		t.Errorf("wanted errors %v, but got %v", expectedErrors, errs)
	}
}

func TestGetJobStatusJobsTerminated(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockJobService := NewMockJobService(mockCtrl)

	configs := []runConfig{{id: "config-1"}, {id: "config-2"}, {id: "config-3"}}
	runner := PerfRunner{
		configs:    configs,
		jobService: mockJobService,
	}
	expectedStatuses := []*JobRunStatus{
		&JobRunStatus{Status: dcp.JobSuccess},
		&JobRunStatus{Status: dcp.JobFailed},
		&JobRunStatus{Status: dcp.JobSuccess},
	}

	for i := range expectedStatuses {
		mockJobService.EXPECT().GetJobStatus(configs[i].id, defaultRunId).Return(
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

	configs := []runConfig{{id: "config-1"}, {id: "config-2"}, {id: "config-3"}}
	runner := PerfRunner{
		configs:    configs,
		jobService: mockJobService,
	}
	expectedStatuses := []*JobRunStatus{
		&JobRunStatus{Status: dcp.JobSuccess},
		&JobRunStatus{Status: dcp.JobInProgress},
		&JobRunStatus{Status: dcp.JobFailed},
	}

	for i := range expectedStatuses {
		mockJobService.EXPECT().GetJobStatus(configs[i].id, defaultRunId).Return(
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
		configs:    []runConfig{{id: "config-1"}, {id: "config-2"}},
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
		SucceededJobs:  1,
		FailedJobs:     1,
		InProgressJobs: 0,
		TotalTime:      25,
		Counters:       aggregateCounters,
	}
	result := getPerfResult(jobStatuses)
	if !reflect.DeepEqual(result, expectedResult) {
		t.Errorf("expected perf result to be : %v, but found: %v",
			expectedResult, result)
	}
}

func TestGetPerfResultJobsInProgress(t *testing.T) {
	counters1 := dcp.JobCounters{}
	counters1.Unmarshal(`{"counter-1": 20, "counter-2": 30}`)

	counters2 := dcp.JobCounters{}
	counters2.Unmarshal(`{"counter-2": 10, "counter-3": 50}`)

	counters3 := dcp.JobCounters{}
	counters3.Unmarshal(`{"counter-1": 50, "counter-2": 40}`)

	aggregateCounters := dcp.JobCounters{}
	aggregateCounters.Unmarshal(`{"counter-1": 70, "counter-2": 80, "counter-3": 50}`)

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
		&JobRunStatus{
			Status:       dcp.JobInProgress,
			CreationTime: 6,
			Counters:     counters3,
		},
	}

	expectedResult := &PerfResult{
		SucceededJobs:  1,
		FailedJobs:     1,
		InProgressJobs: 1,
		TotalTime:      0,
		Counters:       aggregateCounters,
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

	mockTicker := helpers.NewMockTicker()
	runner := PerfRunner{
		configs:    []runConfig{{id: "config-1"}, {id: "config-2"}},
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

	mockTicker := helpers.NewMockTicker()
	runner := PerfRunner{
		configs:    []runConfig{{id: "config-1"}, {id: "config-2"}},
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

	mockTicker := helpers.NewMockTicker()
	runner := PerfRunner{
		configs:    []runConfig{{id: "config-1"}, {id: "config-2"}},
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

type fakeValidator struct {
	result ValidationResult
}

func (v *fakeValidator) Validate(ctx context.Context) ValidationResult {
	return v.result
}

func newFakeValidator(result ValidationResult) Validator {
	return &fakeValidator{result: result}
}

func TestValidateResults(t *testing.T) {
	successResult := ValidationResult{Name: fakeValidatorName, Success: true, FailureMessage: "", Err: nil}
	failureResult := ValidationResult{Name: fakeValidatorName, Success: false, FailureMessage: "failed", Err: nil}
	errorResult := ValidationResult{Name: fakeValidatorName, Success: false, FailureMessage: "", Err: errFail}
	runner := PerfRunner{
		configs: []runConfig{
			{id: testConfigId, validators: []Validator{}},
			{id: testConfigId, validators: []Validator{newFakeValidator(successResult)}},
			{id: testConfigId, validators: []Validator{newFakeValidator(failureResult)}},
			{id: testConfigId, validators: []Validator{newFakeValidator(errorResult)}},
			{id: testConfigId, validators: []Validator{newFakeValidator(successResult), newFakeValidator(successResult)}},
			{
				id:         testConfigId,
				validators: []Validator{newFakeValidator(successResult), newFakeValidator(failureResult), newFakeValidator(successResult)},
			},
			{
				id:         testConfigId,
				validators: []Validator{newFakeValidator(successResult), newFakeValidator(errorResult), newFakeValidator(successResult)},
			},
		},
	}

	want := []ConfigValidationResult{
		{ConfigId: testConfigId, Success: true},
		{ConfigId: testConfigId, Success: true, Results: []ValidationResult{successResult}},
		{ConfigId: testConfigId, Success: false, Results: []ValidationResult{failureResult}},
		{ConfigId: testConfigId, Success: false, Results: []ValidationResult{errorResult}},
		{ConfigId: testConfigId, Success: true, Results: []ValidationResult{successResult, successResult}},
		{ConfigId: testConfigId, Success: false, Results: []ValidationResult{successResult, failureResult}},
		{ConfigId: testConfigId, Success: false, Results: []ValidationResult{successResult, errorResult}},
	}

	got := runner.ValidateResults(context.Background())

	if !reflect.DeepEqual(want, got) {
		t.Errorf("got %v, wanted %v", got, want)
	}
}

func TestCreateJobsAndMonitorJobsSuccess(t *testing.T) {
	tmpFile := helpers.CreateTmpFile("", "perfrunner-test-", `
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

	mockTicker := helpers.NewMockTicker()
	runner := PerfRunner{
		jobService: mockJobService,
		ticker:     mockTicker,
	}

	if errs := runner.CreateConfigs(context.Background(), tmpFile); len(errs) != 0 {
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
	mockJobService.EXPECT().GetJobStatus(runner.configs[0].id, defaultRunId).Return(
		&JobRunStatus{
			Status:       dcp.JobInProgress,
			CreationTime: 5,
		}, nil)

	mockJobService.EXPECT().GetJobStatus(runner.configs[1].id, defaultRunId).Return(
		&JobRunStatus{
			Status:       dcp.JobFailed,
			CreationTime: 6,
			FinishTime:   11,
			Counters:     counters2,
		}, nil)
	mockTicker.Tick()

	// Transition the in progress job to success.
	mockJobService.EXPECT().GetJobStatus(runner.configs[0].id, defaultRunId).Return(
		&JobRunStatus{
			Status:       dcp.JobSuccess,
			CreationTime: 5,
			FinishTime:   10,
			Counters:     counters1,
		}, nil)

	mockJobService.EXPECT().GetJobStatus(runner.configs[1].id, defaultRunId).Return(
		&JobRunStatus{
			Status:       dcp.JobFailed,
			CreationTime: 6,
			FinishTime:   11,
			Counters:     counters2,
		}, nil)
	mockTicker.Tick()

	wg.Wait()
}
