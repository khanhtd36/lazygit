package updates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	assert.NoError(t, os.WriteFile(path, []byte(content), 0o755))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	assert.NoError(t, err)
	return string(content)
}

func TestMoveAsideAndReplace(t *testing.T) {
	t.Run("parks the old binary as .old", func(t *testing.T) {
		dir := t.TempDir()
		binaryPath := filepath.Join(dir, "lazygit.exe")
		newPath := binaryPath + ".new"
		writeFile(t, binaryPath, "old")
		writeFile(t, newPath, "new")

		assert.NoError(t, moveAsideAndReplace(binaryPath, newPath))

		assert.Equal(t, "new", readFile(t, binaryPath))
		assert.Equal(t, "old", readFile(t, binaryPath+".old"))
		assert.NoFileExists(t, newPath)
	})

	t.Run("replaces the .old left by an earlier update", func(t *testing.T) {
		dir := t.TempDir()
		binaryPath := filepath.Join(dir, "lazygit.exe")
		newPath := binaryPath + ".new"
		writeFile(t, binaryPath, "old")
		writeFile(t, newPath, "new")
		writeFile(t, binaryPath+".old", "older")

		assert.NoError(t, moveAsideAndReplace(binaryPath, newPath))

		assert.Equal(t, "new", readFile(t, binaryPath))
		assert.Equal(t, "old", readFile(t, binaryPath+".old"))
	})

	t.Run("parks under another name when .old can't be removed", func(t *testing.T) {
		dir := t.TempDir()
		binaryPath := filepath.Join(dir, "lazygit.exe")
		newPath := binaryPath + ".new"
		writeFile(t, binaryPath, "old")
		writeFile(t, newPath, "new")
		// A non-empty directory can't be removed, like a running executable
		assert.NoError(t, os.Mkdir(binaryPath+".old", 0o755))
		writeFile(t, filepath.Join(binaryPath+".old", "stuck"), "stuck")

		assert.NoError(t, moveAsideAndReplace(binaryPath, newPath))

		assert.Equal(t, "new", readFile(t, binaryPath))
		parked, err := filepath.Glob(binaryPath + ".old-*")
		assert.NoError(t, err)
		assert.Len(t, parked, 1)
		assert.Equal(t, "old", readFile(t, parked[0]))
	})

	t.Run("puts the old binary back when the new one can't be moved in", func(t *testing.T) {
		dir := t.TempDir()
		binaryPath := filepath.Join(dir, "lazygit.exe")
		writeFile(t, binaryPath, "old")

		err := moveAsideAndReplace(binaryPath, binaryPath+".missing")

		assert.Error(t, err)
		assert.Equal(t, "old", readFile(t, binaryPath))
	})
}

func TestRemoveParkedBinaries(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "lazygit.exe")
	writeFile(t, binaryPath, "current")
	writeFile(t, binaryPath+".old", "old")
	writeFile(t, binaryPath+".old-123", "older")
	writeFile(t, filepath.Join(dir, "other.exe.old"), "unrelated")

	removeParkedBinaries(binaryPath)

	assert.FileExists(t, binaryPath)
	assert.NoFileExists(t, binaryPath+".old")
	assert.NoFileExists(t, binaryPath+".old-123")
	assert.FileExists(t, filepath.Join(dir, "other.exe.old"))
}
