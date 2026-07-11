package max

import (
	"PaymentsBot/internal/payments"
	"context"
	"fmt"
	"log"
	"strings"
	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)


type MaxService struct{
	bot   *maxbot.Api
	Chats map[string]int64
	updateCh <-chan schemes.UpdateInterface
	payments *payments.PaymentsService
}


func (m *MaxService) GetGroupID(groupName string) (int64, error){

	return 0, nil
}

func (m *MaxService) SendMessageInGroupID(chatID int64, message string) error{

	return nil
}
func (m *MaxService) SendMessageInGroupName(nameGroup string, message string) error{

	return nil
}

func (m *MaxService) Updates(u tgbotapi.Update, ctx context.Context) error{
	for {
	select {
	case update, ok := <-m.updateCh:
		if !ok {
			return 
		}
		log.Printf("Received: %#v", update)
		switch upd := update.(type) {
		case *schemes.MessageCreatedUpdate:
			command := upd.Message.Body.Text
			command = strings.Split(command, " ")[0]

			if someFunc, ok := m.Commands[command]; ok {
				go func(upd *schemes.MessageCreatedUpdate) {
					msg := someFunc(upd)
					err := m.bot.Messages.Send(ctx, msg)
					if err != nil {
						fmt.Printf("error send message %v", err)
					}
				}(upd)
			}
		default:
			log.Printf("Unknown type: %#v", upd)
		}
	case <-ctx.Done():
		return nil
	}
	}
}