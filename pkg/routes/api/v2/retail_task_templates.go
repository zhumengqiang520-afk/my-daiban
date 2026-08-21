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
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/conditional"
)

type retailTaskTemplateListBody struct {
	Body Paginated[*models.RetailTaskTemplate]
}

func RegisterRetailTaskTemplateRoutes(api huma.API) {
	if !config.RetailEnabled.GetBool() {
		return
	}
	tags := []string{"retail templates"}
	Register(api, huma.Operation{OperationID: "retail-task-templates-list", Summary: "List retail task templates", Description: "Returns templates visible through the caller's organization scope.", Method: http.MethodGet, Path: "/retail/templates", Tags: tags}, retailTaskTemplatesList)
	Register(api, huma.Operation{OperationID: "retail-task-templates-read", Summary: "Get a retail task template", Description: "Returns the current template definition and checklist.", Method: http.MethodGet, Path: "/retail/templates/{id}", Tags: tags}, retailTaskTemplatesRead)
	Register(api, huma.Operation{OperationID: "retail-task-templates-create", Summary: "Create a retail task template", Description: "Creates an editable template and its first immutable version.", Method: http.MethodPost, Path: "/retail/templates", Tags: tags}, retailTaskTemplatesCreate)
	Register(api, huma.Operation{OperationID: "retail-task-templates-update", Summary: "Update a retail task template", Description: "Creates a new immutable version; PATCH is available for partial updates.", Method: http.MethodPut, Path: "/retail/templates/{id}", Tags: tags}, retailTaskTemplatesUpdate)
	Register(api, huma.Operation{OperationID: "retail-task-templates-delete", Summary: "Deactivate a retail task template", Description: "Stops future dispatches while preserving task provenance.", Method: http.MethodDelete, Path: "/retail/templates/{id}", Tags: tags}, retailTaskTemplatesDelete)
	Register(api, huma.Operation{OperationID: "retail-task-template-dispatch-preview", Summary: "Preview a template dispatch", Description: "Validates target, project, dates and idempotency without creating a task.", Method: http.MethodPost, Path: "/retail/templates/{id}/dispatch-preview", Tags: tags}, retailTaskTemplateDispatchPreview)
	Register(api, huma.Operation{OperationID: "retail-task-template-dispatch", Summary: "Dispatch a retail task template", Description: "Creates one task per target from an immutable version. Repeated idempotency keys return the existing task.", Method: http.MethodPost, Path: "/retail/templates/{id}/dispatch", Tags: tags}, retailTaskTemplateDispatch)
}

func init() { AddRouteRegistrar(RegisterRetailTaskTemplateRoutes) }

func retailTaskTemplatesList(ctx context.Context, in *struct {
	ListParams
	OrgUnitID int64 `query:"org_unit_id" doc:"Return templates owned by this organization only."`
}) (*retailTaskTemplateListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &models.RetailTaskTemplate{OrgUnitID: in.OrgUnitID}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*models.RetailTaskTemplate)
	if !ok {
		return nil, fmt.Errorf("RetailTaskTemplate.ReadAll returned unexpected type %T", result)
	}
	return &retailTaskTemplateListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

type retailTaskTemplateReadBody struct {
	models.RetailTaskTemplate
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"Maximum permission for the requesting user (0=read, 2=admin)."`
}

func retailTaskTemplatesRead(ctx context.Context, in *struct {
	ID int64 `path:"id" doc:"Retail task template ID."`
	conditional.Params
}) (*singleReadBody[retailTaskTemplateReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	template := &models.RetailTaskTemplate{ID: in.ID}
	permission, err := handler.DoReadOne(ctx, template, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &retailTaskTemplateReadBody{RetailTaskTemplate: *template, MaxPermission: models.Permission(permission)}
	return conditionalReadResponse(&in.Params, body, template.Updated, permission)
}

func retailTaskTemplatesCreate(ctx context.Context, in *struct{ Body models.RetailTaskTemplate }) (*singleBody[models.RetailTaskTemplate], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailTaskTemplate]{Body: &in.Body}, nil
}

func retailTaskTemplatesUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"id" doc:"Retail task template ID."`
	Body retailTaskTemplateReadBody
}) (*singleBody[models.RetailTaskTemplate], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	template := &in.Body.RetailTaskTemplate
	template.ID = in.ID
	if err := handler.DoUpdate(ctx, template, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailTaskTemplate]{Body: template}, nil
}

func retailTaskTemplatesDelete(ctx context.Context, in *struct {
	ID int64 `path:"id" doc:"Retail task template ID."`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &models.RetailTaskTemplate{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}

func retailTaskTemplateDispatchPreview(ctx context.Context, in *struct {
	ID   int64 `path:"id" doc:"Retail task template ID."`
	Body struct {
		Targets []models.RetailTemplateDispatchInput `json:"targets" minItems:"1" maxItems:"100" doc:"One or more target task occurrences."`
	}
}) (*singleBody[[]*models.RetailTemplateDispatchPreview], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()
	previews := make([]*models.RetailTemplateDispatchPreview, 0, len(in.Body.Targets))
	for _, target := range in.Body.Targets {
		preview, err := models.PreviewRetailTaskTemplateDispatch(s, in.ID, target, a)
		if err != nil {
			rollbackRetailWorkflow(s)
			return nil, translateDomainError(err)
		}
		previews = append(previews, preview)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[[]*models.RetailTemplateDispatchPreview]{Body: &previews}, nil
}

func retailTaskTemplateDispatch(ctx context.Context, in *struct {
	ID   int64 `path:"id" doc:"Retail task template ID."`
	Body struct {
		Targets []models.RetailTemplateDispatchInput `json:"targets" minItems:"1" maxItems:"100" doc:"One or more target task occurrences."`
	}
}) (*singleBody[[]*models.RetailTemplateDispatchResult], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()
	results := make([]*models.RetailTemplateDispatchResult, 0, len(in.Body.Targets))
	for _, target := range in.Body.Targets {
		result, err := models.DispatchRetailTaskTemplate(s, in.ID, target, a)
		if err != nil {
			rollbackRetailWorkflow(s)
			return nil, translateDomainError(err)
		}
		results = append(results, result)
	}
	if err := commitRetailWorkflow(ctx, s); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[[]*models.RetailTemplateDispatchResult]{Body: &results}, nil
}
