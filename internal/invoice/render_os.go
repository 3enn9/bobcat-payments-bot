package invoice

import (
	"os"
)

func mkTempDir() (string, error) {
	return os.MkdirTemp("", "invoice-render-*")
}

func removeDir(dir string) {
	_ = os.RemoveAll(dir)
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
