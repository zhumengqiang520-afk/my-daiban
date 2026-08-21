// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

type RetailTemplateChecklistItem struct {
	Title    string `json:"title" minLength:"1" maxLength:"500" doc:"The checklist step text."`
	Required bool   `json:"required" doc:"Whether the step must be completed before submission."`
	Position int    `json:"position" minimum:"0" doc:"The display order."`
}

// RetailTaskTemplate is an editable template whose dispatched tasks always point to an immutable version.
type RetailTaskTemplate struct {
	ID               int64                         `xorm:"bigint autoincr not null unique pk" json:"id" param:"id" readOnly:"true" doc:"The unique template ID."`
	OrgUnitID        int64                         `xorm:"bigint not null INDEX" json:"org_unit_id" doc:"The organization scope that owns this template. Immutable after creation."`
	OrgUnitName      string                        `xorm:"-" json:"org_unit_name" readOnly:"true" doc:"The owning organization name."`
	Name             string                        `xorm:"varchar(250) not null INDEX" json:"name" minLength:"1" maxLength:"250" doc:"A manager-facing template name."`
	Title            string                        `xorm:"varchar(500) not null" json:"title" minLength:"1" maxLength:"500" doc:"The generated task title."`
	Description      string                        `xorm:"longtext null" json:"description" doc:"The generated task description."`
	Category         RetailTaskCategory            `xorm:"varchar(40) not null INDEX" json:"category" doc:"The retail work category."`
	EstimatedMinutes int                           `xorm:"int not null default 0 INDEX" json:"estimated_minutes" minimum:"0" maximum:"1440" doc:"Expected effort for each generated task."`
	EvidenceRequired bool                          `xorm:"not null default false INDEX" json:"evidence_required" doc:"Whether generated tasks require attachment evidence."`
	Active           bool                          `xorm:"not null default true INDEX" json:"active" doc:"Whether the template may be dispatched."`
	CurrentVersion   int                           `xorm:"int not null default 1" json:"current_version" readOnly:"true" doc:"The immutable version used by the next dispatch."`
	Checklist        []RetailTemplateChecklistItem `xorm:"-" json:"checklist" doc:"Checklist copied to every generated task."`
	CreatedByID      int64                         `xorm:"bigint not null INDEX" json:"created_by_id" readOnly:"true" doc:"The user who created the template."`
	Created          time.Time                     `xorm:"created not null" json:"created" readOnly:"true" doc:"When the template was created."`
	Updated          time.Time                     `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When the template was last changed."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

func (*RetailTaskTemplate) TableName() string { return "retail_task_templates" }

type RetailTemplateVersion struct {
	ID               int64                         `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The immutable template version ID."`
	TemplateID       int64                         `xorm:"bigint not null INDEX unique(retail_template_version)" json:"template_id" readOnly:"true" doc:"The editable template this version belongs to."`
	Version          int                           `xorm:"int not null unique(retail_template_version)" json:"version" readOnly:"true" doc:"The monotonically increasing version number."`
	Title            string                        `xorm:"varchar(500) not null" json:"title" readOnly:"true"`
	Description      string                        `xorm:"longtext null" json:"description" readOnly:"true"`
	Category         RetailTaskCategory            `xorm:"varchar(40) not null INDEX" json:"category" readOnly:"true"`
	EstimatedMinutes int                           `xorm:"int not null default 0" json:"estimated_minutes" readOnly:"true"`
	EvidenceRequired bool                          `xorm:"not null default false" json:"evidence_required" readOnly:"true"`
	ChecklistJSON    string                        `xorm:"longtext null" json:"-"`
	Checklist        []RetailTemplateChecklistItem `xorm:"-" json:"checklist" readOnly:"true"`
	CreatedByID      int64                         `xorm:"bigint not null INDEX" json:"created_by_id" readOnly:"true"`
	Created          time.Time                     `xorm:"created not null" json:"created" readOnly:"true"`
}

func (*RetailTemplateVersion) TableName() string { return "retail_template_versions" }

