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
	ErrCodeRetailTaskProfileDoesNotExist  = 20020
	ErrCodeRetailTaskProfileAlreadyExists = 20021
	ErrCodeRetailTaskInvalidCategory      = 20022
	ErrCodeRetailTaskInvalidStaff         = 20023
	ErrCodeRetailProjectScopeConflict     = 20024
)

type ErrRetailTaskProfileDoesNotExist struct{ ID int64 }

func (err ErrRetailTaskProfileDoesNotExist) Error() string {
	return fmt.Sprintf("retail task profile does not exist [id: %d]", err.ID)
}

func (err ErrRetailTaskProfileDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusNotFound, Code: ErrCodeRetailTaskProfileDoesNotExist, Message: "This retail task profile does not exist."}
}

type ErrRetailTaskProfileAlreadyExists struct{ TaskID int64 }

func (err ErrRetailTaskProfileAlreadyExists) Error() string {
	return fmt.Sprintf("retail task profile already exists [task: %d]", err.TaskID)
}

func (err ErrRetailTaskProfileAlreadyExists) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusConflict, Code: ErrCodeRetailTaskProfileAlreadyExists, Message: "This task already has a retail profile."}
}

type ErrRetailTaskInvalidCategory struct{ Category RetailTaskCategory }

func (err ErrRetailTaskInvalidCategory) Error() string {
	return fmt.Sprintf("invalid retail task category: %s", err.Category)
}

func (err ErrRetailTaskInvalidCategory) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusBadRequest, Code: ErrCodeRetailTaskInvalidCategory, Message: "Invalid retail task category."}
}

type ErrRetailTaskInvalidStaff struct {
	OrgUnitID int64
	UserID    int64
	Role      string
}

func (err ErrRetailTaskInvalidStaff) Error() string {
	return fmt.Sprintf("invalid retail task %s [organization unit: %d, user: %d]", err.Role, err.OrgUnitID, err.UserID)
}

func (err ErrRetailTaskInvalidStaff) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusBadRequest, Code: ErrCodeRetailTaskInvalidStaff, Message: "The selected assignee or reviewer is not active in the required organization scope."}
}

type ErrRetailProjectScopeConflict struct {
	ProjectID int64
	OrgUnitID int64
}

func (err ErrRetailProjectScopeConflict) Error() string {
	return fmt.Sprintf("project already contains tasks for another retail organization [project: %d, organization unit: %d]", err.ProjectID, err.OrgUnitID)
}

func (err ErrRetailProjectScopeConflict) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusConflict, Code: ErrCodeRetailProjectScopeConflict, Message: "A retail project can only contain tasks for one organization unit."}
}
