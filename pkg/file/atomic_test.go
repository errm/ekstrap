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

package file_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pkg "github.com/errm/ekstrap/pkg/file"
)

var file = &pkg.Atomic{}

func TestWritingToNonExistantFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	check(t, err)
	defer os.RemoveAll(dir) //cleanup

	filename := filepath.Join(dir, "filename")

	err = file.Sync(strings.NewReader("Hello World"), filename, 0640)
	check(t, err)

	contents, err := ioutil.ReadFile(filename)
	check(t, err)
	if string(contents) != "Hello World" {
		t.Errorf("Unexpected file contents: %s", contents)
	}
}

func TestPermissions(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	check(t, err)
	defer os.RemoveAll(dir) //cleanup

	perms := []os.FileMode{
		0640,
		0644,
	}

	for _, perm := range perms {
		filename := filepath.Join(dir, fmt.Sprintf("filename-%s", perm))
		err = file.Sync(strings.NewReader("string"), filename, perm)
		check(t, err)
		info, err := os.Stat(filename)
		check(t, err)
		if info.Mode() != perm {
			t.Errorf("Expecting mode: %s, got %s", perm, info.Mode())
		}
	}

}

func TestOverwrite(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	check(t, err)
	defer os.RemoveAll(dir) //cleanup

	filename := filepath.Join(dir, "filename")

	err = ioutil.WriteFile(filename, []byte("Old contents"), 0644)
	check(t, err)

	err = file.Sync(strings.NewReader("New contents"), filename, 0644)
	check(t, err)

	contents, err := ioutil.ReadFile(filename)
	check(t, err)
	if string(contents) != "New contents" {
		t.Errorf("Unexpected file contents: %s", contents)
	}
}

func TestNoOppOverwrite(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	check(t, err)
	defer os.RemoveAll(dir) //cleanup

	filename := filepath.Join(dir, "filename")

	err = ioutil.WriteFile(filename, []byte("contents"), 0644)
	check(t, err)
	info, err := os.Stat(filename)
	check(t, err)
	mtime := info.ModTime()
	time.Sleep(10 * time.Millisecond)

	err = file.Sync(strings.NewReader("contents"), filename, 0644)
	check(t, err)

	info, err = os.Stat(filename)
	check(t, err)
	if info.ModTime() != mtime {
		t.Errorf("File should not have been rewritten")
	}
}

func check(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Unexpected error %s", err)
	}
}
