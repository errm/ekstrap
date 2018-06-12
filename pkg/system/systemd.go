package system

import (
	"errors"
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
		if err := systemctl("daemon-reload"); err != nil {
			return err
		}
	}

	for service, _ := range s.servicesToRestart {
		if err := enable(service); err != nil {
			return err
		}

		log.Printf("restarting %s service", service)
		if err := systemctl("restart", service); err != nil {
			return err
		}
	}

	return nil
}

func systemctl(args ...string) error {
	output, err := exec.Command("systemctl", args...).CombinedOutput()
	if err != nil {
		return errors.New(string(output))
	}
	return nil
}

func enable(service string) error {
	path := "/etc/systemd/system/multi-user.target.wants/" + service + ".service"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("enabling %s service", service)
		return systemctl("enable", service)
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
