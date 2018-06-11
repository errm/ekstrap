package file_test

import (
	"fmt"
	"github.com/errm/ekstrap/pkg/file"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
