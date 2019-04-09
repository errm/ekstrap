/*
Copyright 2018 Edward Robinson.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package system

import (
	"errors"
	"log"
	"os"
	"os/exec"

	"github.com/coreos/go-systemd/dbus"
)

type dbusConn interface {
	Reload() error
	EnableUnitFiles([]string, bool, bool) (bool, []dbus.EnableUnitFileChange, error)
	RestartUnit(string, string, chan<- string) (int, error)
	ListUnits() ([]dbus.UnitStatus, error)
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

func (s *Systemd) ContainerRuntime() (string, error) {
	candidates := map[string]string{
		"containerd.service": "containerd",
		"docker.service":     "docker",
	}
	units, err := s.Conn.ListUnits()
	if err != nil {
		return "", err
	}
	for _, unit := range units {
		if value, ok := candidates[unit.Name]; ok && unit.LoadState == "loaded" {
			return value, nil
		}
	}
	return "", errors.New("couldn't work out what container runtime is installed")
}
