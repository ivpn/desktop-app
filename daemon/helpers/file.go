//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
//
//  This file is part of the Daemon for IVPN Client Desktop.
//
//  The Daemon for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The Daemon for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the Daemon for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

package helpers

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/ivpn/desktop-app/daemon/service/platform/filerights"
)

func FileChmod(file string, fileMode os.FileMode) error {
	// set correct file rights
	if runtime.GOOS == "windows" {
		// only for Windows: Golang is not able to change file permissins in Windows style
		if err := filerights.WindowsChmod(file, fileMode); err != nil { // read\write only for privileged user
			os.Remove(file)
			return err
		}
	} else {
		if err := os.Chmod(file, fileMode); err != nil {
			os.Remove(file)
			return err
		}
	}
	return nil
}

// WriteFile writes data to the named file, creating it if necessary.
// It ensures correct file permissions and only then is writing data.
// If File exists - it will be truncated before writing.
func WriteFile(file string, data []byte, fileMode os.FileMode) error {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return err
	}
	defer f.Close()

	// Ensure file has correct permissions
	if err := FileChmod(file, fileMode); err != nil {
		return err
	}

	// Write data to the file
	if _, err = f.Write(data); err != nil {
		return err
	}

	// Ensure changes are immediately flushed to disk
	return f.Sync()
}

// FileExists checks if a file exists and is not a directory
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// CopyFile creates a copy of file
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if sfi.IsDir() {
		return fmt.Errorf("unable to copy ('%s' is directory)", src)
	}
	if !sfi.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
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
