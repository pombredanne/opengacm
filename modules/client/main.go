// opengacm-client is a client agent that can manage local go applications,
// such as building applications, opening, shutting down and restarting
// operations; and you can also use opengacm-client to diagnose running Go
// applications.

// In addition, as an important part of the opengacm project, opengacm-client
// is a machine-level proxy unit that will assume data acquisition, data
// reporting, application remote management, and application deployment.
package main

import (
	"fmt"
	"os"

	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	INFO = `opengacm-client is a client agent that can manage local go applications,
such as building applications, opening, shutting down and restarting
operations; and you can also use opengacm-client to diagnose running Go
applications.`
)

var (
	Name    = "client"
	Version = "0.0.0"
	Commit  = ""
	// opengacm-client start/restart/stop.
	client            = kingpin.New(Name, "Local golang applications Manager system.")
	start             = client.Command("start", "Start opengacm-client daemon.")
	startConfigFile   = start.Flag("config-file", "Config file of opengacm-client daemon.").String()
	stop              = client.Command("stop", "Stop opengacm-client daemon.")
	stopConfigFile    = stop.Flag("config-file", "Config file of opengacm-client daemon.").String()
	restart           = client.Command("restart", "Restart opengacm-client daemon.")
	restartConfigFile = restart.Flag("config-file", "Config file of opengacm-client daemon.").String()
	// Manage Command List.
	login      = client.Command("login", "Login opengacm-client daemon and manage applications.")
	version    = client.Command("version", "Get opengacm-client version.")
	info       = client.Command("info", "Get opengacm-client information.")
	cstack     = client.Command("stack", "Prints the stack trace..")
	cgc        = client.Command("gc", "Runs the garbage collector and blocks until successful.")
	cmemstats  = client.Command("memstats", "Prints the allocation and garbage collection stats.")
	cversion   = client.Command("goversion", "Prints the Go version used to build the program.")
	cpprofHeap = client.Command("pprof-heap", `Reads the heap profile and launches "go tool pprof".`)
	cpprofCPU  = client.Command("pprof-cpu", `Reads the CPU profile and launches "go tool pprof".`)
	cstats     = client.Command("stats", "Prints the vital runtime stats.")
	ctrace     = client.Command("trace", `Runs the runtime tracer for 5 secs and launches "go tool trace".`)
)

// showVersion is a function that get the version information.
func showVersion() {
	fmt.Printf("%s version: %s, commit: %s\n", Name, Version, Commit)
}

// showInfo is a function that get the process information.
func showInfo() {
	fmt.Printf("%s info: %s\n", Name, INFO)
}

func main() {
	if len(os.Args) < 2 {
		processes()
		return
	}
	switch kingpin.MustParse(client.Parse(os.Args[1:2])) {
	case start.FullCommand():
		daemon := NewClientDaemon()
		daemon.startDaemon()
	case stop.FullCommand():
		stopDaemon()
	case restart.FullCommand():
		showVersion()
	case login.FullCommand():
		manageApplications()
	case cstack.FullCommand():
		command(os.Args, stackTrace)
	case cgc.FullCommand():
		command(os.Args, gc)
	case cmemstats.FullCommand():
		command(os.Args, memStats)
	case cversion.FullCommand():
		command(os.Args, goVersion)
	case cpprofHeap.FullCommand():
		command(os.Args, pprofHeap)
	case cpprofCPU.FullCommand():
		command(os.Args, pprofCPU)
	case cstats.FullCommand():
		command(os.Args, stats)
	case ctrace.FullCommand():
		command(os.Args, trace)
	case version.FullCommand():
		showVersion()
	case info.FullCommand():
		showInfo()
	}
}
