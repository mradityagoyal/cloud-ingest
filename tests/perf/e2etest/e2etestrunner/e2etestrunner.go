/* Copyright 2018 Google Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// The end-to-end test runner tool generates data, runs the ecosystem, and runs the perf tool
// to validate everything is working together and procucing correct results.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/GoogleCloudPlatform/cloud-ingest/tests/perf/e2etest"
	"github.com/GoogleCloudPlatform/cloud-ingest/tests/perf/proto"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"golang.org/x/oauth2/google"
	"io/ioutil"
)

const (
	testRunsRootDir         = "test_runs"
	localBackendAPIEndpoint = "http://0.0.0.0:8080/"
	systemWarmupWaitTime    = 5 * time.Second
	defaultDatagenTemplate  = "tests/perf/e2etest/e2etestrunner/datagen-template.proto.txt"
	unsetPathTemplate       = "<UNSET>"
)

func createTestRunDir() string {
	timeFormat := time.Now().Format("2006-01-02T15-04-05")
	dir := filepath.Join(testRunsRootDir, fmt.Sprintf("test-run-%s-%d", timeFormat, os.Getpid()))
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		glog.Fatalf("Failed to create directory %s", dir)
	}

	return dir
}

// getEnvPathBin retrieves a path in an environment variable, and tacks on
// a bin folder.
func getEnvPathBin(env string) string {
	path := os.Getenv(env)
	if path == "" {
		glog.Fatalf("%s environment variable must be defined.", env)
	}

	return filepath.Join(path, "bin")
}

// redirectOutput creates stdout/stderr files in our test directory, so we have all output.
// Callers need to close all file handles created (and returned) by this function.
func redirectOutput(cmd *e2etest.CommandDescription, testRunDir string) []*os.File {
	outputDir := filepath.Join(testRunDir, cmd.Label)
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		glog.Fatalf("Failed to create directory %s", outputDir)
	}

	stdout, err := os.Create(filepath.Join(outputDir, "stdout.log"))
	if err != nil {
		glog.Fatalf("Failed to create file in directory %s.", outputDir)
	}
	stderr, err := os.Create(filepath.Join(outputDir, "stderr.log"))
	if err != nil {
		glog.Fatalf("Failed to create file in directory %s.", outputDir)
	}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return []*os.File{stdout, stderr}
}

func closeOpenFiles(files ...*os.File) {
	for _, file := range files {
		glog.Infof("Closing %s.", file.Name())
		err := file.Close()
		if err != nil {
			glog.Errorf("Failed to close %s: %v.", file.Name(), err)
		}
	}
}

// writeProto writes an arbitrary protobuf message to a file in testRunDir.
func writeProto(testRunDir, filename string, msg proto.Message) string {
	templateFilename := filepath.Join(testRunDir, filename)
	file, err := os.Create(templateFilename)
	if err != nil {
		glog.Fatalf("Failed to create %s: %v.", templateFilename, err)
	}

	err = proto.MarshalText(file, msg)
	if err != nil {
		file.Close()
		glog.Fatalf("Failed to write proto to file %s: %v.", templateFilename, err)
	}

	err = file.Close()
	if err != nil {
		glog.Fatalf("Failed to close file %s: %v.", file.Name(), err)
	}

	return templateFilename
}

// writeTestRunTemplate creates a standard test run proto file with the requested
// source directory.
func writeTestRunTemplate(testRunDir, sourceDir string) string {
	// Populate from scratch (only source dir is needed)
	config := &pb_perf.LoadTestingConfiguration{
		Name: "e2e-test",
		Config: []*pb_perf.JobConfig{
			{
				SourceDir: sourceDir,
				Validators: []pb_perf.JobConfig_TestRunValidator{
					pb_perf.JobConfig_METADATA_VALIDATOR,
					pb_perf.JobConfig_DEEP_COMPARISON_VALIDATOR,
				},
			},
		},
	}

	// Write it to our test run directory.
	return writeProto(testRunDir, "test_config_proto.txt", config)
}

// writeDataGenerationTemplate copies a data generation template
// to test dir, with the destination path changed to our test location.
func writeDataGenerationTemplate(testRunDir, datagenTemplate string) (string, string) {
	data, err := ioutil.ReadFile(datagenTemplate)
	if err != nil {
		glog.Fatalf("Failed to read template file %s: %v.", datagenTemplate, err)
	}

	config := &pb_perf.DataGeneratorConfig{}
	if err := proto.UnmarshalText(string(data), config); err != nil {
		glog.Fatalf("Failed to unmarshall datagen proto: %v.", err)
	}

	// Populate our own directory if one isn't already provided.
	dir := config.GetFileSystem().Dir
	if dir.Path == "" || dir.Path == unsetPathTemplate {
		dir.Path = filepath.Join(testRunDir, "generated-data")
	}

	sourceDir, err := filepath.Abs(dir.Path)
	if err != nil {
		glog.Fatalf("Could not get absolute path for %s: %v.", dir.Path, err)
	}

	// Write it to our test run directory
	templateFilename := writeProto(testRunDir, "data_generation_proto.txt", config)
	return sourceDir, templateFilename
}

func runSingleCommand(ctx context.Context, cmd e2etest.CommandDescription) error {
	dispatcher := e2etest.NewRunnerDispatcher()
	exitCh := make(chan e2etest.CmdExit)
	dispatcher.Dispatch(ctx, cmd, exitCh)
	result := <-exitCh
	return result.Err
}

// generateData generates perf data to be used with the test.
// We always make a copy of the template used to the test dir.
func generateData(testRunDir, datagenTemplate string) string {
	sourceDir, datagenTemplate := writeDataGenerationTemplate(testRunDir, datagenTemplate)

	datagenCmd := e2etest.CommandDescription{
		Label: "data-generator",
		Name:  filepath.Join(getEnvPathBin("GOPATH"), "datageneratortool"),
		Args:  []string{"-msg-file", datagenTemplate},
	}
	outputFiles := redirectOutput(&datagenCmd, testRunDir)
	defer closeOpenFiles(outputFiles...)

	err := runSingleCommand(context.Background(), datagenCmd)
	if err != nil {
		glog.Fatalf("Failed to generate data: %v.", err)
	}

	return sourceDir
}

func main() {
	defer glog.Flush()
	projectID := flag.String(
		"project-id", "",
		"(optional) The project id to associate with this DCP. By default, local default credentials "+
			"(gcloud application default credentials file locally, or metadata service on ComputeEngine) "+
			"are looked up to extract the project id.")
	sourceDir := flag.String(
		"source-dir", "",
		"(optional) Specify an existing source dir (relative or full path), rather than generating data.")
	preserveSourceDir := flag.Bool("preserve-source-dir", false,
		"(optional) Do not delete generated source directories we would normally delete by default.")
	datagenTemplate := flag.String(
		"datagen-template", "",
		fmt.Sprintf("(optional) Specify a location (relative or full path) for data generation "+
			"template file. Defaults to %s", defaultDatagenTemplate))
	timeout := flag.Duration(
		"timeout", 0,
		"(optional) Pass a timeout to the test execution, using duration notation (e.g. \"5m30s\"). "+
			"Defaults to 0, meaning no timeout.")
	flag.Parse()

	// Try to grab the default projectId if one is not provided.
	if *projectID == "" {
		creds, err := google.FindDefaultCredentials(context.Background())
		if err != nil {
			glog.Fatalf("No project ID provided, and failed to find default: %v.", err)
		}

		if creds.ProjectID == "" {
			glog.Fatalf("No project ID provided, and no default set in your environment.")
		}

		*projectID = creds.ProjectID
		glog.Infof("Using default projectID: %s.", *projectID)
	}

	// Setup test environment.
	testRunDir := createTestRunDir()
	glog.Infof("Created test run dir %s", testRunDir)

	// If sourceDir is provided, we're done.
	// Otherwise, use a (possibly-default) datagen template to generate
	// data and use that data as the sourceDir.
	deleteSourceDir := false
	if *sourceDir == "" {
		if *datagenTemplate == "" {
			*datagenTemplate = defaultDatagenTemplate
		}

		glog.Infof("No source dir specified; generating data.")
		*sourceDir = generateData(testRunDir, *datagenTemplate)

		// If we generated a bunch of data, make sure to clean it up later.
		if !*preserveSourceDir {
			deleteSourceDir = true
		}
	} else {
		// Expand sourceDir, to support relative paths as well.
		var err error
		*sourceDir, err = filepath.Abs(*sourceDir)
		if err != nil {
			glog.Fatalf("Could not get absolute path for %s: %v.", *sourceDir, err)
		}
	}

	testRunTemplate := writeTestRunTemplate(testRunDir, *sourceDir)
	glog.Infof("Test run template located at: %s.", testRunTemplate)

	// Setup agent command.
	agentCmd := e2etest.CommandDescription{
		Label: "agent",
		Name:  filepath.Join(getEnvPathBin("GOPATH"), "agentmain"),
		Args: []string{"-projectid", *projectID,
			"-log_dir", filepath.Join(testRunDir, "agent")},
	}

	// Setup DCP command.
	dcpCmd := e2etest.CommandDescription{
		Label: "dcp",
		Name:  filepath.Join(getEnvPathBin("GOPATH"), "dcpmain"),
		Args: []string{"-projectid", *projectID, "-disablelogprocessing",
			"-log_dir", filepath.Join(testRunDir, "dcp")},
	}

	// Setup Backend command.
	// NOTE: The relative path in the args assumes we're running this from the Makefile.
	backendCmd := e2etest.CommandDescription{
		Label: "backend",
		Name:  filepath.Join(getEnvPathBin("FULL_OPI_BACKEND_VIRTUALENV_PATH"), "python"),
		Args:  []string{"webconsole/backend/main.py", "-sdcp"},
	}

	// Setup Perftool command.
	perftoolCmd := e2etest.CommandDescription{
		Label: "perf-tool",
		Name:  filepath.Join(getEnvPathBin("GOPATH"), "perftool"),
		Args: []string{
			"-project-id", *projectID,
			"-msg-file", testRunTemplate,
			"-api-endpoint", localBackendAPIEndpoint,
			"-timeout", timeout.String(),
		},
	}

	// Have stdout/stderr go to files.
	openFiles := []*os.File{}
	openFiles = append(openFiles, redirectOutput(&agentCmd, testRunDir)...)
	openFiles = append(openFiles, redirectOutput(&dcpCmd, testRunDir)...)
	openFiles = append(openFiles, redirectOutput(&backendCmd, testRunDir)...)
	openFiles = append(openFiles, redirectOutput(&perftoolCmd, testRunDir)...)

	// Display just the perf tool's output on the console as well, so we have something to look at.
	perftoolCmd.Stdout = io.MultiWriter(perftoolCmd.Stdout, os.Stdout)
	perftoolCmd.Stderr = io.MultiWriter(perftoolCmd.Stderr, os.Stderr)

	// Run the tests and clean up after ourselves.
	err := func() error {
		defer closeOpenFiles(openFiles...)
		defer func() {
			if deleteSourceDir {
				glog.Infof("Deleting source directory %s...", *sourceDir)
				os.RemoveAll(*sourceDir)
			}
		}()

		// Run the cloud-ingest ecosystem locally.
		systemCommands := []e2etest.CommandDescription{agentCmd, dcpCmd, backendCmd}
		ctx, cancel := context.WithCancel(context.Background())
		systemRunner := e2etest.NewDispatcherSystemRunner()

		glog.Infof("Starting OPI Ecosystem...")
		systemRunner.Start(ctx, cancel, systemCommands)

		glog.Infof("Waiting %s for the components to be ready to take requests.", systemWarmupWaitTime)
		time.Sleep(systemWarmupWaitTime)

		// Kick off the test itself. Perftool should always complete.
		glog.Infof("Running end-to-end test...")
		perfErr := runSingleCommand(ctx, perftoolCmd)
		glog.Infof("Done running tests. Got result: %v.", perfErr)

		// Tear down the running ecosystem.
		glog.Infof("Shutting down OPI Ecosystem...")
		systemRunner.Stop()

		return perfErr
	}()

	glog.Infof("Test run output can be found in: %s", testRunDir)
	if err != nil {
		glog.Info("Test run FAILED.")
		os.Exit(1)
	}

	glog.Info("Test run SUCCEEDED.")
}
