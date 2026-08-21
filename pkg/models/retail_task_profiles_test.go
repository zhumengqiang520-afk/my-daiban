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

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetailTaskProfileLifecycleAndPermissions(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	insertRetailOrgFixture(t, s)
	_, err := s.Where("id > ?", 0).Delete(&RetailTaskProfile{})
	require.NoError(t, err)

	owner := &user.User{ID: 1, Username: "user1"}
	staff := &user.User{ID: 2, Username: "user2"}
	outsider := &user.User{ID: 3, Username: "user3"}
	staffMembership := &RetailMembership{OrgUnitID: 1, Username: "user2", JobTitle: "Sales Associate", Active: true}
	require.NoError(t, staffMembership.Create(s, owner))

	profile := &RetailTaskProfile{
		TaskID:            1,
		OrgUnitID:         1,
		Category:          RetailTaskCategoryOpening,
		PrimaryAssigneeID: 2,
		ReviewerID:        1,
		EstimatedMinutes:  30,
		Source:            RetailTaskSourceManual,
		EvidenceRequired:  true,
	}
	canCreate, err := profile.CanCreate(s, owner)
	require.NoError(t, err)
	assert.True(t, canCreate)
	canCreate, err = profile.CanCreate(s, staff)
	require.NoError(t, err)
	assert.False(t, canCreate)

	require.NoError(t, profile.Create(s, owner))
	assert.NotZero(t, profile.ID)
	assert.Equal(t, RetailTaskStatusAssigned, profile.Status)
	assert.Equal(t, "task #1", profile.TaskTitle)
	assert.Equal(t, "user2", profile.PrimaryAssignee)

	hasProjectAccess, err := s.Where("project_id = ?", 1).And("team_id = ?", 1).And("permission >= ?", PermissionWrite).Exist(&TeamProject{})
	require.NoError(t, err)
	assert.True(t, hasProjectAccess)
	hasAssignee, err := s.Where("task_id = ?", 1).And("user_id = ?", 2).Exist(&TaskAssginee{})
	require.NoError(t, err)
	assert.True(t, hasAssignee)

	canRead, permission, err := profile.CanRead(s, staff)
	require.NoError(t, err)
	assert.True(t, canRead)
	assert.Equal(t, int(PermissionRead), permission)
	canRead, _, err = profile.CanRead(s, outsider)
	require.NoError(t, err)
	assert.False(t, canRead)

	profile.Category = RetailTaskCategoryDisplay
	profile.EstimatedMinutes = 45
	profile.Status = RetailTaskStatusCompleted
	require.NoError(t, profile.Update(s, owner))
	assert.Equal(t, RetailTaskStatusAssigned, profile.Status, "generic updates cannot bypass the workflow state machine")
	assert.Equal(t, 45, profile.EstimatedMinutes)

	conflict := &RetailTaskProfile{TaskID: 2, OrgUnitID: 2, Category: RetailTaskCategoryOther, Source: RetailTaskSourceManual}
	err = conflict.Create(s, owner)
	var scopeConflict ErrRetailProjectScopeConflict
	require.ErrorAs(t, err, &scopeConflict)

	require.NoError(t, profile.Delete(s, owner))
	exists, err := s.ID(profile.ID).Exist(&RetailTaskProfile{})
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRetailTaskProfileValidation(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	insertRetailOrgFixture(t, s)

	profile := &RetailTaskProfile{TaskID: 1, OrgUnitID: 1, Category: "invalid", Source: RetailTaskSourceManual}
	err := profile.Create(s, &user.User{ID: 1, Username: "user1"})
	var invalidCategory ErrRetailTaskInvalidCategory
	require.ErrorAs(t, err, &invalidCategory)

	profile.Category = RetailTaskCategoryOpening
	profile.PrimaryAssigneeID = 3
	err = profile.Create(s, &user.User{ID: 1, Username: "user1"})
	var invalidStaff ErrRetailTaskInvalidStaff
	require.ErrorAs(t, err, &invalidStaff)
}
