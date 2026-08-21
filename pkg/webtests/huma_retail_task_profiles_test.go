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

func TestHumaRetailTaskProfiles(t *testing.T) {
	_, err := setupTestEnv()
	require.NoError(t, err)
	oldRetailEnabled := config.RetailEnabled.GetBool()
	config.RetailEnabled.Set(true)
	defer config.RetailEnabled.Set(oldRetailEnabled)
	e := routes.NewEcho()
	routes.RegisterRoutes(e)

	orgHandler := webHandlerTestV2{user: &testuser1, basePath: "/api/v2/retail/org-units", idParam: "id", t: t, e: e}
	companyRec, err := orgHandler.testCreateWithUser(nil, nil, `{"type":"company","name":"Bedding Group","code":"COMP","active":true}`)
	require.NoError(t, err)
	var company models.RetailOrgUnit
	require.NoError(t, json.Unmarshal(companyRec.Body.Bytes(), &company))

	membershipHandler := webHandlerTestV2{user: &testuser1, basePath: "/api/v2/retail/memberships", idParam: "id", t: t, e: e}
	membershipPayload := fmt.Sprintf(`{"org_unit_id":%d,"username":"user2","job_title":"Sales Associate","manager_user_id":1,"primary":true,"admin":false,"active":true}`, company.ID)
	membershipRec, err := membershipHandler.testCreateWithUser(nil, nil, membershipPayload)
	require.NoError(t, err)
	var membership models.RetailMembership
	require.NoError(t, json.Unmarshal(membershipRec.Body.Bytes(), &membership))

	owner := webHandlerTestV2{user: &testuser1, basePath: "/api/v2/retail/task-profiles", idParam: "id", t: t, e: e}
	staff := webHandlerTestV2{user: &testuser2, basePath: owner.basePath, idParam: "id", t: t, e: e}
	outsider := webHandlerTestV2{user: &testuser10, basePath: owner.basePath, idParam: "id", t: t, e: e}
	payload := fmt.Sprintf(`{"task_id":1,"org_unit_id":%d,"category":"opening","primary_assignee_id":2,"reviewer_id":1,"estimated_minutes":30,"source":"manual","evidence_required":true}`, company.ID)

	_, err = outsider.testCreateWithUser(nil, nil, payload)
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))

	createdRec, err := owner.testCreateWithUser(nil, nil, payload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, createdRec.Code)
	var profile models.RetailTaskProfile
	require.NoError(t, json.Unmarshal(createdRec.Body.Bytes(), &profile))
	require.NotZero(t, profile.ID)
	assert.Equal(t, models.RetailTaskStatusAssigned, profile.Status)
	assert.Equal(t, "task #1", profile.TaskTitle)

	readRec, err := staff.testReadOneWithUser(nil, map[string]string{"id": fmt.Sprint(profile.ID)})
	require.NoError(t, err)
	assert.Contains(t, readRec.Body.String(), `"max_permission":0`)
	_, err = outsider.testReadOneWithUser(nil, map[string]string{"id": fmt.Sprint(profile.ID)})
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))

	listRec, err := owner.testReadAllWithUser(nil, nil)
	require.NoError(t, err)
	assert.Contains(t, listRec.Body.String(), `"total":1`)
	assert.Contains(t, listRec.Body.String(), `"category":"opening"`)

	updatePayload := fmt.Sprintf(`{"task_id":1,"org_unit_id":%d,"category":"display","primary_assignee_id":2,"reviewer_id":1,"estimated_minutes":45,"status":"completed","source":"manual","evidence_required":false}`, company.ID)
	updatedRec, err := owner.testUpdateWithUser(nil, map[string]string{"id": fmt.Sprint(profile.ID)}, updatePayload)
	require.NoError(t, err)
	assert.Contains(t, updatedRec.Body.String(), `"category":"display"`)
	assert.Contains(t, updatedRec.Body.String(), `"status":"assigned"`, "PUT cannot bypass the workflow state machine")

	_, err = owner.testDeleteWithUser(nil, map[string]string{"id": fmt.Sprint(profile.ID)})
	require.NoError(t, err)
	_, err = membershipHandler.testDeleteWithUser(nil, map[string]string{"id": fmt.Sprint(membership.ID)})
	require.NoError(t, err)
	_, err = orgHandler.testDeleteWithUser(nil, map[string]string{"id": fmt.Sprint(company.ID)})
	require.NoError(t, err)
}
