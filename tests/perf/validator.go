package perf

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"google.golang.org/api/iterator"
)

const (
	metadataValidatorName       = "metadata validator"
	deepComparisonValidatorName = "deep comparison validator"
)

type ValidationResult struct {
	Name           string
	Success        bool
	FailureMessage string
	Err            error
}

type Validator interface {
	Validate(ctx context.Context) ValidationResult
}

type metadataValidator struct {
	gcs        gcloud.GCS
	sourceDir  string
	bucketName string
}

type fileValidationFunc func(
	path, relPath string, objectMetadata *dcp.ObjectMetadata, info os.FileInfo) (bool, string, error)

func buildGCSAttrMap(ctx context.Context, gcs gcloud.GCS, bucketName string) (map[string]*storage.ObjectAttrs, error) {
	listIter := gcs.ListObjects(ctx, bucketName, nil)

	// Build a map of object name => object attributes.
	gcsAttrMap := make(map[string]*storage.ObjectAttrs)
	for {
		attrs, err := listIter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return nil, err
		}

		gcsAttrMap[attrs.Name] = attrs
	}

	return gcsAttrMap, nil
}

func gcsListingBasedValidation(ctx context.Context, gcs gcloud.GCS, sourceDir, bucketName string,
	fileValidationFunc fileValidationFunc) ValidationResult {
	result := ValidationResult{}

	// Build a map of object name => object attributes.
	// TODO: Rather than storing it all in memory, verify that Walk and list are in the same
	//       order, and then redo this to have the Walk and the GCS.List iterated in parallel.
	gcsAttrMap, err := buildGCSAttrMap(ctx, gcs, bucketName)
	if err != nil {
		result.Err = err
		return result
	}

	// Iterate source directory and check that everything is present.
	result.Success = true
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		// Bail immediately if anything is wrong.
		if err != nil {
			return err
		}

		// Skip directories.
		if info.IsDir() {
			return nil
		}

		relPath := helpers.GetRelPathOsAgnostic(sourceDir, path)

		attrs, ok := gcsAttrMap[relPath]
		if !ok {
			result.Success = false
			result.FailureMessage = fmt.Sprintf("file %s not found in GCS", relPath)
			return filepath.SkipDir
		}

		objectMetadata, err := dcp.GCSAttrToObjectMetadata(attrs)
		if err != nil {
			result.Success = false
			result.FailureMessage = fmt.Sprintf("file %s metadata in GCS failed to parse: %v", relPath, err)
			return filepath.SkipDir
		}

		success, failureMessage, err := fileValidationFunc(path, relPath, objectMetadata, info)
		if err != nil {
			return err
		}

		if !success {
			result.Success = false
			result.FailureMessage = failureMessage
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		result.Err = err
		result.Success = false
	}

	return result
}

func NewMetadataValidator(gcs gcloud.GCS, sourceDir, bucketName string) Validator {
	return &metadataValidator{gcs: gcs, sourceDir: sourceDir, bucketName: bucketName}
}

func (v *metadataValidator) Validate(ctx context.Context) ValidationResult {
	result := gcsListingBasedValidation(ctx, v.gcs, v.sourceDir, v.bucketName,
		func(path, relPath string, objectMetadata *dcp.ObjectMetadata, info os.FileInfo) (bool, string, error) {
			if objectMetadata.Size != info.Size() {
				failureMessage := fmt.Sprintf("file %s has size %d in GCS, but should be %d",
					relPath, objectMetadata.Size, info.Size())
				return false, failureMessage, nil
			} else if objectMetadata.Mtime != info.ModTime().Unix() {
				failureMessage := fmt.Sprintf("file %s has mtime %d in GCS, but should be %d",
					relPath, objectMetadata.Mtime, info.ModTime().Unix())
				return false, failureMessage, nil
			}

			return true, "", nil
		})

	result.Name = metadataValidatorName
	return result
}

type deepComparisonValidator struct {
	gcs        gcloud.GCS
	sourceDir  string
	bucketName string
}

func NewDeepComparisonValidator(gcs gcloud.GCS, sourceDir, bucketName string) Validator {
	return &deepComparisonValidator{gcs: gcs, sourceDir: sourceDir, bucketName: bucketName}
}

// readFileMD5 computes and MD5 checksum of the contents of a file,
// and returns it as a byte slice.
func readFileMD5(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := md5.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func (v *deepComparisonValidator) Validate(ctx context.Context) ValidationResult {
	result := gcsListingBasedValidation(ctx, v.gcs, v.sourceDir, v.bucketName,
		func(path, relPath string, objectMetadata *dcp.ObjectMetadata, info os.FileInfo) (bool, string, error) {
			fileMD5, err := readFileMD5(path)
			if err != nil {
				return false, "", err
			}

			if !bytes.Equal(objectMetadata.MD5, fileMD5) {
				failureMessage := fmt.Sprintf("file %s has MD5 %x in GCS, but %x locally",
					relPath, objectMetadata.MD5, fileMD5)
				return false, failureMessage, nil
			}

			return true, "", nil
		})

	result.Name = deepComparisonValidatorName
	return result
}
