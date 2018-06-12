package file_test

import (
	pkg "github.com/errm/ekstrap/pkg/file"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var file = &pkg.Atomic{}

func TestWritingToNonExistantFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	check(t, err)
	defer os.RemoveAll(dir) //cleanup

	filename := filepath.Join(dir, "filename")

	ok, err := file.Sync(strings.NewReader("Hello World"), filename)
	if ok != true {
		t.Error("expected ok to be true")
	}
	check(t, err)

	contents, err := ioutil.ReadFile(filename)
	check(t, err)

	if string(contents) != "Hello World" {
		t.Errorf("Unexpected file contents: %s", contents)
	}

	perm := os.FileMode(0640)
	info, err := os.Stat(filename)
	check(t, err)
	if info.Mode() != perm {
		t.Errorf("Expecting mode: %s, got %s", perm, info.Mode())
	}
}

func TestOverwrite(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	check(t, err)
	defer os.RemoveAll(dir) //cleanup

	filename := filepath.Join(dir, "filename")

	err = ioutil.WriteFile(filename, []byte("Old contents"), 0644)
	check(t, err)

	ok, err := file.Sync(strings.NewReader("New contents"), filename)
	if ok != true {
		t.Error("expected ok to be true")
	}
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

	ok, err := file.Sync(strings.NewReader("contents"), filename)
	if ok != false {
		t.Error("expected ok to be false")
	}
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
