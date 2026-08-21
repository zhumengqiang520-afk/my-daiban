// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"sort"
	"strings"
	"time"

	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

type RetailTaskCategory string

const (
	RetailTaskCategoryOpening          RetailTaskCategory = "opening"
	RetailTaskCategoryClosing          RetailTaskCategory = "closing"
	RetailTaskCategoryDisplay          RetailTaskCategory = "display"
	RetailTaskCategoryInventory        RetailTaskCategory = "inventory"
	RetailTaskCategoryCustomerFollowup RetailTaskCategory = "customer_followup"
	RetailTaskCategoryDelivery         RetailTaskCategory = "delivery"
	RetailTaskCategoryAfterSales       RetailTaskCategory = "after_sales"
	RetailTaskCategoryOther            RetailTaskCategory = "other"
)

type RetailTaskStatus string

const (
	RetailTaskStatusDraft         RetailTaskStatus = "draft"
	RetailTaskStatusAssigned      RetailTaskStatus = "assigned"
	RetailTaskStatusInProgress    RetailTaskStatus = "in_progress"
	RetailTaskStatusPendingReview RetailTaskStatus = "pending_review"
	RetailTaskStatusRejected      RetailTaskStatus = "rejected"
	RetailTaskStatusCompleted     RetailTaskStatus = "completed"
	RetailTaskStatusCancelled     RetailTaskStatus = "cancelled"
)

type RetailTaskSource string

const (
	RetailTaskSourceManual   RetailTaskSource = "manual"
	RetailTaskSourceTemplate RetailTaskSource = "template"
)

