package gocmd

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/mod/modfile"
)

func TestFindGoMod(t *testing.T) {
	// t.Parallel()
	// move working directory to testdata to avoid hitting module's "go.mod"
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chdir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		err := os.Chdir(wd)
		if err != nil {
			t.Fatal(err)
		}
	})

	assert := func(t *testing.T, mod *modfile.File, path string) {
		t.Helper()

		for _, c := range mod.Module.Syntax.Comments.Before {
			if c.Token == "// "+path {
				return
			}
		}
		t.Log("unexpected mod file")
		data, _ := mod.Format()
		t.Fatal(string(data))
	}

	testCases := []struct {
		path string
		test func(t *testing.T, mod *modfile.File, err error)
	}{
		{
			path: "root",
			test: func(t *testing.T, mod *modfile.File, err error) {
				if err != nil {
					t.Fatal(err)
				}
				assert(t, mod, "go.mod")
			},
		}, {
			path: "root/a/b/c",
			test: func(t *testing.T, mod *modfile.File, err error) {
				if err != nil {
					t.Fatal(err)
				}
				assert(t, mod, "a/b/go.mod")
			},
		}, {
			path: "root/a/b/c/d",
			test: func(t *testing.T, mod *modfile.File, err error) {
				if err != nil {
					t.Fatal(err)
				}
				assert(t, mod, "a/b/c/d/go.mod")
			},
		}, {
			path: "invalid/x/y",
			test: func(t *testing.T, mod *modfile.File, err error) {
				if err == nil {
					t.Log("unexpected mod file")
					data, _ := mod.Format()
					t.Fatal(string(data))
				}
				if !errors.Is(err, fs.ErrNotExist) {
					t.Fatalf("unexpected error: %s", err)
				}
			},
		}, {
			path: "invalid/x/y/z/empty",
			test: func(t *testing.T, mod *modfile.File, err error) {
				if err == nil {
					t.Log("unexpected mod file")
					data, _ := mod.Format()
					t.Fatal(string(data))
				}
			},
		}, {
			path: "invalid/x/y/z/bad",
			test: func(t *testing.T, mod *modfile.File, err error) {
				if err == nil {
					t.Log("unexpected mod file")
					data, _ := mod.Format()
					t.Fatal(string(data))
				}
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()

			p := filepath.FromSlash(tc.path)
			mod, err := findGoMod(p)
			tc.test(t, mod, err)
		})
	}
}

func TestValidModuleGoVersion(t *testing.T) {
	t.Parallel()

	const dir = "testdata/root"

	err := ValidModuleGoVersion(dir, "go1.19")
	if err != nil {
		t.Fatal(err)
	}

	err = ValidModuleGoVersion(dir, "go1.19beta1")
	if err != nil {
		t.Fatal(err)
	}

	err = ValidModuleGoVersion(dir, "go1.18.5")
	if err == nil {
		t.Fatalf("unexpected success")
	}
	if !errors.Is(err, ErrUnexpectedGoVersion) {
		t.Fatalf("unexpected error: %s", err)
	}

	err = ValidModuleGoVersion(dir, "unknown")
	if err == nil {
		t.Fatalf("unexpected success")
	}
	if !errors.Is(err, ErrInvalidVersion) {
		t.Fatalf("unexpected error: %s", err)
	}
}
