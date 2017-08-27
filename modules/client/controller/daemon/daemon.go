// daemon package is the main package that keeps everything operating as it
// should be. It is a background service application. It's responsible for starting,
// stopping and deleting applications. It also will keep an eye on the Watcher in
// case a application dies so it can restart it again.

package daemon

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/wgliang/opengacm/modules/client/controller/application"
	"github.com/wgliang/opengacm/modules/client/controller/preparable"
	"github.com/wgliang/opengacm/modules/client/controller/utils"
	"github.com/wgliang/opengacm/modules/client/controller/watcher"
)

// Daemon is the main module that keeps everything in place and execute
// the necessary actions to keep the application running as they should be.
type Daemon struct {
	sync.Mutex

	SysFolder string           // SysFolder is the main APM folder where the necessary config files will be stored.
	PidFile   string           // PidFille is the APM pid file path.
	OutFile   string           // OutFile is the APM output log file path.
	ErrFile   string           // ErrFile is the APM err log file path.
	Watcher   *watcher.Watcher // Watcher is a watcher instance.

	Applications map[string]application.ApplicationContainer // Applications is a map containing all procs started on APM.
}

// DecodableDaemon is a struct that the config toml file will decode to.
// It is needed because toml decoder doesn't decode to interfaces, so the
// Applications map can't be decoded as long as we use the ApplicationContainer interface
type DecodableDaemon struct {
	SysFolder string
	PidFile   string
	OutFile   string
	ErrFile   string

	Watcher *watcher.Watcher

	Applications map[string]*application.Application
}

// InitMaster will start a daemon instance with configFile.
// It returns a Daemon instance.
func InitMaster(configFile string) *Daemon {
	watcher := watcher.InitWatcher()
	decodableDaemon := &DecodableDaemon{}
	decodableDaemon.Applications = make(map[string]*application.Application)

	err := utils.SafeReadTomlFile(configFile, decodableDaemon)
	if err != nil {
		panic(err)
	}

	applications := make(map[string]application.ApplicationContainer)
	for k, v := range decodableDaemon.Applications {
		applications[k] = v
	}
	// We need this hack because toml decoder doesn't decode to interfaces
	daemon := &Daemon{
		SysFolder:    decodableDaemon.SysFolder,
		PidFile:      decodableDaemon.PidFile,
		OutFile:      decodableDaemon.OutFile,
		ErrFile:      decodableDaemon.ErrFile,
		Watcher:      decodableDaemon.Watcher,
		Applications: applications,
	}

	if daemon.SysFolder == "" {
		os.MkdirAll(path.Dir(configFile), 0777)
		daemon.SysFolder = path.Dir(configFile) + "/"
	}
	daemon.Watcher = watcher
	daemon.Revive()
	log.Infof("All applications revived...")
	go daemon.WatchApplications()
	go daemon.SaveApplicationsLoop()
	go daemon.UpdateStatus()
	return daemon
}

// WatchProcs will keep the applications running forever.
func (daemon *Daemon) WatchApplications() {
	for application := range daemon.Watcher.RestartApplication() {
		if !application.ShouldKeepAlive() {
			daemon.Lock()
			daemon.updateStatus(application)
			daemon.Unlock()
			log.Infof("Application %s does not have keep alive set. Will not be restarted.", application.Identifier())
			continue
		}
		log.Infof("Restarting application %s.", application.Identifier())
		if application.IsAlive() {
			log.Warnf("Application %s was supposed to be dead, but it is alive.", application.Identifier())
		}
		daemon.Lock()
		application.AddRestart()
		err := daemon.restart(application)
		daemon.Unlock()
		if err != nil {
			log.Warnf("Could not restart application %s due to %s.", application.Identifier(), err)
		}
	}
}

// Prepare will compile the source code into a binary and return a preparable
// ready to be executed.
func (daemon *Daemon) Prepare(sourcePath string, name string, language string, keepAlive bool, args []string) (preparable.ProcPreparable, []byte, error) {
	applicationPreparable := &preparable.Preparable{
		Name:       name,
		SourcePath: sourcePath,
		SysFolder:  daemon.SysFolder,
		Language:   language,
		KeepAlive:  keepAlive,
		Args:       args,
	}
	output, err := applicationPreparable.PrepareBin()
	return applicationPreparable, output, err
}

// RunPreparable will run applicationPreparable and add it to the watch list in case everything goes well.
func (daemon *Daemon) RunPreparable(applicationPreparable preparable.ProcPreparable) error {
	daemon.Lock()
	defer daemon.Unlock()
	if _, ok := daemon.Applications[applicationPreparable.Identifier()]; ok {
		log.Warnf("Application %s already exist.", applicationPreparable.Identifier())
		return errors.New("Trying to start a application that already exist.")
	}
	application, err := applicationPreparable.Start()
	if err != nil {
		return err
	}
	daemon.Applications[application.Identifier()] = application
	daemon.saveApplicationsWrapper()
	daemon.Watcher.AddApplicationWatcher(application)
	application.SetStatus("running")
	return nil
}

// ListProcs will return a list of all procs.
func (daemon *Daemon) ListApplications() []application.ApplicationContainer {
	procsList := []application.ApplicationContainer{}
	for _, v := range daemon.Applications {
		procsList = append(procsList, v)
	}
	return procsList
}

