package max

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

var ErrUnsupportedUpdate = errors.New("unsupported update type")

func ParseWebhookUpdate(data []byte) (schemes.UpdateInterface, error) {
	var base schemes.Update
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, err
	}

	switch base.UpdateType {
	case schemes.TypeMessageCreated:
		var upd schemes.MessageCreatedUpdate
		if err := json.Unmarshal(data, &upd); err != nil {
			return nil, err
		}
		return &upd, nil
	case schemes.TypeMessageCallback:
		var upd schemes.MessageCallbackUpdate
		if err := json.Unmarshal(data, &upd); err != nil {
			return nil, err
		}
		return &upd, nil
	case schemes.TypeMessageEdited:
		var upd schemes.MessageEditedUpdate
		if err := json.Unmarshal(data, &upd); err != nil {
			return nil, err
		}
		return &upd, nil
	case schemes.TypeBotStarted:
		var upd schemes.BotStartedUpdate
		if err := json.Unmarshal(data, &upd); err != nil {
			return nil, err
		}
		return &upd, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedUpdate, base.UpdateType)
	}
}
