// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"strings"
	"time"

	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

type RetailSubmission struct {
	ID                    int64     `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique numeric ID of this completion submission."`
	ProfileID             int64     `xorm:"bigint not null INDEX" json:"profile_id" readOnly:"true" doc:"The retail task profile submitted for review."`
	SubmittedByID         int64     `xorm:"bigint not null INDEX" json:"submitted_by_id" readOnly:"true" doc:"The user who submitted the task for review."`
	Note                  string    `xorm:"longtext null" json:"note" doc:"An optional completion note for the reviewer."`
	EvidenceAttachmentIDs []int64   `xorm:"-" json:"evidence_attachment_ids" doc:"Task attachment IDs used as completion evidence."`
	Created               time.Time `xorm:"created not null INDEX" json:"created" readOnly:"true" doc:"When this completion was submitted."`
}

func (*RetailSubmission) TableName() string { return "retail_submissions" }

type RetailSubmissionFile struct {
	ID           int64 `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique numeric ID of this evidence relation."`
	SubmissionID int64 `xorm:"bigint not null INDEX unique(retail_submission_attachment)" json:"submission_id" readOnly:"true" doc:"The completion submission this evidence belongs to."`
	AttachmentID int64 `xorm:"bigint not null INDEX unique(retail_submission_attachment)" json:"attachment_id" readOnly:"true" doc:"The Vikunja task attachment used as evidence."`
}

func (*RetailSubmissionFile) TableName() string { return "retail_submission_files" }

type RetailReviewDecision string

const (
	RetailReviewApproved RetailReviewDecision = "approved"
	RetailReviewRejected RetailReviewDecision = "rejected"
)

type RetailReview struct {
	ID           int64                `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique numeric ID of this review."`
	ProfileID    int64                `xorm:"bigint not null INDEX" json:"profile_id" readOnly:"true" doc:"The reviewed retail task profile."`
	SubmissionID int64                `xorm:"bigint not null INDEX" json:"submission_id" doc:"The completion submission being reviewed."`
	ReviewerID   int64                `xorm:"bigint not null INDEX" json:"reviewer_id" readOnly:"true" doc:"The user who made the review decision."`
	Decision     RetailReviewDecision `xorm:"varchar(20) not null INDEX" json:"decision" doc:"The decision: approved or rejected."`
	Comment      string               `xorm:"longtext null" json:"comment" doc:"The review comment. Required when rejecting."`
	Created      time.Time            `xorm:"created not null INDEX" json:"created" readOnly:"true" doc:"When the review decision was made."`
}

func (*RetailReview) TableName() string { return "retail_reviews" }

type RetailTaskTransition struct {
	ID        int64            `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique numeric ID of this workflow transition."`
	ProfileID int64            `xorm:"bigint not null INDEX" json:"profile_id" readOnly:"true" doc:"The retail task profile whose state changed."`
	From      RetailTaskStatus `xorm:"varchar(30) not null INDEX" json:"from" readOnly:"true" doc:"The workflow status before the transition."`
	To        RetailTaskStatus `xorm:"varchar(30) not null INDEX" json:"to" readOnly:"true" doc:"The workflow status after the transition."`
	ActorID   int64            `xorm:"bigint not null INDEX" json:"actor_id" readOnly:"true" doc:"The user who caused this transition."`
	Reason    string           `xorm:"longtext null" json:"reason" readOnly:"true" doc:"The note or review reason associated with the transition."`
	Created   time.Time        `xorm:"created not null INDEX" json:"created" readOnly:"true" doc:"When the transition occurred."`
}

func (*RetailTaskTransition) TableName() string { return "retail_task_transitions" }

type RetailTaskWorkflow struct {
	Profile     *RetailTaskProfile      `json:"profile" readOnly:"true" doc:"The retail task profile and current workflow status."`
	Checklist   []*RetailChecklistItem  `json:"checklist" readOnly:"true" doc:"The ordered completion checklist."`
	Submissions []*RetailSubmission     `json:"submissions" readOnly:"true" doc:"Every completion submission, oldest first."`
	Reviews     []*RetailReview         `json:"reviews" readOnly:"true" doc:"Every review decision, oldest first."`
	Transitions []*RetailTaskTransition `json:"transitions" readOnly:"true" doc:"The immutable business workflow history."`
}

