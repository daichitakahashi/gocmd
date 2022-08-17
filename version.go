package gocmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var (
	m                sync.Mutex
	cache            []byte
	cacheAllVersions map[string]bool
)

type version struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
	// Files []any `json:"files"`
}

func fetchAllVersions() (map[string]bool, error) {
	m.Lock()
	defer m.Unlock()

	if cacheAllVersions != nil {
		return cacheAllVersions, nil
	}
	if cache == nil {
		var c http.Client
		resp, err := c.Get("https://go.dev/dl/?mode=json&include=all")
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, errors.New(http.StatusText(resp.StatusCode))
		}
		if err != nil {
			return nil, err
		}
		cache = data
	}
	var v []version
	err := json.Unmarshal(cache, &v)
	if err != nil {
		return nil, err
	}

	m := map[string]bool{}
	for _, vv := range v {
		m[vv.Version] = vv.Stable
	}
	cacheAllVersions = m
	return m, nil
}

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

	versions, err := fetchAllVersions()
	if err != nil {
		return err
	}
	_, ok := versions[version]
	if !ok {
		return ErrInvalidVersion
	}
	return nil
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

	versions, err := fetchAllVersions()
	if err != nil {
		return false, err
	}
	return versions[version], nil
}

func commandVersion(cmd string) (string, error) {
	gotVersion, err := exec.Command(cmd, "env", "GOVERSION").Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(gotVersion)), nil
}

// CurrentVersion returns the version of "go" command.
func CurrentVersion() (string, error) {
	return commandVersion("go")
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
// Behavior is similar to Lookup, but it collects versions that have the same MINOR version.
// This finds the executable that has the latest version in the collected list.
// If "go" command has the same MINOR version, it is prioritized.
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

func findCandidates(expectedVer string) []string {
	v := &byLatestGoVersion{
		expectedVer: expectedVer,
	}
	for vv := range cacheAllVersions {
		if strings.HasPrefix(vv, expectedVer) {
			v.versions = append(v.versions, vv)
		}
	}
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
