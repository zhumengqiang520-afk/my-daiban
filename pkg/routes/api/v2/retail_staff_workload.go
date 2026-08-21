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

func RegisterRetailStaffWorkloadRoutes(api huma.API) {
	if !config.RetailEnabled.GetBool() {
		return
	}
	tags := []string{"retail workforce"}
	Register(api, huma.Operation{OperationID: "retail-staff-workload", Summary: "Get retail staff workload", Description: "Returns daily capacity, assigned minutes, task count and overload warnings for visible staff. Date range is limited to 32 days.", Method: http.MethodGet, Path: "/retail/staff/workload", Tags: tags}, retailStaffWorkload)
	Register(api, huma.Operation{OperationID: "retail-staff-capacity-set", Summary: "Set staff capacity for one day", Description: "Creates or replaces a daily capacity override. Zero minutes represents an unavailable day.", Method: http.MethodPut, Path: "/retail/staff/{id}/capacity", Tags: tags}, retailStaffCapacitySet)
}

func init() { AddRouteRegistrar(RegisterRetailStaffWorkloadRoutes) }

func retailStaffWorkload(ctx context.Context, in *struct {
	OrgUnitID int64  `query:"org_unit_id" minimum:"1" doc:"Organization scope to include, including descendant stores."`
	DateFrom  string `query:"date_from" minLength:"10" maxLength:"10" doc:"First business day in YYYY-MM-DD format."`
	DateTo    string `query:"date_to" minLength:"10" maxLength:"10" doc:"Last business day in YYYY-MM-DD format."`
}) (*singleBody[[]*models.RetailStaffWorkload], error) {
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
	items, err := models.GetRetailStaffWorkload(s, in.OrgUnitID, from, to, a)
	if err != nil {
		rollbackRetailWorkflow(s)
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[[]*models.RetailStaffWorkload]{Body: &items}, nil
}

func retailStaffCapacitySet(ctx context.Context, in *struct {
	UserID int64 `path:"id" minimum:"1" doc:"Retail staff user ID."`
	Body   struct {
		OrgUnitID   int64  `json:"org_unit_id" minimum:"1" doc:"The staff member's organization assignment."`
		CapacityDay string `json:"capacity_day" minLength:"10" maxLength:"10" doc:"Business day in YYYY-MM-DD format."`
		Minutes     int    `json:"minutes" minimum:"0" maximum:"1440" doc:"Available minutes for the day."`
		Reason      string `json:"reason" maxLength:"500" doc:"Why the default capacity was overridden."`
	}
}) (*singleBody[models.RetailStaffCapacity], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	day, err := time.ParseInLocation("2006-01-02", in.Body.CapacityDay, config.GetTimeZone())
	if err != nil {
		return nil, translateDomainError(models.ErrInvalidData{Message: "capacity_day must use YYYY-MM-DD."})
	}
	s := db.NewSession()
	defer s.Close()
	capacity, err := models.SetRetailStaffCapacity(s, in.Body.OrgUnitID, in.UserID, day, in.Body.Minutes, in.Body.Reason, a)
	if err != nil {
		rollbackRetailWorkflow(s)
		return nil, translateDomainError(err)
	}
	if err := commitRetailWorkflow(ctx, s); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.RetailStaffCapacity]{Body: capacity}, nil
}