type RetailTemplateDispatch struct {
	ID                int64     `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true"`
	TemplateVersionID int64     `xorm:"bigint not null INDEX" json:"template_version_id" readOnly:"true"`
	TargetOrgUnitID   int64     `xorm:"bigint not null INDEX" json:"target_org_unit_id" readOnly:"true"`
	ProjectID         int64     `xorm:"bigint not null INDEX" json:"project_id" readOnly:"true"`
	TaskID            int64     `xorm:"bigint not null INDEX unique" json:"task_id" readOnly:"true"`
	ScheduledFor      time.Time `xorm:"datetime not null INDEX" json:"scheduled_for" readOnly:"true"`
	IdempotencyKey    string    `xorm:"varchar(250) not null unique INDEX" json:"idempotency_key" readOnly:"true"`
	CreatedByID       int64     `xorm:"bigint not null INDEX" json:"created_by_id" readOnly:"true"`
	Created           time.Time `xorm:"created not null" json:"created" readOnly:"true"`
}

func (*RetailTemplateDispatch) TableName() string { return "retail_template_dispatches" }

func GetRetailTaskTemplateByID(s *xorm.Session, id int64) (*RetailTaskTemplate, error) {
	template := &RetailTaskTemplate{ID: id}
	exists, err := s.Get(template)
	if err != nil {
		return template, err
	}
	if !exists {
		return template, ErrRetailTaskTemplateDoesNotExist{ID: id}
	}
	return template, addRetailTaskTemplateInfo(s, []*RetailTaskTemplate{template})
}

func (r *RetailTaskTemplate) validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Title = strings.TrimSpace(r.Title)
	if r.Name == "" || r.Title == "" {
		return ErrInvalidData{Message: "Template name and task title are required."}
	}
	if !validRetailTaskCategory(r.Category) {
		return ErrRetailTaskInvalidCategory{Category: r.Category}
	}
	if r.EstimatedMinutes < 0 || r.EstimatedMinutes > 1440 {
		return ErrInvalidData{Message: "Estimated minutes must be between 0 and 1440."}
	}
	for index := range r.Checklist {
		r.Checklist[index].Title = strings.TrimSpace(r.Checklist[index].Title)
		if r.Checklist[index].Title == "" || len([]rune(r.Checklist[index].Title)) > 500 || r.Checklist[index].Position < 0 {
			return ErrInvalidData{Message: "Every checklist item needs a valid title and position."}
		}
	}
	sort.SliceStable(r.Checklist, func(i, j int) bool { return r.Checklist[i].Position < r.Checklist[j].Position })
	return nil
}

func validRetailTaskCategory(category RetailTaskCategory) bool {
	switch category {
	case RetailTaskCategoryOpening, RetailTaskCategoryClosing, RetailTaskCategoryDisplay, RetailTaskCategoryInventory,
		RetailTaskCategoryCustomerFollowup, RetailTaskCategoryDelivery, RetailTaskCategoryAfterSales, RetailTaskCategoryOther:
		return true
	default:
		return false
	}
}

func (r *RetailTaskTemplate) Create(s *xorm.Session, a web.Auth) error {
	creator, err := user.GetFromAuth(a)
	if err != nil {
		return err
	}
	if _, err = GetRetailOrgUnitByID(s, r.OrgUnitID); err != nil {
		return err
	}
	if err = r.validate(); err != nil {
		return err
	}
	r.ID = 0
	r.CurrentVersion = 1
	r.CreatedByID = creator.ID
	if _, err = s.Insert(r); err != nil {
		return err
	}
	if err = createRetailTemplateVersion(s, r, creator.ID); err != nil {
		return err
	}
	return addRetailTaskTemplateInfo(s, []*RetailTaskTemplate{r})
}

func (r *RetailTaskTemplate) ReadOne(s *xorm.Session, _ web.Auth) error {
	template, err := GetRetailTaskTemplateByID(s, r.ID)
	if template != nil {
		*r = *template
	}
	return err
}

