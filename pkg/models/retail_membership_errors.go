// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/web"
)

const (
	ErrCodeRetailMembershipDoesNotExist   = 20010
	ErrCodeRetailMembershipAlreadyExists  = 20011
	ErrCodeRetailMembershipInvalidPeriod  = 20012
	ErrCodeRetailMembershipInvalidManager = 20013
	ErrCodeRetailOrgUnitNeedsAdmin        = 20014
)

type ErrRetailMembershipDoesNotExist struct{ ID int64 }

func (err ErrRetailMembershipDoesNotExist) Error() string {
	return fmt.Sprintf("retail membership does not exist [id: %d]", err.ID)
}

func (err ErrRetailMembershipDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusNotFound, Code: ErrCodeRetailMembershipDoesNotExist, Message: "This retail membership does not exist."}
}

type ErrRetailMembershipAlreadyExists struct {
	OrgUnitID int64
	UserID    int64
}

func (err ErrRetailMembershipAlreadyExists) Error() string {
	return fmt.Sprintf("retail membership already exists [organization unit: %d, user: %d]", err.OrgUnitID, err.UserID)
}

func (err ErrRetailMembershipAlreadyExists) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusConflict, Code: ErrCodeRetailMembershipAlreadyExists, Message: "This user already belongs to the organization unit."}
}

type ErrRetailMembershipInvalidPeriod struct{}

func (ErrRetailMembershipInvalidPeriod) Error() string {
	return "temporary retail membership has an invalid effective period"
}

func (ErrRetailMembershipInvalidPeriod) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusBadRequest, Code: ErrCodeRetailMembershipInvalidPeriod, Message: "Temporary membership requires an end time after its start time and cannot start in the future."}
}

type ErrRetailMembershipInvalidManager struct{ ManagerUserID int64 }

func (err ErrRetailMembershipInvalidManager) Error() string {
	return fmt.Sprintf("invalid retail membership manager [user: %d]", err.ManagerUserID)
}

func (err ErrRetailMembershipInvalidManager) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusBadRequest, Code: ErrCodeRetailMembershipInvalidManager, Message: "The manager must be another active member of the same organization unit."}
}

type ErrRetailOrgUnitNeedsAdmin struct{ OrgUnitID int64 }

func (err ErrRetailOrgUnitNeedsAdmin) Error() string {
	return fmt.Sprintf("retail organization unit needs at least one administrator [id: %d]", err.OrgUnitID)
}

func (err ErrRetailOrgUnitNeedsAdmin) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusConflict, Code: ErrCodeRetailOrgUnitNeedsAdmin, Message: "An organization unit must keep at least one active administrator."}
}
