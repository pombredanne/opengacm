package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/wgliang/opengacm/modules/client/controller/cli"

	log "github.com/Sirupsen/logrus"
)

const (
	EXIT    = "EXIT"
	START   = "START"
	STOP    = "STOP"
	RESTART = "RESTART"
	ADD     = "ADD"
	GET     = "GET"
	LIST    = "LIST"
	STATUS  = "STATUS"
	SAVE    = "SAVE"
	DELETE  = "DELETE"
	RECOVER = "RECOVER"
)

func manageApplications() {
	running := true
	reader := bufio.NewReader(os.Stdin)
	dns := "127.0.0.1:9653"
	timeout := time.Duration(30) * time.Second
	for running {
		fmt.Print("gacmgc#")

		command, _, _ := reader.ReadLine()
		commands := strings.Split(string(command), " ")
		if len(commands) <= 0 {
			continue
		}

		operation := strings.ToUpper(commands[0])
		switch operation {
		case EXIT:
			os.Exit(0)
		case START:
			cli := cli.InitCli(dns, timeout)
			cli.StartApplications(commands[1])
		case STOP:
			cli := cli.InitCli(dns, timeout)
			cli.StopApplications(commands[1])
		case RESTART:
			cli := cli.InitCli(dns, timeout)
			cli.RestartApplications(commands[1])
		case GET:
			cli := cli.InitCli(dns, timeout)
			cli.StartGoApplication(commands[2], commands[1], true, []string{})
		case DELETE:
			cli := cli.InitCli(dns, timeout)
			cli.DeleteApplications(commands[1])
		case SAVE:
			cli := cli.InitCli(dns, timeout)
			cli.Save()
		case RECOVER:
			cli := cli.InitCli(dns, timeout)
			cli.Resurrect()
		case STATUS:
			cli := cli.InitCli(dns, timeout)
			cli.Status()
		case LIST:
			cli := cli.InitCli(dns, timeout)
			cli.Status()
		}

	}

	signalsKill := make(chan os.Signal, 1)
	signal.Notify(signalsKill,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-signalsKill
	log.Info("Received signal to stop...")
	os.Exit(0)
}
