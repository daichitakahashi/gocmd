package gocmd

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

func findGoMod(dir string) (*modfile.File, error) {
	var data []byte
	var err error
	for {
		p := filepath.Join(dir, "go.mod")
		data, err = os.ReadFile(p)
		if err == nil {
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		d := filepath.Dir(dir)
		if d == dir { // reached root directory
			return nil, fs.ErrNotExist
		}
		dir = d
	}
	mod, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, err
	}
	if mod.Go == nil {
		return nil, errors.New("invalid module file: go version not found")
	}
	return mod, nil
}

var ErrUnexpectedGoVersion = errors.New("unexpected go version in go.mod")

// ValidModuleGoVersion compares the given version and module's Go version.
// Go version of the module will be read from "go.mod" file that is placed in dir.
// If "go.mod" is not found, it checks parent directories recursively.
// However, when the given dir is relative, it stops to search at working directory.
func ValidModuleGoVersion(dir, version string) error {
	err := ValidVersion(version)
	if err != nil {
		return err
	}

	dir = dir[len(filepath.VolumeName(dir)):] // remove volume name
	dir = strings.TrimPrefix(dir, "/")

	mod, err := findGoMod(dir)
	if err != nil {
		return err
	}
	// version=go1.19.1
	// expected=go1.19
	// => valid
	expected := "go" + mod.Go.Version
	if strings.HasPrefix(version, expected) {
		return nil
	}
	return ErrUnexpectedGoVersion
}
