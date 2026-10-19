package app

import (
	"fmt"
	"os"

	"github.com/go-errors/errors"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/updates"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Runs `lazygit update` and exits.
func selfUpdate(appConfig config.AppConfigurer, cmn *common.Common) {
	if err := runSelfUpdate(appConfig, cmn); err != nil {
		var upToDate *updates.UpToDateError
		if errors.As(err, &upToDate) {
			fmt.Println(err)
			os.Exit(0)
		}

		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Exit(0)
}

func runSelfUpdate(appConfig config.AppConfigurer, cmn *common.Common) error {
	updater, err := updates.NewUpdater(cmn, appConfig)
	if err != nil {
		return err
	}

	var newVersion string
	var checkErr error
	updater.CheckForNewUpdate(func(version string, err error) error {
		newVersion, checkErr = version, err
		return nil
	}, true)
	if checkErr != nil {
		return checkErr
	}

	fmt.Println(utils.ResolvePlaceholderString(
		cmn.Tr.UpdatingToVersion, map[string]string{"newVersion": newVersion},
	))
	if err := updater.Update(newVersion); err != nil {
		return errors.New(utils.ResolvePlaceholderString(
			cmn.Tr.UpdateFailedErr, map[string]string{"errMessage": err.Error()},
		))
	}

	fmt.Println(cmn.Tr.UpdateCompleted)
	return nil
}
