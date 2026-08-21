// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"testing"
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatchDueRetailTemplateSchedules(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	insertRetailOrgFixture(t, s)
	owner := &user.User{ID: 1, Username: "user1"}
	require.NoError(t, (&RetailMembership{OrgUnitID: 1, Username: "user2", JobTitle: "Sales Associate", Active: true}).Create(s, owner))
	template := &RetailTaskTemplate{
		OrgUnitID: 1, Name: "Daily closing", Title: "Close the store", Category: RetailTaskCategoryClosing,
		EstimatedMinutes: 30, Active: true, Checklist: []RetailTemplateChecklistItem{{Title: "Lock the doors", Required: true}},
	}
	require.NoError(t, template.Create(s, owner))
	firstRun := time.Now().Add(-time.Minute).Truncate(time.Second)
	schedule := &RetailTemplateSchedule{
		TemplateID: template.ID, TargetOrgUnitID: 1, ProjectID: 1, PrimaryAssigneeID: 2, ReviewerID: 1,
		Frequency: RetailScheduleDaily, Interval: 1, DueOffsetMinutes: 60, NextRunAt: firstRun, Active: true,
	}
	require.NoError(t, schedule.Create(s, owner))
	require.NoError(t, s.Commit())
	require.NoError(t, s.Close())

	dispatchDueRetailTemplateSchedulesAt(time.Now())

	verify := db.NewSession()
	defer verify.Close()
	updated, err := GetRetailTemplateScheduleByID(verify, schedule.ID)
	require.NoError(t, err)
	assert.WithinDuration(t, firstRun, updated.LastRunAt, time.Second)
	assert.WithinDuration(t, firstRun.AddDate(0, 0, 1), updated.NextRunAt, time.Second)
	dispatches := []*RetailTemplateDispatch{}
	require.NoError(t, verify.Where("idempotency_key = ?", retailScheduleIdempotencyKey(&RetailTemplateSchedule{ID: schedule.ID, NextRunAt: firstRun})).Find(&dispatches))
	require.Len(t, dispatches, 1)
	profile, err := GetRetailTaskProfileByTaskID(verify, dispatches[0].TaskID)
	require.NoError(t, err)
	assert.Equal(t, RetailTaskStatusAssigned, profile.Status)
	assert.Equal(t, 30, profile.EstimatedMinutes)

	dispatchDueRetailTemplateSchedulesAt(time.Now())
	count, err := verify.Where("template_version_id = ?", dispatches[0].TemplateVersionID).Count(&RetailTemplateDispatch{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "the next occurrence is not due and must not duplicate the task")
}

func TestNextRetailScheduleRunClampsShortMonths(t *testing.T) {
	oldTimezone := config.ServiceTimeZone.GetString()
	config.ServiceTimeZone.Set("Asia/Shanghai")
	defer config.ServiceTimeZone.Set(oldTimezone)
	location := config.GetTimeZone()
	schedule := &RetailTemplateSchedule{
		Frequency: RetailScheduleMonthly, Interval: 1, AnchorDay: 31,
		NextRunAt: time.Date(2027, time.January, 31, 8, 30, 0, 0, location),
	}
	february := nextRetailScheduleRun(schedule)
	assert.Equal(t, time.Date(2027, time.February, 28, 8, 30, 0, 0, location), february)
	schedule.NextRunAt = february
	march := nextRetailScheduleRun(schedule)
	assert.Equal(t, time.Date(2027, time.March, 31, 8, 30, 0, 0, location), march)
}
