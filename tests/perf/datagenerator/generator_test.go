package datagenerator

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
)

func TestNewGeneratorFromProtoFileNotExist(t *testing.T) {
	_, err := NewGeneratorFromProtoFile("does-not-exist-file")
	if err == nil {
		t.Errorf("expected parsing proto error, but found err is nil.")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected does not exist err but found %v.", err)
	}
}

func TestNewGeneratorFromProtoFileFailure(t *testing.T) {
	tmpFile := helpers.CreateTmpFile("", "generator-test-", "This is corrupted proto")
	defer os.Remove(tmpFile) // clean up
	if _, err := NewGeneratorFromProtoFile(tmpFile); err == nil {
		t.Errorf("expected parsing proto error, but found err is nil.")
	}
}

func TestGeneratorNoFileSystemInProto(t *testing.T) {
	tmpFile := helpers.CreateTmpFile("", "generator-test-", "")
	defer os.Remove(tmpFile) // clean up
	g, err := NewGeneratorFromProtoFile(tmpFile)
	if err != nil {
		t.Errorf("expected getting generator, but found err %v.", err)
	}
	errs := g.GenerateObjects()
	if len(errs) != 1 {
		t.Errorf("expected getting 1 error but got %v", errs)
	}
	expectedErr := "DataGeneratorConfig.DataSource has not implemented type"
	if !strings.Contains(errs[0].Error(), expectedErr) {
		t.Errorf("expected error %s, found %v.", expectedErr, errs[0])
	}
}

func TestGeneratorNoDistributionInProto(t *testing.T) {
	tmpFile := helpers.CreateTmpFile("", "generator-test-", `
fileSystem: {
  dir: {
    path: "path"
    totalNumberFiles:  10
  }
  maxNodesInDir: 3
}`)
	defer os.Remove(tmpFile) // clean up
	g, err := NewGeneratorFromProtoFile(tmpFile)
	if err != nil {
		t.Errorf("expected getting generator, but found err %v.", err)
	}
	errs := g.GenerateObjects()
	if len(errs) != 1 {
		t.Errorf("expected getting 1 error but got %v", errs)
	}
	expectedErr := "FileSystem.FileSizeDistribution has not implemented type"
	if !strings.Contains(errs[0].Error(), expectedErr) {
		t.Errorf("expected error %s, found %v.", expectedErr, errs[0])
	}
}

func checkDirectoryFiles(
	t *testing.T, dirPath string, stIndex int, endIndex int, fileSize int64) {
	numberFiles := endIndex - stIndex
	files, _ := ioutil.ReadDir(dirPath)
	if len(files) != numberFiles {
		t.Errorf("expected %d files but found %d", numberFiles, len(files))
	}
	for i, f := range files {
		if f.IsDir() {
			t.Errorf("not expected dir in the generated files: %v.", f)
		}
		if f.Size() != fileSize {
			t.Errorf("expected files sized of %d byt got %d", fileSize, f.Size())
		}
		expectedName := fmt.Sprintf("%s%d", filePrefix, i+stIndex)
		if f.Name() != expectedName {
			t.Errorf("expected filename %s, but found %s.", expectedName, f.Name())
		}
	}
}

func TestGeneratorSingleDir(t *testing.T) {
	tmpDir := helpers.CreateTmpDir("", "generator-test-")
	tmpFile := helpers.CreateTmpFile("", "generator-test-", fmt.Sprintf(`
fileSystem: {
  dir: {
    path: "%s"
    totalNumberFiles:  10
  }
  uniformDistribution: {
    min: 1024
    max: 1025
  }
  maxNodesInDir: 10
}`, tmpDir))

	defer os.Remove(tmpFile) // clean up
	defer os.RemoveAll(tmpDir)

	g, err := NewGeneratorFromProtoFile(tmpFile)
	if err != nil {
		t.Errorf("expected getting generator, but found err %v.", err)
	}
	errs := g.GenerateObjects()
	if len(errs) != 0 {
		t.Errorf("expected success but got %v", errs)
	}
	// Check all the files are there.
	checkDirectoryFiles(t, tmpDir, 0, 10, 1024)
}

