package max

import (
	"PaymentsBot/internal/db"
)

const garageGroupID int64 = -78319159032544

func (m *MaxService) SendGarageWorkReport(item db.GarageWorkLog) error {
	text := db.FormatGarageWorkMessage(item)
	return m.SendMessageInGroupID(garageGroupID, text)
}
