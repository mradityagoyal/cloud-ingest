package profile

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/golang/glog"
)

const (
	heapProfile = "heap"
	profileDir  = "profiles"
)

// ContinuouslyRecord will write the specified profiles (either heap, CPU or both) to a profileDir
// created under the given logDir every profileFreq seconds. Profiling continues until the given
// context is cancelled.
func ContinuouslyRecord(ctx context.Context, logDir string, heap, cpu bool, profileFreq time.Duration) error {
	profiler := profiler{
		profileCPU: cpu,
		frequency:  profileFreq,
		logDir:     path.Join(logDir, profileDir),
	}
	if heap {
		profiler.profiles = []string{heapProfile}
	}

	return profiler.startProfiling(ctx)
}

type profiler struct {
	profileCPU bool
	profiles   []string

	frequency time.Duration // ContinuouslyRecord frequency in seconds
	logDir    string

	cpuTmpFile *os.File
}

func (p *profiler) startProfiling(ctx context.Context) error {
	if err := createProfileDirIfNotExist(p.logDir); err != nil {
		return fmt.Errorf("failed to create profile dir %s: %v", p.logDir, err)
	}
	if p.profileCPU {
		if err := p.startCPUProfile(); err != nil {
			return fmt.Errorf("failed to start the cpu profile: %v", err)
		}
	}

	t := time.NewTicker(p.frequency)
	for {
		select {
		case <-t.C:
			if err := p.emitProfile(); err != nil {
				return fmt.Errorf("failed to emit usage profile")
			}
		case <-ctx.Done():
			t.Stop()
			glog.Infof("Stopping profiler...")
			if p.profileCPU {
				if err := p.stopCPUProfile(); err != nil {
					return fmt.Errorf("failed to stop the cpu profile: %v", err)
				}
			}
			return nil
		}
	}
}

func createProfileDirIfNotExist(profileDir string) error {
	fd, err := os.Stat(profileDir)

	if os.IsNotExist(err) {
		if err2 := os.MkdirAll(profileDir, 0777); err2 != nil {
			return err2
		}
	} else if err != nil {
		return err
	} else if fd.Mode().IsRegular() {
		return fmt.Errorf("profile dir %s is a file", profileDir)
	}
	return nil
}

func (p *profiler) startCPUProfile() error {
	var err error
	p.cpuTmpFile, err = ioutil.TempFile("", "agent-cpu-profile")
	if err != nil {
		return err
	}
	err = pprof.StartCPUProfile(p.cpuTmpFile)
	return err
}

func (p *profiler) stopCPUProfile() error {
	pprof.StopCPUProfile()
	return p.cpuTmpFile.Close()
}

// emitProfile stops the CPU profiling and dumps all profiles tracked by profiler into files in
// profileDir, with the current timestamp included in the filenames. It then restarts
// CPU profiling.
func (p *profiler) emitProfile() error {
	suffix := time.Now().Format("2006-01-02-15-04-05.000")

	// Emit the CPU profile.
	if p.profileCPU {
		cpuFileName := filepath.Join(p.logDir, fmt.Sprintf("profile.cpu.%s", suffix))
		if err := p.stopCPUProfile(); err != nil {
			return fmt.Errorf("failed to stop the cpu profile: %v", err)
		}
		if err := os.Rename(p.cpuTmpFile.Name(), cpuFileName); err != nil {
			return err
		}
	}

	for _, profile := range p.profiles {
		fileName := filepath.Join(p.logDir, fmt.Sprintf("profile.%s.%s", profile, suffix))
		if err := dumpProfile(fileName, profile); err != nil {
			return err
		}
	}

	// Restart the CPU profile.
	return p.startCPUProfile()
}

// dumpProfile writes a dump of an arbitrary profile to the file specified by the given filename.
func dumpProfile(filename, profile string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return pprof.Lookup(profile).WriteTo(f, 1)
}