func TestGeneratorFullTree(t *testing.T) {
	tmpDir := helpers.CreateTmpDir("", "generator-test-")
	tmpFile := helpers.CreateTmpFile("", "generator-test-", fmt.Sprintf(`
fileSystem: {
  dir: {
    path: "%s"
    totalNumberFiles:  9
  }
  uniformDistribution: {
    min: 1024
    max: 1025
  }
  maxNodesInDir: 3
}`, tmpDir))

	defer os.Remove(tmpFile) // clean up
	defer os.RemoveAll(tmpDir)

	g, _ := NewGeneratorFromProtoFile(tmpFile)
	errs := g.GenerateObjects()
	if len(errs) != 0 {
		t.Errorf("expected success but got %v", errs)
	}
	// Check all the files are there.
	dirs, _ := ioutil.ReadDir(tmpDir)
	if len(dirs) != 3 {
		t.Errorf("expected 3 dirs but found %d", len(dirs))
	}
	for i, d := range dirs {
		if !d.IsDir() {
			t.Errorf("expected dir but found: %v.", d)
		}
		dirPath := path.Join(tmpDir, d.Name())
		checkDirectoryFiles(t, dirPath, i*3, i*3+3, 1024)
	}
}

func TestGeneratorSubDirs(t *testing.T) {
	tmpDir := helpers.CreateTmpDir("", "generator-test-")
	tmpFile := helpers.CreateTmpFile("", "generator-test-", fmt.Sprintf(`
fileSystem: {
  dir: {
    path: "%s"
    totalNumberFiles:  10
    subDir: {
      path: "subdir",
      totalNumberFiles: 10
    }
  }
  uniformDistribution: {
    min: 1024
    max: 1025
  }
  maxNodesInDir: 10
}`, tmpDir))

	defer os.Remove(tmpFile) // clean up
	defer os.RemoveAll(tmpDir)

	g, err := NewGeneratorFromProtoFile(tmpFile)
	if err != nil {
		t.Errorf("expected getting generator, but found err %v.", err)
	}
	errs := g.GenerateObjects()
	if len(errs) != 0 {
		t.Errorf("expected success but got %v", errs)
	}
	// The tmpDir should contain subdir and dir-0 and dir-1.
	dirs, _ := ioutil.ReadDir(tmpDir)
	if len(dirs) != 3 {
		t.Errorf("expected 10 files but found %d", len(dirs))
	}
	for _, d := range dirs {
		if !d.IsDir() {
			t.Errorf("expected dir in %v.", d)
		}
	}
	checkDirectoryFiles(t, path.Join(tmpDir, dirPrefix+"0"), 0, 5, 1024)
	checkDirectoryFiles(t, path.Join(tmpDir, dirPrefix+"1"), 5, 10, 1024)
	checkDirectoryFiles(t, path.Join(tmpDir, "subdir"), 0, 10, 1024)
}

func TestGeneratorErrorWritingFiles(t *testing.T) {
	tmpDir := helpers.CreateTmpDir("", "generator-test-")
	tmpFile := helpers.CreateTmpFile("", "generator-test-", fmt.Sprintf(`
fileSystem: {
  dir: {
    path: "%s"
    totalNumberFiles:  10
  }
  uniformDistribution: {
    min: 1024
    max: 1025
  }
  maxNodesInDir: 10
}`, tmpDir))

	defer os.Remove(tmpFile) // clean up
	defer os.RemoveAll(tmpDir)

	// Changing the permissions of the directory to disable writing on it.
	os.Chmod(tmpDir, 0640)

	g, err := NewGeneratorFromProtoFile(tmpFile)
	if err != nil {
		t.Errorf("expected getting generator, but found err %v.", err)
	}
	errs := g.GenerateObjects()
	if len(errs) != 10 {
		t.Errorf("expected 10 failures but got %v", errs)
	}
	// Check all the files are there.
	for _, err := range errs {
		if !os.IsPermission(err) {
			t.Errorf("expected permission denied error but got %v.", err)
		}
	}
}