func GetRetailTaskProfileByTaskID(s *xorm.Session, taskID int64) (*RetailTaskProfile, error) {
	profile := &RetailTaskProfile{}
	exists, err := s.Where("task_id = ?", taskID).Get(profile)
	if err != nil {
		return profile, err
	}
	if !exists {
		return profile, ErrRetailTaskProfileDoesNotExist{ID: taskID}
	}
	if err := addRetailTaskProfileInfo(s, []*RetailTaskProfile{profile}); err != nil {
		return profile, err
	}
	return profile, nil
}

func StartRetailTask(s *xorm.Session, taskID int64, a web.Auth) (*RetailTaskWorkflow, error) {
	profile, err := GetRetailTaskProfileByTaskID(s, taskID)
	if err != nil {
		return nil, err
	}
	if err := requireRetailTaskWorkerOrAdmin(s, profile, a); err != nil {
		return nil, err
	}
	if profile.Status != RetailTaskStatusAssigned && profile.Status != RetailTaskStatusRejected {
		return nil, ErrRetailTaskInvalidTransition{From: profile.Status, To: RetailTaskStatusInProgress}
	}
	if err := transitionRetailTask(s, profile, RetailTaskStatusInProgress, a, ""); err != nil {
		return nil, err
	}
	return GetRetailTaskWorkflow(s, taskID, a)
}

func SetRetailChecklistItemDone(s *xorm.Session, itemID int64, done bool, a web.Auth) (*RetailChecklistItem, error) {
	item, err := GetRetailChecklistItemByID(s, itemID)
	if err != nil {
		return nil, err
	}
	profile, err := GetRetailTaskProfileByID(s, item.ProfileID)
	if err != nil {
		return nil, err
	}
	if err := requireRetailTaskWorkerOrAdmin(s, profile, a); err != nil {
		return nil, err
	}
	if profile.Status != RetailTaskStatusInProgress {
		return nil, ErrRetailTaskInvalidTransition{From: profile.Status, To: RetailTaskStatusInProgress}
	}
	item.Done = done
	if done {
		item.DoneByID = a.GetID()
		item.DoneAt = time.Now()
	} else {
		item.DoneByID = 0
		item.DoneAt = time.Time{}
	}
	_, err = s.ID(item.ID).Cols("done", "done_by_id", "done_at").Update(item)
	return item, err
}

func SubmitRetailTask(s *xorm.Session, taskID int64, note string, attachmentIDs []int64, a web.Auth) (*RetailTaskWorkflow, error) {
	profile, err := GetRetailTaskProfileByTaskID(s, taskID)
	if err != nil {
		return nil, err
	}
	if err := requireRetailTaskWorkerOrAdmin(s, profile, a); err != nil {
		return nil, err
	}
	if profile.Status != RetailTaskStatusInProgress {
		return nil, ErrRetailTaskInvalidTransition{From: profile.Status, To: RetailTaskStatusPendingReview}
	}
	incomplete, err := s.Where("profile_id = ?", profile.ID).And("required = ?", true).And("done = ?", false).Count(&RetailChecklistItem{})
	if err != nil {
		return nil, err
	}
	if incomplete > 0 {
		return nil, ErrRetailTaskChecklistIncomplete{ProfileID: profile.ID}
	}
	attachmentIDs = uniqueInt64s(attachmentIDs)
	if profile.EvidenceRequired && len(attachmentIDs) == 0 {
		return nil, ErrRetailTaskEvidenceRequired{ProfileID: profile.ID}
	}
	for _, attachmentID := range attachmentIDs {
		exists, attachmentErr := s.Where("id = ?", attachmentID).And("task_id = ?", taskID).Exist(&TaskAttachment{})
		if attachmentErr != nil {
			return nil, attachmentErr
		}
		if !exists {
			return nil, ErrRetailTaskInvalidAttachment{TaskID: taskID, AttachmentID: attachmentID}
		}
	}
	submission := &RetailSubmission{ProfileID: profile.ID, SubmittedByID: a.GetID(), Note: strings.TrimSpace(note)}
	if _, err := s.Insert(submission); err != nil {
		return nil, err
	}
	for _, attachmentID := range attachmentIDs {
		if _, err := s.Insert(&RetailSubmissionFile{SubmissionID: submission.ID, AttachmentID: attachmentID}); err != nil {
			return nil, err
		}
	}
	next := RetailTaskStatusPendingReview
	if profile.ReviewerID == 0 {
		next = RetailTaskStatusCompleted
	}
	if err := transitionRetailTask(s, profile, next, a, submission.Note); err != nil {
		return nil, err
	}
	if next == RetailTaskStatusCompleted {
		if err := setRetailUnderlyingTaskDone(s, taskID, true, a); err != nil {
			return nil, err
		}
	}
	return GetRetailTaskWorkflow(s, taskID, a)
}

