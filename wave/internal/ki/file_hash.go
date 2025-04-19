package ki

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func getHashedFilenameFromPath(filePath string, originalFileName string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	buf := make([]byte, 32*1024)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			hash.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}

	return toOutputFileName(hash, originalFileName), nil
}

func getHashedFilenameFromBytes(content []byte, originalFileName string) string {
	hash := sha256.New()
	hash.Write(content)
	return toOutputFileName(hash, originalFileName)
}

func toOutputFileName(hash hash.Hash, originalFileName string) string {
	hashedSuffix := fmt.Sprintf("%x", hash.Sum(nil))[:12]
	ext := filepath.Ext(originalFileName)
	return fmt.Sprintf("%s_%s%s", strings.TrimSuffix(originalFileName, ext), hashedSuffix, ext)
}
