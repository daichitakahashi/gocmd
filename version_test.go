package gocmd

import (
	"bytes"
	"errors"
	"os/exec"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var inputs = []struct {
	version string
	exists  bool
	stable  bool
	error   bool
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
	}, {
		version: "unknown",
		exists:  false,
		stable:  false,
	}, {
		version: "../invalid",
		error:   true,
	},
}

func TestValidVersion(t *testing.T) {
	t.Parallel()

	for _, i := range inputs {
		err := ValidVersion(i.version)
		if err != nil && !errors.Is(err, ErrInvalidVersion) {
			t.Fatal(err)
		}
		if i.error {
			if err == nil {
				t.Error("error expected")
			}
		} else if i.exists && err != nil {
			t.Errorf("the go version %q expected to be valid", i.version)
		} else if !i.exists && err == nil {
			t.Errorf("the go version %q expected not to be valid", i.version)
		}
	}
}

func TestStableVersion(t *testing.T) {
	t.Parallel()

	for _, i := range inputs {
		valid, err := StableVersion(i.version)
		if err != nil && !errors.Is(err, ErrInvalidVersion) {
			t.Fatal(err)
		}
		if i.stable && !valid {
			t.Errorf("the go version %q expected to be stable", i.version)
		} else if !i.stable && valid {
			t.Errorf("the go version %q expected not to exist or be stable", i.version)
		}
	}
}

// current version must be larger than "go1.19"
func currentVersion(t *testing.T) string {
	t.Helper()

	result, err := exec.Command("go", "env", "GOVERSION").Output()
	if err != nil {
		t.Fatal(err)
	}
	v := string(bytes.TrimSpace(result))
	prefix := versionRe.FindString(v)
	if prefix < "go1.19" {
		t.Skipf("test skipped because version of go command is less than go1.19: %s", v)
	}
	return v
}

// check following
//   - go1.18.4 exists
//   - go1.18.5 exists
//   - go1.19rc2 not exists
func checkPrerequisites(t *testing.T) {
	t.Helper()

	_, err := exec.LookPath("go1.18.4")
	if err != nil {
		t.Skipf("test skipped because go1.18.4 command not exists: %s", err)
	}
	_, err = exec.LookPath("go1.18.5")
	if err != nil {
		t.Skipf("test skipped because go1.18.5 command not exists: %s", err)
	}
	_, err = exec.LookPath("go1.19rc2")
	if err == nil {
		t.Skipf("tes skipped because go1.19rc2 command exists")
	}
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestLookup(t *testing.T) {
	t.Parallel()
	checkPrerequisites(t)

	assertCommand := func(t *testing.T, path, version string) bool {
		t.Helper()
		result, err := exec.Command(path, "version").Output()
		if err != nil {
			t.Fatal(err)
		}
		return bytes.Contains(result, []byte(" "+version+" "))
	}

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

	path, err = Lookup("unknown")
	if !errors.Is(err, ErrInvalidVersion) {
		t.Fatalf("unknown: expected error: %v, got error: %v, got path: %s", ErrInvalidVersion, err, path)
	}
}

func TestFindCandidates(t *testing.T) {
	t.Parallel()

	const version = "go1.15"
	err := ValidVersion(version)
	if err != nil {
		t.Fatal(err)
	}
	candidates := findCandidates(version)

	diff := cmp.Diff([]string{
		"go1.15.15",
		"go1.15.14",
		"go1.15.13",
		"go1.15.12",
		"go1.15.11",
		"go1.15.10",
		"go1.15.9",
		"go1.15.8",
		"go1.15.7",
		"go1.15.6",
		"go1.15.5",
		"go1.15.4",
		"go1.15.3",
		"go1.15.2",
		"go1.15.1",
		"go1.15",
		"go1.15rc2",
		"go1.15rc1",
		"go1.15beta1",
	}, candidates)
	if diff != "" {
		t.Fatal(diff)
	}
}

func TestLookupLatest(t *testing.T) {
	t.Parallel()
	checkPrerequisites(t)

	assertCommand := func(t *testing.T, path, version string) bool {
		t.Helper()
		result, err := exec.Command(path, "version").Output()
		if err != nil {
			t.Fatal(err)
		}
		return bytes.Contains(result, []byte(" "+version))
	}

	cur := currentVersion(t)

	// current version results "go"
	path, err := LookupLatest(cur)
	if err != nil {
		t.Fatal(err)
	}
	if path != "go" {
		t.Fatalf("expected path: %q, got path: %q", "go", path)
	}
	assertCommand(t, path, versionRe.FindString(cur))

	path, err = LookupLatest("go1.18")
	if err != nil {
		t.Fatal(err)
	}
	assertCommand(t, path, "go1.18.5")

	path, err = LookupLatest("unknown")
	if !errors.Is(err, ErrInvalidVersion) {
		t.Fatalf("unknown: expected error: %v, got error: %v, got path: %s", ErrInvalidVersion, err, path)
	}
}
