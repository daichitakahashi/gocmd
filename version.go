package gocmd

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/daichitakahashi/gocmd/internal"
)

var ErrInvalidVersion = errors.New("invalid version")

// ValidVersion returns whether the given go version exists.
// The source of correctness is the following URL:
//
//	https://go.dev/dl/?mode=json&include=all
func ValidVersion(version string) error {
	// handle non-version string as error
	if filepath.Base(version) != version {
		return ErrInvalidVersion
	}

	var ok bool
	internal.Versions(func(versions map[string]bool) {
		_, ok = versions[version]
	})
	if ok {
		return nil
	}
	fetched, err := internal.FetchOnce()
	if err != nil {
		return err
	}
	if fetched {
		internal.Versions(func(versions map[string]bool) {
			_, ok = versions[version]
		})
		if ok {
			return nil
		}
	}
	return ErrInvalidVersion

}

// StableVersion returns whether the given go version exists and is stable.
// The source of correctness is the following URL:
//
//	https://go.dev/dl/?mode=json&include=all
func StableVersion(version string) (bool, error) {
	// handle non-version string as error
	if filepath.Base(version) != version {
		return false, ErrInvalidVersion
	}

	var stable, ok bool
	internal.Versions(func(versions map[string]bool) {
		stable, ok = versions[version]
	})
	if ok {
		return stable, nil
	}
	fetched, err := internal.FetchOnce()
	if err != nil {
		return false, err
	}
	if fetched {
		internal.Versions(func(versions map[string]bool) {
			stable, ok = versions[version]
		})
		if ok {
			return stable, nil
		}
	}
	return false, ErrInvalidVersion
}

var (
	verCache = map[string]string{}
	vm       sync.Mutex
)

func commandVersion(cmd string) (string, error) {
	vm.Lock()
	defer vm.Unlock()
	if v, ok := verCache[cmd]; ok {
		return v, nil
	}
	gotVersion, err := exec.Command(cmd, "env", "GOVERSION").Output()
	if err != nil {
		return "", err
	}
	v := string(bytes.TrimSpace(gotVersion))
	verCache[cmd] = v
	return v, nil
}

// CurrentVersion returns the version of "go" command.
func CurrentVersion() (string, error) {
	return commandVersion("go")
}

// MajorVersion returns major version of the given version.
// If the given version is invalid, returned value is an empty string.
func MajorVersion(version string) string {
	return versionRe.FindString(version)
}

var ErrNotFound = exec.ErrNotFound

func checkCommandVersion(cmd, version string) error {
	gotVersion, err := commandVersion(cmd)
	if err != nil {
		return err
	}
	if version != gotVersion {
		return fmt.Errorf("got unexpected version %q from %q", gotVersion, cmd)
	}
	return nil
}

// Lookup finds a go executable having exact given version.
// Firstly, it checks the given version with ValidVersion, and returns ErrInvalidVersion if the version is invalid.
// After that, it checks versions of "go" and specific executable(golang.org/dl/go1.N).
// When an executable with GOVERSION={given version} exists, it returns the executable's path.
// If no executable exists, it returns ErrNotFound.
func Lookup(version string) (string, error) {
	err := ValidVersion(version)
	if err != nil {
		return "", err
	}

	var full string
	var goErr, verErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		goErr = checkCommandVersion("go", version)
	}()
	go func() {
		defer wg.Done()
		full, verErr = exec.LookPath(version)
		if verErr != nil {
			return
		}
		verErr = checkCommandVersion(full, version)
	}()
	wg.Wait()

	if goErr == nil {
		return "go", nil
	}
	if verErr == nil {
		return full, nil
	}

	if errors.Is(verErr, exec.ErrNotFound) {
		return "", ErrNotFound
	}
	return "", verErr
}

var versionRe = regexp.MustCompile(`^go[1-9][0-9]*\.(?:0|[1-9][0-9]*)`)

