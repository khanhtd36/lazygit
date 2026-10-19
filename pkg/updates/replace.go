package updates

import "os"

// Swaps the binary at newPath in for the one at binaryPath.
func replaceBinary(binaryPath, newPath string) error {
	return os.Rename(newPath, binaryPath)
}