func (r *RetailTaskTemplate) ReadAll(s *xorm.Session, a web.Auth, search string, page, perPage int) (interface{}, int, int64, error) {
	all := []*RetailTaskTemplate{}
	query := s.Asc("id")
	if r.OrgUnitID > 0 {
		query = query.Where("org_unit_id = ?", r.OrgUnitID)
	}
	if err := query.Find(&all); err != nil {
		return nil, 0, 0, err
	}
	result := make([]*RetailTaskTemplate, 0, len(all))
	needle := strings.ToLower(strings.TrimSpace(search))
	for _, template := range all {
		can, _, err := (&RetailOrgUnit{ID: template.OrgUnitID}).CanRead(s, a)
		if err != nil {
			return nil, 0, 0, err
		}
		if !can || needle != "" && !strings.Contains(strings.ToLower(template.Name+" "+template.Title), needle) {
			continue
		}
		result = append(result, template)
	}
	if err := addRetailTaskTemplateInfo(s, result); err != nil {
		return nil, 0, 0, err
	}
	total := int64(len(result))
	limit, start := getLimitFromPageIndex(page, perPage)
	if limit > 0 && start < len(result) {
		result = result[start:min(start+limit, len(result))]
	} else if limit > 0 {
		result = []*RetailTaskTemplate{}
	}
	return result, len(result), total, nil
}

func (r *RetailTaskTemplate) Update(s *xorm.Session, a web.Auth) error {
	existing, err := GetRetailTaskTemplateByID(s, r.ID)
	if err != nil {
		return err
	}
	r.OrgUnitID = existing.OrgUnitID
	if err = r.validate(); err != nil {
		return err
	}
	r.CurrentVersion = existing.CurrentVersion + 1
	r.CreatedByID = existing.CreatedByID
	if _, err = s.ID(r.ID).Cols("name", "title", "description", "category", "estimated_minutes", "evidence_required", "active", "current_version").Update(r); err != nil {
		return err
	}
	if err = createRetailTemplateVersion(s, r, a.GetID()); err != nil {
		return err
	}
	if !r.Active {
		if _, err = s.Where("template_id = ?", r.ID).Cols("active").Update(&RetailTemplateSchedule{Active: false}); err != nil {
			return err
		}
	}
	return r.ReadOne(s, a)
}

func (r *RetailTaskTemplate) Delete(s *xorm.Session, _ web.Auth) error {
	if _, err := s.ID(r.ID).Cols("active").Update(&RetailTaskTemplate{Active: false}); err != nil {
		return err
	}
	_, err := s.Where("template_id = ?", r.ID).Cols("active").Update(&RetailTemplateSchedule{Active: false})
	return err
}

func createRetailTemplateVersion(s *xorm.Session, template *RetailTaskTemplate, creatorID int64) error {
	checklistJSON, err := json.Marshal(template.Checklist)
	if err != nil {
		return err
	}
	version := &RetailTemplateVersion{
		TemplateID: template.ID, Version: template.CurrentVersion, Title: template.Title, Description: template.Description,
		Category: template.Category, EstimatedMinutes: template.EstimatedMinutes, EvidenceRequired: template.EvidenceRequired,
		ChecklistJSON: string(checklistJSON), Checklist: template.Checklist, CreatedByID: creatorID,
	}
	_, err = s.Insert(version)
	return err
}

func addRetailTaskTemplateInfo(s *xorm.Session, templates []*RetailTaskTemplate) error {
	for _, template := range templates {
		org, err := GetRetailOrgUnitByID(s, template.OrgUnitID)
		if err != nil {
			return err
		}
		template.OrgUnitName = org.Name
		version := &RetailTemplateVersion{}
		exists, err := s.Where("template_id = ?", template.ID).And("version = ?", template.CurrentVersion).Get(version)
		if err != nil {
			return err
		}
		if exists && version.ChecklistJSON != "" {
			if err := json.Unmarshal([]byte(version.ChecklistJSON), &template.Checklist); err != nil {
				return err
			}
		}
	}
	return nil
}

