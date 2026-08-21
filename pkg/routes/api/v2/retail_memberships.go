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

type retailMembershipListBody struct {
	Body Paginated[*models.RetailMembership]
}

func RegisterRetailMembershipRoutes(api huma.API) {
	if !config.RetailEnabled.GetBool() {
		return
	}
	tags := []string{"retail staff"}
	Register(api, huma.Operation{OperationID: "retail-memberships-list", Summary: "List retail staff memberships", Description: "Returns only staff assignments in organization scopes visible to the caller. Filter by organization unit with org_unit_id.", Method: http.MethodGet, Path: "/retail/memberships", Tags: tags}, retailMembershipsList)
	Register(api, huma.Operation{OperationID: "retail-memberships-read", Summary: "Get a retail staff membership", Description: "Returns employment metadata and the caller's effective organization permission.", Method: http.MethodGet, Path: "/retail/memberships/{id}", Tags: tags}, retailMembershipsRead)
	Register(api, huma.Operation{OperationID: "retail-memberships-create", Summary: "Assign staff to a retail organization unit", Description: "Creates the staff profile and grants the matching Vikunja team access. Only an administrator of the unit or an ancestor may assign staff.", Method: http.MethodPost, Path: "/retail/memberships", Tags: tags}, retailMembershipsCreate)
	Register(api, huma.Operation{OperationID: "retail-memberships-update", Summary: "Update a retail staff membership", Description: "Updates job, manager, primary assignment, temporary period, active state, and organization-admin access. Organization and user are immutable; PATCH is available for partial updates.", Method: http.MethodPut, Path: "/retail/memberships/{id}", Tags: tags}, retailMembershipsUpdate)
	Register(api, huma.Operation{OperationID: "retail-memberships-delete", Summary: "Remove a retail staff membership", Description: "Removes the profile and revokes the matching team access. The last active organization administrator cannot be removed.", Method: http.MethodDelete, Path: "/retail/memberships/{id}", Tags: tags}, retailMembershipsDelete)
}

func init() { AddRouteRegistrar(RegisterRetailMembershipRoutes) }

func retailMembershipsList(ctx context.Context, in *struct {
	ListParams
	OrgUnitID int64 `query:"org_unit_id" doc:"Return assignments for this organization unit only."`
}) (*retailMembershipListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	model := &models.RetailMembership{OrgUnitID: in.OrgUnitID}
	result, _, total, err := handler.DoReadAll(ctx, model, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*models.RetailMembership)
	if !ok {
		return nil, fmt.Errorf("RetailMembership.ReadAll returned unexpected type %T", result)
	}
	return &retailMembershipListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

type retailMembershipReadBody struct {
	models.RetailMembership
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"Maximum permission for the requesting user (0=read, 2=admin)."`
}

func retailMembershipsRead(ctx context.Context, in *struct {
	ID int64 `path:"id" doc:"Staff membership ID."`
	conditional.Params
}) (*singleReadBody[retailMembershipReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	membership := &models.RetailMembership{ID: in.ID}
	permission, err := handler.DoReadOne(ctx, membership, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &retailMembershipReadBody{RetailMembership: *membership, MaxPermission: models.Permission(permission)}
	return conditionalReadResponse(&in.Params, body, membership.Updated, permission)
}

func retailMembershipsCreate(ctx context.Context, in *struct{ Body models.RetailMembership }) (*singleBody[models.RetailMembership], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailMembership]{Body: &in.Body}, nil
}

func retailMembershipsUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"id" doc:"Staff membership ID."`
	Body retailMembershipReadBody
}) (*singleBody[models.RetailMembership], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	membership := &in.Body.RetailMembership
	membership.ID = in.ID
	if err := handler.DoUpdate(ctx, membership, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailMembership]{Body: membership}, nil
}

func retailMembershipsDelete(ctx context.Context, in *struct {
	ID int64 `path:"id" doc:"Staff membership ID."`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &models.RetailMembership{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}
