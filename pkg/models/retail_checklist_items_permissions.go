// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"code.vikunja.io/api/pkg/web"
	"xorm.io/xorm"
)

func (r *RetailChecklistItem) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	profile, err := GetRetailTaskProfileByID(s, r.ProfileID)
	if err != nil {
		return false, err
	}
	return (&RetailOrgUnit{ID: profile.OrgUnitID}).hasInheritedAdminAccess(s, a)
}

func (r *RetailChecklistItem) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	item, err := GetRetailChecklistItemByID(s, r.ID)
	if err != nil {
		return false, 0, err
	}
	profile, err := GetRetailTaskProfileByID(s, item.ProfileID)
	if err != nil {
		return false, 0, err
	}
	return (&RetailOrgUnit{ID: profile.OrgUnitID}).CanRead(s, a)
}

func (r *RetailChecklistItem) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return r.canManage(s, a)
}

func (r *RetailChecklistItem) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return r.canManage(s, a)
}

func (r *RetailChecklistItem) canManage(s *xorm.Session, a web.Auth) (bool, error) {
	item, err := GetRetailChecklistItemByID(s, r.ID)
	if err != nil {
		return false, err
	}
	profile, err := GetRetailTaskProfileByID(s, item.ProfileID)
	if err != nil {
		return false, err
	}
	return (&RetailOrgUnit{ID: profile.OrgUnitID}).hasInheritedAdminAccess(s, a)
}
