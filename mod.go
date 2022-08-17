package gocmd

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/mod/modfile"
)

// ModuleGoVersion reads module's go version from "go.mod" with the path from `go env GOMOD`.
func ModuleGoVersion() (string, error) {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return "", err
	}
	path := string(bytes.TrimSpace(out))
	if path == "" || path == os.DevNull {
		return "", fs.ErrNotExist
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	mod, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return "", err
	}
	if mod.Go == nil {
		return "", errors.New("invalid module file: go version not found")
	}
	return "go" + mod.Go.Version, nil
}

var ErrUnexpectedGoVersion = errors.New("unexpected go version in go.mod")

// ValidModuleGoVersion compares the given version and module's Go version.
// Go version of the module will be read from "go.mod" with the path from `go env GOMOD`.
func ValidModuleGoVersion(version string) error {
	err := ValidVersion(version)
	if err != nil {
		return err
	}

	expected, err := ModuleGoVersion()
	if err != nil {
		return err
	}

	// version=go1.19.1
	// expected=go1.19
	// => valid
	if strings.HasPrefix(version, expected) {
		return nil
	}
	return ErrUnexpectedGoVersion
}
