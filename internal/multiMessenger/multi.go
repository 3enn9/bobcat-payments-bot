package multi

import (
	"PaymentsBot/internal/usecase"
	"fmt"
)

type MultiMessenger struct {
	messengers []usecase.SendMessanger
}

func NewMultiMessenger(messengers []usecase.SendMessanger) *MultiMessenger {
	return &MultiMessenger{messengers: messengers}
}

func (m *MultiMessenger) SendMessageInGroupName(groupName string, message string) error {
	for _, messenger := range m.messengers {
		err := messenger.SendMessageInGroupName(groupName, message)
		if err != nil {
			fmt.Printf("error send messageInGroupName")
		}
	}
	return nil
}
