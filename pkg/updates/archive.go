package updates

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

// Fails unless checksums (the contents of a goreleaser checksums.txt) lists
// the sha256 of the file at filePath under assetName.
func verifyChecksum(filePath, assetName, checksums string) error {
	expected := ""
	for _, line := range strings.Split(checksums, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == assetName {
			expected = strings.ToLower(fields[0])
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("no checksum listed for %s", assetName)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	if actual := hex.EncodeToString(hash.Sum(nil)); actual != expected {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", assetName, expected, actual)
	}

	return nil
}

// Writes the file called binaryName found anywhere in the .zip or .tar.gz
// archive to destPath.
func extractBinary(archivePath, binaryName, destPath string) error {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractFromZip(archivePath, binaryName, destPath)
	}
	return extractFromTarGz(archivePath, binaryName, destPath)
}

func extractFromZip(archivePath, binaryName, destPath string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() || path.Base(entry.Name) != binaryName {
			continue
		}

		src, err := entry.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		return writeExecutable(destPath, src)
	}

	return fmt.Errorf("%s not found in %s", binaryName, archivePath)
}

func extractFromTarGz(archivePath, binaryName, destPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%s not found in %s", binaryName, archivePath)
		}
		if err != nil {
			return err
		}

		if header.Typeflag == tar.TypeReg && path.Base(header.Name) == binaryName {
			return writeExecutable(destPath, tarReader)
		}
	}
}

func writeExecutable(destPath string, src io.Reader) error {
	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, src); err != nil {
		out.Close()
		return err
	}

	return out.Close()
}
