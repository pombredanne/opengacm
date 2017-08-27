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

// application is a package that controls applications,start.stop or restart
// applications.

package application

// ApplicationStatus is a wrapper with the process current status.
type ApplicationStatus struct {
	Status   string
	Restarts int
}

// SetStatus will set the process string status.
func (as *ApplicationStatus) SetStatus(status string) {
	as.Status = status
}

// AddRestart will add one restart to the process status.
func (as *ApplicationStatus) AddRestart() {
	as.Restarts++
}