// RetailTaskProfile holds the retail workflow fields for a Vikunja task.
type RetailTaskProfile struct {
	ID                int64              `xorm:"bigint autoincr not null unique pk" json:"id" param:"id" readOnly:"true" doc:"The unique numeric ID of this retail task profile."`
	TaskID            int64              `xorm:"bigint not null unique INDEX" json:"task_id" doc:"The Vikunja task this profile extends. Immutable after creation."`
	TaskTitle         string             `xorm:"-" json:"task_title" readOnly:"true" doc:"The current Vikunja task title."`
	DueDate           time.Time          `xorm:"-" json:"due_date,omitzero" readOnly:"true" doc:"The current Vikunja task due date."`
	OrgUnitID         int64              `xorm:"bigint not null INDEX" json:"org_unit_id" doc:"The store, warehouse, or other organization unit responsible for this task. Immutable after creation."`
	OrgUnitName       string             `xorm:"-" json:"org_unit_name" readOnly:"true" doc:"The organization unit display name."`
	Category          RetailTaskCategory `xorm:"varchar(40) not null INDEX" json:"category" doc:"The retail work category."`
	PrimaryAssigneeID int64              `xorm:"bigint null INDEX" json:"primary_assignee_id" doc:"The primary staff member responsible for completion. Zero leaves the profile in draft."`
	PrimaryAssignee   string             `xorm:"-" json:"primary_assignee" readOnly:"true" doc:"The primary assignee's display name."`
	ReviewerID        int64              `xorm:"bigint null INDEX" json:"reviewer_id" doc:"The organization manager responsible for reviewing completion. Zero means no dedicated reviewer."`
	Reviewer          string             `xorm:"-" json:"reviewer" readOnly:"true" doc:"The reviewer's display name."`
	EstimatedMinutes  int                `xorm:"int not null default 0 INDEX" json:"estimated_minutes" minimum:"0" maximum:"1440" doc:"Estimated effort in minutes, from 0 to 1440."`
	Status            RetailTaskStatus   `xorm:"varchar(30) not null INDEX" json:"status" readOnly:"true" doc:"The server-controlled retail workflow status."`
	Source            RetailTaskSource   `xorm:"varchar(30) not null INDEX" json:"source" doc:"How this task was created: manual or template."`
	SourceID          int64              `xorm:"bigint null INDEX" json:"source_id" readOnly:"true" doc:"The template version or other source record ID, when applicable."`
	EvidenceRequired  bool               `xorm:"not null default false INDEX" json:"evidence_required" doc:"Whether at least one completion evidence attachment is required before submission."`
	CreatedByID       int64              `xorm:"bigint not null INDEX" json:"created_by_id" readOnly:"true" doc:"The user ID that created this retail task profile."`
	Created           time.Time          `xorm:"created not null" json:"created" readOnly:"true" doc:"When this retail task profile was created."`
	Updated           time.Time          `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When this retail task profile was last updated."`

	FilterStatus RetailTaskStatus `xorm:"-" json:"-"`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

func (*RetailTaskProfile) TableName() string { return "retail_task_profiles" }

func GetRetailTaskProfileByID(s *xorm.Session, id int64) (*RetailTaskProfile, error) {
	profile := &RetailTaskProfile{ID: id}
	if id < 1 {
		return profile, ErrRetailTaskProfileDoesNotExist{ID: id}
	}
	exists, err := s.Get(profile)
	if err != nil {
		return profile, err
	}
	if !exists {
		return profile, ErrRetailTaskProfileDoesNotExist{ID: id}
	}
	if err := addRetailTaskProfileInfo(s, []*RetailTaskProfile{profile}); err != nil {
		return profile, err
	}
	return profile, nil
}

func (r *RetailTaskProfile) validate(s *xorm.Session) (*Task, *RetailOrgUnit, error) {
	taskValue, err := GetTaskByIDSimple(s, r.TaskID)
	if err != nil {
		return nil, nil, err
	}
	org, err := GetRetailOrgUnitByID(s, r.OrgUnitID)
	if err != nil {
		return nil, nil, err
	}
	if !org.Active {
		return nil, nil, ErrInvalidData{Message: "The retail organization unit is inactive."}
	}
	switch r.Category {
	case RetailTaskCategoryOpening, RetailTaskCategoryClosing, RetailTaskCategoryDisplay, RetailTaskCategoryInventory,
		RetailTaskCategoryCustomerFollowup, RetailTaskCategoryDelivery, RetailTaskCategoryAfterSales, RetailTaskCategoryOther:
	default:
		return nil, nil, ErrRetailTaskInvalidCategory{Category: r.Category}
	}
	if r.Source == "" {
		r.Source = RetailTaskSourceManual
	}
	if r.Source != RetailTaskSourceManual && r.Source != RetailTaskSourceTemplate {
		return nil, nil, ErrInvalidData{Message: "Invalid retail task source."}
	}
	if r.EstimatedMinutes < 0 || r.EstimatedMinutes > 1440 {
		return nil, nil, ErrInvalidData{Message: "Estimated minutes must be between 0 and 1440."}
	}
	if r.PrimaryAssigneeID > 0 {
		active, memberErr := isActiveRetailMember(s, r.OrgUnitID, r.PrimaryAssigneeID)
		if memberErr != nil {
			return nil, nil, memberErr
		}
		if !active {
			return nil, nil, ErrRetailTaskInvalidStaff{OrgUnitID: r.OrgUnitID, UserID: r.PrimaryAssigneeID, Role: "primary assignee"}
		}
	}
	if r.ReviewerID > 0 {
		if r.ReviewerID == r.PrimaryAssigneeID {
			return nil, nil, ErrRetailTaskInvalidStaff{OrgUnitID: r.OrgUnitID, UserID: r.ReviewerID, Role: "reviewer"}
		}
		canAdmin, adminErr := (&RetailOrgUnit{ID: r.OrgUnitID}).hasInheritedAdminAccess(s, &user.User{ID: r.ReviewerID})
		if adminErr != nil {
			return nil, nil, adminErr
		}
		if !canAdmin {
			return nil, nil, ErrRetailTaskInvalidStaff{OrgUnitID: r.OrgUnitID, UserID: r.ReviewerID, Role: "reviewer"}
		}
	}
	conflict, err := s.Table("retail_task_profiles").
		Join("INNER", "tasks", "tasks.id = retail_task_profiles.task_id").
		Where("tasks.project_id = ?", taskValue.ProjectID).
		And("retail_task_profiles.org_unit_id != ?", r.OrgUnitID).
		And("retail_task_profiles.id != ?", r.ID).
		Exist(&RetailTaskProfile{})
	if err != nil {
		return nil, nil, err
	}
	if conflict {
		return nil, nil, ErrRetailProjectScopeConflict{ProjectID: taskValue.ProjectID, OrgUnitID: r.OrgUnitID}
	}
	return &taskValue, org, nil
}

func (r *RetailTaskProfile) Create(s *xorm.Session, a web.Auth) error {
	creator, err := user.GetFromAuth(a)
	if err != nil {
		return err
	}
	exists, err := s.Where("task_id = ?", r.TaskID).Exist(&RetailTaskProfile{})
	if err != nil {
		return err
	}
	if exists {
		return ErrRetailTaskProfileAlreadyExists{TaskID: r.TaskID}
	}
	task, org, err := r.validate(s)
	if err != nil {
		return err
	}
	if err := ensureRetailProjectTeamAccess(s, task.ProjectID, org.TeamID, a); err != nil {
		return err
	}
	if r.PrimaryAssigneeID > 0 {
		if err := ensureRetailTaskAssignee(s, r.TaskID, r.PrimaryAssigneeID, a); err != nil {
			return err
		}
		r.Status = RetailTaskStatusAssigned
	} else {
		r.Status = RetailTaskStatusDraft
	}
	r.ID = 0
	r.CreatedByID = creator.ID
	if _, err = s.Insert(r); err != nil {
		return err
	}
	return addRetailTaskProfileInfo(s, []*RetailTaskProfile{r})
}

func (r *RetailTaskProfile) ReadOne(s *xorm.Session, _ web.Auth) error {
	profile, err := GetRetailTaskProfileByID(s, r.ID)
	if profile != nil {
		*r = *profile
	}
	return err
}

func (r *RetailTaskProfile) ReadAll(s *xorm.Session, a web.Auth, search string, page, perPage int) (interface{}, int, int64, error) {
	if _, is := a.(*LinkSharing); is {
		return nil, 0, 0, ErrGenericForbidden{}
	}
	all := []*RetailTaskProfile{}
	query := s.Asc("id")
	if r.OrgUnitID > 0 {
		query = query.Where("org_unit_id = ?", r.OrgUnitID)
	}
	if r.TaskID > 0 {
		query = query.Where("task_id = ?", r.TaskID)
	}
	if r.FilterStatus != "" {
		query = query.Where("status = ?", r.FilterStatus)
	}
	if err := query.Find(&all); err != nil {
		return nil, 0, 0, err
	}
	if err := addRetailTaskProfileInfo(s, all); err != nil {
		return nil, 0, 0, err
	}
	needle := strings.ToLower(strings.TrimSpace(search))
	result := make([]*RetailTaskProfile, 0, len(all))
	access := map[int64]bool{}
	for _, profile := range all {
		can, known := access[profile.OrgUnitID]
		if !known {
			var accessErr error
			can, _, accessErr = (&RetailOrgUnit{ID: profile.OrgUnitID}).CanRead(s, a)
			if accessErr != nil {
				return nil, 0, 0, accessErr
			}
			access[profile.OrgUnitID] = can
		}
		if !can {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(profile.TaskTitle), needle) && !strings.Contains(strings.ToLower(string(profile.Category)), needle) {
			continue
		}
		result = append(result, profile)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	total := int64(len(result))
	limit, start := getLimitFromPageIndex(page, perPage)
	if limit > 0 && start < len(result) {
		result = result[start:min(start+limit, len(result))]
	} else if limit > 0 {
		result = []*RetailTaskProfile{}
	}
	return result, len(result), total, nil
}

func (r *RetailTaskProfile) Update(s *xorm.Session, a web.Auth) error {
	existing, err := GetRetailTaskProfileByID(s, r.ID)
	if err != nil {
		return err
	}
	r.TaskID = existing.TaskID
	r.OrgUnitID = existing.OrgUnitID
	r.Status = existing.Status
	r.SourceID = existing.SourceID
	_, _, err = r.validate(s)
	if err != nil {
		return err
	}
	if r.PrimaryAssigneeID > 0 {
		if err := ensureRetailTaskAssignee(s, r.TaskID, r.PrimaryAssigneeID, a); err != nil {
			return err
		}
		if r.Status == RetailTaskStatusDraft {
			r.Status = RetailTaskStatusAssigned
		}
	} else if r.Status == RetailTaskStatusAssigned {
		r.Status = RetailTaskStatusDraft
	}
	_, err = s.ID(r.ID).Cols("category", "primary_assignee_id", "reviewer_id", "estimated_minutes", "status", "source", "evidence_required").Update(r)
	if err != nil {
		return err
	}
	return r.ReadOne(s, a)
}

func (r *RetailTaskProfile) Delete(s *xorm.Session, _ web.Auth) error {
	submissionIDs := []int64{}
	if err := s.Table("retail_submissions").Cols("id").Where("profile_id = ?", r.ID).Find(&submissionIDs); err != nil {
		return err
	}
	if len(submissionIDs) > 0 {
		if _, err := s.In("submission_id", submissionIDs).Delete(&RetailSubmissionFile{}); err != nil {
			return err
		}
	}
	for _, model := range []interface{}{&RetailReview{}, &RetailSubmission{}, &RetailTaskTransition{}, &RetailChecklistItem{}} {
		if _, err := s.Where("profile_id = ?", r.ID).Delete(model); err != nil {
			return err
		}
	}
	_, err := s.ID(r.ID).Delete(&RetailTaskProfile{})
	return err
}

func ensureRetailProjectTeamAccess(s *xorm.Session, projectID, teamID int64, a web.Auth) error {
	relation := &TeamProject{}
	exists, err := s.Where("project_id = ?", projectID).And("team_id = ?", teamID).Get(relation)
	if err != nil {
		return err
	}
	if !exists {
		return (&TeamProject{ProjectID: projectID, TeamID: teamID, Permission: PermissionWrite}).Create(s, a)
	}
	if relation.Permission >= PermissionWrite {
		return nil
	}
	_, err = s.ID(relation.ID).Cols("permission").Update(&TeamProject{Permission: PermissionWrite})
	return err
}

func ensureRetailTaskAssignee(s *xorm.Session, taskID, userID int64, a web.Auth) error {
	exists, err := s.Where("task_id = ?", taskID).And("user_id = ?", userID).Exist(&TaskAssginee{})
	if err != nil || exists {
		return err
	}
	return (&TaskAssginee{TaskID: taskID, UserID: userID}).Create(s, a)
}

func isActiveRetailMember(s *xorm.Session, orgUnitID, userID int64) (bool, error) {
	membership := &RetailMembership{}
	exists, err := s.Where("org_unit_id = ?", orgUnitID).And("user_id = ?", userID).And("active = ?", true).Get(membership)
	if err != nil || !exists {
		return false, err
	}
	return membership.EndsAt.IsZero() || membership.EndsAt.After(time.Now()), nil
}

func addRetailTaskProfileInfo(s *xorm.Session, profiles []*RetailTaskProfile) error {
	if len(profiles) == 0 {
		return nil
	}
	taskIDs := make([]int64, 0, len(profiles))
	orgIDs := make([]int64, 0, len(profiles))
	userIDs := make([]int64, 0, len(profiles)*2)
	for _, profile := range profiles {
		taskIDs = append(taskIDs, profile.TaskID)
		orgIDs = append(orgIDs, profile.OrgUnitID)
		if profile.PrimaryAssigneeID > 0 {
			userIDs = append(userIDs, profile.PrimaryAssigneeID)
		}
		if profile.ReviewerID > 0 {
			userIDs = append(userIDs, profile.ReviewerID)
		}
	}
	tasks := []*Task{}
	if err := s.In("id", taskIDs).Find(&tasks); err != nil {
		return err
	}
	orgs := []*RetailOrgUnit{}
	if err := s.In("id", orgIDs).Find(&orgs); err != nil {
		return err
	}
	users, err := user.GetUsersByIDs(s, userIDs)
	if err != nil {
		return err
	}
	taskByID := make(map[int64]*Task, len(tasks))
	for _, task := range tasks {
		taskByID[task.ID] = task
	}
	orgByID := make(map[int64]*RetailOrgUnit, len(orgs))
	for _, org := range orgs {
		orgByID[org.ID] = org
	}
	for _, profile := range profiles {
		if task := taskByID[profile.TaskID]; task != nil {
			profile.TaskTitle = task.Title
			profile.DueDate = task.DueDate
		}
		if org := orgByID[profile.OrgUnitID]; org != nil {
			profile.OrgUnitName = org.Name
		}
		if assignee := users[profile.PrimaryAssigneeID]; assignee != nil {
			profile.PrimaryAssignee = assignee.GetName()
		}
		if reviewer := users[profile.ReviewerID]; reviewer != nil {
			profile.Reviewer = reviewer.GetName()
		}
	}
	return nil
}
