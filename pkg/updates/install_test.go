package updates

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManagerOf(t *testing.T) {
	scenarios := []struct {
		path     string
		expected string
	}{
		{`C:\Users\me\AppData\Local\Microsoft\WinGet\Packages\khanhtd36.lazygit-khanhtd36_Microsoft.Winget.Source_8wekyb3d8bbwe\lazygit.exe`, "winget"},
		{`C:\Users\me\AppData\Local\Microsoft\WinGet\Links\lazygit.exe`, "winget"},
		{"/opt/homebrew/Cellar/lazygit-khanhtd36/0.65.2/bin/lazygit", "Homebrew"},
		{"/home/linuxbrew/.linuxbrew/bin/lazygit", "Homebrew"},
		{"/usr/local/Cellar/lazygit-khanhtd36/0.65.2/bin/lazygit", "Homebrew"},
		{"/home/me/go/bin/lazygit", "go install"},
		{`C:\Users\me\go\bin\lazygit.exe`, "go install"},
		{"/nix/store/abc123-lazygit-0.65.2/bin/lazygit", "Nix"},
		{"/home/me/.local/bin/lazygit", ""},
		{`C:\Users\me\AppData\Local\Programs\lazygit\lazygit.exe`, ""},
		{"/usr/local/bin/lazygit", ""},
	}

	for _, s := range scenarios {
		t.Run(s.path, func(t *testing.T) {
			manager := managerOf(s.path)

			if s.expected == "" {
				assert.Nil(t, manager)
			} else {
				assert.NotNil(t, manager)
				assert.Equal(t, s.expected, manager.name)
			}
		})
	}
}

func TestManagedInstallErr(t *testing.T) {
	updater := newTestUpdater("", "0.65.1")

	assert.NoError(t, updater.managedInstallErr("/home/me/.local/bin/lazygit"))
	assert.EqualError(t,
		updater.managedInstallErr("/opt/homebrew/Cellar/lazygit-khanhtd36/0.65.2/bin/lazygit"),
		"lazygit was installed with Homebrew, which has to update it. Run: brew upgrade lazygit-khanhtd36")
	assert.EqualError(t,
		updater.managedInstallErr("/nix/store/abc123-lazygit-0.65.2/bin/lazygit"),
		"lazygit was installed with Nix, which has to update it.")
}
