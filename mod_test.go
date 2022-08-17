package gocmd

import (
	"errors"
	"io/fs"
	"os"
	"testing"
)

func chdir(t *testing.T, path string) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chdir(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		err := os.Chdir(wd)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestValidModuleGoVersion(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		// move working directory to testdata to avoid hitting module's "go.mod"
		chdir(t, "testdata/valid")

		err := ValidModuleGoVersion("go1.19")
		if err != nil {
			t.Fatal(err)
		}

		err = ValidModuleGoVersion("go1.19beta1")
		if err != nil {
			t.Fatal(err)
		}

		err = ValidModuleGoVersion("go1.18.5")
		if err == nil {
			t.Fatalf("unexpected success")
		}
		if !errors.Is(err, ErrUnexpectedGoVersion) {
			t.Fatalf("unexpected error: %s", err)
		}

		err = ValidModuleGoVersion("unknown")
		if err == nil {
			t.Fatalf("unexpected success")
		}
		if !errors.Is(err, ErrInvalidVersion) {
			t.Fatalf("unexpected error: %s", err)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		chdir(t, "testdata/invalid")

		err := ValidModuleGoVersion("go1.19")
		if err == nil {
			t.Fatal("unexpected success")
		}
	})

	t.Run("empty", func(t *testing.T) {
		chdir(t, "testdata/empty")

		err := ValidModuleGoVersion("go1.19")
		if err == nil {
			t.Fatal("unexpected success")
		}
	})

	t.Run("not found", func(t *testing.T) {
		chdir(t, t.TempDir())

		err := ValidModuleGoVersion("go1.19")
		if err == nil {
			t.Fatal("unexpected success")
		}
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("unexpected error: %s", err)
		}
	})
}
