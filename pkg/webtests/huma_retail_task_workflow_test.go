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
	"net/http/httptest"
	"testing"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/routes"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHumaRetailTaskWorkflow(t *testing.T) {
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

	profileHandler := webHandlerTestV2{user: &testuser1, basePath: "/api/v2/retail/task-profiles", idParam: "id", t: t, e: e}
	profilePayload := fmt.Sprintf(`{"task_id":1,"org_unit_id":%d,"category":"opening","primary_assignee_id":2,"reviewer_id":1,"estimated_minutes":30,"source":"manual","evidence_required":false}`, company.ID)
	profileRec, err := profileHandler.testCreateWithUser(nil, nil, profilePayload)
	require.NoError(t, err)
	var profile models.RetailTaskProfile
	require.NoError(t, json.Unmarshal(profileRec.Body.Bytes(), &profile))

	checklistHandler := webHandlerTestV2{user: &testuser1, basePath: "/api/v2/retail/checklist-items", idParam: "id", t: t, e: e}
	checklistPayload := fmt.Sprintf(`{"profile_id":%d,"title":"Open the display area","required":true,"position":1}`, profile.ID)
	checklistRec, err := checklistHandler.testCreateWithUser(nil, nil, checklistPayload)
	require.NoError(t, err)
	var checklist models.RetailChecklistItem
	require.NoError(t, json.Unmarshal(checklistRec.Body.Bytes(), &checklist))

	ownerToken := humaTokenFor(t, &testuser1)
	staffToken := humaTokenFor(t, &testuser2)
	outsiderToken := humaTokenFor(t, &testuser10)

	startPath := "/api/v2/retail/tasks/1/start"
	rec := humaRequest(t, e, http.MethodPost, startPath, "", outsiderToken, "")
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = humaRequest(t, e, http.MethodPost, startPath, "", staffToken, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	workflow := decodeRetailWorkflow(t, rec)
	assert.Equal(t, models.RetailTaskStatusInProgress, workflow.Profile.Status)

	rec = humaRequest(t, e, http.MethodPut, fmt.Sprintf("/api/v2/retail/checklist-items/%d/completion", checklist.ID), `{"done":true}`, staffToken, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var completedItem models.RetailChecklistItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &completedItem))
	assert.True(t, completedItem.Done)
	assert.Equal(t, int64(2), completedItem.DoneByID)

	rec = humaRequest(t, e, http.MethodPost, "/api/v2/retail/tasks/1/submissions", `{"note":"opening complete","evidence_attachment_ids":[]}`, staffToken, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	workflow = decodeRetailWorkflow(t, rec)
	require.Len(t, workflow.Submissions, 1)
	assert.Equal(t, models.RetailTaskStatusPendingReview, workflow.Profile.Status)
	submissionID := workflow.Submissions[0].ID

	reviewPath := "/api/v2/retail/tasks/1/reviews"
	reviewBody := fmt.Sprintf(`{"submission_id":%d,"decision":"rejected","comment":"redo the display"}`, submissionID)
	rec = humaRequest(t, e, http.MethodPost, reviewPath, reviewBody, staffToken, "")
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = humaRequest(t, e, http.MethodPost, reviewPath, reviewBody, ownerToken, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	workflow = decodeRetailWorkflow(t, rec)
	assert.Equal(t, models.RetailTaskStatusRejected, workflow.Profile.Status)
	require.Len(t, workflow.Reviews, 1)
	assert.Equal(t, models.RetailReviewRejected, workflow.Reviews[0].Decision)

	rec = humaRequest(t, e, http.MethodPost, startPath, "", staffToken, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	rec = humaRequest(t, e, http.MethodPost, "/api/v2/retail/tasks/1/submissions", `{"note":"display corrected","evidence_attachment_ids":[]}`, staffToken, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	workflow = decodeRetailWorkflow(t, rec)
	require.Len(t, workflow.Submissions, 2)

	approveBody := fmt.Sprintf(`{"submission_id":%d,"decision":"approved","comment":"accepted"}`, workflow.Submissions[1].ID)
	rec = humaRequest(t, e, http.MethodPost, reviewPath, approveBody, ownerToken, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	workflow = decodeRetailWorkflow(t, rec)
	assert.Equal(t, models.RetailTaskStatusCompleted, workflow.Profile.Status)
	require.Len(t, workflow.Reviews, 2)
	assert.Equal(t, models.RetailReviewApproved, workflow.Reviews[1].Decision)

	rec = humaRequest(t, e, http.MethodGet, "/api/v2/retail/tasks/1/workflow", "", staffToken, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	workflow = decodeRetailWorkflow(t, rec)
	assert.Len(t, workflow.Transitions, 6)
	assert.Equal(t, models.RetailTaskStatusCompleted, workflow.Transitions[5].To)

	_, err = profileHandler.testDeleteWithUser(nil, map[string]string{"id": fmt.Sprint(profile.ID)})
	require.NoError(t, err)
	_, err = membershipHandler.testDeleteWithUser(nil, map[string]string{"id": fmt.Sprint(membership.ID)})
	require.NoError(t, err)
	_, err = orgHandler.testDeleteWithUser(nil, map[string]string{"id": fmt.Sprint(company.ID)})
	require.NoError(t, err)
}

func decodeRetailWorkflow(t *testing.T, rec *httptest.ResponseRecorder) *models.RetailTaskWorkflow {
	t.Helper()
	workflow := &models.RetailTaskWorkflow{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), workflow))
	return workflow
}
