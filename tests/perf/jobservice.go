package perf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
)

// JobRunStatus represents a job run status.
type JobRunStatus struct {
	Status       int64
	CreationTime int64
	FinishTime   int64
	Counters     dcp.JobCounters
}

// JobService provides an interface to create and get statuses of transfer jobs.
type JobService interface {
	// CreateJobConfig creates a transfer job config.
	CreateJobConfig(configId string, sourceDir string, destinationBucket string) error

	// GetJobStatus gets a transfer job run status.
	GetJobStatus(configId string, runId string) (*JobRunStatus, error)
}

// IngestService implements JobService interface for a real service implementation.
type IngestService struct {
	projectId string

	// Webconsole backend API endpoint to hit for all requests.
	apiEndpoint string

	// Defined to distinguish between actual http post and fake http post in unit
	// tests.
	httpPostFn func(url string, contentType string, body io.Reader) (
		resp *http.Response, err error)

	// Defined to distinguish between actual http get and fake http get in unit
	// tests.
	httpGetFn func(url string) (resp *http.Response, err error)
}

// NewIngestService creates a new IngestService based on projectId and
// http.Client object.
func NewIngestService(projectId, apiEndpoint string, httpClient *http.Client) *IngestService {
	return &IngestService{
		projectId:   projectId,
		apiEndpoint: apiEndpoint,
		httpPostFn:  httpClient.Post,
		httpGetFn:   httpClient.Get,
	}
}

func (s IngestService) CreateJobConfig(
	configId string, sourceDir string, destinationBucket string) error {
	url := s.apiEndpoint + path.Join("projects", s.projectId, "jobconfigs")
	requestBody := map[string]string{
		"jobConfigId":         configId,
		"fileSystemDirectory": sourceDir,
		"gcsBucket":           destinationBucket,
	}
	requestJson, _ := json.Marshal(requestBody)
	res, err := s.httpPostFn(
		url, "application/json", bytes.NewBuffer(requestJson))
	if err != nil {
		return fmt.Errorf(
			"error in request: %s, with request body: %v, err: %v",
			url, requestBody, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf(
			"unexpected response code for request: %s, with request body: %v, "+
				"return code: %d, response body: %s",
			url, requestBody, res.StatusCode, string(body))
	}
	return nil
}

func (s IngestService) GetJobStatus(
	configId string, runId string) (*JobRunStatus, error) {
	url := s.apiEndpoint + path.Join("projects", s.projectId, "jobrun", configId)
	res, err := s.httpGetFn(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(res.Body)
		err := fmt.Errorf(
			"unexpected response code for request: %s, return code: %d, response body: %s",
			url, res.StatusCode, string(body))
		return nil, err
	}

	resJson := make(map[string]interface{})
	d := json.NewDecoder(res.Body)
	d.UseNumber()
	if err := d.Decode(&resJson); err != nil {
		return nil, fmt.Errorf(
			"failed to decode response %v, with err: %v.", res.Body, err)
	}

	jStatus, _ := resJson["Status"].(json.Number).Int64()
	jCreationTime, _ := resJson["JobCreationTime"].(json.Number).Int64()
	var jFinishTime int64
	if resJson["JobFinishTime"] != nil {
		jFinishTime, _ = resJson["JobFinishTime"].(json.Number).Int64()
	}

	b, _ := json.Marshal(resJson["Counters"])
	var jCounters dcp.JobCounters
	jCounters.Unmarshal(string(b))

	return &JobRunStatus{
		Status:       jStatus,
		CreationTime: jCreationTime,
		FinishTime:   jFinishTime,
		Counters:     jCounters,
	}, nil
}
