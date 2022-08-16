package gocmd

import (
	"bytes"
	"errors"
	"os/exec"
	"testing"
)

var inputs = []struct {
	version string
	exists  bool
	stable  bool
}{
	{
		version: "go1.19",
		exists:  true,
		stable:  true,
	}, {
		version: "go1.19beta1",
		exists:  true,
		stable:  false,
	}, {
		version: "go1.19rc2",
		exists:  true,
		stable:  false,
	}, {
		version: "go1.18beta9",
		exists:  false,
		stable:  false,
	},
}

func TestValidVersion(t *testing.T) {
	t.Parallel()

	for _, i := range inputs {
		valid, err := ValidVersion(i.version)
		if err != nil {
			t.Fatal(err)
		}
		if i.exists && !valid {
			t.Errorf("the go version %q expected to be valid", i.version)
		} else if !i.exists && valid {
			t.Errorf("the go version %q expected not to be valid", i.version)
		}
	}
}

func TestStableVersion(t *testing.T) {
	t.Parallel()

	for _, i := range inputs {
		valid, err := StableVersion(i.version)
		if err != nil {
			t.Fatal(err)
		}
		if i.stable && !valid {
			t.Errorf("the go version %q expected to be stable", i.version)
		} else if !i.stable && valid {
			t.Errorf("the go version %q expected not to exist or be stable", i.version)
		}
	}
}

func TestLookup(t *testing.T) {
	t.Parallel()

	cur := currentVersion(t)

	// current version results "go"
	path, err := Lookup(cur)
	if err != nil {
		t.Fatal(err)
	}
	if path != "go" {
		t.Fatalf("expected path: %q, got path: %q", "go", path)
	}
	assertCommand(t, path, cur)

	path, err = Lookup("go1.18.5")
	if err != nil {
		t.Fatal(err)
	}
	assertCommand(t, path, "go1.18.5")

	path, err = Lookup("go1.19rc2")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("go1.19rc2: expected error: %v, got error: %v, got path: %s", ErrNotFound, err, path)
	}
}

// current version must be larger than "go1.19"
func currentVersion(t *testing.T) string {
	t.Helper()

	result, err := exec.Command("go", "env", "GOVERSION").Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes.TrimSpace(result))
}

func assertCommand(t *testing.T, path, version string) bool {
	t.Helper()
	result, err := exec.Command(path, "version").Output()
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Contains(result, []byte(" "+version+" "))
}
