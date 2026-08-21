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
	ErrCodeRetailOrgUnitDoesNotExist  = 20001
	ErrCodeRetailOrgUnitInvalidType   = 20002
	ErrCodeRetailOrgUnitInvalidParent = 20003
	ErrCodeRetailOrgUnitHasChildren   = 20004
	ErrCodeRetailOrgUnitHasTasks      = 20005
)

type ErrRetailOrgUnitDoesNotExist struct{ ID int64 }

func (err ErrRetailOrgUnitDoesNotExist) Error() string {
	return fmt.Sprintf("retail organization unit does not exist [id: %d]", err.ID)
}

func (err ErrRetailOrgUnitDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusNotFound, Code: ErrCodeRetailOrgUnitDoesNotExist, Message: "This retail organization unit does not exist."}
}

type ErrRetailOrgUnitInvalidType struct{ Type RetailOrgUnitType }

func (err ErrRetailOrgUnitInvalidType) Error() string {
	return fmt.Sprintf("invalid retail organization unit type: %s", err.Type)
}

func (err ErrRetailOrgUnitInvalidType) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusBadRequest, Code: ErrCodeRetailOrgUnitInvalidType, Message: "Invalid retail organization unit type."}
}

type ErrRetailOrgUnitInvalidParent struct {
	Type       RetailOrgUnitType
	ParentType RetailOrgUnitType
}

func (err ErrRetailOrgUnitInvalidParent) Error() string {
	return fmt.Sprintf("invalid parent type %s for retail organization unit type %s", err.ParentType, err.Type)
}

func (err ErrRetailOrgUnitInvalidParent) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusBadRequest, Code: ErrCodeRetailOrgUnitInvalidParent, Message: "Invalid parent for this retail organization unit type."}
}

type ErrRetailOrgUnitHasChildren struct{ ID int64 }

func (err ErrRetailOrgUnitHasChildren) Error() string {
	return fmt.Sprintf("retail organization unit has children [id: %d]", err.ID)
}

func (err ErrRetailOrgUnitHasChildren) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusConflict, Code: ErrCodeRetailOrgUnitHasChildren, Message: "Remove or move child organization units before deleting this one."}
}

type ErrRetailOrgUnitHasTasks struct{ ID int64 }

func (err ErrRetailOrgUnitHasTasks) Error() string {
	return fmt.Sprintf("retail organization unit has tasks [id: %d]", err.ID)
}

func (err ErrRetailOrgUnitHasTasks) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusConflict, Code: ErrCodeRetailOrgUnitHasTasks, Message: "Remove or reassign retail task profiles before deleting this organization unit."}
}
