// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"context"
	"strconv"
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/cron"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/log"
)

func RegisterRetailTemplateScheduleCron() {
	if !config.RetailEnabled.GetBool() {
		return
	}
	if err := cron.Schedule("* * * * *", func() { dispatchDueRetailTemplateSchedulesAt(time.Now()) }); err != nil {
		log.Errorf("Could not register retail template schedule cron: %s", err)
	}
}

func dispatchDueRetailTemplateSchedulesAt(now time.Time) {
	lookup := db.NewSession()
	schedules := []*RetailTemplateSchedule{}
	err := lookup.Where("active = ?", true).And("next_run_at <= ?", now).Asc("next_run_at", "id").Limit(100).Find(&schedules)
	_ = lookup.Rollback()
	_ = lookup.Close()
	if err != nil {
		log.Errorf("Could not load due retail template schedules: %s", err)
		return
	}
	for _, schedule := range schedules {
		dispatchRetailTemplateSchedule(schedule, now)
	}
}

func dispatchRetailTemplateSchedule(schedule *RetailTemplateSchedule, now time.Time) {
	s := db.NewSession()
	defer s.Close()
	current, err := GetRetailTemplateScheduleByID(s, schedule.ID)
	if err != nil || !current.Active || current.NextRunAt.After(now) {
		_ = s.Rollback()
		return
	}
	creator, err := retailScheduleCreator(s, current)
	if err != nil {
		_ = s.Rollback()
		log.Errorf("Could not load creator for retail template schedule %d: %s", current.ID, err)
		return
	}
	_, err = DispatchRetailTaskTemplate(s, current.TemplateID, RetailTemplateDispatchInput{
		TargetOrgUnitID: current.TargetOrgUnitID, ProjectID: current.ProjectID, PrimaryAssigneeID: current.PrimaryAssigneeID,
		ReviewerID: current.ReviewerID, ScheduledFor: current.NextRunAt,
		DueDate:        current.NextRunAt.Add(time.Duration(current.DueOffsetMinutes) * time.Minute),
		IdempotencyKey: retailScheduleIdempotencyKey(current),
	}, creator)
	if err != nil {
		_ = s.Rollback()
		events.CleanupPending(s)
		log.Errorf("Could not dispatch retail template schedule %d: %s", current.ID, err)
		return
	}
	lastRun := current.NextRunAt
	nextRun := nextRetailScheduleRun(current)
	if _, err = s.ID(current.ID).Cols("last_run_at", "next_run_at").Update(&RetailTemplateSchedule{LastRunAt: lastRun, NextRunAt: nextRun}); err != nil {
		_ = s.Rollback()
		events.CleanupPending(s)
		log.Errorf("Could not advance retail template schedule %d: %s", current.ID, err)
		return
	}
	if err = s.Commit(); err != nil {
		events.CleanupPending(s)
		log.Errorf("Could not commit retail template schedule %d: %s", current.ID, err)
		return
	}
	events.DispatchPending(context.Background(), s)
}

func retailScheduleIdempotencyKey(schedule *RetailTemplateSchedule) string {
	return "retail-schedule:" + strconv.FormatInt(schedule.ID, 10) + ":" + schedule.NextRunAt.UTC().Format(time.RFC3339Nano)
}
