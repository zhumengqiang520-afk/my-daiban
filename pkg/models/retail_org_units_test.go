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
	"xorm.io/xorm"
)

func insertRetailOrgFixture(t *testing.T, s *xorm.Session) {
	t.Helper()
	_, err := s.Where("id > ?", 0).Delete(&RetailMembership{})
	require.NoError(t, err)
	_, err = s.Where("id > ?", 0).Delete(&RetailOrgUnit{})
	require.NoError(t, err)
	units := []*RetailOrgUnit{
		{ID: 1, Type: RetailOrgUnitCompany, Name: "Sleep Retail", Code: "COMP", TeamID: 1, Active: true, CreatedByID: 1},
		{ID: 2, ParentID: 1, Type: RetailOrgUnitRegion, Name: "East Region", Code: "EAST", TeamID: 9, Active: true, CreatedByID: 1},
		{ID: 3, ParentID: 2, Type: RetailOrgUnitStore, Name: "Shanghai Store", Code: "SH001", TeamID: 10, Active: true, CreatedByID: 1},
	}
	_, err = s.Insert(units)
	require.NoError(t, err)
	_, err = s.Insert(&RetailMembership{ID: 1, OrgUnitID: 1, UserID: 1, JobTitle: "Owner", IsPrimary: true, StartsAt: time.Now(), Active: true, CreatedByID: 1})
	require.NoError(t, err)
}

func TestRetailOrgUnitCreate(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	_, err := s.Where("id > ?", 0).Delete(&RetailOrgUnit{})
	require.NoError(t, err)

	unit := &RetailOrgUnit{Type: RetailOrgUnitCompany, Name: " Bedding Group ", Code: " BED ", Active: true}
	err = unit.Create(s, &user.User{ID: 1, Username: "user1"})
	require.NoError(t, err)
	require.NoError(t, s.Commit())

	assert.NotZero(t, unit.ID)
	assert.NotZero(t, unit.TeamID)
	assert.Equal(t, "Bedding Group", unit.Name)
	db.AssertExists(t, "teams", map[string]interface{}{"id": unit.TeamID, "name": "[Retail BED] Bedding Group"}, false)
	db.AssertExists(t, "team_members", map[string]interface{}{"team_id": unit.TeamID, "user_id": int64(1), "admin": true}, false)
}

func TestRetailOrgUnitPermissions(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	insertRetailOrgFixture(t, s)

	companyAdmin := &user.User{ID: 1}
	companyMember := &user.User{ID: 2}
	storeMember := &user.User{ID: 3}
	outsider := &user.User{ID: 4}
	store := &RetailOrgUnit{ID: 3}

	can, permission, err := store.CanRead(s, companyAdmin)
	require.NoError(t, err)
	assert.True(t, can)
	assert.Equal(t, int(PermissionAdmin), permission, "company admin inherits admin access to stores")

	can, permission, err = store.CanRead(s, companyMember)
	require.NoError(t, err)
	assert.True(t, can)
	assert.Equal(t, int(PermissionRead), permission, "ordinary ancestor members inherit read access")

	can, permission, err = store.CanRead(s, storeMember)
	require.NoError(t, err)
	assert.True(t, can)
	assert.Equal(t, int(PermissionRead), permission)

	can, _, err = (&RetailOrgUnit{ID: 2}).CanRead(s, storeMember)
	require.NoError(t, err)
	assert.False(t, can, "store membership must not grant upward or sibling access")

	can, _, err = store.CanRead(s, outsider)
	require.NoError(t, err)
	assert.False(t, can)

	canUpdate, err := store.CanUpdate(s, companyAdmin)
	require.NoError(t, err)
	assert.True(t, canUpdate)
	canUpdate, err = store.CanUpdate(s, companyMember)
	require.NoError(t, err)
	assert.False(t, canUpdate)

	canDeleteTeam, err := (&Team{ID: 1}).CanDelete(s, companyAdmin)
	require.NoError(t, err)
	assert.False(t, canDeleteTeam, "retail-managed teams may only be deleted through the organization endpoint")
	canUpdateTeam, err := (&Team{ID: 1}).CanUpdate(s, companyAdmin)
	require.NoError(t, err)
	assert.False(t, canUpdateTeam, "retail-managed team names stay synchronized with organization units")
}

func TestRetailOrgUnitReadAllScopesDescendants(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	insertRetailOrgFixture(t, s)

	result, count, total, err := (&RetailOrgUnit{}).ReadAll(s, &user.User{ID: 1}, "", 1, 50)
	require.NoError(t, err)
	assert.Len(t, result.([]*RetailOrgUnit), 3)
	assert.Equal(t, 3, count)
	assert.Equal(t, int64(3), total)

	result, count, total, err = (&RetailOrgUnit{}).ReadAll(s, &user.User{ID: 3}, "", 1, 50)
	require.NoError(t, err)
	units := result.([]*RetailOrgUnit)
	require.Len(t, units, 1)
	assert.Equal(t, int64(3), units[0].ID)
	assert.Equal(t, 1, count)
	assert.Equal(t, int64(1), total)

	result, count, total, err = (&RetailOrgUnit{}).ReadAll(s, &user.User{ID: 1}, "sh001", 1, 50)
	require.NoError(t, err)
	assert.Len(t, result.([]*RetailOrgUnit), 1)
	assert.Equal(t, 1, count)
	assert.Equal(t, int64(1), total)
}

func TestRetailOrgUnitHierarchyAndDeleteGuard(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	insertRetailOrgFixture(t, s)

	err := (&RetailOrgUnit{ParentID: 1, Type: RetailOrgUnitStore}).validateHierarchy(s)
	var invalidParent ErrRetailOrgUnitInvalidParent
	require.ErrorAs(t, err, &invalidParent)

	err = (&RetailOrgUnit{ID: 2}).Delete(s, &user.User{ID: 1})
	var hasChildren ErrRetailOrgUnitHasChildren
	require.ErrorAs(t, err, &hasChildren)
}
