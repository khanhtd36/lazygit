package updates

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kardianos/osext"
)

// Swaps the binary at newPath in for the one at binaryPath.
func replaceBinary(binaryPath, newPath string) error {
	if runtime.GOOS == "windows" {
		return moveAsideAndReplace(binaryPath, newPath)
	}

	// Renaming over a running executable is fine on Unix.
	return os.Rename(newPath, binaryPath)
}

// Windows can't overwrite or delete a running executable, but it can rename
// one. So park the current binary under a different name and move the new
// one into its place; Cleanup removes the parked file on a later start.
func moveAsideAndReplace(binaryPath, newPath string) error {
	parkedPath := binaryPath + ".old"
	// The copy parked by an earlier update may still be running in another
	// session, in which case it can't be removed and we need another name.
	if err := os.Remove(parkedPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		parkedPath = fmt.Sprintf("%s.old-%d", binaryPath, time.Now().UnixNano())
	}

	if err := os.Rename(binaryPath, parkedPath); err != nil {
		return err
	}

	if err := os.Rename(newPath, binaryPath); err != nil {
		_ = os.Rename(parkedPath, binaryPath)
		return err
	}

	return nil
}

// Cleanup removes the binaries that earlier updates parked next to the running
// one. Ones that are still running can't be removed and are left for a later
// start.
func Cleanup() {
	if runtime.GOOS != "windows" {
		return
	}

	if binaryPath, err := osext.Executable(); err == nil {
		removeParkedBinaries(binaryPath)
	}
}

func removeParkedBinaries(binaryPath string) {
	entries, err := os.ReadDir(filepath.Dir(binaryPath))
	if err != nil {
		return
	}

	parkedPrefix := filepath.Base(binaryPath) + ".old"
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), parkedPrefix) {
			_ = os.Remove(filepath.Join(filepath.Dir(binaryPath), entry.Name()))
		}
	}
}