func TestGeneratorErrorWritingDirs(t *testing.T) {
	tmpDir := helpers.CreateTmpDir("", "generator-test-")
	tmpFile := helpers.CreateTmpFile("", "generator-test-", fmt.Sprintf(`
fileSystem: {
  dir: {
    path: "%s"
    totalNumberFiles:  27
  }
  uniformDistribution: {
    min: 1024
    max: 1025
  }
  maxNodesInDir: 3
}`, tmpDir))

	defer os.Remove(tmpFile) // clean up
	defer os.RemoveAll(tmpDir)

	// Pre-create the subdir with permissions that does.
	os.Mkdir(path.Join(tmpDir, dirPrefix+"1"), 0640)

	g, err := NewGeneratorFromProtoFile(tmpFile)
	if err != nil {
		t.Errorf("expected getting generator, but found err %v.", err)
	}
	errs := g.GenerateObjects()
	if len(errs) != 3 {
		t.Errorf("expected 1 failures but got %v", errs)
	}
	for i, err := range errs {
		expectedErr := fmt.Sprintf(
			"skipped generating %d files [%d, %d)", 3, 9+i*3, 12+i*3)
		if !strings.Contains(err.Error(), expectedErr) {
			t.Errorf(
				"expected error string %s, but found %v.", expectedErr, err.Error())
		}
	}
}

func getMaxMinHeightAndCount(
	filePath string, currH int, currMax int, currMin int) (int, int, int) {
	file, err := os.Lstat(filePath)
	if err != nil {
		log.Fatal(err)
	}
	if !file.IsDir() {
		if currH > currMax {
			currMax = currH
		}
		if currH < currMin {
			currMin = currH
		}
		return currMax, currMin, 1
	}

	files, err := ioutil.ReadDir(filePath)
	if err != nil {
		log.Fatal(err)
	}
	count := 0
	for _, f := range files {
		max, min, c := getMaxMinHeightAndCount(
			path.Join(filePath, f.Name()), currH+1, currMax, currMin)
		if max > currMax {
			currMax = max
		}
		if min < currMin {
			currMin = min
		}
		count += c
	}
	return currMax, currMin, count
}

func TestGeneratorBalancedDir(t *testing.T) {
	for i := 1; i <= 30; i++ {
		tmpDir := helpers.CreateTmpDir("", "generator-test-")
		tmpFile := helpers.CreateTmpFile("", "generator-test-", fmt.Sprintf(`
fileSystem: {
  dir: {
    path: "%s"
    totalNumberFiles:  %d
  }
  uniformDistribution: {
    min: 1024
    max: 1025
  }
  maxNodesInDir: 3
}`, tmpDir, i))

		defer os.Remove(tmpFile) // clean up
		defer os.RemoveAll(tmpDir)

		g, err := NewGeneratorFromProtoFile(tmpFile)
		if err != nil {
			t.Errorf("expected getting generator, but found err %v.", err)
		}
		errs := g.GenerateObjects()
		if len(errs) != 0 {
			t.Errorf("expected success but got %v", errs)
		}
		min, max, count := getMaxMinHeightAndCount(tmpDir, 0, 0, int(^uint(0)>>1))
		if max > min+1 {
			t.Errorf(
				"expected generation of a balanced tree files, min file heigh: %d, "+
					"max file height: %d. %s, %s", min, max, tmpDir, tmpFile)
		}
		if count != i {
			t.Errorf("expected %d number of files but found %v.", i, count)
		}
	}
}
