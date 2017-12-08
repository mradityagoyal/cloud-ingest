package perf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
)

const (
	testApiEndpoint = "test_api_endpoint/"
)

// serviceTestWrapper is wrapper to provide a mock implementation of http post
// and http get.
type serviceTestWrapper struct {
	projectId         string
	configId          string
	runId             string
	sourceDir         string
	destinationBucket string

	// Expected returned error from mock http post/get.
	returnErr error

	// Expected returned status code from mock http post/get.
	returnStatusCode int

	// Expected returned response body from mock http post/get.
	returnResponseBody string

	// testing.T object to assert that the passed urls to post/get are correct.
	t *testing.T
}

func (w serviceTestWrapper) mockHTTPPost(
	url string, contentType string, body io.Reader) (resp *http.Response, err error) {
	expectedUrl := testApiEndpoint + "projects/" + w.projectId + "/jobconfigs"
	if expectedUrl != url {
		w.t.Errorf(
			"expected to post job config url to be %s, found %s", expectedUrl, url)
	}

	expectedRequestMap := map[string]string{
		"jobConfigId":         w.configId,
		"fileSystemDirectory": w.sourceDir,
		"gcsBucket":           w.destinationBucket,
	}
	expectedRequest, _ := json.Marshal(expectedRequestMap)

	requestBodyBuf := new(bytes.Buffer)
	requestBodyBuf.ReadFrom(body)
	requestBody := requestBodyBuf.String()

	if !dcp.AreEqualJSON(requestBody, string(expectedRequest)) {
		w.t.Errorf("expected request body to be %s, found %s", string(expectedRequest), requestBody)
	}

	if w.returnErr != nil {
		return nil, w.returnErr
	}
	return &http.Response{
		StatusCode: w.returnStatusCode,
		Body:       dcp.NewStringReadCloser(w.returnResponseBody),
	}, nil
}

func (w serviceTestWrapper) mockHTTPGet(url string) (resp *http.Response, err error) {
	expectedUrl := testApiEndpoint + "projects/" + w.projectId +
		"/jobrun/" + w.configId
	if expectedUrl != url {
		w.t.Errorf(
			"expected to get job status url to be %s, found %s", expectedUrl, url)
	}

	if w.returnErr != nil {
		return nil, w.returnErr
	}
	return &http.Response{
		StatusCode: w.returnStatusCode,
		Body:       dcp.NewStringReadCloser(w.returnResponseBody),
	}, nil
}

func TestCreateJobConfigRequestError(t *testing.T) {
	w := serviceTestWrapper{
		projectId:         "dummy-project",
		configId:          "dummy-config",
		sourceDir:         "dummy-source",
		destinationBucket: "dummy-bucket",
		returnErr:         fmt.Errorf("cannot complete the request"),
		t:                 t,
	}

	j := IngestService{
		projectId:   w.projectId,
		apiEndpoint: testApiEndpoint,
		httpPostFn:  w.mockHTTPPost,
	}
	err := j.CreateJobConfig(w.configId, w.sourceDir, w.destinationBucket)
	if err == nil {
		t.Errorf("expecting error on creating job config, but found nil")
	}
	if !strings.Contains(err.Error(), w.returnErr.Error()) {
		t.Errorf("expecting returned error \"%s\" to contain \"%s\".",
			err.Error(), w.returnErr.Error())
	}
}

func TestCreateJobConfigNotCreatedStatusCode(t *testing.T) {
	w := serviceTestWrapper{
		projectId:         "dummy-project",
		configId:          "dummy-config",
		sourceDir:         "dummy-source",
		destinationBucket: "dummy-bucket",
		returnStatusCode:  http.StatusConflict,
		t:                 t,
	}

	j := IngestService{
		projectId:   w.projectId,
		apiEndpoint: testApiEndpoint,
		httpPostFn:  w.mockHTTPPost,
	}
	err := j.CreateJobConfig(w.configId, w.sourceDir, w.destinationBucket)
	if err == nil {
		t.Errorf("expecting error with conflict status code, but found nil")
	}

	if !strings.Contains(err.Error(), "unexpected response code for request") {
		t.Errorf("expecting returned error \"%s\" to contain "+
			"\"unexpected response code for request\".", err.Error())
	}
}

