package invoice

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// StampSupplierPDF overlays the supplier's transparent stamp on the last page
// and keeps the original PDF pages and text intact.
func StampSupplierPDF(raw []byte, data *PDFData) ([]byte, error) {
	api.DisableConfigDir()
	if len(raw) == 0 {
		return nil, fmt.Errorf("пустой PDF")
	}
	if data == nil {
		return nil, fmt.Errorf("данные счета не переданы")
	}
	stamp := supplierStamp(data.SupplierName, data.SupplierINN)
	if len(stamp) == 0 {
		return nil, fmt.Errorf("для поставщика %q не найдена печать", data.SupplierName)
	}

	pageCount, err := api.PageCount(bytes.NewReader(raw), nil)
	if err != nil {
		return nil, fmt.Errorf("не удалось определить страницы PDF: %w", err)
	}
	if pageCount < 1 {
		return nil, fmt.Errorf("в PDF нет страниц")
	}

	// Match the sizes used in generated invoices: 48 mm for ANN and 80% for
	// other stamps. ADN, SAN and SST are enlarged by 20% from that size.
	scale := 0.184
	stampCode := supplierStampCode(data.SupplierName, data.SupplierINN)
	if stampCode == "ann" {
		scale = 0.23
	} else if isStamp20PercentLarger(stampCode) {
		scale *= 1.2
	}
	offsetX, offsetY := 32, 55
	if isEntrepreneurName(data.SupplierName) {
		offsetX, offsetY = 83, 105
	}
	if x, y, ok := stampPositionFromText(raw, isEntrepreneurName(data.SupplierName)); ok {
		offsetX, offsetY = x, y
	}
	description := fmt.Sprintf("scale:%.3f rel, pos:bl, off:%d %d, rot:0, op:1", scale, offsetX, offsetY)
	watermark, err := api.ImageWatermarkForReader(bytes.NewReader(stamp), description, true, false, types.MILLIMETRES)
	if err != nil {
		return nil, fmt.Errorf("не удалось подготовить печать: %w", err)
	}

	var stamped bytes.Buffer
	if err := api.AddWatermarks(bytes.NewReader(raw), &stamped, []string{strconv.Itoa(pageCount)}, watermark, nil); err != nil {
		return nil, fmt.Errorf("не удалось наложить печать на PDF: %w", err)
	}
	return stamped.Bytes(), nil
}

// stampPositionFromText locates a signature label on the last page and keeps
// the stamp aligned with it when invoice line items change the document layout.
func stampPositionFromText(raw []byte, entrepreneur bool) (int, int, bool) {
	path, err := exec.LookPath("pdftotext")
	if err != nil {
		return 0, 0, false
	}
	cmd := exec.Command(path, "-bbox", "-", "-")
	cmd.Stdin = bytes.NewReader(raw)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, false
	}

	var doc struct {
		Pages []struct {
			Width  float64 `xml:"width,attr"`
			Height float64 `xml:"height,attr"`
			Words  []struct {
				XMin float64 `xml:"xMin,attr"`
				YMax float64 `xml:"yMax,attr"`
				Text string  `xml:",chardata"`
			} `xml:"word"`
		} `xml:"body>doc>page"`
	}
	if err := xml.Unmarshal(out, &doc); err != nil || len(doc.Pages) == 0 {
		return 0, 0, false
	}
	page := doc.Pages[len(doc.Pages)-1]
	if page.Width <= 0 || page.Height <= 0 {
		return 0, 0, false
	}

	anchors := []string{"руководитель", "директор", "подпись", "м.п."}
	if entrepreneur {
		anchors = []string{"предприниматель"}
	}
	const pointsPerMM = 72.0 / 25.4
	for _, word := range page.Words {
		text := strings.ToLower(strings.TrimSpace(word.Text))
		text = strings.Trim(text, ":.,;()")
		for _, anchor := range anchors {
			if text != anchor {
				continue
			}
			xMM := word.XMin / pointsPerMM
			yFromBottomMM := (page.Height - word.YMax) / pointsPerMM
			if entrepreneur {
				// Keep the same placement relative to the label as the existing
				// ANN placement (83 mm X, 105 mm Y on the supplied invoice).
				return int(xMM + 70), int(yFromBottomMM - 25), true
			}
			// Place company stamps over the signature area to the right of its label.
			return int(xMM + 20), int(yFromBottomMM - 3), true
		}
	}
	return 0, 0, false
}
