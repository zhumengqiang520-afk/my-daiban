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

type retailChecklistItemListBody struct {
	Body Paginated[*models.RetailChecklistItem]
}

func RegisterRetailChecklistItemRoutes(api huma.API) {
	if !config.RetailEnabled.GetBool() {
		return
	}
	tags := []string{"retail tasks"}
	Register(api, huma.Operation{OperationID: "retail-checklist-items-list", Summary: "List a retail task checklist", Description: "Returns the ordered checklist for one visible retail task profile. profile_id is required.", Method: http.MethodGet, Path: "/retail/checklist-items", Tags: tags}, retailChecklistItemsList)
	Register(api, huma.Operation{OperationID: "retail-checklist-items-read", Summary: "Get a retail checklist item", Description: "Returns one checklist definition and its current completion state.", Method: http.MethodGet, Path: "/retail/checklist-items/{id}", Tags: tags}, retailChecklistItemsRead)
	Register(api, huma.Operation{OperationID: "retail-checklist-items-create", Summary: "Add a retail checklist item", Description: "Adds a required or optional verification step. Only organization administrators may change checklist definitions.", Method: http.MethodPost, Path: "/retail/checklist-items", Tags: tags}, retailChecklistItemsCreate)
	Register(api, huma.Operation{OperationID: "retail-checklist-items-update", Summary: "Update a retail checklist item", Description: "Updates title, required policy, and display position. Completion is changed through the dedicated completion action; PATCH is available for partial definition updates.", Method: http.MethodPut, Path: "/retail/checklist-items/{id}", Tags: tags}, retailChecklistItemsUpdate)
	Register(api, huma.Operation{OperationID: "retail-checklist-items-delete", Summary: "Delete a retail checklist item", Description: "Deletes a checklist definition. Only organization administrators may do this.", Method: http.MethodDelete, Path: "/retail/checklist-items/{id}", Tags: tags}, retailChecklistItemsDelete)
}

func init() { AddRouteRegistrar(RegisterRetailChecklistItemRoutes) }

func retailChecklistItemsList(ctx context.Context, in *struct {
	ListParams
	ProfileID int64 `query:"profile_id" minimum:"1" doc:"The retail task profile whose checklist to return."`
}) (*retailChecklistItemListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &models.RetailChecklistItem{ProfileID: in.ProfileID}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*models.RetailChecklistItem)
	if !ok {
		return nil, fmt.Errorf("RetailChecklistItem.ReadAll returned unexpected type %T", result)
	}
	return &retailChecklistItemListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

type retailChecklistItemReadBody struct {
	models.RetailChecklistItem
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"Maximum permission for the requesting user (0=read, 2=admin)."`
}

func retailChecklistItemsRead(ctx context.Context, in *struct {
	ID int64 `path:"id" doc:"Checklist item ID."`
	conditional.Params
}) (*singleReadBody[retailChecklistItemReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	item := &models.RetailChecklistItem{ID: in.ID}
	permission, err := handler.DoReadOne(ctx, item, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &retailChecklistItemReadBody{RetailChecklistItem: *item, MaxPermission: models.Permission(permission)}
	return conditionalReadResponse(&in.Params, body, item.Updated, permission)
}

func retailChecklistItemsCreate(ctx context.Context, in *struct{ Body models.RetailChecklistItem }) (*singleBody[models.RetailChecklistItem], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailChecklistItem]{Body: &in.Body}, nil
}

func retailChecklistItemsUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"id" doc:"Checklist item ID."`
	Body retailChecklistItemReadBody
}) (*singleBody[models.RetailChecklistItem], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	item := &in.Body.RetailChecklistItem
	item.ID = in.ID
	if err := handler.DoUpdate(ctx, item, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailChecklistItem]{Body: item}, nil
}

func retailChecklistItemsDelete(ctx context.Context, in *struct {
	ID int64 `path:"id" doc:"Checklist item ID."`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &models.RetailChecklistItem{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}
