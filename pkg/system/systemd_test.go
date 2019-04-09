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
	unitStatuses    []dbus.UnitStatus
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

func (f *fakeDbusConn) ListUnits() ([]dbus.UnitStatus, error) {
	return f.unitStatuses, f.errors["list"]
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
		tC := tC
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
		tC := tC
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

func TestContainerRuntime(t *testing.T) {
	testCases := []struct {
		desc         string
		unitStatuses []dbus.UnitStatus
		expected     string
	}{
		{
			desc: "When docker is loaded",
			unitStatuses: []dbus.UnitStatus{
				{
					Name:      "docker.service",
					LoadState: "loaded",
				},
			},
			expected: "docker",
		},
		{
			desc: "When containerd is loaded",
			unitStatuses: []dbus.UnitStatus{
				{
					Name:      "containerd.service",
					LoadState: "loaded",
				},
			},
			expected: "containerd",
		},
		{
			desc: "When containerd is loaded but a docker unit is also listed (but not loaded)",
			unitStatuses: []dbus.UnitStatus{
				{
					Name:      "docker.service",
					LoadState: "Not found",
				},
				{
					Name:      "containerd.service",
					LoadState: "loaded",
				},
			},
			expected: "containerd",
		},
	}
	for _, tC := range testCases {
		tC := tC
		t.Run(tC.desc, func(t *testing.T) {
			d := &fakeDbusConn{unitStatuses: tC.unitStatuses}
			s := &system.Systemd{Conn: d}
			runtime, err := s.ContainerRuntime()
			if err != nil {
				t.Errorf("Unexpected error:  %v", err)
			}
			if runtime != tC.expected {
				t.Errorf("Expected container runtime %v to be detected, got %v", tC.expected, runtime)
			}
		})
	}
}

func TestContainerRuntimeErrors(t *testing.T) {
	testCases := []struct {
		desc         string
		unitStatuses []dbus.UnitStatus
		expected     error
		error        error
	}{
		{
			desc: "When there is a systemd error",
			unitStatuses: []dbus.UnitStatus{
				{
					Name:      "docker.service",
					LoadState: "loaded",
				},
			},
			expected: errors.New("a systemd error"),
			error:    errors.New("a systemd error"),
		},
		{
			desc: "When no container runtime is loaded",
			unitStatuses: []dbus.UnitStatus{
				{
					Name:      "containerd.service",
					LoadState: "Not found",
				},
			},
			expected: errors.New("couldn't work out what container runtime is installed"),
		},
	}
	for _, tC := range testCases {
		tC := tC
		t.Run(tC.desc, func(t *testing.T) {
			d := &fakeDbusConn{unitStatuses: tC.unitStatuses, errors: map[string]error{"list": tC.error}}
			s := &system.Systemd{Conn: d}
			_, err := s.ContainerRuntime()
			if err == nil {
				t.Errorf("Expected an error!")
			}
			if err.Error() != tC.expected.Error() {
				t.Errorf("Expected error to be %v but was: %v", tC.expected, err)
			}
		})
	}
}
