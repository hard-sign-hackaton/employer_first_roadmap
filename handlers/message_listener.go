package handlers

import (
	"efr_bot/utils"
	"strings"

	"github.com/max-messenger/maxbot"
)

func GlobalMessageListener(ctx maxbot.Context) error {
	userID := ctx.Update().UserID
	if strings.TrimSpace(ctx.Update().Message.Body.Text) == "/start" {
		return StartBot(ctx)
	}
	if strings.TrimSpace(ctx.Update().Message.Body.Text) == "/restart" {
		return RestartScenario(ctx)
	}
	if strings.TrimSpace(ctx.Update().Message.Body.Text) == "/roadmap" {
		return OpenCurrentRoadmap(ctx)
	}

	userState := utils.GetUserState(userID)
	if strings.HasPrefix(string(userState), "small_survey_") {
		return handleSmallSurveyMessage(ctx)
	}
	if strings.HasPrefix(string(userState), "big_survey_") {
		return handleBigSurveyMessage(ctx)
	}
	if strings.HasPrefix(string(userState), "roadmap_") {
		return handleRoadmapMessage(ctx)
	}

	return nil
}
