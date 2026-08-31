//go:build unix

package platform_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/aayush/torrcli/internal/platform"
)

func TestAcquireLockPreventsSecondOwner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "torrd.lock")
	first, err := platform.AcquireLock(path)
	if err != nil {
		t.Fatalf("acquire first lock: %v", err)
	}
	defer first.Close()

	second, err := platform.AcquireLock(path)
	if second != nil {
		second.Close()
		t.Fatal("second lock acquisition unexpectedly succeeded")
	}
	if !errors.Is(err, platform.ErrLocked) {
		t.Fatalf("second lock error = %v, want ErrLocked", err)
	}
}
