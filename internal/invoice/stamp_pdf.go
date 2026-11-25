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

	if stampCode == "sst" {
		painting, err := stampFiles.ReadFile("stamps/painting.png")
		if err != nil {
			return nil, fmt.Errorf("не удалось прочитать подпись бухгалтера: %w", err)
		}
		if x, y, ok := paintingPositionFromText(raw); ok {
			description := fmt.Sprintf("scale:0.130 rel, pos:bl, off:%d %d, rot:0, op:1", x, y)
			paintingWatermark, err := api.ImageWatermarkForReader(bytes.NewReader(painting), description, true, false, types.MILLIMETRES)
			if err != nil {
				return nil, fmt.Errorf("не удалось подготовить подпись бухгалтера: %w", err)
			}
			var signed bytes.Buffer
			if err := api.AddWatermarks(bytes.NewReader(stamped.Bytes()), &signed, []string{strconv.Itoa(pageCount)}, paintingWatermark, nil); err != nil {
				return nil, fmt.Errorf("не удалось наложить подпись бухгалтера на PDF: %w", err)
			}
			return signed.Bytes(), nil
		}
	}
	return stamped.Bytes(), nil
}

// stampPositionFromText locates a signature label on the last page and keeps
// the stamp aligned with it when invoice line items change the document layout.
func stampPositionFromText(raw []byte, entrepreneur bool) (int, int, bool) {
	page, ok := lastPDFTextPage(raw)
	if !ok {
		return 0, 0, false
	}
	if page.Width <= 0 || page.Height <= 0 {
		return 0, 0, false
	}

	anchors := []string{"руководитель", "директор", "подпись", "м.п."}
	if entrepreneur {
		// The buyer block can also contain "Индивидуальный предприниматель".
		// Use a matching label only in the lower signature area of the last page,
		// and choose the lowest occurrence when the phrase appears more than once.
		lowestSignatureY := page.Height * 0.55
		selected := -1
		for i, word := range page.Words {
			if normalizedPDFWord(word.Text) != "предприниматель" || word.YMax < lowestSignatureY {
				continue
			}
			if selected < 0 || word.YMax > page.Words[selected].YMax {
				selected = i
			}
		}
		if selected < 0 {
			return 0, 0, false
		}
		word := page.Words[selected]
		const pointsPerMM = 72.0 / 25.4
		xMM := word.XMin / pointsPerMM
		yFromBottomMM := (page.Height - word.YMax) / pointsPerMM
		// Keep the same placement relative to the supplier's signature label
		// as the established entrepreneur stamp placement.
		return int(xMM + 70), int(yFromBottomMM - 25), true
	}
	const pointsPerMM = 72.0 / 25.4
	for _, word := range page.Words {
		text := normalizedPDFWord(word.Text)
		for _, anchor := range anchors {
			if text != anchor {
				continue
			}
			xMM := word.XMin / pointsPerMM
			yFromBottomMM := (page.Height - word.YMax) / pointsPerMM
			// Lower company stamps so their center aligns with the signature label
			// instead of covering the invoice terms immediately above it.
			return int(xMM + 20), int(yFromBottomMM - 23), true
		}
	}
	return 0, 0, false
}

func paintingPositionFromText(raw []byte) (int, int, bool) {
	page, ok := lastPDFTextPage(raw)
	if !ok || page.Width <= 0 || page.Height <= 0 {
		return 0, 0, false
	}
	const pointsPerMM = 72.0 / 25.4
	for _, word := range page.Words {
		if normalizedPDFWord(word.Text) != "бухгалтер" {
			continue
		}
		xMM := word.XMin / pointsPerMM
		yFromBottomMM := (page.Height - word.YMax) / pointsPerMM
		// Position the autograph over the accountant's signature line, just
		// after the label. Its width is about 27 mm on an A4 page.
		return int(xMM + 18), int(yFromBottomMM - 7), true
	}
	return 0, 0, false
}

type pdfTextPage struct {
	Width  float64 `xml:"width,attr"`
	Height float64 `xml:"height,attr"`
	Words  []struct {
		XMin float64 `xml:"xMin,attr"`
		YMax float64 `xml:"yMax,attr"`
		Text string  `xml:",chardata"`
	} `xml:"word"`
}

func lastPDFTextPage(raw []byte) (pdfTextPage, bool) {
	path, err := exec.LookPath("pdftotext")
	if err != nil {
		return pdfTextPage{}, false
	}
	cmd := exec.Command(path, "-bbox", "-", "-")
	cmd.Stdin = bytes.NewReader(raw)
	out, err := cmd.Output()
	if err != nil {
		return pdfTextPage{}, false
	}

	var doc struct {
		Pages []pdfTextPage `xml:"body>doc>page"`
	}
	if err := xml.Unmarshal(out, &doc); err != nil || len(doc.Pages) == 0 {
		return pdfTextPage{}, false
	}
	return doc.Pages[len(doc.Pages)-1], true
}

func normalizedPDFWord(word string) string {
	text := strings.ToLower(strings.TrimSpace(word))
	return strings.Trim(text, ":.,;()")
}
