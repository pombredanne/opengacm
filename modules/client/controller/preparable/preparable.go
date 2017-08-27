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

// preparable is a package that through the project path to obtain the
// project, compile and export the executable program, and finally
// automatically deploy the application.

package preparable

import (
	"os/exec"
	"strings"

	"github.com/wgliang/opengacm/modules/client/controller/application"
)

type ProcPreparable interface {
	PrepareBin() ([]byte, error)
	Start() (application.ApplicationContainer, error)
	getPath() string
	Identifier() string
	getBinPath() string
	getPidPath() string
	getOutPath() string
	getErrPath() string
}

// ProcPreparable is a preparable with all the necessary informations to run
// a application. To actually run a application, call the Start() method.
type Preparable struct {
	Name       string
	SourcePath string
	Cmd        string
	SysFolder  string
	Language   string
	KeepAlive  bool
	Args       []string
}

// PrepareBin will compile the Golang project from SourcePath and populate Cmd with the proper
// command for the application to be executed.
// Returns the compile command output.
func (preparable *Preparable) PrepareBin() ([]byte, error) {
	// Remove the last character '/' if present
	if preparable.SourcePath[len(preparable.SourcePath)-1] == '/' {
		preparable.SourcePath = strings.TrimSuffix(preparable.SourcePath, "/")
	}
	cmd := ""
	cmdArgs := []string{}
	binPath := preparable.getBinPath()
	if preparable.Language == "go" {
		cmd = "go"
		cmdArgs = []string{"build", "-o", binPath, preparable.SourcePath + "/."}
	}

	preparable.Cmd = preparable.getBinPath()
	return exec.Command(cmd, cmdArgs...).Output()
}

// Start will execute the application based on the information presented on the preparable.
// This function should be called from inside the master to make sure
// all the watchers and application handling are done correctly.
// Returns a tuple with the application and an error in case there's any.
func (preparable *Preparable) Start() (application.ApplicationContainer, error) {
	proc := &application.Application{
		Name:      preparable.Name,
		Cmd:       preparable.Cmd,
		Args:      preparable.Args,
		Path:      preparable.getPath(),
		Pidfile:   preparable.getPidPath(),
		Outfile:   preparable.getOutPath(),
		Errfile:   preparable.getErrPath(),
		KeepAlive: preparable.KeepAlive,
		Status:    &application.ApplicationStatus{},
	}

	err := proc.Start()
	return proc, err
}

func (preparable *Preparable) Identifier() string {
	return preparable.Name
}

func (preparable *Preparable) getPath() string {
	if preparable.SysFolder[len(preparable.SysFolder)-1] == '/' {
		preparable.SysFolder = strings.TrimSuffix(preparable.SysFolder, "/")
	}
	return preparable.SysFolder + "/" + preparable.Name
}

func (preparable *Preparable) getBinPath() string {
	return preparable.getPath() + "/" + preparable.Name
}

func (preparable *Preparable) getPidPath() string {
	return preparable.getBinPath() + ".pid"
}

func (preparable *Preparable) getOutPath() string {
	return preparable.getBinPath() + ".out"
}

func (preparable *Preparable) getErrPath() string {
	return preparable.getBinPath() + ".err"
}
