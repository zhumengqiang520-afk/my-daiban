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
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetailOperationsDashboardMetrics(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	insertRetailOrgFixture(t, s)
	owner := &user.User{ID: 1, Username: "user1"}
	staff := &user.User{ID: 2, Username: "user2"}
	require.NoError(t, (&RetailMembership{OrgUnitID: 1, Username: "user2", JobTitle: "Sales Associate", Active: true}).Create(s, owner))
	now := time.Now().Truncate(time.Second)
	_, err := s.ID(1).Cols("due_date", "done", "done_at").Update(&Task{DueDate: now.Add(-time.Hour)})
	require.NoError(t, err)
	_, err = s.ID(2).Cols("due_date", "done", "done_at").Update(&Task{DueDate: now.Add(time.Hour)})
	require.NoError(t, err)
	overdue := &RetailTaskProfile{TaskID: 1, OrgUnitID: 1, Category: RetailTaskCategoryOpening, PrimaryAssigneeID: 2, ReviewerID: 1, EstimatedMinutes: 30, Source: RetailTaskSourceManual}
	require.NoError(t, overdue.Create(s, owner))
	completed := &RetailTaskProfile{TaskID: 2, OrgUnitID: 1, Category: RetailTaskCategoryClosing, PrimaryAssigneeID: 2, EstimatedMinutes: 20, Source: RetailTaskSourceManual}
	require.NoError(t, completed.Create(s, owner))
	_, err = StartRetailTask(s, 2, staff)
	require.NoError(t, err)
	_, err = SubmitRetailTask(s, 2, "done", nil, staff)
	require.NoError(t, err)

	day := retailDay(now)
	dashboard, err := GetRetailOperationsDashboard(s, 1, day, day, "", 0, now, owner)
	require.NoError(t, err)
	assert.Equal(t, 2, dashboard.Total)
	assert.Equal(t, 1, dashboard.Completed)
	assert.Equal(t, 1, dashboard.Overdue)
	assert.Equal(t, 1, dashboard.OnTimeCompleted)
	assert.Equal(t, 50, dashboard.CompletionRate)
	assert.Equal(t, 100, dashboard.OnTimeRate)
	require.Len(t, dashboard.OverdueTasks, 1)
	assert.Equal(t, int64(1), dashboard.OverdueTasks[0].TaskID)

	_, err = GetRetailOperationsDashboard(s, 1, day, day, "", 0, now, &user.User{ID: 10})
	var forbidden ErrGenericForbidden
	require.ErrorAs(t, err, &forbidden)
}