type RetailTemplateDispatchInput struct {
	TargetOrgUnitID   int64     `json:"target_org_unit_id" minimum:"1" doc:"The store or warehouse receiving the task."`
	ProjectID         int64     `json:"project_id" minimum:"1" doc:"The Vikunja project where the task will be created."`
	PrimaryAssigneeID int64     `json:"primary_assignee_id" minimum:"1" doc:"The staff member accountable for completion."`
	ReviewerID        int64     `json:"reviewer_id" minimum:"0" doc:"The manager who reviews completion; zero enables automatic completion."`
	ScheduledFor      time.Time `json:"scheduled_for" doc:"The occurrence time used for idempotency and task start time."`
	DueDate           time.Time `json:"due_date" doc:"The generated task due date."`
	IdempotencyKey    string    `json:"idempotency_key" maxLength:"250" doc:"Optional caller key. When omitted, template version, target and occurrence form the key."`
}

type RetailTemplateDispatchResult struct {
	Dispatch *RetailTemplateDispatch `json:"dispatch" readOnly:"true"`
	Profile  *RetailTaskProfile      `json:"profile" readOnly:"true"`
	Reused   bool                    `json:"reused" readOnly:"true" doc:"True when an earlier dispatch with the same key was returned."`
}

type RetailTemplateDispatchPreview struct {
	TemplateVersionID int64              `json:"template_version_id" readOnly:"true"`
	TargetOrgUnitID   int64              `json:"target_org_unit_id" readOnly:"true"`
	TargetOrgUnitName string             `json:"target_org_unit_name" readOnly:"true"`
	ProjectID         int64              `json:"project_id" readOnly:"true"`
	Title             string             `json:"title" readOnly:"true"`
	Category          RetailTaskCategory `json:"category" readOnly:"true"`
	EstimatedMinutes  int                `json:"estimated_minutes" readOnly:"true"`
	ScheduledFor      time.Time          `json:"scheduled_for" readOnly:"true"`
	DueDate           time.Time          `json:"due_date" readOnly:"true"`
	IdempotencyKey    string             `json:"idempotency_key" readOnly:"true"`
	AlreadyDispatched bool               `json:"already_dispatched" readOnly:"true"`
}

func PreviewRetailTaskTemplateDispatch(s *xorm.Session, templateID int64, input RetailTemplateDispatchInput, a web.Auth) (*RetailTemplateDispatchPreview, error) {
	version, target, key, err := prepareRetailTemplateDispatch(s, templateID, input, a)
	if err != nil {
		return nil, err
	}
	already, err := s.Where("idempotency_key = ?", key).Exist(&RetailTemplateDispatch{})
	if err != nil {
		return nil, err
	}
	return &RetailTemplateDispatchPreview{
		TemplateVersionID: version.ID, TargetOrgUnitID: target.ID, TargetOrgUnitName: target.Name,
		ProjectID: input.ProjectID, Title: version.Title, Category: version.Category, EstimatedMinutes: version.EstimatedMinutes,
		ScheduledFor: input.ScheduledFor, DueDate: input.DueDate, IdempotencyKey: key, AlreadyDispatched: already,
	}, nil
}

func DispatchRetailTaskTemplate(s *xorm.Session, templateID int64, input RetailTemplateDispatchInput, a web.Auth) (*RetailTemplateDispatchResult, error) {
	version, target, key, err := prepareRetailTemplateDispatch(s, templateID, input, a)
	if err != nil {
		return nil, err
	}
	dispatch := &RetailTemplateDispatch{}
	exists, err := s.Where("idempotency_key = ?", key).Get(dispatch)
	if err != nil {
		return nil, err
	}
	if exists {
		profile, profileErr := GetRetailTaskProfileByTaskID(s, dispatch.TaskID)
		return &RetailTemplateDispatchResult{Dispatch: dispatch, Profile: profile, Reused: true}, profileErr
	}
	task := &Task{Title: version.Title, Description: version.Description, ProjectID: input.ProjectID, StartDate: input.ScheduledFor, DueDate: input.DueDate}
	if err = task.Create(s, a); err != nil {
		return nil, err
	}
	profile := &RetailTaskProfile{
		TaskID: task.ID, OrgUnitID: target.ID, Category: version.Category, PrimaryAssigneeID: input.PrimaryAssigneeID,
		ReviewerID: input.ReviewerID, EstimatedMinutes: version.EstimatedMinutes, Source: RetailTaskSourceTemplate,
		SourceID: version.ID, EvidenceRequired: version.EvidenceRequired,
	}
	if err = profile.Create(s, a); err != nil {
		return nil, err
	}
	for _, definition := range version.Checklist {
		item := &RetailChecklistItem{ProfileID: profile.ID, Title: definition.Title, Required: definition.Required, Position: definition.Position}
		if err = item.Create(s, a); err != nil {
			return nil, err
		}
	}
	dispatch = &RetailTemplateDispatch{
		TemplateVersionID: version.ID, TargetOrgUnitID: target.ID, ProjectID: input.ProjectID, TaskID: task.ID,
		ScheduledFor: input.ScheduledFor, IdempotencyKey: key, CreatedByID: a.GetID(),
	}
	if _, err = s.Insert(dispatch); err != nil {
		return nil, err
	}
	return &RetailTemplateDispatchResult{Dispatch: dispatch, Profile: profile}, nil
}

