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
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/routes"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHumaRetailOrgUnits(t *testing.T) {
	_, err := setupTestEnv()
	require.NoError(t, err)
	oldRetailEnabled := config.RetailEnabled.GetBool()
	config.RetailEnabled.Set(true)
	defer config.RetailEnabled.Set(oldRetailEnabled)
	e := routes.NewEcho()
	routes.RegisterRoutes(e)

	s := db.NewSession()
	_, err = s.Where("id > ?", 0).Delete(&models.RetailOrgUnit{})
	require.NoError(t, err)
	require.NoError(t, s.Commit())
	require.NoError(t, s.Close())

	owner := webHandlerTestV2{user: &testuser1, basePath: "/api/v2/retail/org-units", idParam: "id", t: t, e: e}
	outsider := webHandlerTestV2{user: &testuser2, basePath: owner.basePath, idParam: "id", t: t, e: e}

	companyRec, err := owner.testCreateWithUser(nil, nil, `{"type":"company","name":"Bedding Group","code":"COMP","active":true}`)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, companyRec.Code)
	var company models.RetailOrgUnit
	require.NoError(t, json.Unmarshal(companyRec.Body.Bytes(), &company))
	require.NotZero(t, company.ID)
	require.NotZero(t, company.TeamID)

	regionPayload := fmt.Sprintf(`{"parent_id":%d,"type":"region","name":"East Region","code":"EAST","active":true}`, company.ID)
	_, err = outsider.testCreateWithUser(nil, nil, regionPayload)
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err), "only parent administrators may create child units")

	regionRec, err := owner.testCreateWithUser(nil, nil, regionPayload)
	require.NoError(t, err)
	var region models.RetailOrgUnit
	require.NoError(t, json.Unmarshal(regionRec.Body.Bytes(), &region))
	require.NotZero(t, region.ID)

	readRec, err := owner.testReadOneWithUser(nil, map[string]string{"id": fmt.Sprint(region.ID)})
	require.NoError(t, err)
	assert.Contains(t, readRec.Body.String(), `"max_permission":2`)
	assert.NotEmpty(t, readRec.Header().Get("ETag"))

	_, err = outsider.testReadOneWithUser(nil, map[string]string{"id": fmt.Sprint(region.ID)})
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err), "unrelated users must not see another store hierarchy")

	listRec, err := owner.testReadAllWithUser(nil, nil)
	require.NoError(t, err)
	assert.Contains(t, listRec.Body.String(), `"total":2`)
	assert.Contains(t, listRec.Body.String(), `"code":"EAST"`)

	updatePayload := `{"name":"Eastern Region","code":"EAST","active":true}`
	updatedRec, err := owner.testUpdateWithUser(nil, map[string]string{"id": fmt.Sprint(region.ID)}, updatePayload)
	require.NoError(t, err)
	assert.Contains(t, updatedRec.Body.String(), `"name":"Eastern Region"`)

	_, err = outsider.testUpdateWithUser(nil, map[string]string{"id": fmt.Sprint(region.ID)}, updatePayload)
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))

	deleteRec, err := owner.testDeleteWithUser(nil, map[string]string{"id": fmt.Sprint(region.ID)})
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, deleteRec.Code)
	_, err = owner.testDeleteWithUser(nil, map[string]string{"id": fmt.Sprint(company.ID)})
	require.NoError(t, err)
}

func TestHumaRetailOrgUnitsDisabled(t *testing.T) {
	_, err := setupTestEnv()
	require.NoError(t, err)
	oldRetailEnabled := config.RetailEnabled.GetBool()
	config.RetailEnabled.Set(false)
	defer config.RetailEnabled.Set(oldRetailEnabled)

	e := routes.NewEcho()
	routes.RegisterRoutes(e)
	rec := humaRequest(t, e, http.MethodGet, "/api/v2/retail/org-units", "", humaTokenFor(t, &testuser1), "")
	assert.Equal(t, http.StatusNotFound, rec.Code, "retail routes must be absent while retail.enabled is false")
}
