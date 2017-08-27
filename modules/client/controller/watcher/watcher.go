// Copyright (c) wgliang 2017. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// watcher is a package that daemon applications, when the state of
// some application becomes stop, watcher will send a signal to start
// the application right now.

package watcher

import (
	"os"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/wgliang/opengacm/modules/client/controller/application"
)

// ApplicationStatus is a wrapper with the application state and an error in case there's any.
type ApplicationStatus struct {
	state *os.ProcessState
	err   error
}

// ApplicationWatcher is a wrapper that act as a object that watches a application.
type ApplicationWatcher struct {
	applicationStatus chan *ApplicationStatus
	application       application.ApplicationContainer
	stopWatcher       chan bool
}

// Watcher is responsible for watching a list of processes and report to Master in
// case the application dies at some point.
type Watcher struct {
	sync.Mutex
	restartApplication chan application.ApplicationContainer
	watchApplications  map[string]*ApplicationWatcher
}

// InitWatcher will create a Watcher instance.
// Returns a Watcher instance.
func InitWatcher() *Watcher {
	watcher := &Watcher{
		restartApplication: make(chan application.ApplicationContainer),
		watchApplications:  make(map[string]*ApplicationWatcher),
	}
	return watcher
}

// RestartApplication is a wrapper to export the channel restartProc. It basically keeps track of
// all the applications that died and need to be restarted.
// Returns a channel with the dead applications that need to be restarted.
func (watcher *Watcher) RestartApplication() chan application.ApplicationContainer {
	return watcher.restartApplication
}

// AddApplicationWatcher will add a watcher on application.
func (watcher *Watcher) AddApplicationWatcher(application application.ApplicationContainer) {
	watcher.Lock()
	defer watcher.Unlock()
	if _, ok := watcher.watchApplications[application.Identifier()]; ok {
		log.Warnf("A watcher for this application already exists.")
		return
	}
	applicationWatcher := &ApplicationWatcher{
		applicationStatus: make(chan *ApplicationStatus, 1),
		application:       application,
		stopWatcher:       make(chan bool, 1),
	}
	watcher.watchApplications[application.Identifier()] = applicationWatcher
	go func() {
		log.Infof("Starting watcher on application %s", application.Identifier())
		state, err := application.Watch()
		applicationWatcher.applicationStatus <- &ApplicationStatus{
			state: state,
			err:   err,
		}
	}()
	go func() {
		defer delete(watcher.watchApplications, applicationWatcher.application.Identifier())
		select {
		case applicationStatus := <-applicationWatcher.applicationStatus:
			log.Infof("Application %s is dead, advising master...", applicationWatcher.application.Identifier())
			log.Infof("State is %s", applicationStatus.state.String())
			watcher.restartApplication <- applicationWatcher.application
			break
		case <-applicationWatcher.stopWatcher:
			break
		}
	}()
}

// StopWatcher will stop a running watcher on a application with identifier 'identifier'
// Returns a channel that will be populated when the watcher is finally done.
func (watcher *Watcher) StopWatcher(identifier string) chan bool {
	if watcher, ok := watcher.watchApplications[identifier]; ok {
		log.Infof("Stopping watcher on application %s", identifier)
		watcher.stopWatcher <- true
		waitStop := make(chan bool, 1)
		go func() {
			<-watcher.applicationStatus
			waitStop <- true
		}()
		return waitStop
	}
	return nil
}
