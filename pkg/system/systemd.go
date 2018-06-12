package system

import (
	"log"
	"os"
	"os/exec"
)

type Systemd struct {
	servicesToRestart map[string]bool
}

func (s *Systemd) NeedsRestart(name string) {
	s.servicesToRestart = set(s.servicesToRestart, name)
}

func (s *Systemd) RestartServices() error {
	if len(s.servicesToRestart) > 0 {
		log.Print("reloading systemd configuration")
		if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
			return err
		}
	}

	for service, _ := range s.servicesToRestart {
		log.Printf("restarting %s service", service)
		if err := exec.Command("systemctl", "restart", service).Run(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Systemd) SetHostname(hostname string) error {
	if currHostname, err := os.Hostname(); err != nil || currHostname == hostname {
		return err
	}
	log.Printf("setting hostname to %s", hostname)
	return exec.Command("hostnamectl", "set-hostname", hostname).Run()
}
