package reminders

import (
	"context"
	"fmt"

	maxapi "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
)

// MaxSender is the production Sender implementation backed by MAX Bot API.
type MaxSender struct {
	api *maxapi.Api
}

func NewMaxSender(api *maxapi.Api) *MaxSender {
	return &MaxSender{api: api}
}

func (s *MaxSender) Send(ctx context.Context, userID int64, text string) error {
	if s.api == nil {
		return fmt.Errorf("MAX API client is not configured")
	}
	keyboard := model.NewKeyboard()
	keyboard.AddRow().AddCallBack("Открыть roadmap", "/open_roadmap")
	_, err := s.api.Messages.Send(ctx, maxapi.NewMessage().SetUser(userID).SetText(text).AddKeyboard(keyboard))
	return err
}