func TestCreateJobConfigSuccess(t *testing.T) {
	w := serviceTestWrapper{
		projectId:         "dummy-project",
		configId:          "dummy-config",
		sourceDir:         "dummy-source",
		destinationBucket: "dummy-bucket",
		returnStatusCode:  http.StatusOK,
		t:                 t,
	}

	j := IngestService{
		projectId:   w.projectId,
		apiEndpoint: testApiEndpoint,
		httpPostFn:  w.mockHTTPPost,
	}
	if err := j.CreateJobConfig(
		w.configId, w.sourceDir, w.destinationBucket); err != nil {
		t.Errorf(
			"expecting creating job config to succeed but got an error: %v", err)
	}
}

func TestGetJobStatusRequestError(t *testing.T) {
	w := serviceTestWrapper{
		projectId: "dummy-project",
		configId:  "dummy-config",
		runId:     "dummy-run",
		returnErr: fmt.Errorf("cannot complete the request"),
		t:         t,
	}

	j := IngestService{
		projectId:   w.projectId,
		apiEndpoint: testApiEndpoint,
		httpGetFn:   w.mockHTTPGet,
	}
	_, err := j.GetJobStatus(w.configId, w.runId)
	if err == nil {
		t.Errorf("expecting error on getting job status, but found nil")
	}
	if !strings.Contains(err.Error(), w.returnErr.Error()) {
		t.Errorf("expecting returned error \"%s\" to contain \"%s\".",
			err.Error(), w.returnErr.Error())
	}
}

func TestGetJobStatusNotOKStatusCode(t *testing.T) {
	w := serviceTestWrapper{
		projectId:        "dummy-project",
		configId:         "dummy-config",
		runId:            "dummy-run",
		returnStatusCode: http.StatusNotFound,
		t:                t,
	}

	j := IngestService{
		projectId:   w.projectId,
		apiEndpoint: testApiEndpoint,
		httpGetFn:   w.mockHTTPGet,
	}
	_, err := j.GetJobStatus(w.configId, w.runId)
	if err == nil {
		t.Errorf("expecting error with not found status code, but found nil")
	}

	if !strings.Contains(err.Error(), "unexpected response code for request") {
		t.Errorf("expecting returned error \"%s\" to contain "+
			"\"unexpected response code for request\".", err.Error())
	}
}

func TestGetJobStatusErrorDecodingResponse(t *testing.T) {
	responseString := "Not a json format response"
	w := serviceTestWrapper{
		projectId:          "dummy-project",
		configId:           "dummy-config",
		runId:              "dummy-run",
		returnStatusCode:   http.StatusOK,
		returnResponseBody: responseString,
		t:                  t,
	}

	j := IngestService{
		projectId:   w.projectId,
		apiEndpoint: testApiEndpoint,
		httpGetFn:   w.mockHTTPGet,
	}

	_, err := j.GetJobStatus(w.configId, w.runId)
	if err == nil {
		t.Errorf("expecting error with not found status code, but found nil")
	}

	if !strings.Contains(err.Error(), "failed to decode response") {
		t.Errorf("expecting returned error \"%s\" to contain "+
			"\"failed to decode response\".", err.Error())
	}
}

func TestGetJobStatusSuccess(t *testing.T) {
	responseString := `
{
  "Counters": {
    "counter-1": 20,
    "counter-2": 30
  },
  "JobCreationTime": 5,
  "JobFinishTime": 10,
  "Status": 3
}`
	w := serviceTestWrapper{
		projectId:          "dummy-project",
		configId:           "dummy-config",
		runId:              "dummy-run",
		returnStatusCode:   http.StatusOK,
		returnResponseBody: responseString,
		t:                  t,
	}

	j := IngestService{
		projectId:   w.projectId,
		apiEndpoint: testApiEndpoint,
		httpGetFn:   w.mockHTTPGet,
	}
	job, err := j.GetJobStatus(w.configId, w.runId)
	if err != nil {
		t.Errorf(
			"expecting getting job status to succeed but got an error: %v", err)
	}
	if job.Status != 3 {
		t.Errorf("expecting job status to be 3, but found %v", job.Status)
	}
	if job.CreationTime != 5 {
		t.Errorf("expecting creation time to be 5, but found %v", job.CreationTime)
	}
	if job.FinishTime != 10 {
		t.Errorf("expecting creation time to be 10, but found %v", job.FinishTime)
	}
	expectedCounters := dcp.JobCounters{}
	expectedCounters.Unmarshal(`{"counter-1": 20, "counter-2": 30}`)
	if !reflect.DeepEqual(expectedCounters, job.Counters) {
		t.Errorf("expected counters to be %v, but found %v.", expectedCounters, job.Counters)
	}
}
