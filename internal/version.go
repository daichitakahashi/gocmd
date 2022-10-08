//go:generate go run ../cmd/genvers -dst=./versions.gen.go -pkg=internal -var=versions
package internal

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
)

var (
	m       sync.Mutex
	fetched bool
)

type version struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
	// Files []any `json:"files"`
}

func Versions(fn func(versions map[string]bool)) {
	m.Lock()
	defer m.Unlock()
	fn(versions)
}

func FetchOnce() (bool, error) {
	m.Lock()
	defer m.Unlock()
	if fetched {
		return false, nil
	}
	v, err := fetch()
	if err != nil {
		return false, err
	}
	versions = v
	fetched = true
	return true, nil
}

func fetch() (map[string]bool, error) {
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

	var v []version
	err = json.Unmarshal(data, &v)
	if err != nil {
		return nil, err
	}

	m := map[string]bool{}
	for _, vv := range v {
		m[vv.Version] = vv.Stable
	}
	return m, nil
}
