package system

import (
	"log"
	"os"
	"os/exec"
)

type Systemd struct{}

func (s *Systemd) RestartService(name string) error {
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return err
	}
	return exec.Command("systemctl", "restart", name).Run()
}

func (s *Systemd) SetHostname(hostname string) error {
	if currHostname, err := os.Hostname(); err != nil || currHostname == hostname {
		return err
	}
	log.Printf("setting hostname to %s", hostname)
	return exec.Command("hostnamectl", "set-hostname", hostname).Run()
}
