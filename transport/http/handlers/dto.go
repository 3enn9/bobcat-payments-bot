package handlers

type PaymentDTO struct {
	GroupID     int64
	Title       string
	Operation   string
	Description string
	Amount      float64
}
