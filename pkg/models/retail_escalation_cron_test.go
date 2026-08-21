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

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/notifications"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetailEscalationLevelsAreIdempotent(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	insertRetailOrgFixture(t, s)
	owner := &user.User{ID: 1, Username: "user1"}
	require.NoError(t, (&RetailMembership{OrgUnitID: 1, Username: "user2", JobTitle: "Sales Associate", ManagerUserID: 1, Active: true}).Create(s, owner))
	due := time.Now().Add(-3 * time.Hour).Truncate(time.Second)
	_, err := s.ID(1).Cols("due_date", "done").Update(&Task{DueDate: due, Done: false})
	require.NoError(t, err)
	profile := &RetailTaskProfile{
		TaskID: 1, OrgUnitID: 1, Category: RetailTaskCategoryOpening, PrimaryAssigneeID: 2,
		ReviewerID: 1, EstimatedMinutes: 30, Source: RetailTaskSourceManual,
	}
	require.NoError(t, profile.Create(s, owner))
	require.NoError(t, s.Commit())
	require.NoError(t, s.Close())

	now := due.Add(3 * time.Hour)
	sendRetailEscalationsAt(now)

	verify := db.NewSession()
	defer verify.Close()
	deliveries := []*RetailNotificationDelivery{}
	require.NoError(t, verify.Where("profile_id = ?", profile.ID).Asc("level").Find(&deliveries))
	require.Len(t, deliveries, 3)
	notificationCount, err := verify.Where("name = ?", "retail.task.escalation").Count(&notifications.DatabaseNotification{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), notificationCount)
	levels := map[string]bool{}
	for _, delivery := range deliveries {
		levels[delivery.Level] = true
	}
	assert.True(t, levels[RetailEscalationAssignee])
	assert.True(t, levels[RetailEscalationManager])
	assert.True(t, levels[RetailEscalationArea])

	sendRetailEscalationsAt(now.Add(time.Minute))
	count, err := verify.Where("profile_id = ?", profile.ID).Count(&RetailNotificationDelivery{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}
