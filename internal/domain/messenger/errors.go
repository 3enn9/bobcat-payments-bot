package messenger

import "errors"

var ErrPinMessage = errors.New("error pin message")
var ErrSendMessage = errors.New("error send message")
var ErrGroupExists = errors.New("error group does not exists")
var ErrMessageEmpty = errors.New("error message is empty")
