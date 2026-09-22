package tag

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var Pull = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Pull all tags from a remote into the tags panel",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.CloneIntoRemote("origin")
		shell.CreateLightweightTag("new-tag", "HEAD")
		shell.RunCommand([]string{"git", "push", "origin", "tag", "new-tag"})
		shell.RunCommand([]string{"git", "tag", "-d", "new-tag"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Tags().
			Focus().
			IsEmpty().
			Press(keys.Universal.Pull).
			Tap(func() {
				t.ExpectPopup().Prompt().
					Title(Equals("Remote to pull tags from:")).
					InitialText(Equals("origin")).
					SuggestionLines(
						Contains("origin"),
					).
					Confirm()
			}).
			Lines(
				MatchesRegexp(`new-tag.*initial commit`).IsSelected(),
			)
	},
})
