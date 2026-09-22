package tag

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForcePull = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Force-pull tags when a local tag has diverged from the remote's",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("first commit")
		shell.CreateLightweightTag("new-tag", "HEAD")
		shell.CloneIntoRemote("origin")

		shell.EmptyCommit("second commit")
		shell.RunCommand([]string{"git", "tag", "-f", "new-tag", "HEAD"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Tags().
			Focus().
			Lines(
				MatchesRegexp(`new-tag.*second commit`).IsSelected(),
			).
			Press(keys.Universal.Pull).
			Tap(func() {
				t.ExpectPopup().Prompt().
					Title(Equals("Remote to pull tags from:")).
					InitialText(Equals("origin")).
					Confirm()
			}).
			Tap(func() {
				t.ExpectPopup().Confirmation().
					Title(Equals("Force pull tags")).
					Content(Contains("Press <esc> to cancel, or <enter> to overwrite them with the remote's tags.")).
					Confirm()
			}).
			Lines(
				MatchesRegexp(`new-tag.*first commit`).IsSelected(),
			)
	},
})
