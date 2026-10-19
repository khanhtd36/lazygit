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

	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// The fork's releases are the only source of updates; upstream is never consulted.
const forkRepo = "khanhtd36/lazygit"

// UpToDateError is returned by a check that finds nothing newer than the
// running version, which callers may not want to treat as a failure.
type UpToDateError struct {
	message string
}

func (e *UpToDateError) Error() string {
	return e.message
}

// Updater checks for updates and does updates
type Updater struct {
	*common.Common
	Config config.AppConfigurer

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
func NewUpdater(cmn *common.Common, config config.AppConfigurer) (*Updater, error) {
	return &Updater{
		Common:  cmn,
		Config:  config,
		apiBase: "https://api.github.com",
		goos:    runtime.GOOS,
		goarch:  runtime.GOARCH,
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
		return "", &UpToDateError{u.Tr.OnLatestVersionErr}
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

	if err := u.checkNotManaged(); err != nil {
		return "", err
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
	if err := u.checkNotManaged(); err != nil {
		return err
	}

	rel, err := u.getRelease("tags/" + newVersion)
	if err != nil {
		return err
	}

	archiveName := u.archiveName(newVersion)
	archiveURL, ok := rel.assetURL(archiveName)
	if !ok {
		return errors.New(utils.ResolvePlaceholderString(
			u.Tr.CouldNotFindBinaryErr, map[string]string{
				"url": rel.HTMLURL,
			},
		))
	}

	checksumsURL, ok := rel.assetURL("checksums.txt")
	if !ok {
		return fmt.Errorf("release %s has no checksums.txt", newVersion)
	}

	binaryPath, err := osext.Executable()
	if err != nil {
		return err
	}
	u.Log.Info("Binary path is " + binaryPath)

	tempDir, err := os.MkdirTemp("", "lazygit-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, archiveName)
	u.Log.Info("Downloading " + archiveURL)
	if err := download(archiveURL, archivePath); err != nil {
		return err
	}

	checksumsPath := filepath.Join(tempDir, "checksums.txt")
	if err := download(checksumsURL, checksumsPath); err != nil {
		return err
	}
	checksums, err := os.ReadFile(checksumsPath)
	if err != nil {
		return err
	}
	if err := verifyChecksum(archivePath, archiveName, string(checksums)); err != nil {
		return err
	}

	// Next to the binary, so that replacing it never crosses file systems
	newBinaryPath := binaryPath + ".new"
	if err := extractBinary(archivePath, u.binaryName(), newBinaryPath); err != nil {
		return err
	}

	if err := replaceBinary(binaryPath, newBinaryPath); err != nil {
		os.Remove(newBinaryPath)
		return err
	}
	u.Log.Info("Update complete!")

	return nil
}

func (u *Updater) binaryName() string {
	if u.goos == "windows" {
		return "lazygit.exe"
	}

	return "lazygit"
}

func download(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error while trying to download %s: %s", url, resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		return err
	}

	return out.Close()
}

// Fails with a message saying how to update, if the running binary belongs to
// a package manager.
func (u *Updater) checkNotManaged() error {
	binaryPath, err := osext.Executable()
	if err != nil {
		return err
	}

	return u.managedInstallErr(binaryPath)
}

func (u *Updater) managedInstallErr(binaryPath string) error {
	manager := managerOf(binaryPath)
	if manager == nil {
		return nil
	}

	message := u.Tr.ManagedInstallNoCmdErr
	if manager.upgradeCommand != "" {
		message = u.Tr.ManagedInstallErr
	}

	return errors.New(utils.ResolvePlaceholderString(message, map[string]string{
		"manager": manager.name,
		"command": manager.upgradeCommand,
	}))
}