// RestartProcess will restart a application.
func (daemon *Daemon) RestartApplications(name string) error {
	err := daemon.StopApplications(name)
	if err != nil {
		return err
	}
	return daemon.StartApplications(name)
}

// StartProcess will a start a application.
func (daemon *Daemon) StartApplications(name string) error {
	daemon.Lock()
	defer daemon.Unlock()
	if application, ok := daemon.Applications[name]; ok {
		return daemon.start(application)
	}
	return errors.New("Unknown application.")
}

// StopProcess will stop a application with the given name.
func (daemon *Daemon) StopApplications(name string) error {
	daemon.Lock()
	defer daemon.Unlock()
	if application, ok := daemon.Applications[name]; ok {
		return daemon.stop(application)
	}
	return errors.New("Unknown application.")
}

// DeleteProcess will delete a application and all its files and childs forever.
func (daemon *Daemon) DeleteApplications(name string) error {
	daemon.Lock()
	defer daemon.Unlock()
	log.Infof("Trying to delete application %s", name)
	if application, ok := daemon.Applications[name]; ok {
		err := daemon.stop(application)
		if err != nil {
			return err
		}
		delete(daemon.Applications, name)
		err = daemon.delete(application)
		if err != nil {
			return err
		}
		log.Infof("Successfully deleted application %s", name)
	}
	return nil
}

// Revive will revive all procs listed on ListProcs. This should ONLY be called
// during daemon startup.
func (daemon *Daemon) Revive() error {
	daemon.Lock()
	defer daemon.Unlock()
	applications := daemon.ListApplications()
	log.Info("Reviving all processes")
	for id := range applications {
		application := applications[id]
		if !application.ShouldKeepAlive() {
			log.Infof("Application %s does not have KeepAlive set. Will not revive it.", application.Identifier())
			continue
		}
		log.Infof("Reviving application %s", application.Identifier())
		err := daemon.start(application)
		if err != nil {
			return fmt.Errorf("Failed to revive application %s due to %s", application.Identifier(), err)
		}
	}
	return nil
}

// NOT thread safe method. Lock should be acquire before calling it.
func (daemon *Daemon) start(application application.ApplicationContainer) error {
	if !application.IsAlive() {
		err := application.Start()
		if err != nil {
			return err
		}
		daemon.Watcher.AddApplicationWatcher(application)
		application.SetStatus("running")
	}
	return nil
}

func (daemon *Daemon) delete(application application.ApplicationContainer) error {
	return application.Delete()
}

// NOT thread safe method. Lock should be acquire before calling it.
func (daemon *Daemon) stop(application application.ApplicationContainer) error {
	if application.IsAlive() {
		waitStop := daemon.Watcher.StopWatcher(application.Identifier())
		err := application.GracefullyStop()
		if err != nil {
			return err
		}
		if waitStop != nil {
			<-waitStop
			application.NotifyStopped()
			application.SetStatus("stopped")
		}
		log.Infof("Application %s successfully stopped.", application.Identifier())
	}
	return nil
}

// UpdateStatus will update a application status every 30s.
func (daemon *Daemon) UpdateStatus() {
	for {
		daemon.Lock()
		applications := daemon.ListApplications()
		for id := range applications {
			application := applications[id]
			daemon.updateStatus(application)
		}
		daemon.Unlock()
		time.Sleep(30 * time.Second)
	}
}

func (daemon *Daemon) updateStatus(application application.ApplicationContainer) {
	if application.IsAlive() {
		application.SetStatus("running")
	} else {
		application.NotifyStopped()
		application.SetStatus("stopped")
	}
}

// NOT thread safe method. Lock should be acquire before calling it.
func (daemon *Daemon) restart(application application.ApplicationContainer) error {
	err := daemon.stop(application)
	if err != nil {
		return err
	}
	return daemon.start(application)
}

// SaveApplicationsLoop will loop forever to save the list of applications onto the application file.
func (daemon *Daemon) SaveApplicationsLoop() {
	for {
		log.Infof("Saving list of applications.")
		daemon.Lock()
		daemon.saveApplicationsWrapper()
		daemon.Unlock()
		time.Sleep(5 * time.Minute)
	}
}

// Stop will stop APM and all of its running applications.
func (daemon *Daemon) Stop() error {
	log.Info("Stopping APM...")
	applications := daemon.ListApplications()
	for id := range applications {
		application := applications[id]
		log.Info("Stopping application %s", application.Identifier())
		daemon.stop(application)
	}
	log.Info("Saving and returning list of applications.")
	return daemon.saveApplicationsWrapper()
}

// SaveApplications will save a list of applications onto a file inside configPath.
// Returns an error in case there's any.
func (daemon *Daemon) SaveApplications() error {
	daemon.Lock()
	defer daemon.Unlock()
	return daemon.saveApplicationsWrapper()
}

// NOT Thread Safe. Lock should be acquired before calling it.
func (daemon *Daemon) saveApplicationsWrapper() error {
	configPath := daemon.getConfigPath()
	return utils.SafeWriteTomlFile(daemon, configPath)
}

func (daemon *Daemon) getConfigPath() string {
	return path.Join(daemon.SysFolder, "config.toml")
}
