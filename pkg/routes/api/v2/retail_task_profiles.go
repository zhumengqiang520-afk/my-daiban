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
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/conditional"
)

type retailTaskProfileListBody struct {
	Body Paginated[*models.RetailTaskProfile]
}

func RegisterRetailTaskProfileRoutes(api huma.API) {
	if !config.RetailEnabled.GetBool() {
		return
	}
	tags := []string{"retail tasks"}
	Register(api, huma.Operation{OperationID: "retail-task-profiles-list", Summary: "List retail task profiles", Description: "Returns retail tasks in organization scopes visible to the caller. Supports organization, task, status, and free-text filtering.", Method: http.MethodGet, Path: "/retail/task-profiles", Tags: tags}, retailTaskProfilesList)
	Register(api, huma.Operation{OperationID: "retail-task-profiles-read", Summary: "Get a retail task profile", Description: "Returns the retail workflow fields together with current task, organization, assignee, and reviewer display data.", Method: http.MethodGet, Path: "/retail/task-profiles/{id}", Tags: tags}, retailTaskProfilesRead)
	Register(api, huma.Operation{OperationID: "retail-task-profiles-create", Summary: "Convert a Vikunja task into a retail task", Description: "Binds an existing task to one organization unit, grants that unit's team write access to the dedicated project, and assigns the primary staff member. Requires project and organization administration.", Method: http.MethodPost, Path: "/retail/task-profiles", Tags: tags}, retailTaskProfilesCreate)
	Register(api, huma.Operation{OperationID: "retail-task-profiles-update", Summary: "Update a retail task profile", Description: "Updates category, staff, effort, source, and evidence policy. Task, organization, source ID, and workflow status are server-controlled; PATCH is available for partial updates.", Method: http.MethodPut, Path: "/retail/task-profiles/{id}", Tags: tags}, retailTaskProfilesUpdate)
	Register(api, huma.Operation{OperationID: "retail-task-profiles-delete", Summary: "Remove a retail task profile", Description: "Removes only the retail extension; the underlying Vikunja task remains. Requires organization administration.", Method: http.MethodDelete, Path: "/retail/task-profiles/{id}", Tags: tags}, retailTaskProfilesDelete)
}

func init() { AddRouteRegistrar(RegisterRetailTaskProfileRoutes) }

func retailTaskProfilesList(ctx context.Context, in *struct {
	ListParams
	OrgUnitID int64                   `query:"org_unit_id" doc:"Return tasks for this organization unit only."`
	TaskID    int64                   `query:"task_id" doc:"Return the profile for this Vikunja task only."`
	Status    models.RetailTaskStatus `query:"status" doc:"Return profiles in this retail workflow status only."`
}) (*retailTaskProfileListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	model := &models.RetailTaskProfile{OrgUnitID: in.OrgUnitID, TaskID: in.TaskID, FilterStatus: in.Status}
	result, _, total, err := handler.DoReadAll(ctx, model, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*models.RetailTaskProfile)
	if !ok {
		return nil, fmt.Errorf("RetailTaskProfile.ReadAll returned unexpected type %T", result)
	}
	return &retailTaskProfileListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

type retailTaskProfileReadBody struct {
	models.RetailTaskProfile
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"Maximum permission for the requesting user (0=read, 2=admin)."`
}

func retailTaskProfilesRead(ctx context.Context, in *struct {
	ID int64 `path:"id" doc:"Retail task profile ID."`
	conditional.Params
}) (*singleReadBody[retailTaskProfileReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	profile := &models.RetailTaskProfile{ID: in.ID}
	permission, err := handler.DoReadOne(ctx, profile, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &retailTaskProfileReadBody{RetailTaskProfile: *profile, MaxPermission: models.Permission(permission)}
	return conditionalReadResponse(&in.Params, body, profile.Updated, permission)
}

func retailTaskProfilesCreate(ctx context.Context, in *struct{ Body models.RetailTaskProfile }) (*singleBody[models.RetailTaskProfile], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailTaskProfile]{Body: &in.Body}, nil
}

func retailTaskProfilesUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"id" doc:"Retail task profile ID."`
	Body retailTaskProfileReadBody
}) (*singleBody[models.RetailTaskProfile], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	profile := &in.Body.RetailTaskProfile
	profile.ID = in.ID
	if err := handler.DoUpdate(ctx, profile, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailTaskProfile]{Body: profile}, nil
}

func retailTaskProfilesDelete(ctx context.Context, in *struct {
	ID int64 `path:"id" doc:"Retail task profile ID."`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &models.RetailTaskProfile{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}
