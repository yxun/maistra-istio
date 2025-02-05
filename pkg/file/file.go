// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package file

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// AtomicCopy copies file by reading the file then writing atomically into the target directory
func AtomicCopy(srcFilepath, targetDir, targetFilename string) error {
	in, err := os.Open(srcFilepath)
	if err != nil {
		return err
	}
	defer in.Close()

	perm, err := in.Stat()
	if err != nil {
		return err
	}

	return AtomicWriteReader(filepath.Join(targetDir, targetFilename), in, perm.Mode())
}

func Copy(srcFilepath, targetDir, targetFilename string) error {
	in, err := os.Open(srcFilepath)
	if err != nil {
		return err
	}
	defer in.Close()

	perm, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(filepath.Join(targetDir, targetFilename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// Write atomically by writing to a temporary file in the same directory then renaming
func AtomicWrite(path string, data []byte, mode os.FileMode) error {
	return AtomicWriteReader(path, bytes.NewReader(data), mode)
}

func AtomicWriteReader(path string, data io.Reader, mode os.FileMode) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp.")
	if err != nil {
		return err
	}
	defer func() {
		if Exists(tmpFile.Name()) {
			if rmErr := os.Remove(tmpFile.Name()); rmErr != nil {
				if err != nil {
					err = fmt.Errorf("%s: %w", rmErr.Error(), err)
				} else {
					err = rmErr
				}
			}
		}
	}()

	if err := os.Chmod(tmpFile.Name(), mode); err != nil {
		return err
	}

	if _, err := io.Copy(tmpFile, data); err != nil {
		if closeErr := tmpFile.Close(); closeErr != nil {
			err = fmt.Errorf("%s: %w", closeErr.Error(), err)
		}
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	return os.Rename(tmpFile.Name(), path)
}

func Exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

const (
	// PrivateFileMode grants owner to read/write a file.
	PrivateFileMode = 0o600
)

// IsDirWriteable checks if dir is writable by writing and removing a file
// to dir. It returns nil if dir is writable.
// Inspired by etcd fileutil.
func IsDirWriteable(dir string) error {
	f := filepath.Join(dir, ".touch")
	if err := os.WriteFile(f, []byte(""), PrivateFileMode); err != nil {
		return err
	}
	return os.Remove(f)
}

// DirEquals check if two directories are referring to the same directory
func DirEquals(a, b string) (bool, error) {
	aa, err := filepath.Abs(a)
	if err != nil {
		return false, err
	}
	bb, err := filepath.Abs(b)
	if err != nil {
		return false, err
	}
	return aa == bb, nil
}
