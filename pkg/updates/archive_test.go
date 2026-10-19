package updates

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func writeZip(t *testing.T, path string, files map[string]string) {
	t.Helper()

	out, err := os.Create(path)
	assert.NoError(t, err)
	defer out.Close()

	writer := zip.NewWriter(out)
	for name, content := range files {
		entry, err := writer.Create(name)
		assert.NoError(t, err)
		_, err = entry.Write([]byte(content))
		assert.NoError(t, err)
	}
	assert.NoError(t, writer.Close())
}

func writeTarGz(t *testing.T, path string, files map[string]string) {
	t.Helper()

	out, err := os.Create(path)
	assert.NoError(t, err)
	defer out.Close()

	gzipWriter := gzip.NewWriter(out)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, content := range files {
		assert.NoError(t, tarWriter.WriteHeader(&tar.Header{
			Name:     name,
			Typeflag: tar.TypeReg,
			Mode:     0o644,
			Size:     int64(len(content)),
		}))
		_, err := tarWriter.Write([]byte(content))
		assert.NoError(t, err)
	}
	assert.NoError(t, tarWriter.Close())
	assert.NoError(t, gzipWriter.Close())
}

func TestExtractBinary(t *testing.T) {
	archiveContents := map[string]string{
		"LICENSE":   "license",
		"README.md": "readme",
		"lazygit":   "the binary",
	}

	scenarios := []struct {
		name  string
		write func(t *testing.T, path string, files map[string]string)
		ext   string
	}{
		{"zip", writeZip, ".zip"},
		{"tar.gz", writeTarGz, ".tar.gz"},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			dir := t.TempDir()
			archivePath := filepath.Join(dir, "lazygit"+s.ext)
			s.write(t, archivePath, archiveContents)
			destPath := filepath.Join(dir, "lazygit.new")

			assert.NoError(t, extractBinary(archivePath, "lazygit", destPath))

			content, err := os.ReadFile(destPath)
			assert.NoError(t, err)
			assert.Equal(t, "the binary", string(content))
		})

		t.Run(s.name+" finds the binary in a subdirectory", func(t *testing.T) {
			dir := t.TempDir()
			archivePath := filepath.Join(dir, "lazygit"+s.ext)
			s.write(t, archivePath, map[string]string{"lazygit_1.0.0/lazygit": "the binary"})
			destPath := filepath.Join(dir, "lazygit.new")

			assert.NoError(t, extractBinary(archivePath, "lazygit", destPath))

			content, err := os.ReadFile(destPath)
			assert.NoError(t, err)
			assert.Equal(t, "the binary", string(content))
		})

		t.Run(s.name+" without the binary", func(t *testing.T) {
			dir := t.TempDir()
			archivePath := filepath.Join(dir, "lazygit"+s.ext)
			s.write(t, archivePath, map[string]string{"LICENSE": "license"})

			err := extractBinary(archivePath, "lazygit", filepath.Join(dir, "lazygit.new"))

			assert.ErrorContains(t, err, "lazygit not found")
		})
	}
}

func TestVerifyChecksum(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "lazygit_1.0.0_linux_x86_64.tar.gz")
	assert.NoError(t, os.WriteFile(archivePath, []byte("archive contents"), 0o644))

	sum := sha256.Sum256([]byte("archive contents"))
	goodSum := hex.EncodeToString(sum[:])
	otherSum := hex.EncodeToString(make([]byte, sha256.Size))

	const name = "lazygit_1.0.0_linux_x86_64.tar.gz"

	scenarios := []struct {
		name        string
		checksums   string
		expectedErr string
	}{
		{
			name:      "matching checksum",
			checksums: otherSum + "  other_asset.zip\n" + goodSum + "  " + name + "\n",
		},
		{
			name:        "corrupted archive",
			checksums:   otherSum + "  " + name + "\n",
			expectedErr: "checksum mismatch",
		},
		{
			name:        "asset not listed",
			checksums:   goodSum + "  other_asset.zip\n",
			expectedErr: "no checksum listed for " + name,
		},
		{
			name:        "empty checksums file",
			checksums:   "",
			expectedErr: "no checksum listed for " + name,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			err := verifyChecksum(archivePath, name, s.checksums)

			if s.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, s.expectedErr)
			}
		})
	}
}
