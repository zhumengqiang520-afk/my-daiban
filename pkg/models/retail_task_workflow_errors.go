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
	ErrCodeRetailChecklistItemDoesNotExist = 20030
	ErrCodeRetailTaskInvalidTransition     = 20031
	ErrCodeRetailTaskChecklistIncomplete   = 20032
	ErrCodeRetailTaskEvidenceRequired      = 20033
	ErrCodeRetailTaskInvalidAttachment     = 20034
	ErrCodeRetailSubmissionDoesNotExist    = 20035
	ErrCodeRetailReviewInvalidDecision     = 20036
)

type ErrRetailChecklistItemDoesNotExist struct{ ID int64 }

func (err ErrRetailChecklistItemDoesNotExist) Error() string {
	return fmt.Sprintf("retail checklist item does not exist [id: %d]", err.ID)
}

func (err ErrRetailChecklistItemDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusNotFound, Code: ErrCodeRetailChecklistItemDoesNotExist, Message: "This retail checklist item does not exist."}
}

type ErrRetailTaskInvalidTransition struct {
	From RetailTaskStatus
	To   RetailTaskStatus
}

func (err ErrRetailTaskInvalidTransition) Error() string {
	return fmt.Sprintf("invalid retail task transition from %s to %s", err.From, err.To)
}

func (err ErrRetailTaskInvalidTransition) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusConflict, Code: ErrCodeRetailTaskInvalidTransition, Message: "This retail task status transition is not allowed."}
}

type ErrRetailTaskChecklistIncomplete struct{ ProfileID int64 }

func (err ErrRetailTaskChecklistIncomplete) Error() string {
	return fmt.Sprintf("required retail checklist items are incomplete [profile: %d]", err.ProfileID)
}

func (err ErrRetailTaskChecklistIncomplete) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusConflict, Code: ErrCodeRetailTaskChecklistIncomplete, Message: "Complete every required checklist item before submitting."}
}

type ErrRetailTaskEvidenceRequired struct{ ProfileID int64 }

func (err ErrRetailTaskEvidenceRequired) Error() string {
	return fmt.Sprintf("retail task evidence is required [profile: %d]", err.ProfileID)
}

func (err ErrRetailTaskEvidenceRequired) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusConflict, Code: ErrCodeRetailTaskEvidenceRequired, Message: "Attach at least one valid task file before submitting."}
}

type ErrRetailTaskInvalidAttachment struct {
	TaskID       int64
	AttachmentID int64
}

func (err ErrRetailTaskInvalidAttachment) Error() string {
	return fmt.Sprintf("attachment does not belong to retail task [task: %d, attachment: %d]", err.TaskID, err.AttachmentID)
}

func (err ErrRetailTaskInvalidAttachment) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusBadRequest, Code: ErrCodeRetailTaskInvalidAttachment, Message: "Every evidence attachment must belong to this task."}
}

type ErrRetailSubmissionDoesNotExist struct{ ID int64 }

func (err ErrRetailSubmissionDoesNotExist) Error() string {
	return fmt.Sprintf("retail submission does not exist [id: %d]", err.ID)
}

func (err ErrRetailSubmissionDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusNotFound, Code: ErrCodeRetailSubmissionDoesNotExist, Message: "This retail task submission does not exist."}
}

type ErrRetailReviewInvalidDecision struct{ Decision RetailReviewDecision }

func (err ErrRetailReviewInvalidDecision) Error() string {
	return fmt.Sprintf("invalid retail review decision: %s", err.Decision)
}

func (err ErrRetailReviewInvalidDecision) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusBadRequest, Code: ErrCodeRetailReviewInvalidDecision, Message: "Review decision must be approved or rejected."}
}
