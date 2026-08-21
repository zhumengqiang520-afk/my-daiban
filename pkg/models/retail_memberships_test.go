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

func TestRetailMembershipLifecycleAndPermissions(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	insertRetailOrgFixture(t, s)

	owner := &user.User{ID: 1, Username: "user1"}
	staff := &user.User{ID: 2, Username: "user2"}
	outsider := &user.User{ID: 3, Username: "user3"}
	membership := &RetailMembership{
		OrgUnitID:     1,
		Username:      "user2",
		JobTitle:      "Sales Associate",
		ManagerUserID: 1,
		IsPrimary:     true,
		Active:        true,
	}

	canCreate, err := membership.CanCreate(s, owner)
	require.NoError(t, err)
	assert.True(t, canCreate)
	canCreate, err = membership.CanCreate(s, staff)
	require.NoError(t, err)
	assert.False(t, canCreate)

	require.NoError(t, membership.Create(s, owner))
	assert.NotZero(t, membership.ID)
	assert.Equal(t, int64(2), membership.UserID)
	assert.Equal(t, "user2", membership.Username)
	assert.Equal(t, "user1", membership.ManagerName)

	canRead, permission, err := membership.CanRead(s, staff)
	require.NoError(t, err)
	assert.True(t, canRead)
	assert.Equal(t, int(PermissionRead), permission)
	canRead, _, err = membership.CanRead(s, outsider)
	require.NoError(t, err)
	assert.False(t, canRead)

	membership.JobTitle = "Senior Sales Associate"
	membership.Admin = false
	membership.Active = false
	require.NoError(t, membership.Update(s, owner))
	assert.False(t, membership.Active)
	exists, err := s.Where("team_id = ?", 1).And("user_id = ?", 2).Exist(&TeamMember{})
	require.NoError(t, err)
	assert.False(t, exists)

	membership.Active = true
	require.NoError(t, membership.Update(s, owner))
	exists, err = s.Where("team_id = ?", 1).And("user_id = ?", 2).And("admin = ?", false).Exist(&TeamMember{})
	require.NoError(t, err)
	assert.True(t, exists)

	require.NoError(t, membership.Delete(s, owner))
	exists, err = s.ID(membership.ID).Exist(&RetailMembership{})
	require.NoError(t, err)
	assert.False(t, exists)
	exists, err = s.Where("team_id = ?", 1).And("user_id = ?", 2).Exist(&TeamMember{})
	require.NoError(t, err)
	assert.False(t, exists)

	err = (&RetailMembership{ID: 1}).Delete(s, owner)
	var needsAdmin ErrRetailOrgUnitNeedsAdmin
	require.ErrorAs(t, err, &needsAdmin)
}

func TestRetailMembershipValidation(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	insertRetailOrgFixture(t, s)

	membership := &RetailMembership{
		OrgUnitID: 1,
		Username:  "user2",
		Temporary: true,
		StartsAt:  time.Now().Add(time.Hour),
		EndsAt:    time.Now().Add(2 * time.Hour),
		Active:    true,
	}
	err := membership.Create(s, &user.User{ID: 1, Username: "user1"})
	var invalidPeriod ErrRetailMembershipInvalidPeriod
	require.ErrorAs(t, err, &invalidPeriod)

	membership.StartsAt = time.Now()
	membership.EndsAt = time.Now().Add(time.Hour)
	membership.ManagerUserID = 3
	err = membership.Create(s, &user.User{ID: 1, Username: "user1"})
	var invalidManager ErrRetailMembershipInvalidManager
	require.ErrorAs(t, err, &invalidManager)
}

func TestExpireRetailMemberships(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	insertRetailOrgFixture(t, s)
	_, err := s.Insert(&RetailMembership{
		ID:          2,
		OrgUnitID:   1,
		UserID:      2,
		JobTitle:    "Temporary Support",
		Temporary:   true,
		StartsAt:    time.Now().Add(-2 * time.Hour),
		EndsAt:      time.Now().Add(-time.Minute),
		Active:      true,
		CreatedByID: 1,
	})
	require.NoError(t, err)
	require.NoError(t, s.Commit())
	require.NoError(t, s.Close())

	expireRetailMembershipsAt(time.Now())

	db.AssertExists(t, "retail_memberships", map[string]interface{}{"id": int64(2), "active": false}, false)
	db.AssertMissing(t, "team_members", map[string]interface{}{"team_id": int64(1), "user_id": int64(2)})
}
