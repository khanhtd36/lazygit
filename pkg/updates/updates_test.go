package updates

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/stretchr/testify/assert"
)

type fakeConfig struct {
	config.AppConfigurer
	version     string
	buildSource string
}

func (self *fakeConfig) GetVersion() string     { return self.version }
func (self *fakeConfig) GetBuildSource() string { return self.buildSource }
func (self *fakeConfig) SaveAppState() error    { return nil }

func newTestUpdater(apiBase, version string) *Updater {
	return &Updater{
		Common: common.NewDummyCommon(),
		Config: &fakeConfig{
			AppConfigurer: config.NewDummyAppConfig(),
			version:       version,
			buildSource:   "binaryRelease",
		},
		apiBase: apiBase,
		goos:    "linux",
		goarch:  "amd64",
	}
}

func TestParseVersion(t *testing.T) {
	scenarios := []struct {
		version  string
		expected [3]int
		ok       bool
	}{
		{"v0.65.2", [3]int{0, 65, 2}, true},
		{"0.65.2", [3]int{0, 65, 2}, true},
		{"v1.10.0", [3]int{1, 10, 0}, true},
		{"unversioned", [3]int{}, false},
		{"d964f4f", [3]int{}, false},
		{"v0.65", [3]int{}, false},
		{"v0.65.2-rc1", [3]int{}, false},
		{"v0.65.2-khanhtd36.1", [3]int{}, false},
		{"v0.65.-2", [3]int{}, false},
		{"", [3]int{}, false},
	}

	for _, s := range scenarios {
		t.Run(s.version, func(t *testing.T) {
			actual, ok := parseVersion(s.version)
			assert.Equal(t, s.ok, ok)
			assert.Equal(t, s.expected, actual)
		})
	}
}

func TestIsNewer(t *testing.T) {
	scenarios := []struct {
		name     string
		newV     string
		current  string
		expected bool
	}{
		{"patch bump", "v0.65.2", "v0.65.1", true},
		{"minor bump beats patch", "v0.66.0", "v0.65.9", true},
		{"numeric not lexicographic", "v0.10.0", "v0.9.0", true},
		{"same version", "v0.65.2", "v0.65.2", false},
		{"older", "v0.65.1", "v0.65.2", false},
		{"current is a source build", "v0.65.2", "unversioned", false},
		{"new is not comparable", "v0.65.2-rc1", "v0.65.1", false},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			assert.Equal(t, s.expected, isNewer(s.newV, s.current))
		})
	}
}

func TestArchiveName(t *testing.T) {
	scenarios := []struct {
		goos, goarch string
		expected     string
	}{
		{"linux", "amd64", "lazygit_0.65.2_linux_x86_64.tar.gz"},
		{"linux", "arm64", "lazygit_0.65.2_linux_arm64.tar.gz"},
		{"linux", "arm", "lazygit_0.65.2_linux_armv6.tar.gz"},
		{"linux", "386", "lazygit_0.65.2_linux_32-bit.tar.gz"},
		{"darwin", "arm64", "lazygit_0.65.2_darwin_arm64.tar.gz"},
		{"windows", "amd64", "lazygit_0.65.2_windows_x86_64.zip"},
	}

	for _, s := range scenarios {
		t.Run(s.goos+"/"+s.goarch, func(t *testing.T) {
			updater := newTestUpdater("", "0.65.1")
			updater.goos = s.goos
			updater.goarch = s.goarch
			assert.Equal(t, s.expected, updater.archiveName("v0.65.2"))
		})
	}
}

// Serves the latest release as the given tag with the given asset names.
func newReleaseServer(t *testing.T, tag string, assets ...string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/khanhtd36/lazygit/releases/latest" &&
			r.URL.Path != "/repos/khanhtd36/lazygit/releases/tags/"+tag {
			http.NotFound(w, r)
			return
		}

		assetsJSON := ""
		for i, name := range assets {
			if i > 0 {
				assetsJSON += ","
			}
			assetsJSON += fmt.Sprintf(`{"name":%q,"browser_download_url":"https://example.com/%s"}`, name, name)
		}
		fmt.Fprintf(w, `{"tag_name":%q,"html_url":"https://example.com/release","assets":[%s]}`, tag, assetsJSON)
	}))
	t.Cleanup(server.Close)

	return server
}

func TestCheckForNewUpdate(t *testing.T) {
	const asset = "lazygit_0.65.2_linux_x86_64.tar.gz"

	scenarios := []struct {
		name            string
		current         string
		tag             string
		assets          []string
		expectedVersion string
		expectedErr     string
	}{
		{
			name:            "newer release",
			current:         "0.65.1",
			tag:             "v0.65.2",
			assets:          []string{asset},
			expectedVersion: "v0.65.2",
		},
		{
			name:        "already on the latest release",
			current:     "0.65.2",
			tag:         "v0.65.2",
			assets:      []string{asset},
			expectedErr: "You already have the latest version",
		},
		{
			name:        "release has no asset for this platform",
			current:     "0.65.1",
			tag:         "v0.65.2",
			assets:      []string{"lazygit_0.65.2_windows_x86_64.zip"},
			expectedErr: "Could not find any binary at https://example.com/release",
		},
		{
			name:        "major version bump",
			current:     "0.65.2",
			tag:         "v1.0.0",
			assets:      []string{"lazygit_1.0.0_linux_x86_64.tar.gz"},
			expectedErr: "New version (v1.0.0) has non-backwards compatible changes compared to the current version (v0.65.2)",
		},
		{
			name:        "source build never sees updates",
			current:     "unversioned",
			tag:         "v0.65.2",
			assets:      []string{asset},
			expectedErr: "You already have the latest version",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			server := newReleaseServer(t, s.tag, s.assets...)
			updater := newTestUpdater(server.URL, s.current)

			version, err := updater.checkForNewUpdate()

			if s.expectedErr != "" {
				assert.EqualError(t, err, s.expectedErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, s.expectedVersion, version)
		})
	}
}

func TestCheckForNewUpdateServerError(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	_, err := newTestUpdater(server.URL, "0.65.1").checkForNewUpdate()

	assert.ErrorContains(t, err, "404")
}

func TestSkipUpdateCheck(t *testing.T) {
	scenarios := []struct {
		name        string
		version     string
		buildSource string
		expected    bool
	}{
		{"goreleaser build", "0.65.1", "binaryRelease", false},
		{"source build", "unversioned", "binaryRelease", true},
		{"other build source", "0.65.1", "buildBinary", true},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			updater := newTestUpdater("", s.version)
			updater.Config.(*fakeConfig).buildSource = s.buildSource

			assert.Equal(t, s.expected, updater.skipUpdateCheck())
		})
	}
}
