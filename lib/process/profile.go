/*
Copyright 2018 Gravitational, Inc.

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

package process

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sync/atomic"
	"time"

	"github.com/gravitational/gravity/lib/defaults"

	log "github.com/sirupsen/logrus"
	"github.com/gravitational/trace"
)

var profilingStarted int32

// StartProfiling starts profiling endpoint, will return AlreadyExists
// if profiling has been initiated
func StartProfiling(ctx context.Context, httpEndpoint, profileDir string) error {
	if !atomic.CompareAndSwapInt32(&profilingStarted, 0, 1) {
		return trace.AlreadyExists("profiling has been already started")
	}

	log.Infof("[PROFILING] http %v, profiles in %v", httpEndpoint, profileDir)

	go func() {
		log.Println(http.ListenAndServe(httpEndpoint, nil))
	}()

	if profileDir == "" {
		return nil
	}

	profileDir = filepath.Join(profileDir, fmt.Sprintf("%v", os.Getpid()))
	if err := os.MkdirAll(profileDir, defaults.SharedDirMask); err != nil {
		return trace.Wrap(err, "failed to create directory %v", profileDir)
	}

	log.Infof("setting up periodic profile dumps in %v", profileDir)
	go func() {
		ticker := time.NewTicker(defaults.ProfilingInterval)
		for {
			select {
			case <-ticker.C:
				f, err := ioutil.TempFile(profileDir, "stacks")
				if err == nil {
					err = pprof.Lookup("goroutine").WriteTo(f, 1)
					if err != nil {
						log.Errorf("failed to dump goroutines: %v", trace.DebugReport(err))
					}
					f.Close()
				}
				f, err = ioutil.TempFile(profileDir, "heap")
				if err == nil {
					err = pprof.WriteHeapProfile(f)
					if err != nil {
						log.Errorf("failed to dump heap: %v", trace.DebugReport(err))
					}
					f.Close()
				}
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	return nil
}