func prepareRetailTemplateDispatch(s *xorm.Session, templateID int64, input RetailTemplateDispatchInput, a web.Auth) (*RetailTemplateVersion, *RetailOrgUnit, string, error) {
	template, err := GetRetailTaskTemplateByID(s, templateID)
	if err != nil {
		return nil, nil, "", err
	}
	admin, err := (&RetailOrgUnit{ID: template.OrgUnitID}).hasInheritedAdminAccess(s, a)
	if err != nil {
		return nil, nil, "", err
	}
	if !admin {
		return nil, nil, "", ErrGenericForbidden{}
	}
	if !template.Active {
		return nil, nil, "", ErrRetailTaskTemplateInactive{ID: template.ID}
	}
	target, err := GetRetailOrgUnitByID(s, input.TargetOrgUnitID)
	if err != nil {
		return nil, nil, "", err
	}
	if !target.Active || !retailOrgIsWithinScope(s, target, template.OrgUnitID) {
		return nil, nil, "", ErrGenericForbidden{}
	}
	projectAdmin, err := (&Project{ID: input.ProjectID}).IsAdmin(s, a)
	if err != nil {
		return nil, nil, "", err
	}
	if !projectAdmin {
		return nil, nil, "", ErrGenericForbidden{}
	}
	if input.ScheduledFor.IsZero() || input.DueDate.IsZero() || input.DueDate.Before(input.ScheduledFor) {
		return nil, nil, "", ErrInvalidData{Message: "A scheduled time and a due date on or after it are required."}
	}
	version := &RetailTemplateVersion{}
	exists, err := s.Where("template_id = ?", template.ID).And("version = ?", template.CurrentVersion).Get(version)
	if err != nil {
		return nil, nil, "", err
	}
	if !exists {
		return nil, nil, "", ErrRetailTemplateVersionDoesNotExist{TemplateID: template.ID, Version: template.CurrentVersion}
	}
	if err := json.Unmarshal([]byte(version.ChecklistJSON), &version.Checklist); err != nil {
		return nil, nil, "", err
	}
	key := strings.TrimSpace(input.IdempotencyKey)
	if key == "" {
		key = fmt.Sprintf("retail-template:%d:%d:%s", version.ID, target.ID, input.ScheduledFor.UTC().Format(time.RFC3339Nano))
	}
	if len(key) > 250 {
		return nil, nil, "", ErrInvalidData{Message: "Idempotency key must be at most 250 characters."}
	}
	return version, target, key, nil
}

func retailOrgIsWithinScope(s *xorm.Session, target *RetailOrgUnit, scopeID int64) bool {
	visited := map[int64]bool{}
	current := target
	for current != nil && current.ID > 0 && !visited[current.ID] {
		if current.ID == scopeID {
			return true
		}
		visited[current.ID] = true
		if current.ParentID == 0 {
			return false
		}
		parent, err := GetRetailOrgUnitByID(s, current.ParentID)
		if err != nil {
			return false
		}
		current = parent
	}
	return false
}
