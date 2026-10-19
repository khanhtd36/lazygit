package updates

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-errors/errors"

	"github.com/kardianos/osext"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// The fork's releases are the only source of updates; upstream is never consulted.
const forkRepo = "khanhtd36/lazygit"

// Updater checks for updates and does updates
type Updater struct {
	*common.Common
	Config    config.AppConfigurer
	OSCommand *oscommands.OSCommand

	// These are fields so that tests can point the updater at a fake server and
	// pretend to run on another platform.
	apiBase string
	goos    string
	goarch  string
}

// Updaterer implements the check and update methods
type Updaterer interface {
	CheckForNewUpdate()
	Update()
}

type release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func (r release) assetURL(name string) (string, bool) {
	for _, asset := range r.Assets {
		if asset.Name == name {
			return asset.URL, true
		}
	}
	return "", false
}

// NewUpdater creates a new updater
func NewUpdater(cmn *common.Common, config config.AppConfigurer, osCommand *oscommands.OSCommand) (*Updater, error) {
	return &Updater{
		Common:    cmn,
		Config:    config,
		OSCommand: osCommand,
		apiBase:   "https://api.github.com",
		goos:      runtime.GOOS,
		goarch:    runtime.GOARCH,
	}, nil
}

// path is "latest" or "tags/<tag>"
func (u *Updater) getRelease(path string) (release, error) {
	var rel release

	req, err := http.NewRequest("GET", u.apiBase+"/repos/"+forkRepo+"/releases/"+path, nil)
	if err != nil {
		return rel, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return rel, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return rel, fmt.Errorf("could not fetch release information: %s", resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(&rel)
	return rel, err
}

// RecordLastUpdateCheck records last time an update check was performed
func (u *Updater) RecordLastUpdateCheck() error {
	u.Config.GetAppState().LastUpdateCheck = time.Now().Unix()
	return u.Config.SaveAppState()
}

// expecting version to be of the form `v12.34.56`
func (u *Updater) majorVersionDiffers(oldVersion, newVersion string) bool {
	if oldVersion == "unversioned" {
		return false
	}
	oldVersion = strings.TrimPrefix(oldVersion, "v")
	newVersion = strings.TrimPrefix(newVersion, "v")
	return strings.Split(oldVersion, ".")[0] != strings.Split(newVersion, ".")[0]
}

// Only plain `X.Y.Z` versions (optionally with a leading `v`) are comparable.
func parseVersion(version string) ([3]int, bool) {
	var result [3]int

	parts := strings.Split(strings.TrimPrefix(version, "v"), ".")
	if len(parts) != len(result) {
		return result, false
	}

	for i, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return [3]int{}, false
		}
		result[i] = number
	}

	return result, true
}

// Returns false if either version isn't comparable, so that e.g. a build from
// source never sees updates.
func isNewer(newVersion, currentVersion string) bool {
	newParts, ok := parseVersion(newVersion)
	if !ok {
		return false
	}
	currentParts, ok := parseVersion(currentVersion)
	if !ok {
		return false
	}

	for i := range newParts {
		if newParts[i] != currentParts[i] {
			return newParts[i] > currentParts[i]
		}
	}
	return false
}

func (u *Updater) currentVersion() string {
	versionNumber := u.Config.GetVersion()
	if versionNumber == "unversioned" {
		return versionNumber
	}

	return fmt.Sprintf("v%s", u.Config.GetVersion())
}

func (u *Updater) checkForNewUpdate() (string, error) {
	u.Log.Info("Checking for an updated version")
	currentVersion := u.currentVersion()
	if err := u.RecordLastUpdateCheck(); err != nil {
		return "", err
	}

	rel, err := u.getRelease("latest")
	if err != nil {
		return "", err
	}
	newVersion := rel.TagName
	u.Log.Info("Current version is " + currentVersion)
	u.Log.Info("New version is " + newVersion)

	if !isNewer(newVersion, currentVersion) {
		return "", errors.New(u.Tr.OnLatestVersionErr)
	}

	if u.majorVersionDiffers(currentVersion, newVersion) {
		errMessage := utils.ResolvePlaceholderString(
			u.Tr.MajorVersionErr, map[string]string{
				"newVersion":     newVersion,
				"currentVersion": currentVersion,
			},
		)
		return "", errors.New(errMessage)
	}

	if _, ok := rel.assetURL(u.archiveName(newVersion)); !ok {
		errMessage := utils.ResolvePlaceholderString(
			u.Tr.CouldNotFindBinaryErr, map[string]string{
				"url": rel.HTMLURL,
			},
		)

		return "", errors.New(errMessage)
	}
	u.Log.Info("Verified resource is available, ready to update")

	return newVersion, nil
}

