package updates

import "strings"

// A package manager that owns the lazygit binary. It must be the one to update
// it, or it would disagree with the binary about what is installed.
type packageManager struct {
	name string
	// The command that upgrades lazygit, or empty if there is no single
	// command that does.
	upgradeCommand string
}

// Returns the package manager that owns the binary at binaryPath, going by
// where it lives, or nil if lazygit can update it itself.
func managerOf(binaryPath string) *packageManager {
	path := strings.ToLower(strings.ReplaceAll(binaryPath, `\`, "/"))

	switch {
	case containsAny(path, "/cellar/", "/homebrew/", "/linuxbrew/"):
		return &packageManager{"Homebrew", "brew upgrade lazygit-khanhtd36"}
	case strings.Contains(path, "/microsoft/winget/"):
		return &packageManager{"winget", "winget upgrade khanhtd36.lazygit-khanhtd36"}
	case strings.Contains(path, "/go/bin/"):
		return &packageManager{"go install", "go install github.com/khanhtd36/lazygit@latest"}
	case strings.Contains(path, "/nix/store/"):
		return &packageManager{"Nix", ""}
	}

	return nil
}

func containsAny(s string, substrings ...string) bool {
	for _, substring := range substrings {
		if strings.Contains(s, substring) {
			return true
		}
	}
	return false
}
