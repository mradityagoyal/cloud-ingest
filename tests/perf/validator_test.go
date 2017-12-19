package perf

import (
	"context"
	"crypto/md5"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/mock/gomock"
)

const (
	// By convention, "fileN" will be N bytes.
	file1Contents = "1"
	file2Contents = "22"
	file3Contents = "333"
	file4Contents = "4444"

	bucketName = "bucket-name"
)

var (
	// Stick them into the past, at different times.
	file1Time = time.Now().Add(-1 * 24 * time.Hour)
	file2Time = time.Now().Add(-2 * 24 * time.Hour)
	file3Time = time.Now().Add(-3 * 24 * time.Hour)
	file4Time = time.Now().Add(-4 * 24 * time.Hour)

	// MD5s for the file contents.
	file1MD5 = localMD5(file1Contents)
	file2MD5 = localMD5(file2Contents)
	file3MD5 = localMD5(file3Contents)
	file4MD5 = localMD5(file4Contents)
)

func localMD5(content string) []byte {
	result := md5.Sum([]byte(content))
	return result[:]
}

// createTestFileFarm returns a standard bunch of files we'll use in all these tests.
// Until there's a need for customizability, this just returns 4 files in order.
// It's the caller's responsibility to delete this when done.
func createTestFileFarm() (string, string, string, string, string) {
	// Top level directory.
	dir, err := ioutil.TempDir("", "validator-test")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v\n", err)
	}

	// Subdirectory for some of the files.
	subdir, err := ioutil.TempDir(dir, "subdir")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v\n", err)
	}

	// Create files.
	file1 := helpers.CreateTmpFile(dir, "test-file-1", file1Contents)
	file2 := helpers.CreateTmpFile(dir, "test-file-2", file2Contents)
	file3 := helpers.CreateTmpFile(subdir, "test-file-3", file3Contents)
	file4 := helpers.CreateTmpFile(subdir, "test-file-4", file4Contents)

	// Set times.
	os.Chtimes(file1, file1Time, file1Time)
	os.Chtimes(file2, file2Time, file2Time)
	os.Chtimes(file3, file3Time, file3Time)
	os.Chtimes(file4, file4Time, file4Time)

	return dir,
		helpers.GetRelPathOsAgnostic(dir, file1),
		helpers.GetRelPathOsAgnostic(dir, file2),
		helpers.GetRelPathOsAgnostic(dir, file3),
		helpers.GetRelPathOsAgnostic(dir, file4)
}

func deleteTestFileFarm(dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		log.Fatalf("Failed to delete directory %s, with error: %v", dir, err)
	}
}

func newObjectAttrs(name string, size int64, time time.Time, md5 []byte) *storage.ObjectAttrs {
	return &storage.ObjectAttrs{
		Name: name,
		Size: size,
		MD5:  md5,
		Metadata: map[string]string{
			dcp.MTIME_ATTR_NAME: strconv.FormatInt(time.Unix(), 10),
		},
	}
}

func TestMetadataValidator_GCSError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedErr := errors.New("gcs error occurred")
	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().ListObjects(gomock.Any(), bucketName, nil).Return(gcloud.NewObjectIterator(expectedErr))

	validator := NewMetadataValidator(mockGCS, "", bucketName)
	got := validator.Validate(context.Background())

	want := ValidationResult{
		Name: metadataValidatorName,
		Err:  expectedErr,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestMetadataValidator_FileReadError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().ListObjects(gomock.Any(), bucketName, nil).Return(gcloud.NewObjectIterator())

	validator := NewMetadataValidator(mockGCS, "~~~", bucketName)
	got := validator.Validate(context.Background())

	if got.Name != metadataValidatorName {
		t.Errorf("wanted name %s, but got %s", metadataValidatorName, got.Name)
	}

	if got.Err == nil {
		t.Error("wanted an error, but got none")
	}
}

