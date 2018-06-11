package file

import (
	"github.com/dchest/safefile"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

type Atomic struct{}

// Sync atomicly writes data to a file at the given path with the given permissions
//
// If the parent directory does not exit it is created
// If the file allready exists and diff returns 0 then this command is a noopp
// Requires the diff utility to be present on the system, since it is specified in POSIX we assume it is
func (a AtomicWriter) Sync(data io.Reader, path string, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0710); err != nil {
		return err
	}
	f, err := safefile.Create(path, perm)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = io.Copy(f, data); err != nil {
		return err
	}
	if output, needsWrite := diff(path, f.Name()); needsWrite {
		log.Printf("File: %s will be updated:", path)
		log.Printf("%s", output)
		return f.Commit()
	}
	return nil
}

func diff(path, new string) ([]byte, bool) {
	old := path
	if _, err := os.Stat(path); os.IsNotExist(err) {
		old = "/dev/null"
	}
	diff := exec.Command("diff", old, new)
	if output, err := diff.Output(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				if status.ExitStatus() == 1 {
					return output, true
				}
			}
		}
	}
	return []byte{}, false
}
