package max

import (
	"encoding/json"
	"fmt"

	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

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
	default:
		return nil, fmt.Errorf("unsupported update type: %s", base.UpdateType)
	}
}