func ReviewRetailTask(s *xorm.Session, taskID, submissionID int64, decision RetailReviewDecision, comment string, a web.Auth) (*RetailTaskWorkflow, error) {
	profile, err := GetRetailTaskProfileByTaskID(s, taskID)
	if err != nil {
		return nil, err
	}
	if err := requireRetailTaskReviewerOrAdmin(s, profile, a); err != nil {
		return nil, err
	}
	if profile.Status != RetailTaskStatusPendingReview {
		return nil, ErrRetailTaskInvalidTransition{From: profile.Status, To: RetailTaskStatusCompleted}
	}
	if decision != RetailReviewApproved && decision != RetailReviewRejected {
		return nil, ErrRetailReviewInvalidDecision{Decision: decision}
	}
	comment = strings.TrimSpace(comment)
	if decision == RetailReviewRejected && comment == "" {
		return nil, ErrInvalidData{Message: "A rejection comment is required."}
	}
	submission := &RetailSubmission{ID: submissionID}
	exists, err := s.Where("id = ?", submissionID).And("profile_id = ?", profile.ID).Get(submission)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrRetailSubmissionDoesNotExist{ID: submissionID}
	}
	review := &RetailReview{ProfileID: profile.ID, SubmissionID: submissionID, ReviewerID: a.GetID(), Decision: decision, Comment: comment}
	if _, err := s.Insert(review); err != nil {
		return nil, err
	}
	next := RetailTaskStatusCompleted
	if decision == RetailReviewRejected {
		next = RetailTaskStatusRejected
	}
	if err := transitionRetailTask(s, profile, next, a, comment); err != nil {
		return nil, err
	}
	if err := setRetailUnderlyingTaskDone(s, taskID, decision == RetailReviewApproved, a); err != nil {
		return nil, err
	}
	return GetRetailTaskWorkflow(s, taskID, a)
}

func CancelRetailTask(s *xorm.Session, taskID int64, reason string, a web.Auth) (*RetailTaskWorkflow, error) {
	profile, err := GetRetailTaskProfileByTaskID(s, taskID)
	if err != nil {
		return nil, err
	}
	admin, err := (&RetailOrgUnit{ID: profile.OrgUnitID}).hasInheritedAdminAccess(s, a)
	if err != nil {
		return nil, err
	}
	if !admin {
		return nil, ErrGenericForbidden{}
	}
	if profile.Status == RetailTaskStatusCompleted || profile.Status == RetailTaskStatusCancelled {
		return nil, ErrRetailTaskInvalidTransition{From: profile.Status, To: RetailTaskStatusCancelled}
	}
	if err := transitionRetailTask(s, profile, RetailTaskStatusCancelled, a, strings.TrimSpace(reason)); err != nil {
		return nil, err
	}
	if err := setRetailUnderlyingTaskDone(s, taskID, false, a); err != nil {
		return nil, err
	}
	return GetRetailTaskWorkflow(s, taskID, a)
}

