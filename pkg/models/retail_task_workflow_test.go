// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

func setupRetailTaskWorkflow(t *testing.T, s *xorm.Session, evidenceRequired bool) (*RetailTaskProfile, *RetailChecklistItem) {
	t.Helper()
	insertRetailOrgFixture(t, s)
	owner := &user.User{ID: 1, Username: "user1"}
	require.NoError(t, (&RetailMembership{OrgUnitID: 1, Username: "user2", JobTitle: "Sales Associate", Active: true}).Create(s, owner))
	profile := &RetailTaskProfile{
		TaskID:            1,
		OrgUnitID:         1,
		Category:          RetailTaskCategoryOpening,
		PrimaryAssigneeID: 2,
		ReviewerID:        1,
		EstimatedMinutes:  30,
		Source:            RetailTaskSourceManual,
		EvidenceRequired:  evidenceRequired,
	}
	require.NoError(t, profile.Create(s, owner))
	item := &RetailChecklistItem{ProfileID: profile.ID, Title: "Entrance is clean", Required: true, Position: 1}
	require.NoError(t, item.Create(s, owner))
	return profile, item
}

func TestRetailTaskWorkflowReviewCycle(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	profile, item := setupRetailTaskWorkflow(t, s, true)
	owner := &user.User{ID: 1, Username: "user1"}
	staff := &user.User{ID: 2, Username: "user2"}
	outsider := &user.User{ID: 3, Username: "user3"}

	_, err := StartRetailTask(s, 1, outsider)
	assert.True(t, IsErrGenericForbidden(err))
	workflow, err := StartRetailTask(s, 1, staff)
	require.NoError(t, err)
	assert.Equal(t, RetailTaskStatusInProgress, workflow.Profile.Status)

	_, err = SubmitRetailTask(s, 1, "done", nil, staff)
	var incomplete ErrRetailTaskChecklistIncomplete
	require.ErrorAs(t, err, &incomplete)

	_, err = SetRetailChecklistItemDone(s, item.ID, true, outsider)
	assert.True(t, IsErrGenericForbidden(err))
	checked, err := SetRetailChecklistItemDone(s, item.ID, true, staff)
	require.NoError(t, err)
	assert.True(t, checked.Done)

	_, err = SubmitRetailTask(s, 1, "done", nil, staff)
	var evidenceRequired ErrRetailTaskEvidenceRequired
	require.ErrorAs(t, err, &evidenceRequired)
	_, err = s.Insert(&TaskAttachment{ID: 100, TaskID: 1, FileID: 1, CreatedByID: 2})
	require.NoError(t, err)
	workflow, err = SubmitRetailTask(s, 1, "first attempt", []int64{100}, staff)
	require.NoError(t, err)
	assert.Equal(t, RetailTaskStatusPendingReview, workflow.Profile.Status)
	require.Len(t, workflow.Submissions, 1)
	firstSubmissionID := workflow.Submissions[0].ID

	_, err = ReviewRetailTask(s, 1, firstSubmissionID, RetailReviewApproved, "", staff)
	assert.True(t, IsErrGenericForbidden(err))
	_, err = ReviewRetailTask(s, 1, firstSubmissionID, RetailReviewRejected, "", owner)
	assert.True(t, IsErrInvalidData(err))
	workflow, err = ReviewRetailTask(s, 1, firstSubmissionID, RetailReviewRejected, "Photo is unclear", owner)
	require.NoError(t, err)
	assert.Equal(t, RetailTaskStatusRejected, workflow.Profile.Status)

	workflow, err = StartRetailTask(s, 1, staff)
	require.NoError(t, err)
	assert.Equal(t, RetailTaskStatusInProgress, workflow.Profile.Status)
	workflow, err = SubmitRetailTask(s, 1, "clear photo", []int64{100}, staff)
	require.NoError(t, err)
	require.Len(t, workflow.Submissions, 2)
	secondSubmissionID := workflow.Submissions[1].ID
	workflow, err = ReviewRetailTask(s, 1, secondSubmissionID, RetailReviewApproved, "Looks good", owner)
	require.NoError(t, err)
	assert.Equal(t, RetailTaskStatusCompleted, workflow.Profile.Status)
	assert.Len(t, workflow.Reviews, 2)
	assert.Len(t, workflow.Transitions, 6)

	task, err := GetTaskByIDSimple(s, 1)
	require.NoError(t, err)
	assert.True(t, task.Done)
	_, err = StartRetailTask(s, 1, staff)
	var invalidTransition ErrRetailTaskInvalidTransition
	require.ErrorAs(t, err, &invalidTransition)

	stored, err := GetRetailTaskProfileByID(s, profile.ID)
	require.NoError(t, err)
	assert.Equal(t, RetailTaskStatusCompleted, stored.Status)
}

func TestRetailTaskWorkflowAutoCompletesWithoutReviewer(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	profile, item := setupRetailTaskWorkflow(t, s, false)
	owner := &user.User{ID: 1, Username: "user1"}
	staff := &user.User{ID: 2, Username: "user2"}
	profile.ReviewerID = 0
	require.NoError(t, profile.Update(s, owner))
	_, err := StartRetailTask(s, 1, staff)
	require.NoError(t, err)
	_, err = SetRetailChecklistItemDone(s, item.ID, true, staff)
	require.NoError(t, err)
	workflow, err := SubmitRetailTask(s, 1, "done", nil, staff)
	require.NoError(t, err)
	assert.Equal(t, RetailTaskStatusCompleted, workflow.Profile.Status)
}
