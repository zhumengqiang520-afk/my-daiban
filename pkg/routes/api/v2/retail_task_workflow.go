// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package apiv2

import (
	"context"
	"net/http"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/models"

	"github.com/danielgtaylor/huma/v2"
	"xorm.io/xorm"
)

func RegisterRetailTaskWorkflowRoutes(api huma.API) {
	if !config.RetailEnabled.GetBool() {
		return
	}
	tags := []string{"retail task workflow"}
	Register(api, huma.Operation{OperationID: "retail-task-workflow-read", Summary: "Get a retail task workflow", Description: "Returns the profile, checklist, every submission and review, and the immutable transition history visible to the caller.", Method: http.MethodGet, Path: "/retail/tasks/{task}/workflow", Tags: tags}, retailTaskWorkflowRead)
	Register(api, huma.Operation{OperationID: "retail-task-start", Summary: "Start a retail task", Description: "Moves an assigned or rejected task to in_progress. The primary assignee or an organization administrator may start it.", Method: http.MethodPost, Path: "/retail/tasks/{task}/start", Tags: tags}, retailTaskStart)
	Register(api, huma.Operation{OperationID: "retail-checklist-completion", Summary: "Set checklist completion", Description: "Checks or unchecks one item while the task is in progress. The primary assignee or an organization administrator may do this.", Method: http.MethodPut, Path: "/retail/checklist-items/{id}/completion", Tags: tags}, retailChecklistCompletion)
	Register(api, huma.Operation{OperationID: "retail-task-submit", Summary: "Submit a retail task for review", Description: "Requires every mandatory checklist item and, when configured, at least one attachment belonging to the task. Tasks without a dedicated reviewer complete automatically.", Method: http.MethodPost, Path: "/retail/tasks/{task}/submissions", Tags: tags}, retailTaskSubmit)
	Register(api, huma.Operation{OperationID: "retail-task-review", Summary: "Review a retail task submission", Description: "Approves or rejects a pending submission. Rejection requires a comment. Only the configured reviewer or an organization administrator may decide.", Method: http.MethodPost, Path: "/retail/tasks/{task}/reviews", Tags: tags}, retailTaskReview)
	Register(api, huma.Operation{OperationID: "retail-task-cancel", Summary: "Cancel a retail task", Description: "Moves a non-completed task to cancelled. Only an organization administrator may cancel.", Method: http.MethodPost, Path: "/retail/tasks/{task}/cancel", Tags: tags}, retailTaskCancel)
}

func init() { AddRouteRegistrar(RegisterRetailTaskWorkflowRoutes) }

func retailTaskWorkflowRead(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"Vikunja task ID."`
}) (*singleBody[models.RetailTaskWorkflow], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()
	workflow, err := models.GetRetailTaskWorkflow(s, in.TaskID, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailTaskWorkflow]{Body: workflow}, nil
}

func retailTaskStart(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"Vikunja task ID."`
}) (*singleBody[models.RetailTaskWorkflow], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()
	workflow, err := models.StartRetailTask(s, in.TaskID, a)
	if err != nil {
		rollbackRetailWorkflow(s)
		return nil, translateDomainError(err)
	}
	if err := commitRetailWorkflow(ctx, s); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailTaskWorkflow]{Body: workflow}, nil
}

func retailChecklistCompletion(ctx context.Context, in *struct {
	ID   int64 `path:"id" doc:"Checklist item ID."`
	Body struct {
		Done bool `json:"done" doc:"The desired completion state."`
	}
}) (*singleBody[models.RetailChecklistItem], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()
	item, err := models.SetRetailChecklistItemDone(s, in.ID, in.Body.Done, a)
	if err != nil {
		rollbackRetailWorkflow(s)
		return nil, translateDomainError(err)
	}
	if err := commitRetailWorkflow(ctx, s); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailChecklistItem]{Body: item}, nil
}

func retailTaskSubmit(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"Vikunja task ID."`
	Body   struct {
		Note                  string  `json:"note" doc:"Optional completion note for the reviewer."`
		EvidenceAttachmentIDs []int64 `json:"evidence_attachment_ids" doc:"IDs of attachments already uploaded to this task."`
	}
}) (*singleBody[models.RetailTaskWorkflow], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()
	workflow, err := models.SubmitRetailTask(s, in.TaskID, in.Body.Note, in.Body.EvidenceAttachmentIDs, a)
	if err != nil {
		rollbackRetailWorkflow(s)
		return nil, translateDomainError(err)
	}
	if err := commitRetailWorkflow(ctx, s); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailTaskWorkflow]{Body: workflow}, nil
}

func retailTaskReview(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"Vikunja task ID."`
	Body   struct {
		SubmissionID int64                       `json:"submission_id" minimum:"1" doc:"The submission being reviewed."`
		Decision     models.RetailReviewDecision `json:"decision" doc:"The decision: approved or rejected."`
		Comment      string                      `json:"comment" doc:"Review comment; required for rejection."`
	}
}) (*singleBody[models.RetailTaskWorkflow], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()
	workflow, err := models.ReviewRetailTask(s, in.TaskID, in.Body.SubmissionID, in.Body.Decision, in.Body.Comment, a)
	if err != nil {
		rollbackRetailWorkflow(s)
		return nil, translateDomainError(err)
	}
	if err := commitRetailWorkflow(ctx, s); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailTaskWorkflow]{Body: workflow}, nil
}

func retailTaskCancel(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"Vikunja task ID."`
	Body   struct {
		Reason string `json:"reason" doc:"Why the task is being cancelled."`
	}
}) (*singleBody[models.RetailTaskWorkflow], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()
	workflow, err := models.CancelRetailTask(s, in.TaskID, in.Body.Reason, a)
	if err != nil {
		rollbackRetailWorkflow(s)
		return nil, translateDomainError(err)
	}
	if err := commitRetailWorkflow(ctx, s); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailTaskWorkflow]{Body: workflow}, nil
}

func rollbackRetailWorkflow(s *xorm.Session) {
	_ = s.Rollback()
	events.CleanupPending(s)
}

func commitRetailWorkflow(ctx context.Context, s *xorm.Session) error {
	if err := s.Commit(); err != nil {
		events.CleanupPending(s)
		return err
	}
	events.DispatchPending(ctx, s)
	return nil
}
