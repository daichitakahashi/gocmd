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
	"sync"
)

var (
	m                   sync.Mutex
	cache               []byte
	cacheStableVersions map[string]bool
)

type version struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
	// Files []any `json:"files"`
}

func fetchStableVersions() (map[string]bool, error) {
	m.Lock()
	defer m.Unlock()

	if cacheStableVersions != nil {
		return cacheStableVersions, nil
	}
	if cache == nil {
		var c http.Client
		resp, err := c.Get("https://go.dev/dl/?mode=json&include=all")
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
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
	cacheStableVersions = m
	return m, nil
}

// ValidVersion returns whether the given go version exists.
// The source of correctness is the following URL:
//
//	https://go.dev/dl/?mode=json&include=all
func ValidVersion(version string) (bool, error) {
	versions, err := fetchStableVersions()
	if err != nil {
		return false, err
	}
	_, ok := versions[version]
	return ok, nil
}

// StableVersion returns whether the given go version exists and is stable.
// The source of correctness is the following URL:
//
//	https://go.dev/dl/?mode=json&include=all
func StableVersion(version string) (bool, error) {
	versions, err := fetchStableVersions()
	if err != nil {
		return false, err
	}
	return versions[version], nil
}

var (
	ErrInvalidVersion = errors.New("invalid version")
	ErrNotFound       = exec.ErrNotFound
)

func checkCommandVersion(cmd, version string) error {
	gotVersion, err := exec.Command(cmd, "env", "GOVERSION").Output()
	if err != nil {
		return err
	}
	if version != string(bytes.TrimSpace(gotVersion)) {
		return fmt.Errorf("got unexpected version %q from %q", gotVersion, cmd)
	}
	return nil
}

// Lookup finds go executable having given version.
// Firstly, check the given version with ValidVersion.
// After that, check versions of "go" and specific executable(golang.org/dl/go1.N).
// If an executable with GOVERSION={given version} exists, it returns the executable's path.
func Lookup(version string) (string, error) {
	// Lookup handles non-version string as error
	if filepath.Base(version) != version {
		return "", ErrInvalidVersion
	}
	valid, err := ValidVersion(version)
	if err != nil {
		return "", err
	}
	if !valid {
		return "", ErrInvalidVersion
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
