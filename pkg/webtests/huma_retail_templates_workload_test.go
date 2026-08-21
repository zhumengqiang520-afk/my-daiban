// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package webtests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/routes"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHumaRetailTemplatesAndWorkload(t *testing.T) {
	_, err := setupTestEnv()
	require.NoError(t, err)
	oldRetailEnabled := config.RetailEnabled.GetBool()
	oldTimezone := config.ServiceTimeZone.GetString()
	config.RetailEnabled.Set(true)
	config.ServiceTimeZone.Set("Asia/Shanghai")
	defer config.RetailEnabled.Set(oldRetailEnabled)
	defer config.ServiceTimeZone.Set(oldTimezone)
	e := routes.NewEcho()
	routes.RegisterRoutes(e)

	orgHandler := webHandlerTestV2{user: &testuser1, basePath: "/api/v2/retail/org-units", idParam: "id", t: t, e: e}
	companyRec, err := orgHandler.testCreateWithUser(nil, nil, `{"type":"company","name":"Bedding Group","code":"COMP","active":true}`)
	require.NoError(t, err)
	var company models.RetailOrgUnit
	require.NoError(t, json.Unmarshal(companyRec.Body.Bytes(), &company))

	membershipHandler := webHandlerTestV2{user: &testuser1, basePath: "/api/v2/retail/memberships", idParam: "id", t: t, e: e}
	membershipPayload := fmt.Sprintf(`{"org_unit_id":%d,"username":"user2","job_title":"Sales Associate","manager_user_id":1,"primary":true,"admin":false,"active":true}`, company.ID)
	_, err = membershipHandler.testCreateWithUser(nil, nil, membershipPayload)
	require.NoError(t, err)

	templateHandler := webHandlerTestV2{user: &testuser1, basePath: "/api/v2/retail/templates", idParam: "id", t: t, e: e}
	outsiderTemplateHandler := webHandlerTestV2{user: &testuser10, basePath: templateHandler.basePath, idParam: "id", t: t, e: e}
	templatePayload := fmt.Sprintf(`{"org_unit_id":%d,"name":"Daily opening","title":"Open the store","description":"Prepare the bedding displays","category":"opening","estimated_minutes":60,"evidence_required":false,"active":true,"checklist":[{"title":"Check sample beds","required":true,"position":1}]}`, company.ID)
	_, err = outsiderTemplateHandler.testCreateWithUser(nil, nil, templatePayload)
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
	templateRec, err := templateHandler.testCreateWithUser(nil, nil, templatePayload)
	require.NoError(t, err)
	var template models.RetailTaskTemplate
	require.NoError(t, json.Unmarshal(templateRec.Body.Bytes(), &template))
	assert.Equal(t, 1, template.CurrentVersion)

	scheduleHandler := webHandlerTestV2{user: &testuser1, basePath: "/api/v2/retail/template-schedules", idParam: "id", t: t, e: e}
	outsiderScheduleHandler := webHandlerTestV2{user: &testuser10, basePath: scheduleHandler.basePath, idParam: "id", t: t, e: e}
	schedulePayload := fmt.Sprintf(`{"template_id":%d,"target_org_unit_id":%d,"project_id":1,"primary_assignee_id":2,"reviewer_id":1,"frequency":"daily","interval":1,"due_offset_minutes":120,"next_run_at":"2026-08-23T08:00:00+08:00","active":true}`, template.ID, company.ID)
	_, err = outsiderScheduleHandler.testCreateWithUser(nil, nil, schedulePayload)
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
	scheduleRec, err := scheduleHandler.testCreateWithUser(nil, nil, schedulePayload)
	require.NoError(t, err)
	var schedule models.RetailTemplateSchedule
	require.NoError(t, json.Unmarshal(scheduleRec.Body.Bytes(), &schedule))
	assert.Equal(t, models.RetailScheduleDaily, schedule.Frequency)
	staffScheduleHandler := webHandlerTestV2{user: &testuser2, basePath: scheduleHandler.basePath, idParam: "id", t: t, e: e}
	readSchedule, err := staffScheduleHandler.testReadOneWithUser(nil, map[string]string{"id": fmt.Sprint(schedule.ID)})
	require.NoError(t, err)
	assert.Contains(t, readSchedule.Body.String(), `"max_permission":0`)

	ownerToken := humaTokenFor(t, &testuser1)
	staffToken := humaTokenFor(t, &testuser2)
	dispatchBody := fmt.Sprintf(`{"targets":[{"target_org_unit_id":%d,"project_id":1,"primary_assignee_id":2,"reviewer_id":1,"scheduled_for":"2026-08-22T08:00:00+08:00","due_date":"2026-08-22T10:00:00+08:00"}]}`, company.ID)
	previewPath := fmt.Sprintf("/api/v2/retail/templates/%d/dispatch-preview", template.ID)
	rec := humaRequest(t, e, http.MethodPost, previewPath, dispatchBody, ownerToken, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var previews []*models.RetailTemplateDispatchPreview
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &previews))
	require.Len(t, previews, 1)
	assert.False(t, previews[0].AlreadyDispatched)

	dispatchPath := fmt.Sprintf("/api/v2/retail/templates/%d/dispatch", template.ID)
	rec = humaRequest(t, e, http.MethodPost, dispatchPath, dispatchBody, staffToken, "")
	assert.Equal(t, http.StatusForbidden, rec.Code)
	rec = humaRequest(t, e, http.MethodPost, dispatchPath, dispatchBody, ownerToken, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var first []*models.RetailTemplateDispatchResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &first))
	require.Len(t, first, 1)
	assert.False(t, first[0].Reused)
	assert.Equal(t, models.RetailTaskSourceTemplate, first[0].Profile.Source)

	rec = humaRequest(t, e, http.MethodPost, dispatchPath, dispatchBody, ownerToken, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var second []*models.RetailTemplateDispatchResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &second))
	require.Len(t, second, 1)
	assert.True(t, second[0].Reused)
	assert.Equal(t, first[0].Dispatch.TaskID, second[0].Dispatch.TaskID)

	capacityBody := fmt.Sprintf(`{"org_unit_id":%d,"capacity_day":"2026-08-22","minutes":30,"reason":"short shift"}`, company.ID)
	rec = humaRequest(t, e, http.MethodPut, "/api/v2/retail/staff/2/capacity", capacityBody, ownerToken, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/retail/staff/workload?org_unit_id=%d&date_from=2026-08-22&date_to=2026-08-22", company.ID), "", ownerToken, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var workload []*models.RetailStaffWorkload
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &workload))
	found := false
	for _, row := range workload {
		if row.UserID == 2 {
			found = true
			assert.Equal(t, 60, row.AssignedMinutes)
			assert.Equal(t, 30, row.CapacityMinutes)
			assert.True(t, row.Overloaded)
		}
	}
	assert.True(t, found)

	rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/retail/dashboard/operations?org_unit_id=%d&date_from=2026-08-22&date_to=2026-08-22", company.ID), "", ownerToken, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var dashboard models.RetailOperationsDashboard
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &dashboard))
	assert.Equal(t, 1, dashboard.Total)
	assert.Equal(t, 0, dashboard.Completed)
}
