package main

import (
	"testing"
	"time"
)

func TestClientDaemon(t *testing.T) {
	daemon := NewClientDaemon()
	daemon.startDaemon()
	time.Sleep(3 * time.Second)
	stopDaemon()
}

func TestGC(t *testing.T) {
	command([]string{"test", "stack", "89448"}, gc)
	command([]string{"test", "stack"}, gc)
}

func TestMemStats(t *testing.T) {
	command([]string{"test", "stack", "89448"}, memStats)
}

func TestGoVersion(t *testing.T) {
	command([]string{"test", "stack", "89448"}, goVersion)
}

func TestPprofHeap(t *testing.T) {
	command([]string{"test", "stack", "89448"}, pprofHeap)
}

func TestPprofCPU(t *testing.T) {
	command([]string{"test", "stack", "89448"}, pprofCPU)
}

func TestProcesses(t *testing.T) {
	processes()
}
