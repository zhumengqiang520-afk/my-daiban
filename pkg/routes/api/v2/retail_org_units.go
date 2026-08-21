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

type retailOrgUnitListBody struct {
	Body Paginated[*models.RetailOrgUnit]
}

func RegisterRetailOrgUnitRoutes(api huma.API) {
	if !config.RetailEnabled.GetBool() {
		return
	}
	tags := []string{"retail organization"}
	Register(api, huma.Operation{OperationID: "retail-org-units-list", Summary: "List retail organization units", Method: http.MethodGet, Path: "/retail/org-units", Tags: tags}, retailOrgUnitsList)
	Register(api, huma.Operation{OperationID: "retail-org-units-read", Summary: "Get a retail organization unit", Method: http.MethodGet, Path: "/retail/org-units/{id}", Tags: tags}, retailOrgUnitsRead)
	Register(api, huma.Operation{OperationID: "retail-org-units-create", Summary: "Create a retail organization unit", Method: http.MethodPost, Path: "/retail/org-units", Tags: tags}, retailOrgUnitsCreate)
	Register(api, huma.Operation{OperationID: "retail-org-units-update", Summary: "Update a retail organization unit", Method: http.MethodPut, Path: "/retail/org-units/{id}", Tags: tags}, retailOrgUnitsUpdate)
	Register(api, huma.Operation{OperationID: "retail-org-units-delete", Summary: "Delete a retail organization unit", Method: http.MethodDelete, Path: "/retail/org-units/{id}", Tags: tags}, retailOrgUnitsDelete)
}

func init() { AddRouteRegistrar(RegisterRetailOrgUnitRoutes) }

func retailOrgUnitsList(ctx context.Context, in *struct{ ListParams }) (*retailOrgUnitListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &models.RetailOrgUnit{}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*models.RetailOrgUnit)
	if !ok {
		return nil, fmt.Errorf("RetailOrgUnit.ReadAll returned unexpected type %T", result)
	}
	return &retailOrgUnitListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

type retailOrgUnitReadBody struct {
	models.RetailOrgUnit
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"Maximum permission for the requesting user (0=read, 2=admin)."`
}

func retailOrgUnitsRead(ctx context.Context, in *struct {
	ID int64 `path:"id" doc:"Organization unit ID."`
	conditional.Params
}) (*singleReadBody[retailOrgUnitReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	unit := &models.RetailOrgUnit{ID: in.ID}
	permission, err := handler.DoReadOne(ctx, unit, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &retailOrgUnitReadBody{RetailOrgUnit: *unit, MaxPermission: models.Permission(permission)}
	return conditionalReadResponse(&in.Params, body, unit.Updated, permission)
}

func retailOrgUnitsCreate(ctx context.Context, in *struct{ Body models.RetailOrgUnit }) (*singleBody[models.RetailOrgUnit], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailOrgUnit]{Body: &in.Body}, nil
}

func retailOrgUnitsUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"id" doc:"Organization unit ID."`
	Body retailOrgUnitReadBody
}) (*singleBody[models.RetailOrgUnit], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	unit := &in.Body.RetailOrgUnit
	unit.ID = in.ID
	if err := handler.DoUpdate(ctx, unit, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailOrgUnit]{Body: unit}, nil
}

func retailOrgUnitsDelete(ctx context.Context, in *struct {
	ID int64 `path:"id" doc:"Organization unit ID."`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &models.RetailOrgUnit{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}
