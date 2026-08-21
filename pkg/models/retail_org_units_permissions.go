// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"time"

	"code.vikunja.io/api/pkg/web"
	"xorm.io/xorm"
)

func (r *RetailOrgUnit) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	if _, is := a.(*LinkSharing); is {
		return false, nil
	}
	if r.Type == RetailOrgUnitCompany && r.ParentID == 0 {
		return true, nil
	}
	if err := r.validateHierarchy(s); err != nil {
		return false, err
	}
	parent := &RetailOrgUnit{ID: r.ParentID}
	return parent.hasInheritedAdminAccess(s, a)
}

func (r *RetailOrgUnit) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	if _, is := a.(*LinkSharing); is {
		return false, 0, nil
	}
	unit, err := GetRetailOrgUnitByID(s, r.ID)
	if err != nil {
		return false, 0, err
	}
	return unit.inheritedAccess(s, a)
}

func (r *RetailOrgUnit) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return r.hasInheritedAdminAccess(s, a)
}

func (r *RetailOrgUnit) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return r.hasInheritedAdminAccess(s, a)
}

func (r *RetailOrgUnit) hasInheritedAdminAccess(s *xorm.Session, a web.Auth) (bool, error) {
	can, permission, err := r.inheritedAccessByID(s, a)
	return can && permission == int(PermissionAdmin), err
}

func (r *RetailOrgUnit) inheritedAccessByID(s *xorm.Session, a web.Auth) (bool, int, error) {
	unit, err := GetRetailOrgUnitByID(s, r.ID)
	if err != nil {
		return false, 0, err
	}
	return unit.inheritedAccess(s, a)
}

func (r *RetailOrgUnit) inheritedAccess(s *xorm.Session, a web.Auth) (bool, int, error) {
	visited := map[int64]bool{}
	current := r
	maxPermission := int(PermissionRead)
	hasAccess := false
	for current != nil && current.ID > 0 && !visited[current.ID] {
		visited[current.ID] = true
		member := &TeamMember{}
		has, err := s.Where("team_id = ?", current.TeamID).And("user_id = ?", a.GetID()).Get(member)
		if err != nil {
			return false, 0, err
		}
		if has {
			membership := &RetailMembership{}
			hasRetailProfile, profileErr := s.Where("org_unit_id = ?", current.ID).And("user_id = ?", a.GetID()).Get(membership)
			if profileErr != nil {
				return false, 0, profileErr
			}
			if hasRetailProfile && (!membership.Active || !membership.EndsAt.IsZero() && !membership.EndsAt.After(time.Now())) {
				if current.ParentID == 0 {
					break
				}
				current, err = GetRetailOrgUnitByID(s, current.ParentID)
				if err != nil {
					return false, 0, err
				}
				continue
			}
			if member.Admin {
				return true, int(PermissionAdmin), nil
			}
			hasAccess = true
			maxPermission = int(PermissionRead)
		}
		if current.ParentID == 0 {
			break
		}
		current, err = GetRetailOrgUnitByID(s, current.ParentID)
		if err != nil {
			return false, 0, err
		}
	}
	return hasAccess, maxPermission, nil
}
