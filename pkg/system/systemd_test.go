package system

import (
	"testing"

	"github.com/coreos/go-systemd/dbus"
)

type fakeDbusConn struct {
	restartedUnit string
}

func (f *fakeDbusConn) Reload() error {
	return nil
}

func (f *fakeDbusConn) EnableUnitFiles(files []string, runtime, force bool) (bool, []dbus.EnableUnitFileChange, error) {
	return false, []dbus.EnableUnitFileChange{}, nil
}

func (f *fakeDbusConn) RestartUnit(name string, mode string, ch chan<- string) (int, error) {
	f.restartedUnit = name
	return 0, nil
}

func TestEnsureRunning(t *testing.T) {
	testCases := []struct {
		desc string
		unit string
	}{
		{
			desc: "Restart a unit",
			unit: "something",
		},
		{
			desc: "Restart a different unit",
			unit: "else",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			d := &fakeDbusConn{}
			s := &Systemd{d}

			err := s.EnsureRunning(tC.unit)
			if err != nil {
				t.Errorf("Unexpected error:  %v", err)
			}

			if d.restartedUnit != tC.unit {
				t.Errorf("Expected %v to be restarted, got %v", tC.unit, d.restartedUnit)
			}
		})
	}
}
