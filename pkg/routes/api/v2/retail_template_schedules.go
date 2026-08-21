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

type retailTemplateScheduleListBody struct {
	Body Paginated[*models.RetailTemplateSchedule]
}

func RegisterRetailTemplateScheduleRoutes(api huma.API) {
	if !config.RetailEnabled.GetBool() {
		return
	}
	tags := []string{"retail templates"}
	Register(api, huma.Operation{OperationID: "retail-template-schedules-list", Summary: "List recurring template schedules", Description: "Returns daily, weekly and monthly schedules visible to the caller.", Method: http.MethodGet, Path: "/retail/template-schedules", Tags: tags}, retailTemplateSchedulesList)
	Register(api, huma.Operation{OperationID: "retail-template-schedules-read", Summary: "Get a recurring template schedule", Method: http.MethodGet, Path: "/retail/template-schedules/{id}", Tags: tags}, retailTemplateSchedulesRead)
	Register(api, huma.Operation{OperationID: "retail-template-schedules-create", Summary: "Create a recurring template schedule", Description: "The first occurrence fixes the wall-clock time and weekday or month day for later runs.", Method: http.MethodPost, Path: "/retail/template-schedules", Tags: tags}, retailTemplateSchedulesCreate)
	Register(api, huma.Operation{OperationID: "retail-template-schedules-update", Summary: "Update a recurring template schedule", Description: "Changes target, recurrence, next run or activation state; PATCH is available.", Method: http.MethodPut, Path: "/retail/template-schedules/{id}", Tags: tags}, retailTemplateSchedulesUpdate)
	Register(api, huma.Operation{OperationID: "retail-template-schedules-delete", Summary: "Deactivate a recurring template schedule", Method: http.MethodDelete, Path: "/retail/template-schedules/{id}", Tags: tags}, retailTemplateSchedulesDelete)
}

func init() { AddRouteRegistrar(RegisterRetailTemplateScheduleRoutes) }

func retailTemplateSchedulesList(ctx context.Context, in *struct {
	ListParams
	TemplateID      int64 `query:"template_id" doc:"Filter by template."`
	TargetOrgUnitID int64 `query:"target_org_unit_id" doc:"Filter by exact target organization."`
}) (*retailTemplateScheduleListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	model := &models.RetailTemplateSchedule{TemplateID: in.TemplateID, TargetOrgUnitID: in.TargetOrgUnitID}
	result, _, total, err := handler.DoReadAll(ctx, model, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*models.RetailTemplateSchedule)
	if !ok {
		return nil, fmt.Errorf("RetailTemplateSchedule.ReadAll returned unexpected type %T", result)
	}
	return &retailTemplateScheduleListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

type retailTemplateScheduleReadBody struct {
	models.RetailTemplateSchedule
	MaxPermission models.Permission `json:"max_permission" readOnly:"true"`
}

func retailTemplateSchedulesRead(ctx context.Context, in *struct {
	ID int64 `path:"id" doc:"Schedule ID."`
	conditional.Params
}) (*singleReadBody[retailTemplateScheduleReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	schedule := &models.RetailTemplateSchedule{ID: in.ID}
	permission, err := handler.DoReadOne(ctx, schedule, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &retailTemplateScheduleReadBody{RetailTemplateSchedule: *schedule, MaxPermission: models.Permission(permission)}
	return conditionalReadResponse(&in.Params, body, schedule.Updated, permission)
}

func retailTemplateSchedulesCreate(ctx context.Context, in *struct{ Body models.RetailTemplateSchedule }) (*singleBody[models.RetailTemplateSchedule], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailTemplateSchedule]{Body: &in.Body}, nil
}

func retailTemplateSchedulesUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"id" doc:"Schedule ID."`
	Body retailTemplateScheduleReadBody
}) (*singleBody[models.RetailTemplateSchedule], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	schedule := &in.Body.RetailTemplateSchedule
	schedule.ID = in.ID
	if err := handler.DoUpdate(ctx, schedule, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailTemplateSchedule]{Body: schedule}, nil
}

func retailTemplateSchedulesDelete(ctx context.Context, in *struct {
	ID int64 `path:"id" doc:"Schedule ID."`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &models.RetailTemplateSchedule{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}
