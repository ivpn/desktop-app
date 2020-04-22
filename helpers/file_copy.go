package helpers

import (
	"fmt"
	"io"
	"os"
)

// CopyFile creates a copy of file
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if sfi.IsDir() {
		return fmt.Errorf("unable to copy ('%s' is directory)", src)
	}

	_, err = os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			// it is not 'NotExist' error
			return err
		}
	}

	err = copyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named by dst
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
