// application is a package that controls applications,start.stop or restart
// applications.

package application

import (
	"errors"
	"os"
	"strconv"
	"syscall"

	"github.com/wgliang/opengacm/modules/client/controller/utils"
)

type ApplicationContainer interface {
	Start() error
	ForceStop() error
	GracefullyStop() error
	Restart() error
	Delete() error
	IsAlive() bool
	Identifier() string
	ShouldKeepAlive() bool
	AddRestart()
	NotifyStopped()
	SetStatus(status string)
	GetPid() int
	GetStatus() *ApplicationStatus
	Watch() (*os.ProcessState, error)
	release()
}

// Application is a os.Process wrapper with Status and more info that will be used on Daemon to maintain
// the application health.
type Application struct {
	Name      string
	Cmd       string
	Args      []string
	Path      string
	Pidfile   string
	Outfile   string
	Errfile   string
	KeepAlive bool
	Pid       int
	Status    *ApplicationStatus
	process   *os.Process
}

// Start will execute the command Cmd that should run the application. It will also create an out, err and pidfile
// in case they do not exist yet.
// Returns an error in case there's any.
func (application *Application) Start() error {
	outFile, err := utils.GetFile(application.Outfile)
	if err != nil {
		return err
	}
	errFile, err := utils.GetFile(application.Errfile)
	if err != nil {
		return err
	}
	wd, _ := os.Getwd()
	procAtr := &os.ProcAttr{
		Dir: wd,
		Env: os.Environ(),
		Files: []*os.File{
			os.Stdin,
			outFile,
			errFile,
		},
	}
	args := append([]string{application.Name}, application.Args...)
	process, err := os.StartProcess(application.Cmd, args, procAtr)
	if err != nil {
		return err
	}
	application.process = process
	application.Pid = application.process.Pid
	err = utils.WriteFile(application.Pidfile, []byte(strconv.Itoa(application.process.Pid)))
	if err != nil {
		return err
	}

	application.Status.SetStatus("started")
	return nil
}

// ForceStop will forcefully send a SIGKILL signal to application killing it instantly.
// Returns an error in case there's any.
func (application *Application) ForceStop() error {
	if application.process != nil {
		err := application.process.Signal(syscall.SIGKILL)
		application.Status.SetStatus("stopped")
		application.release()
		return err
	}
	return errors.New("Process does not exist.")
}

// GracefullyStop will send a SIGTERM signal asking the application to terminate.
// The application may choose to die gracefully or ignore this signal completely. In that case
// the application will keep running unless you call ForceStop()
// Returns an error in case there's any.
func (application *Application) GracefullyStop() error {
	if application.process != nil {
		err := application.process.Signal(syscall.SIGTERM)
		application.Status.SetStatus("asked to stop")
		return err
	}
	return errors.New("Process does not exist.")
}

// Restart will try to gracefully stop the application and then Start it again.
// Returns an error in case there's any.
func (application *Application) Restart() error {
	if application.IsAlive() {
		err := application.GracefullyStop()
		if err != nil {
			return err
		}
	}
	return application.Start()
}

// Delete will delete everything created by this application, including the out, err and pid file.
// Returns an error in case there's any.
func (application *Application) Delete() error {
	application.release()
	err := utils.DeleteFile(application.Outfile)
	if err != nil {
		return err
	}
	err = utils.DeleteFile(application.Errfile)
	if err != nil {
		return err
	}
	return os.RemoveAll(application.Path)
}

// IsAlive will check if the application is alive or not.
// Returns true if the application is alive or false otherwise.
func (application *Application) IsAlive() bool {
	p, err := os.FindProcess(application.Pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

// Watch will stop execution and wait until the application change its state. Usually changing state, means that the application died.
// Returns a tuple with the new application state and an error in case there's any.
func (application *Application) Watch() (*os.ProcessState, error) {
	return application.process.Wait()
}

// Will release the application and remove its PID file
func (application *Application) release() {
	if application.process != nil {
		application.process.Release()
	}
	utils.DeleteFile(application.Pidfile)
}

// Notify that application was stopped so we can set its PID to -1
func (application *Application) NotifyStopped() {
	application.Pid = -1
}

// Add one restart to application status
func (application *Application) AddRestart() {
	application.Status.AddRestart()
}

// Return application current PID
func (application *Application) GetPid() int {
	return application.Pid
}

// Return application current status
func (application *Application) GetStatus() *ApplicationStatus {
	return application.Status
}

// Set application status
func (application *Application) SetStatus(status string) {
	application.Status.SetStatus(status)
}

// application identifier that will be used by watcher to keep track of its processes
func (application *Application) Identifier() string {
	return application.Name
}

// Returns true if the application should be kept alive or not
func (application *Application) ShouldKeepAlive() bool {
	return application.KeepAlive
}
