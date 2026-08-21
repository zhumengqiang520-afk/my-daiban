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
	ErrCodeRetailTaskTemplateDoesNotExist    = 20037
	ErrCodeRetailTaskTemplateInactive        = 20038
	ErrCodeRetailTemplateVersionDoesNotExist = 20039
)

type ErrRetailTaskTemplateDoesNotExist struct{ ID int64 }

func (err ErrRetailTaskTemplateDoesNotExist) Error() string {
	return fmt.Sprintf("retail task template does not exist [id: %d]", err.ID)
}

func (err ErrRetailTaskTemplateDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusNotFound, Code: ErrCodeRetailTaskTemplateDoesNotExist, Message: "This retail task template does not exist."}
}

type ErrRetailTaskTemplateInactive struct{ ID int64 }

func (err ErrRetailTaskTemplateInactive) Error() string {
	return fmt.Sprintf("retail task template is inactive [id: %d]", err.ID)
}

func (err ErrRetailTaskTemplateInactive) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusConflict, Code: ErrCodeRetailTaskTemplateInactive, Message: "This retail task template is inactive."}
}

type ErrRetailTemplateVersionDoesNotExist struct {
	TemplateID int64
	Version    int
}

func (err ErrRetailTemplateVersionDoesNotExist) Error() string {
	return fmt.Sprintf("retail template version does not exist [template: %d, version: %d]", err.TemplateID, err.Version)
}

func (err ErrRetailTemplateVersionDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusNotFound, Code: ErrCodeRetailTemplateVersionDoesNotExist, Message: "This retail template version does not exist."}
}
