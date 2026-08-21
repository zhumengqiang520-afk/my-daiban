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
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterRetailDashboardRoutes(api huma.API) {
	if !config.RetailEnabled.GetBool() {
		return
	}
	Register(api, huma.Operation{
		OperationID: "retail-operations-dashboard", Summary: "Get the retail operations dashboard",
		Description: "Returns completion, on-time, rejection and overdue metrics plus the actionable overdue queue for a visible organization scope.",
		Method:      http.MethodGet, Path: "/retail/dashboard/operations", Tags: []string{"retail dashboard"},
	}, retailOperationsDashboard)
}

func init() { AddRouteRegistrar(RegisterRetailDashboardRoutes) }

func retailOperationsDashboard(ctx context.Context, in *struct {
	OrgUnitID  int64                     `query:"org_unit_id" minimum:"1" doc:"Organization scope, including descendants."`
	DateFrom   string                    `query:"date_from" minLength:"10" maxLength:"10" doc:"First business day in YYYY-MM-DD format."`
	DateTo     string                    `query:"date_to" minLength:"10" maxLength:"10" doc:"Last business day in YYYY-MM-DD format."`
	Category   models.RetailTaskCategory `query:"category" doc:"Optional retail category filter."`
	AssigneeID int64                     `query:"assignee_id" doc:"Optional primary assignee filter."`
}) (*singleBody[models.RetailOperationsDashboard], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	from, err := time.ParseInLocation("2006-01-02", in.DateFrom, config.GetTimeZone())
	if err != nil {
		return nil, translateDomainError(models.ErrInvalidData{Message: "date_from must use YYYY-MM-DD."})
	}
	to, err := time.ParseInLocation("2006-01-02", in.DateTo, config.GetTimeZone())
	if err != nil {
		return nil, translateDomainError(models.ErrInvalidData{Message: "date_to must use YYYY-MM-DD."})
	}
	s := db.NewSession()
	defer s.Close()
	dashboard, err := models.GetRetailOperationsDashboard(s, in.OrgUnitID, from, to, in.Category, in.AssigneeID, time.Now(), a)
	if err != nil {
		rollbackRetailWorkflow(s)
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailOperationsDashboard]{Body: dashboard}, nil
}