// LookupLatest finds a go executable having the given version.
// Behavior is similar to Lookup, but it collects versions that have the same major version.
// This finds the executable that has the latest version in the collected list.
// If "go" command has the same major version, it is prioritized.
func LookupLatest(version string) (string, error) {
	err := ValidVersion(version)
	if err != nil {
		return "", err
	}

	expectedVer := versionRe.FindString(version)

	// check "go" command
	cur, err := CurrentVersion()
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(cur, expectedVer) {
		return "go", nil
	}

	// find the latest command
	for _, c := range findCandidates(expectedVer) {
		full, err := exec.LookPath(c)
		if err != nil {
			continue
		}
		err = checkCommandVersion(full, c)
		if err == nil {
			return full, nil
		}
	}
	return "", ErrNotFound
}

// this function must be called after internal.FetchAllVersions
func findCandidates(expectedVer string) []string {
	v := &byLatestGoVersion{
		expectedVer: expectedVer,
	}
	internal.Versions(func(versions map[string]bool) {
		for vv := range versions {
			if strings.HasPrefix(vv, expectedVer) {
				v.versions = append(v.versions, vv)
			}
		}
	})
	sort.Sort(v)
	return v.versions
}

// implements sort.Interface.
// It sorts Go versions in descending order.
type byLatestGoVersion struct {
	expectedVer string
	versions    []string
}

func (b *byLatestGoVersion) Len() int {
	return len(b.versions)
}

type typ int

const (
	beta typ = iota + 1
	rc
	stable
)

func getTyp(s string) typ {
	if s == "" {
		return stable
	} else if strings.HasPrefix(s, "beta") {
		return beta
	} else if strings.HasPrefix(s, "rc") {
		return rc
	}
	return stable
}

func (b *byLatestGoVersion) Less(i, j int) bool {
	iv, jv := b.versions[i], b.versions[j]

	var it, jt typ
	it = getTyp(strings.TrimPrefix(iv, b.expectedVer))
	jt = getTyp(strings.TrimPrefix(jv, b.expectedVer))

	// compare typ
	if it != jt {
		return it > jt
	}

	// compare string
	il, jl := len(iv), len(jv)
	if il != jl {
		return il > jl
	}
	return iv > jv
}

func (b *byLatestGoVersion) Swap(i, j int) {
	b.versions[i], b.versions[j] = b.versions[j], b.versions[i]
}

var _ sort.Interface = (*byLatestGoVersion)(nil)

type Mode uint8

const (
	ModeExact Mode = 1 << iota
	ModeLatest
	ModeFallback
)

// Determine go command with given version, and return its path and actual version.
// Following mode is available.
//   - ModeExact determines command by using Lookup
//   - ModeLatest determines command by using LookupLatest
//   - ModeFallback determines command by using LookupLatest, but if no command was found, fallbacks to "go" command
func Determine(version string, mode Mode) (path, ver string, err error) {
	if mode == ModeExact {
		path, err = Lookup(version)
		if err != nil {
			return "", "", fmt.Errorf(`failed to find "go" command which has the version %s exactly`, version)
		}
	} else {
		path, err = LookupLatest(version)
		if err != nil {
			if mode == ModeLatest {
				return "", "", fmt.Errorf(`failed to find "go" command that has major version %s: %w`, MajorVersion(version), err)
			}
			path = "go" // ModeFallback
		}
	}

	if path == "go" {
		goVer, err := CurrentVersion()
		if err != nil {
			return "", "", fmt.Errorf(`failed to get "go" version`)
		}
		return path, goVer, nil
	}
	return path, filepath.Base(path), nil
}

// DetermineFromModuleGoVersion determines go command with the version from go.mod, and returns its path and actual version.
// Every mode uses LookupLatest. In ModeFallback, if no command was found, fallbacks to "go"command.
func DetermineFromModuleGoVersion(mode Mode) (path, ver string, _ error) {
	modVer, err := ModuleGoVersion()
	if err != nil {
		return "", "", fmt.Errorf("failed to read go.mod: %w", err)
	}
	path, err = LookupLatest(modVer)
	if err != nil {
		switch mode {
		case ModeFallback:
			goVer, _ := CurrentVersion() // CurrentVersion is already called and succeeded in LookupLatest
			return "go", goVer, nil
		default: // ModeExact, ModeLatest
			return "", "", fmt.Errorf(`failed to find "go" command that has major version %s: %w`, modVer, err)
		}
	}
	if path == "go" {
		goVer, _ := CurrentVersion()
		return path, goVer, nil
	}
	return path, filepath.Base(path), nil
}
