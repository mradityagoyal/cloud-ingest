package datagenerator

import (
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	pb "github.com/GoogleCloudPlatform/cloud-ingest/tests/perf/proto"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"golang.org/x/time/rate"
)

const (
	numberThreads = 200
	maxIOPS       = 30000
	filePrefix    = "file-"
	dirPrefix     = "dir-"
)

// Generator to generate data files bases on DataGeneratorConfig proto message.
type Generator struct {
	config *pb.DataGeneratorConfig
	errs   []error
	mu     sync.Mutex // Protects errs array.

	// Holds the last generation status for getting a status update.
	lastStatus struct {
		sync.Mutex
		val string
	}
}

// NewGeneratorFromProtoFile creates a Generator based on proto message in a file.
func NewGeneratorFromProtoFile(filePath string) (*Generator, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	config := &pb.DataGeneratorConfig{}
	if err := proto.UnmarshalText(string(data), config); err != nil {
		return nil, err
	}
	return &Generator{
		config: config,
		errs:   []error{}}, nil
}

// GenerateObjects generates objects based on the generator config. Returns list
// of errors propagated during the generation.
func (g *Generator) GenerateObjects() []error {
	if fs := g.config.GetFileSystem(); fs != nil {
		return g.generateFileSystemObjects()
	}
	return []error{fmt.Errorf(
		"DataGeneratorConfig.DataSource has not implemented type %T",
		g.config.DataSource)}
}

// GetStatus gets the status string of the generation.
func (g *Generator) GetStatus() string {
	g.lastStatus.Lock()
	defer g.lastStatus.Unlock()
	return g.lastStatus.val
}

// generateFileSystemObjects generates file system objects.
func (g *Generator) generateFileSystemObjects() []error {
	distribution, err := getFileSizeDistribution(g.config.GetFileSystem())
	if err != nil {
		return []error{err}
	}

	fileGenerator := NewBytesGenerator(distribution)
	filesChan := g.generateFileSystemPaths()
	var wg sync.WaitGroup
	for i := 0; i < numberThreads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			l := rate.NewLimiter(maxIOPS/numberThreads, 1)
			for file := range filesChan {
				l.Wait(context.Background())
				if err := ioutil.WriteFile(
					file, fileGenerator.GetBytes(), 0640); err != nil {
					glog.Errorf("Failed writing file: %s, err: %v", file, err)
					g.mu.Lock()
					g.errs = append(g.errs, err)
					g.mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()

	return g.errs
}

// generateFileSystemPaths returns a channel with the file paths to generate.
// It also creates the subdirectories needed for the generation if any.
func (g *Generator) generateFileSystemPaths() <-chan string {
	c := make(chan string)
	fs := g.config.GetFileSystem()
	go func() {
		defer close(c)
		g.generateFileSystemPathsHelper(
			fs.Dir, 0, int(fs.Dir.TotalNumberFiles), int(fs.MaxNodesInDir), c)
	}()
	return c
}

// generateFileSystemPathsHelper is a recursive message helper for generating
// the file paths. It takes a directory, the start and end indices of the files
// to be generated, max number of nodes in any directory, and the channel to
// send the generated file paths to.
func (g *Generator) generateFileSystemPathsHelper(
	dir *pb.Directory, start int, end int, max int, c chan string) {
	numberFiles := end - start

	// Make sure the directory exists and make it if not.
	if err := os.MkdirAll(dir.Path, 0750); err != nil {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.errs = append(
			g.errs, fmt.Errorf("skipped generating %d files [%d, %d), err: %v",
				numberFiles, start, end, err))
		return
	}

	// Generate the nested directories first.
	for _, d := range dir.SubDir {
		d.Path = path.Join(dir.Path, d.Path)
		g.generateFileSystemPathsHelper(d, 0, int(d.TotalNumberFiles), max, c)
	}

	remainingNodes := max - len(dir.SubDir)
	if numberFiles <= remainingNodes {
		// Leaf level, it should contain only files.
		for ; start < end; start++ {
			filePath := path.Join(dir.Path, filePrefix+strconv.Itoa(start))
			c <- filePath

			// Update the last file sent for processing.
			g.lastStatus.Lock()
			g.lastStatus.val = fmt.Sprintf("Processing file: %s.", filePath)
			g.lastStatus.Unlock()
		}
		return
	}

	// Calculate the number of dirs in this dir.Path. We distributes the files
	// to the dirs evenly.
	numberDirs := int(math.Ceil(float64(numberFiles) / float64(remainingNodes)))
	// Cap the number of dirs to the number of remaining nodes in the current dir.
	if numberDirs > remainingNodes {
		numberDirs = remainingNodes
	}

	// Distribute the files in the dirs.
	filesInEachDir := numberFiles / numberDirs
	remainder := numberFiles % numberDirs

	for i := 0; i < numberDirs; i++ {
		e := start + filesInEachDir
		if remainder > 0 {
			e++
			remainder--
		}

		newDir := &pb.Directory{
			Path: path.Join(dir.Path, dirPrefix+strconv.Itoa(i)),
		}
		g.generateFileSystemPathsHelper(newDir, start, e, max, c)
		start = e
	}
}

func getFileSizeDistribution(fs *pb.FileSystem) (helpers.Distribution, error) {
	if d := fs.GetUniformDistribution(); d != nil {
		return helpers.NewUniformDistribution(int(d.Min), int(d.Max), 0), nil
	}
	return nil, fmt.Errorf(
		"FileSystem.FileSizeDistribution has not implemented type %T",
		fs.FileSizeDistribution)
}