func TestMetadataValidator_Success(t *testing.T) {
	dir, file1, file2, file3, file4 := createTestFileFarm()
	defer deleteTestFileFarm(dir)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().ListObjects(gomock.Any(), bucketName, nil).Return(gcloud.NewObjectIterator(
		newObjectAttrs(file1, 1, file1Time, file1MD5),
		newObjectAttrs(file2, 2, file2Time, file2MD5),
		newObjectAttrs(file3, 3, file3Time, file3MD5),
		newObjectAttrs(file4, 4, file4Time, file4MD5),
		// Extra entries don't matter; just that every file in the directory is up to date.
		newObjectAttrs("misc_extra", 1234, time.Now(), file1MD5),
	))

	validator := NewMetadataValidator(mockGCS, dir, bucketName)
	got := validator.Validate(context.Background())

	want := ValidationResult{
		Name:    metadataValidatorName,
		Success: true,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestMetadataValidator_MissingFileFailure(t *testing.T) {
	dir, file1, file2, _, file4 := createTestFileFarm()
	defer deleteTestFileFarm(dir)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().ListObjects(gomock.Any(), bucketName, nil).Return(gcloud.NewObjectIterator(
		newObjectAttrs(file1, 1, file1Time, file1MD5),
		newObjectAttrs(file2, 2, file2Time, file2MD5),
		// file3 is missing.
		newObjectAttrs(file4, 4, file4Time, file4MD5),
	))

	validator := NewMetadataValidator(mockGCS, dir, bucketName)
	got := validator.Validate(context.Background())

	want := ValidationResult{
		Name:    metadataValidatorName,
		Success: false,
	}

	if got.FailureMessage == "" {
		t.Error("expected failure message but no message was set")
	}

	got.FailureMessage = ""
	if !reflect.DeepEqual(want, got) {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestMetadataValidator_FileWrongSizeFailure(t *testing.T) {
	dir, file1, file2, file3, file4 := createTestFileFarm()
	defer deleteTestFileFarm(dir)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().ListObjects(gomock.Any(), bucketName, nil).Return(gcloud.NewObjectIterator(
		newObjectAttrs(file1, 1, file1Time, file1MD5),
		newObjectAttrs(file2, 2, file2Time, file2MD5),
		// file3 has the wrong size, so metadata doesn't match.
		newObjectAttrs(file3, 2, file3Time, file3MD5),
		newObjectAttrs(file4, 4, file4Time, file4MD5),
	))

	validator := NewMetadataValidator(mockGCS, dir, bucketName)
	got := validator.Validate(context.Background())

	want := ValidationResult{
		Name:    metadataValidatorName,
		Success: false,
	}

	if got.FailureMessage == "" {
		t.Error("expected failure message but no message was set")
	}

	got.FailureMessage = ""
	if !reflect.DeepEqual(want, got) {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestMetadataValidator_FileMissingTimeFailure(t *testing.T) {
	dir, file1, file2, file3, file4 := createTestFileFarm()
	defer deleteTestFileFarm(dir)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().ListObjects(gomock.Any(), bucketName, nil).Return(gcloud.NewObjectIterator(
		newObjectAttrs(file1, 1, file1Time, file1MD5),
		newObjectAttrs(file2, 2, file2Time, file2MD5),
		// file3 has no timestamp data set.
		&storage.ObjectAttrs{Name: file3, Size: 3},
		newObjectAttrs(file4, 4, file4Time, file4MD5),
	))

	validator := NewMetadataValidator(mockGCS, dir, bucketName)
	got := validator.Validate(context.Background())

	want := ValidationResult{
		Name:    metadataValidatorName,
		Success: false,
	}

	if got.FailureMessage == "" {
		t.Error("expected failure message but no message was set")
	}

	got.FailureMessage = ""
	if !reflect.DeepEqual(want, got) {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestMetadataValidator_FileBadMetadataFailure(t *testing.T) {
	dir, file1, file2, file3, file4 := createTestFileFarm()
	defer deleteTestFileFarm(dir)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().ListObjects(gomock.Any(), bucketName, nil).Return(gcloud.NewObjectIterator(
		newObjectAttrs(file1, 1, file1Time, file1MD5),
		newObjectAttrs(file2, 2, file2Time, file2MD5),
		// file3 has no timestamp data set.
		&storage.ObjectAttrs{Name: file3, Size: 3, Metadata: map[string]string{dcp.MTIME_ATTR_NAME: "b0rken"}},
		newObjectAttrs(file4, 4, file4Time, file4MD5),
	))

	validator := NewMetadataValidator(mockGCS, dir, bucketName)
	got := validator.Validate(context.Background())

	want := ValidationResult{
		Name:    metadataValidatorName,
		Success: false,
	}

	if got.FailureMessage == "" {
		t.Error("expected failure message but no message was set")
	}

	got.FailureMessage = ""
	if !reflect.DeepEqual(want, got) {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestMetadataValidator_FileWrongTimeFailure(t *testing.T) {
	dir, file1, file2, file3, file4 := createTestFileFarm()
	defer deleteTestFileFarm(dir)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().ListObjects(gomock.Any(), bucketName, nil).Return(gcloud.NewObjectIterator(
		newObjectAttrs(file1, 1, file1Time, file1MD5),
		newObjectAttrs(file2, 2, file2Time, file2MD5),
		// file3 has the wrong timestamp, so metadata doesn't match.
		newObjectAttrs(file3, 3, file3Time.Add(-1*time.Hour), file3MD5),
		newObjectAttrs(file4, 4, file4Time, file4MD5),
	))

	validator := NewMetadataValidator(mockGCS, dir, bucketName)
	got := validator.Validate(context.Background())

	want := ValidationResult{
		Name:    metadataValidatorName,
		Success: false,
	}

	if got.FailureMessage == "" {
		t.Error("expected failure message but no message was set")
	}

	got.FailureMessage = ""
	if !reflect.DeepEqual(want, got) {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestDeepComparisonValidator_GCSError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedErr := errors.New("gcs error occurred")
	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().ListObjects(gomock.Any(), bucketName, nil).Return(gcloud.NewObjectIterator(expectedErr))

	validator := NewDeepComparisonValidator(mockGCS, "", bucketName)
	got := validator.Validate(context.Background())

	want := ValidationResult{
		Name: deepComparisonValidatorName,
		Err:  expectedErr,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestDeepComparisonValidator_FileReadError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().ListObjects(gomock.Any(), bucketName, nil).Return(gcloud.NewObjectIterator())

	validator := NewDeepComparisonValidator(mockGCS, "~~~", bucketName)
	got := validator.Validate(context.Background())

	if got.Name != deepComparisonValidatorName {
		t.Errorf("wanted name %s, but got %s", deepComparisonValidatorName, got.Name)
	}

	if got.Err == nil {
		t.Error("wanted an error, but got none")
	}
}

func TestDeepComparisonValidator_Success(t *testing.T) {
	dir, file1, file2, file3, file4 := createTestFileFarm()
	defer deleteTestFileFarm(dir)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().ListObjects(gomock.Any(), bucketName, nil).Return(gcloud.NewObjectIterator(
		newObjectAttrs(file1, 1, file1Time, file1MD5),
		newObjectAttrs(file2, 2, file2Time, file2MD5),
		newObjectAttrs(file3, 3, file3Time, file3MD5),
		newObjectAttrs(file4, 4, file4Time, file4MD5),
		// Extra entries don't matter; just that every file in the directory is up to date.
		newObjectAttrs("misc_extra", 1234, time.Now(), file1MD5),
	))

	validator := NewDeepComparisonValidator(mockGCS, dir, bucketName)
	got := validator.Validate(context.Background())

	want := ValidationResult{
		Name:    deepComparisonValidatorName,
		Success: true,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestDeepComparisonValidator_MissingFileFailure(t *testing.T) {
	dir, file1, file2, _, file4 := createTestFileFarm()
	defer deleteTestFileFarm(dir)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().ListObjects(gomock.Any(), bucketName, nil).Return(gcloud.NewObjectIterator(
		newObjectAttrs(file1, 1, file1Time, file1MD5),
		newObjectAttrs(file2, 2, file2Time, file2MD5),
		// file3 is missing.
		newObjectAttrs(file4, 4, file4Time, file4MD5),
	))

	validator := NewDeepComparisonValidator(mockGCS, dir, bucketName)
	got := validator.Validate(context.Background())

	want := ValidationResult{
		Name:    deepComparisonValidatorName,
		Success: false,
	}

	if got.FailureMessage == "" {
		t.Error("expected failure message but no message was set")
	}

	got.FailureMessage = ""
	if !reflect.DeepEqual(want, got) {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestDeepComparisonValidator_WrongMD5Failure(t *testing.T) {
	dir, file1, file2, file3, file4 := createTestFileFarm()
	defer deleteTestFileFarm(dir)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().ListObjects(gomock.Any(), bucketName, nil).Return(gcloud.NewObjectIterator(
		newObjectAttrs(file1, 1, file1Time, file1MD5),
		newObjectAttrs(file2, 2, file2Time, file2MD5),
		// file3 now has file1's MD5
		newObjectAttrs(file3, 3, file3Time, file1MD5),
		newObjectAttrs(file4, 4, file4Time, file4MD5),
	))

	validator := NewDeepComparisonValidator(mockGCS, dir, bucketName)
	got := validator.Validate(context.Background())

	want := ValidationResult{
		Name:    deepComparisonValidatorName,
		Success: false,
	}

	if got.FailureMessage == "" {
		t.Error("expected failure message but no message was set")
	}

	got.FailureMessage = ""
	if !reflect.DeepEqual(want, got) {
		t.Errorf("wanted %v, got %v", want, got)
	}
}
