package system

import (
	"log"
	"os"
	"os/exec"

	"github.com/coreos/go-systemd/dbus"
)

type dbusConn interface {
	Reload() error
	EnableUnitFiles([]string, bool, bool) (bool, []dbus.EnableUnitFileChange, error)
	RestartUnit(string, string, chan<- string) (int, error)
}

// Systemd allows you to interact with the systemd init system.
type Systemd struct {
	Conn dbusConn
}

// EnsureRunning makes sure that the service is running with the latest config.
func (s *Systemd) EnsureRunning(name string) error {
	if err := s.Conn.Reload(); err != nil {
		return err
	}
	if _, _, err := s.Conn.EnableUnitFiles([]string{name}, false, true); err != nil {
		return err
	}
	_, err := s.Conn.RestartUnit(name, "replace", nil)
	return err
}

// SetHostname sets the hostname.
func (s *Systemd) SetHostname(hostname string) error {
	if currHostname, err := os.Hostname(); err != nil || currHostname == hostname {
		return err
	}
	log.Printf("setting hostname to %s", hostname)
	return exec.Command("hostnamectl", "set-hostname", hostname).Run()
}
