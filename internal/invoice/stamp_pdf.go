package invoice

import (
	"bytes"
	"fmt"
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
	// other stamps. Relative scaling keeps the stamp proportional on A4 variants.
	scale := 0.184
	if strings.TrimSpace(data.SupplierINN) == "645499004617" ||
		(strings.TrimSpace(data.SupplierINN) == "" && SupplierFileCode(data.SupplierName, "") == "анн") {
		scale = 0.23
	}
	offsetX, offsetY := 32, 55
	if isEntrepreneurName(data.SupplierName) {
		offsetX, offsetY = 83, 105
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