func GetRetailTaskWorkflow(s *xorm.Session, taskID int64, a web.Auth) (*RetailTaskWorkflow, error) {
	profile, err := GetRetailTaskProfileByTaskID(s, taskID)
	if err != nil {
		return nil, err
	}
	can, _, err := (&RetailOrgUnit{ID: profile.OrgUnitID}).CanRead(s, a)
	if err != nil {
		return nil, err
	}
	if !can {
		return nil, ErrGenericForbidden{}
	}
	workflow := &RetailTaskWorkflow{Profile: profile, Checklist: []*RetailChecklistItem{}, Submissions: []*RetailSubmission{}, Reviews: []*RetailReview{}, Transitions: []*RetailTaskTransition{}}
	if err := s.Where("profile_id = ?", profile.ID).Asc("position", "id").Find(&workflow.Checklist); err != nil {
		return nil, err
	}
	if err := s.Where("profile_id = ?", profile.ID).Asc("id").Find(&workflow.Submissions); err != nil {
		return nil, err
	}
	if len(workflow.Submissions) > 0 {
		submissionByID := make(map[int64]*RetailSubmission, len(workflow.Submissions))
		ids := make([]int64, 0, len(workflow.Submissions))
		for _, submission := range workflow.Submissions {
			submissionByID[submission.ID] = submission
			ids = append(ids, submission.ID)
		}
		files := []*RetailSubmissionFile{}
		if err := s.In("submission_id", ids).Asc("id").Find(&files); err != nil {
			return nil, err
		}
		for _, file := range files {
			submissionByID[file.SubmissionID].EvidenceAttachmentIDs = append(submissionByID[file.SubmissionID].EvidenceAttachmentIDs, file.AttachmentID)
		}
	}
	if err := s.Where("profile_id = ?", profile.ID).Asc("id").Find(&workflow.Reviews); err != nil {
		return nil, err
	}
	if err := s.Where("profile_id = ?", profile.ID).Asc("id").Find(&workflow.Transitions); err != nil {
		return nil, err
	}
	return workflow, nil
}

func requireRetailTaskWorkerOrAdmin(s *xorm.Session, profile *RetailTaskProfile, a web.Auth) error {
	if profile.PrimaryAssigneeID == a.GetID() {
		return nil
	}
	admin, err := (&RetailOrgUnit{ID: profile.OrgUnitID}).hasInheritedAdminAccess(s, a)
	if err != nil {
		return err
	}
	if !admin {
		return ErrGenericForbidden{}
	}
	return nil
}

func requireRetailTaskReviewerOrAdmin(s *xorm.Session, profile *RetailTaskProfile, a web.Auth) error {
	if profile.ReviewerID > 0 && profile.ReviewerID == a.GetID() {
		return nil
	}
	admin, err := (&RetailOrgUnit{ID: profile.OrgUnitID}).hasInheritedAdminAccess(s, a)
	if err != nil {
		return err
	}
	if !admin {
		return ErrGenericForbidden{}
	}
	return nil
}

func transitionRetailTask(s *xorm.Session, profile *RetailTaskProfile, next RetailTaskStatus, a web.Auth, reason string) error {
	previous := profile.Status
	if _, err := s.ID(profile.ID).Cols("status").Update(&RetailTaskProfile{Status: next}); err != nil {
		return err
	}
	if _, err := s.Insert(&RetailTaskTransition{ProfileID: profile.ID, From: previous, To: next, ActorID: a.GetID(), Reason: reason}); err != nil {
		return err
	}
	profile.Status = next
	return nil
}

func setRetailUnderlyingTaskDone(s *xorm.Session, taskID int64, done bool, a web.Auth) error {
	task, err := GetTaskByIDSimple(s, taskID)
	if err != nil {
		return err
	}
	task.Done = done
	if done {
		task.DoneAt = time.Now()
	} else {
		task.DoneAt = time.Time{}
	}
	if _, err := s.ID(taskID).Cols("done", "done_at").Update(&task); err != nil {
		return err
	}
	if err := updateProjectByTaskID(s, taskID); err != nil {
		return err
	}
	events.DispatchOnCommit(s, &TaskUpdatedEvent{Task: &task, Doer: doerFromAuth(s, a)})
	return nil
}

func uniqueInt64s(values []int64) []int64 {
	seen := make(map[int64]bool, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value < 1 || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
