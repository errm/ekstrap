package system_test

import (
	"errors"
	"testing"

	"github.com/coreos/go-systemd/dbus"

	"github.com/errm/ekstrap/pkg/system"
)

type fakeDbusConn struct {
	systemdReloaded bool
	restartedUnits  []string
	enabledUnits    []string
	errors          map[string]error
}

func (f *fakeDbusConn) Reload() error {
	f.systemdReloaded = true
	return f.errors["reload"]
}

func (f *fakeDbusConn) EnableUnitFiles(files []string, runtime, force bool) (bool, []dbus.EnableUnitFileChange, error) {
	f.enabledUnits = append(f.enabledUnits, files...)
	return false, []dbus.EnableUnitFileChange{}, f.errors["enable"]
}

func (f *fakeDbusConn) RestartUnit(name string, mode string, ch chan<- string) (int, error) {
	f.restartedUnits = append(f.restartedUnits, name)
	return 0, f.errors["restart"]
}

func TestEnsureRunning(t *testing.T) {
	testCases := []struct {
		desc string
		unit string
	}{
		{
			desc: "Ensure a unit is running",
			unit: "something.service",
		},
		{
			desc: "Restart a different unit",
			unit: "else.service",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			d := &fakeDbusConn{}
			s := &system.Systemd{Conn: d}

			err := s.EnsureRunning(tC.unit)
			if err != nil {
				t.Errorf("Unexpected error:  %v", err)
			}

			if len(d.enabledUnits) != 1 {
				t.Errorf("Expected 1 unit to be enabled, but %v were", len(d.enabledUnits))
			}

			if d.enabledUnits[0] != tC.unit {
				t.Errorf("Expected %v to be restarted, got %v", tC.unit, d.enabledUnits[0])
			}

			if len(d.restartedUnits) != 1 {
				t.Errorf("Expected 1 unit to be restarted, but %v were", len(d.restartedUnits))
			}

			if d.restartedUnits[0] != tC.unit {
				t.Errorf("Expected %v to be restarted, got %v", tC.unit, d.restartedUnits[0])
			}

			if !d.systemdReloaded {
				t.Error("Expected systemd daemon config to be reloaded, it wasn't")
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	testCases := []struct {
		desc string
		err  error
		name string
	}{
		{
			desc: "An error in reloading",
			err:  errors.New("Reloading systemd is broken"),
			name: "reload",
		},
		{
			desc: "An error in enabling",
			err:  errors.New("Enabling a unit is broken"),
			name: "enable",
		},
		{
			desc: "An error in restarting",
			err:  errors.New("Restarting a unit is broken"),
			name: "restart",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			errs := make(map[string]error)
			errs[tC.name] = tC.err
			d := &fakeDbusConn{errors: errs}
			s := &system.Systemd{Conn: d}
			err := s.EnsureRunning("kubelet.service")
			if err != tC.err {
				t.Errorf("Got error: %v, expected %v", err, tC.err)
			}
		})
	}
}
