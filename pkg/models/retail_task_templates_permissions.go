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

func (r *RetailTaskTemplate) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	if _, is := a.(*LinkSharing); is {
		return false, nil
	}
	return (&RetailOrgUnit{ID: r.OrgUnitID}).hasInheritedAdminAccess(s, a)
}

func (r *RetailTaskTemplate) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	if _, is := a.(*LinkSharing); is {
		return false, 0, nil
	}
	template, err := GetRetailTaskTemplateByID(s, r.ID)
	if err != nil {
		return false, 0, err
	}
	return (&RetailOrgUnit{ID: template.OrgUnitID}).CanRead(s, a)
}

func (r *RetailTaskTemplate) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return r.canManage(s, a)
}

func (r *RetailTaskTemplate) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return r.canManage(s, a)
}

func (r *RetailTaskTemplate) canManage(s *xorm.Session, a web.Auth) (bool, error) {
	if _, is := a.(*LinkSharing); is {
		return false, nil
	}
	template, err := GetRetailTaskTemplateByID(s, r.ID)
	if err != nil {
		return false, err
	}
	return (&RetailOrgUnit{ID: template.OrgUnitID}).hasInheritedAdminAccess(s, a)
}
