package daemon

import (
	"fmt"
	"net"
	"net/rpc"
	"time"

	"github.com/wgliang/opengacm/modules/client/controller/application"
)

// RemoteDaemon is a struct that holds the daemon instance.
type RemoteDaemon struct {
	daemon *Daemon // Daemon instance
}

// RemoteClient is a struct that holds the remote client instance.
type RemoteClient struct {
	conn *rpc.Client // RpcConnection for the remote client.
}

// GoApplication is a struct that represents the necessary arguments for a go binary to be built.
type GoApplication struct {
	SourcePath string   // SourcePath is the package path. (Ex: github.com/topfreegames/apm)
	Name       string   // Name is the application name that will be given to the application.
	KeepAlive  bool     // KeepAlive will determine whether APM should keep the proc live or not.
	Args       []string // Args is an array containing all the extra args that will be passed to the binary after compilation.
}

type ApplicationDataResponse struct {
	Name      string
	Pid       int
	Status    *application.ApplicationStatus
	KeepAlive bool
}

type ApplicationResponse struct {
	Applications []*ApplicationDataResponse
}

// Save will save the current running and stopped processes onto a file.
// Returns an error in case there's any.
func (rd *RemoteDaemon) Save(req string, ack *bool) error {
	req = ""
	*ack = true
	return rd.daemon.SaveApplications()
}

// Resurrect will restore all previously save processes.
// Returns an error in case there's any.
func (rd *RemoteDaemon) Resurrect(req string, ack *bool) error {
	req = ""
	*ack = true
	return rd.daemon.Revive()
}

// StartGoBin will build a binary based on the arguments passed on goApplication, then it will start the application
// and keep it alive if KeepAlive is set to true.
// It returns an error and binds true to ack pointer.
func (rd *RemoteDaemon) StartGoApplication(goApplication *GoApplication, ack *bool) error {
	preparable, output, err := rd.daemon.Prepare(goApplication.SourcePath, goApplication.Name, "go", goApplication.KeepAlive, goApplication.Args)
	*ack = true
	if err != nil {
		return fmt.Errorf("ERROR: %s OUTPUT: %s", err, string(output))
	}
	return rd.daemon.RunPreparable(preparable)
}

// RestartProcess will restart a application that was previously built using GoApplication.
// It returns an error in case there's any.
func (rd *RemoteDaemon) RestartApplications(applicationName string, ack *bool) error {
	*ack = true
	return rd.daemon.RestartApplications(applicationName)
}

// StartApplications will start a application that was previously built using GoApplication.
// It returns an error in case there's any.
func (rd *RemoteDaemon) StartApplications(applicationName string, ack *bool) error {
	*ack = true
	return rd.daemon.StartApplications(applicationName)
}

// StopProcess will stop a application that is currently running.
// It returns an error in case there's any.
func (rd *RemoteDaemon) StopApplications(applicationName string, ack *bool) error {
	*ack = true
	return rd.daemon.StopApplications(applicationName)
}

// MonitStatus will query for the status of each application and bind it to procs pointer list.
// It returns an error in case there's any.
func (rd *RemoteDaemon) MonitStatus(req string, response *ApplicationResponse) error {
	req = ""
	applications := rd.daemon.ListApplications()
	applicationsResponse := []*ApplicationDataResponse{}
	for id := range applications {
		application := applications[id]
		applicationData := &ApplicationDataResponse{
			Name:      application.Identifier(),
			Pid:       application.GetPid(),
			Status:    application.GetStatus(),
			KeepAlive: application.ShouldKeepAlive(),
		}
		applicationsResponse = append(applicationsResponse, applicationData)
	}
	*response = ApplicationResponse{
		Applications: applicationsResponse,
	}
	return nil
}

// DeleteProcess will delete a application with name applicationName.
// It returns an error in case there's any.
func (rd *RemoteDaemon) DeleteApplications(applicationName string, ack *bool) error {
	*ack = true
	return rd.daemon.DeleteApplications(applicationName)
}

// Stop will stop APM remote server.
// It returns an error in case there's any.
func (rd *RemoteDaemon) Stop() error {
	return rd.daemon.Stop()
}

// StartRemoteMasterServer starts a remote APM server listening on dsn address and binding to
// configFile.
// It returns a RemoteDaemon instance.
func StartRemoteMasterServer(ln net.Listener, configFile string) *RemoteDaemon {
	RemoteDaemon := &RemoteDaemon{
		daemon: InitMaster(configFile),
	}
	rpc.Register(RemoteDaemon)
	go rpc.Accept(ln)
	return RemoteDaemon
}

// StartRemoteClient will start a remote client that can talk to a remote server that
// is already running on dsn address.
// It returns an error in case there's any or it could not connect within the timeout.
func StartRemoteClient(dsn string, timeout time.Duration) (*RemoteClient, error) {
	conn, err := net.DialTimeout("tcp", dsn, timeout)
	if err != nil {
		return nil, err
	}
	return &RemoteClient{conn: rpc.NewClient(conn)}, nil
}

// Save will save a list of procs onto a file.
// Returns an error in case there's any.
func (client *RemoteClient) Save() error {
	var started bool
	return client.conn.Call("RemoteDaemon.Save", "", &started)
}

// Resurrect will restore all previously save processes.
// Returns an error in case there's any.
func (client *RemoteClient) Resurrect() error {
	var started bool
	return client.conn.Call("RemoteDaemon.Resurrect", "", &started)
}

// StartGoBin is a wrapper that calls the remote StartsGoBin.
// It returns an error in case there's any.
func (client *RemoteClient) StartGoApplication(sourcePath string, name string, keepAlive bool, args []string) error {
	goApplication := &GoApplication{
		SourcePath: sourcePath,
		Name:       name,
		KeepAlive:  keepAlive,
		Args:       args,
	}
	var started bool
	return client.conn.Call("RemoteDaemon.StartGoApplication", goApplication, &started)
}

// RestartApplications is a wrapper that calls the remote RestartProcess.
// It returns an error in case there's any.
func (client *RemoteClient) RestartApplications(applicationName string) error {
	var started bool
	return client.conn.Call("RemoteDaemon.RestartApplications", applicationName, &started)
}

// StartApplications is a wrapper that calls the remote StartProcess.
// It returns an error in case there's any.
func (client *RemoteClient) StartApplications(applicationName string) error {
	var started bool
	return client.conn.Call("RemoteDaemon.StartApplications", applicationName, &started)
}

// StopApplications is a wrapper that calls the remote StopProcess.
// It returns an error in case there's any.
func (client *RemoteClient) StopApplications(applicationName string) error {
	var stopped bool
	return client.conn.Call("RemoteDaemon.StopApplications", applicationName, &stopped)
}

// DeleteApplications is a wrapper that calls the remote DeleteProcess.
// It returns an error in case there's any.
func (client *RemoteClient) DeleteApplications(applicationName string) error {
	var deleted bool
	return client.conn.Call("RemoteDaemon.DeleteApplications", applicationName, &deleted)
}

// MonitStatus is a wrapper that calls the remote MonitStatus.
// It returns a tuple with a list of application and an error in case there's any.
func (client *RemoteClient) MonitStatus() (ApplicationResponse, error) {
	var response *ApplicationResponse
	err := client.conn.Call("RemoteDaemon.MonitStatus", "", &response)
	return *response, err
}
