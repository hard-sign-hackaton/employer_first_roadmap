package handlers

import (
	"efr_bot/reminders"
	"efr_bot/services"
)

type Services struct {
	Profile    services.ProfileService
	Reference  services.ReferenceService
	Trajectory services.TrajectoryService
	Roadmap    services.RoadmapService
	Admission  services.AdmissionService
	Reminder   *reminders.Service
}

var app Services

func Configure(services Services) { app = services }
