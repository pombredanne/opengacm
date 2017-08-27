package cli

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/wgliang/opengacm/modules/client/controller/daemon"
)

// Cli is the command line client.
type Cli struct {
	remoteClient *daemon.RemoteClient
}

// InitCli initiates a remote client connecting to dsn.
// Returns a Cli instance.
func InitCli(dsn string, timeout time.Duration) *Cli {
	client, err := daemon.StartRemoteClient(dsn, timeout)
	if err != nil {
		log.Println("Failed to start remote client due to: %+v\n", err)
	}
	return &Cli{
		remoteClient: client,
	}
}

// Save will save all previously saved applications onto a list.
// Display an error in case there's any.
func (cli *Cli) Save() {
	err := cli.remoteClient.Save()
	if err != nil {
		log.Println("Failed to save list of applications due to: %+v\n", err)
	}
}

// Resurrect will restore all previously save applications.
// Display an error in case there's any.
func (cli *Cli) Resurrect() {
	err := cli.remoteClient.Resurrect()
	if err != nil {
		log.Println("Failed to resurrect all previously save applications due to: %+v\n", err)
	}
}

// StartGoApplication will try to start a go binary application.
// Returns a fatal error in case there's any.
func (cli *Cli) StartGoApplication(sourcePath string, name string, keepAlive bool, args []string) {
	err := cli.remoteClient.StartGoApplication(sourcePath, name, keepAlive, args)
	if err != nil {
		log.Println("Failed to start go application binary due to: %+v\n", err)
	}
}

// RestartProcess will try to restart a application with applicationName. Note that this application
// must have been already started through StartGoBin.
func (cli *Cli) RestartApplications(applicationName string) {
	err := cli.remoteClient.RestartApplications(applicationName)
	if err != nil {
		log.Println("Failed to restart application due to: %+v\n", err)
	}
}

// StartProcess will try to start a application with applicationName. Note that this application
// must have been already started through StartGoBin.
func (cli *Cli) StartApplications(applicationName string) {
	err := cli.remoteClient.StartApplications(applicationName)
	if err != nil {
		log.Println("Failed to start application due to: %+v\n", err)
	}
}

// StopProcess will try to stop a process named applicationName.
func (cli *Cli) StopApplications(applicationName string) {
	err := cli.remoteClient.StopApplications(applicationName)
	if err != nil {
		log.Println("Failed to stop application due to: %+v\n", err)
	}
}

// DeleteProcess will stop and delete all dependencies from process applicationName forever.
func (cli *Cli) DeleteApplications(applicationName string) {
	err := cli.remoteClient.DeleteApplications(applicationName)
	if err != nil {
		log.Println("Failed to delete application due to: %+v\n", err)
	}
}

// Status will display the status of all applications started through StartGoBin.
func (cli *Cli) Status() {
	applicationResponse, err := cli.remoteClient.MonitStatus()
	if err != nil {
		log.Println("Failed to get status due to: %+v\n", err)
	}
	maxName := 0
	for id := range applicationResponse.Applications {
		application := applicationResponse.Applications[id]
		maxName = int(math.Max(float64(maxName), float64(len(application.Name))))
	}
	totalSize := maxName + 51
	topBar := ""
	for i := 1; i <= totalSize; i += 1 {
		topBar += "-"
	}
	infoBar := fmt.Sprintf("|%s|%s|%s|%s|",
		PadString("pid", 13),
		PadString("name", maxName+2),
		PadString("status", 16),
		PadString("keep-alive", 15))
	fmt.Println(topBar)
	fmt.Println(infoBar)
	for id := range applicationResponse.Applications {
		application := applicationResponse.Applications[id]
		kp := "True"
		if !application.KeepAlive {
			kp = "False"
		}
		fmt.Printf("|%s|%s|%s|%s|\n",
			PadString(fmt.Sprintf("%d", application.Pid), 13),
			PadString(application.Name, maxName+2),
			PadString(application.Status.Status, 16),
			PadString(kp, 15))
	}
	fmt.Println(topBar)
}

// PadString will add totalSize spaces evenly to the right and left side of str.
// Returns str after applying the pad.
func PadString(str string, totalSize int) string {
	turn := 0
	for {
		if len(str) >= totalSize {
			break
		}
		if turn == 0 {
			str = " " + str
			turn ^= 1
		} else {
			str = str + " "
			turn ^= 1
		}
	}
	return str
}
