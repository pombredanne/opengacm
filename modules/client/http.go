package main

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"

	"github.com/wgliang/opengacm/modules/client/controller/application"
	gapmdaemon "github.com/wgliang/opengacm/modules/client/controller/daemon"

	log "github.com/Sirupsen/logrus"
	"github.com/buaazp/fasthttprouter"
	"github.com/kardianos/osext"
	"github.com/kavu/go_reuseport"
	"github.com/sevlyar/go-daemon"
	"github.com/valyala/fasthttp"
)

const (
	defaultServerAddr   = ":9653"
	defaultTransferAddr = ":9652"
)

var (
	router = fasthttprouter.New()
)

type ClientDaemon struct {
	ServerAddr   string
	TransferAddr string
	AllowedIps   []string
	Applications []application.Application
}

// NewClientDaemon provides the implement of return a ClientDaemon struct.
func NewClientDaemon() *ClientDaemon {
	return &ClientDaemon{}
}

// RegHandler is a function that will register all handler.
func (cd *ClientDaemon) RegHandler(router *fasthttprouter.Router) {
	router.POST("/gacmclient/hello", cd.HandleHello)
	router.POST("/gacmclient/deploy", cd.HandleDeploy)
	router.POST("/gacmclient/policy", cd.HandlePolicy)
	router.POST("/gacmclient/update", cd.HandleUpdate)
	router.POST("/gacmclient/action", cd.HandleAction)
}

// HandleHello is a handler of testing service.
func (cd *ClientDaemon) HandleHello(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, "OpenGACM Client Daemon is OK!  Hi there! Your RequestURI is %q", ctx.RequestURI())
}

// HandleDeploy is a function that provides the ability to remotely
// deploy applications and will receive remote instructions and data
// to complete deployment tasks.
func (cd *ClientDaemon) HandleDeploy(ctx *fasthttp.RequestCtx) {
	// todo:handle the deploy.
}

// HandlePolicy is a function that provides the ability to issue policies
// to the application, it will accept the remote delivery strategy and
// issued to the application, such as black and white list, service open
// and disable and configuration upgrades, etc.
func (cd *ClientDaemon) HandlePolicy(ctx *fasthttp.RequestCtx) {
	// todo:handle the policy.
}

// HandleUpdate is a function that provides the ability to update the
// application, accept remote latest application files to replace local
// applications and update services.
func (cd *ClientDaemon) HandleUpdate(ctx *fasthttp.RequestCtx) {
	// todo:handle the update application file.
}

// HandleAction is a function that provides the ability to operate the
// native Go application by starting, stopping, restarting, and deleting
// the application by accepting remote instructions.
func (cd *ClientDaemon) HandleAction(ctx *fasthttp.RequestCtx) {
	// todo:handle the action of manage applications.
}

// startDaemon is the opengacm-client background guard service, assume the
// communication with the remote management center and local application
// communication function, provides the tcp and http two protocol services
//to meet different needs.
func (cd *ClientDaemon) startDaemon() {
	if cd.ServerAddr == "" || cd.TransferAddr == "" {
		cd.ServerAddr = defaultServerAddr
		cd.TransferAddr = defaultTransferAddr
	}
	ln, err := reuseport.Listen("tcp", cd.ServerAddr)
	if err != nil {
		log.Fatalf("error in net.Listen: %s", err)
	}

	cd.RegHandler(router)

	if *startConfigFile == "" {
		folderPath, err := osext.ExecutableFolder()
		if err != nil {
			log.Fatal(err)
		}
		*startConfigFile = folderPath + "/.apmenv/config.toml"
		os.MkdirAll(path.Dir(*startConfigFile), 0777)
	}

	ctx := &daemon.Context{
		PidFileName: path.Join(filepath.Dir(*startConfigFile), "opengacm-client.pid"),
		PidFilePerm: 0644,
		LogFileName: path.Join(filepath.Dir(*startConfigFile), "opengacm-client.log"),
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
	}
	if ok, _, _ := isDaemonRunning(ctx); ok {
		log.Info("Server is already running.")
		return
	}

	log.Info("Starting daemon...")
	d, err := ctx.Reborn()
	if err != nil {
		log.Fatalf("Failed to reborn daemon due to %+v.", err)
	}

	if d != nil {
		return
	}

	defer ctx.Release()

	log.Info("Starting remote master server...")
	remoteMaster := gapmdaemon.StartRemoteMasterServer(ln, *startConfigFile)

	go func(allowedIPs []string) {
		IPAllowHandler := func(ctx *fasthttp.RequestCtx) {
			ip := ctx.RemoteIP().String()
			isAllowed := false
			for _, v := range allowedIPs {
				if v == ip {
					isAllowed = true
					break
				}
			}
			if !isAllowed {
				log.Println(ip + " Access Unauthorized.")
				ctx.Error("IP Access Unauthorized.", 403)
				return
			}
			router.Handler(ctx)
		}
		if err := fasthttp.Serve(ln, IPAllowHandler); err != nil {
			log.Println("gacmc error:", err)
		}
	}(cd.AllowedIps)
	channelExit := make(chan os.Signal, 1)
	signal.Notify(channelExit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	select {
	case <-channelExit:
	}
	log.Info("Received signal to stop...")
	err = remoteMaster.Stop()
	if err != nil {
		log.Fatal(err)
	}
}

func stopDaemon() {
	if *stopConfigFile == "" {
		folderPath, err := osext.ExecutableFolder()
		if err != nil {
			log.Fatal(err)
		}
		*stopConfigFile = folderPath + "/.apmenv/config.toml"
		os.MkdirAll(path.Dir(*stopConfigFile), 0777)
	}

	ctx := &daemon.Context{
		PidFileName: path.Join(filepath.Dir(*stopConfigFile), "opengacm-client.pid"),
		PidFilePerm: 0644,
		LogFileName: path.Join(filepath.Dir(*stopConfigFile), "opengacm-client.log"),
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
	}

	if ok, p, _ := isDaemonRunning(ctx); ok {
		if err := p.Signal(syscall.Signal(syscall.SIGQUIT)); err != nil {
			log.Fatalf("Failed to kill daemon %v", err)
		}
		log.Info("daemon exit!")
	} else {
		ctx.Release()
		log.Info("instance is not running.")
	}
}

// isDaemonRunning is used to determine whether the current daemon has started.
func isDaemonRunning(ctx *daemon.Context) (bool, *os.Process, error) {
	d, err := ctx.Search()

	if err != nil {
		return false, d, err
	}

	if err := d.Signal(syscall.Signal(0)); err != nil {
		return false, d, err
	}

	return true, d, nil
}
