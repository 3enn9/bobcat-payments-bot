package invoice

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func ExtractPDFText(raw []byte) (string, error) {
	path, err := exec.LookPath("pdftotext")
	if err != nil {
		return "", fmt.Errorf("нужен pdftotext (poppler-utils): %w", err)
	}

	cmd := exec.Command(path, "-layout", "-enc", "UTF-8", "-", "-")
	cmd.Stdin = bytes.NewReader(raw)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("pdftotext: %s", strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("pdftotext: %w", err)
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return "", fmt.Errorf("в PDF нет текстового слоя")
	}
	return text, nil
}
