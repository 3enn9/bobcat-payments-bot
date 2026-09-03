package invoice

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// PDFToImages конвертирует PDF в PNG-страницы через pdftoppm (poppler-utils).
// Возвращает срез [][]byte, по одному на каждую страницу.
func PDFToImages(raw []byte, dpi int) ([][]byte, error) {
	if dpi <= 0 {
		dpi = 150
	}

	path, err := exec.LookPath("pdftoppm")
	if err != nil {
		return nil, fmt.Errorf("нужен pdftoppm (poppler-utils): %w", err)
	}

	dir, err := mkTempDir()
	if err != nil {
		return nil, err
	}
	defer removeDir(dir)

	prefix := filepath.Join(dir, "page")
	cmd := exec.Command(path,
		"-png",
		"-r", fmt.Sprintf("%d", dpi),
		"-", prefix,
	)
	cmd.Stdin = bytes.NewReader(raw)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("pdftoppm: %s: %w", strings.TrimSpace(string(out)), err)
	}

	files, err := filepath.Glob(prefix + "-*.png")
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("pdftoppm не создал PNG")
	}

	// pdftoppm нумерует файлы: page-1.png, page-2.png ...
	// Glob возвращает в алфавитном порядке, что совпадает с порядком страниц.
	var pages [][]byte
	for _, f := range files {
		data, err := readFile(f)
		if err != nil {
			return nil, err
		}
		pages = append(pages, data)
	}
	return pages, nil
}