// CheckForNewUpdate checks if there is an available update
func (u *Updater) CheckForNewUpdate(onFinish func(string, error) error, userRequested bool) {
	if !userRequested && u.skipUpdateCheck() {
		return
	}

	newVersion, err := u.checkForNewUpdate()
	if err = onFinish(newVersion, err); err != nil {
		u.Log.Error(err)
	}
}

func (u *Updater) skipUpdateCheck() bool {
	// will remove the check for windows after adding a manifest file asking for
	// the required permissions
	if u.goos == "windows" {
		u.Log.Info("Updating is currently not supported for windows until we can fix permission issues")
		return true
	}

	if u.Config.GetVersion() == "unversioned" {
		u.Log.Info("Current version is not built from an official release so we won't check for an update")
		return true
	}

	if u.Config.GetBuildSource() != "binaryRelease" {
		u.Log.Info("Binary is not built with the binaryRelease flag so we won't check for an update")
		return true
	}

	userConfig := u.UserConfig()
	if userConfig.Update.Method == "never" {
		u.Log.Info("Update method is set to never so we won't check for an update")
		return true
	}

	currentTimestamp := time.Now().Unix()
	lastUpdateCheck := u.Config.GetAppState().LastUpdateCheck
	days := userConfig.Update.Days

	if (currentTimestamp-lastUpdateCheck)/(60*60*24) < days {
		u.Log.Info("Last update was too recent so we won't check for an update")
		return true
	}

	return false
}

func (u *Updater) mappedArch() string {
	archMap := map[string]string{
		"386":   "32-bit",
		"amd64": "x86_64",
		"arm":   "armv6",
	}
	result, found := archMap[u.goarch]
	if found {
		return result
	}
	return u.goarch
}

func (u *Updater) zipExtension() string {
	if u.goos == "windows" {
		return "zip"
	}

	return "tar.gz"
}

// example: lazygit_0.65.2_windows_x86_64.zip
func (u *Updater) archiveName(version string) string {
	return fmt.Sprintf(
		"lazygit_%s_%s_%s.%s",
		strings.TrimPrefix(version, "v"),
		u.goos,
		u.mappedArch(),
		u.zipExtension(),
	)
}

// Update downloads the latest binary and replaces the current binary with it
func (u *Updater) Update(newVersion string) error {
	return u.update(newVersion)
}

func (u *Updater) update(newVersion string) error {
	rel, err := u.getRelease("tags/" + newVersion)
	if err != nil {
		return err
	}

	rawUrl, ok := rel.assetURL(u.archiveName(newVersion))
	if !ok {
		return errors.New(utils.ResolvePlaceholderString(
			u.Tr.CouldNotFindBinaryErr, map[string]string{
				"url": rel.HTMLURL,
			},
		))
	}

	u.Log.Info("Updating with url " + rawUrl)
	return u.downloadAndInstall(rawUrl)
}

func (u *Updater) downloadAndInstall(rawUrl string) error {
	configDir := u.Config.GetUserConfigDir()
	u.Log.Info("Download directory is " + configDir)

	zipPath := filepath.Join(configDir, "temp_lazygit."+u.zipExtension())
	u.Log.Info("Temp path to tarball/zip file is " + zipPath)

	// remove existing zip file
	if err := os.RemoveAll(zipPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Create the zip file
	out, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(rawUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error while trying to download latest lazygit: %s", resp.Status)
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	u.Log.Info("untarring tarball/unzipping zip file")
	err = u.OSCommand.Cmd.New([]string{"tar", "-zxf", zipPath, "lazygit"}).Run()
	if err != nil {
		return err
	}

	// the `tar` terminal cannot store things in a new location without permission
	// so it creates it in the current directory. As such our path is fairly simple.
	// You won't see it because it's gitignored.
	tempLazygitFilePath := "lazygit"

	u.Log.Infof("Path to temp binary is %s", tempLazygitFilePath)

	// get the path of the current binary
	binaryPath, err := osext.Executable()
	if err != nil {
		return err
	}
	u.Log.Info("Binary path is " + binaryPath)

	// Verify the main file exists
	if _, err := os.Stat(zipPath); err != nil {
		return err
	}

	// swap out the old binary for the new one
	err = os.Rename(tempLazygitFilePath, binaryPath)
	if err != nil {
		return err
	}
	u.Log.Info("Update complete!")

	return nil
}
