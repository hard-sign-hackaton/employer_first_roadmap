package handlers

import (
	"github.com/max-messenger/maxbot"
)

func DemoHandler(ctx maxbot.Context) error {
	return ctx.Reply("Demo response")
}
