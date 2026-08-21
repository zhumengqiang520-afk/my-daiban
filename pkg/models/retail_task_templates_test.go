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

func TestRetailTaskTemplateDispatchAndWorkload(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	insertRetailOrgFixture(t, s)
	owner := &user.User{ID: 1, Username: "user1"}
	staff := &user.User{ID: 2, Username: "user2"}
	outsider := &user.User{ID: 3, Username: "user3"}
	require.NoError(t, (&RetailMembership{OrgUnitID: 1, Username: "user2", JobTitle: "Sales Associate", Active: true}).Create(s, owner))

	template := &RetailTaskTemplate{
		OrgUnitID: 1, Name: "Daily opening", Title: "Open the store", Category: RetailTaskCategoryOpening,
		EstimatedMinutes: 45, EvidenceRequired: false, Active: true,
		Checklist: []RetailTemplateChecklistItem{{Title: "Check sample beds", Required: true, Position: 1}},
	}
	can, err := template.CanCreate(s, owner)
	require.NoError(t, err)
	assert.True(t, can)
	can, err = template.CanCreate(s, staff)
	require.NoError(t, err)
	assert.False(t, can)
	require.NoError(t, template.Create(s, owner))
	assert.Equal(t, 1, template.CurrentVersion)

	template.Title = "Open store and displays"
	template.EstimatedMinutes = 60
	require.NoError(t, template.Update(s, owner))
	assert.Equal(t, 2, template.CurrentVersion)
	versions := []*RetailTemplateVersion{}
	require.NoError(t, s.Where("template_id = ?", template.ID).Asc("version").Find(&versions))
	require.Len(t, versions, 2)
	assert.Equal(t, "Open the store", versions[0].Title, "older versions remain immutable")

	scheduled := time.Date(2026, 8, 22, 8, 0, 0, 0, time.UTC)
	input := RetailTemplateDispatchInput{
		TargetOrgUnitID: 1, ProjectID: 1, PrimaryAssigneeID: 2, ReviewerID: 1,
		ScheduledFor: scheduled, DueDate: scheduled.Add(2 * time.Hour),
	}
	preview, err := PreviewRetailTaskTemplateDispatch(s, template.ID, input, owner)
	require.NoError(t, err)
	assert.False(t, preview.AlreadyDispatched)
	assert.Equal(t, 60, preview.EstimatedMinutes)

	_, err = DispatchRetailTaskTemplate(s, template.ID, input, outsider)
	var forbidden ErrGenericForbidden
	require.ErrorAs(t, err, &forbidden)

	first, err := DispatchRetailTaskTemplate(s, template.ID, input, owner)
	require.NoError(t, err)
	assert.False(t, first.Reused)
	assert.Equal(t, RetailTaskSourceTemplate, first.Profile.Source)
	assert.Equal(t, versions[1].ID, first.Profile.SourceID)
	items := []*RetailChecklistItem{}
	require.NoError(t, s.Where("profile_id = ?", first.Profile.ID).Find(&items))
	require.Len(t, items, 1)

	second, err := DispatchRetailTaskTemplate(s, template.ID, input, owner)
	require.NoError(t, err)
	assert.True(t, second.Reused)
	assert.Equal(t, first.Dispatch.TaskID, second.Dispatch.TaskID)

	capacity, err := SetRetailStaffCapacity(s, 1, 2, scheduled, 30, "short shift", owner)
	require.NoError(t, err)
	assert.Equal(t, 30, capacity.Minutes)
	_, err = SetRetailStaffCapacity(s, 1, 2, scheduled, 30, "", outsider)
	require.ErrorAs(t, err, &forbidden)

	workload, err := GetRetailStaffWorkload(s, 1, scheduled, scheduled, owner)
	require.NoError(t, err)
	found := false
	for _, row := range workload {
		if row.UserID == 2 {
			found = true
			assert.Equal(t, 60, row.AssignedMinutes)
			assert.Equal(t, 30, row.CapacityMinutes)
			assert.True(t, row.Warning)
			assert.True(t, row.Overloaded)
		}
	}
	assert.True(t, found)
}
