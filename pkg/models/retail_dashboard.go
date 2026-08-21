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
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

type RetailDashboardTask struct {
	ProfileID         int64              `json:"profile_id" readOnly:"true"`
	TaskID            int64              `json:"task_id" readOnly:"true"`
	Title             string             `json:"title" readOnly:"true"`
	OrgUnitID         int64              `json:"org_unit_id" readOnly:"true"`
	OrgUnitName       string             `json:"org_unit_name" readOnly:"true"`
	Category          RetailTaskCategory `json:"category" readOnly:"true"`
	Status            RetailTaskStatus   `json:"status" readOnly:"true"`
	PrimaryAssigneeID int64              `json:"primary_assignee_id" readOnly:"true"`
	EstimatedMinutes  int                `json:"estimated_minutes" readOnly:"true"`
	DueDate           time.Time          `json:"due_date" readOnly:"true"`
}

type RetailOperationsDashboard struct {
	OrgUnitID         int64                  `json:"org_unit_id" readOnly:"true"`
	DateFrom          time.Time              `json:"date_from" readOnly:"true"`
	DateTo            time.Time              `json:"date_to" readOnly:"true"`
	Total             int                    `json:"total" readOnly:"true"`
	Completed         int                    `json:"completed" readOnly:"true"`
	Overdue           int                    `json:"overdue" readOnly:"true"`
	PendingReview     int                    `json:"pending_review" readOnly:"true"`
	Rejected          int                    `json:"rejected" readOnly:"true"`
	Cancelled         int                    `json:"cancelled" readOnly:"true"`
	OnTimeCompleted   int                    `json:"on_time_completed" readOnly:"true"`
	CompletionRate    int                    `json:"completion_rate_percent" readOnly:"true"`
	OnTimeRate        int                    `json:"on_time_rate_percent" readOnly:"true"`
	RejectedTaskCount int                    `json:"rejected_task_count" readOnly:"true"`
	RejectionRate     int                    `json:"rejection_rate_percent" readOnly:"true"`
	StatusCounts      map[string]int         `json:"status_counts" readOnly:"true"`
	CategoryCounts    map[string]int         `json:"category_counts" readOnly:"true"`
	OverdueTasks      []*RetailDashboardTask `json:"overdue_tasks" readOnly:"true"`
}

func GetRetailOperationsDashboard(s *xorm.Session, orgUnitID int64, from, to time.Time, category RetailTaskCategory, assigneeID int64, now time.Time, a web.Auth) (*RetailOperationsDashboard, error) {
	can, _, err := (&RetailOrgUnit{ID: orgUnitID}).CanRead(s, a)
	if err != nil {
		return nil, err
	}
	if !can {
		return nil, ErrGenericForbidden{}
	}
	from, to, err = validateRetailDashboardRange(from, to)
	if err != nil {
		return nil, err
	}
	orgs := []*RetailOrgUnit{}
	if err := s.Find(&orgs); err != nil {
		return nil, err
	}
	orgIDs := make([]int64, 0, len(orgs))
	orgNames := make(map[int64]string, len(orgs))
	for _, org := range orgs {
		orgNames[org.ID] = org.Name
		if retailOrgIsWithinScope(s, org, orgUnitID) {
			orgIDs = append(orgIDs, org.ID)
		}
	}
	rows := []*retailDashboardRow{}
	query := s.Table("retail_task_profiles").
		Select("retail_task_profiles.id AS profile_id, tasks.id AS task_id, tasks.title, tasks.due_date, tasks.done_at, retail_task_profiles.org_unit_id, retail_task_profiles.category, retail_task_profiles.status, retail_task_profiles.primary_assignee_id, retail_task_profiles.estimated_minutes").
		Join("INNER", "tasks", "tasks.id = retail_task_profiles.task_id").
		In("retail_task_profiles.org_unit_id", orgIDs).
		And("tasks.due_date >= ?", from).And("tasks.due_date < ?", to.AddDate(0, 0, 1))
	if category != "" {
		if !validRetailTaskCategory(category) {
			return nil, ErrRetailTaskInvalidCategory{Category: category}
		}
		query = query.And("retail_task_profiles.category = ?", category)
	}
	if assigneeID > 0 {
		query = query.And("retail_task_profiles.primary_assignee_id = ?", assigneeID)
	}
	if err := query.Find(&rows); err != nil {
		return nil, err
	}
	dashboard := &RetailOperationsDashboard{
		OrgUnitID: orgUnitID, DateFrom: from, DateTo: to, StatusCounts: map[string]int{}, CategoryCounts: map[string]int{}, OverdueTasks: []*RetailDashboardTask{},
	}
	profileIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		profileIDs = append(profileIDs, row.ProfileID)
		dashboard.Total++
		dashboard.StatusCounts[string(row.Status)]++
		dashboard.CategoryCounts[string(row.Category)]++
		switch row.Status {
		case RetailTaskStatusCompleted:
			dashboard.Completed++
			if !row.DoneAt.IsZero() && !row.DoneAt.After(row.DueDate) {
				dashboard.OnTimeCompleted++
			}
		case RetailTaskStatusPendingReview:
			dashboard.PendingReview++
		case RetailTaskStatusRejected:
			dashboard.Rejected++
		case RetailTaskStatusCancelled:
			dashboard.Cancelled++
		case RetailTaskStatusDraft, RetailTaskStatusAssigned, RetailTaskStatusInProgress:
			// Counted in the generic status map above.
		}
		if row.Status != RetailTaskStatusCompleted && row.Status != RetailTaskStatusCancelled && row.DueDate.Before(now) {
			dashboard.Overdue++
			dashboard.OverdueTasks = append(dashboard.OverdueTasks, &RetailDashboardTask{
				ProfileID: row.ProfileID, TaskID: row.TaskID, Title: row.Title, OrgUnitID: row.OrgUnitID, OrgUnitName: orgNames[row.OrgUnitID],
				Category: row.Category, Status: row.Status, PrimaryAssigneeID: row.PrimaryAssigneeID, EstimatedMinutes: row.EstimatedMinutes, DueDate: row.DueDate,
			})
		}
	}
	denominator := dashboard.Total - dashboard.Cancelled
	if denominator > 0 {
		dashboard.CompletionRate = dashboard.Completed * 100 / denominator
	}
	if dashboard.Completed > 0 {
		dashboard.OnTimeRate = dashboard.OnTimeCompleted * 100 / dashboard.Completed
	}
	if len(profileIDs) > 0 {
		rejectedProfiles := []int64{}
		if err := s.Table("retail_reviews").Distinct("profile_id").In("profile_id", profileIDs).And("decision = ?", RetailReviewRejected).Find(&rejectedProfiles); err != nil {
			return nil, err
		}
		dashboard.RejectedTaskCount = len(rejectedProfiles)
		if denominator > 0 {
			dashboard.RejectionRate = dashboard.RejectedTaskCount * 100 / denominator
		}
	}
	sort.Slice(dashboard.OverdueTasks, func(i, j int) bool {
		return dashboard.OverdueTasks[i].DueDate.Before(dashboard.OverdueTasks[j].DueDate)
	})
	return dashboard, nil
}

type retailDashboardRow struct {
	ProfileID         int64              `xorm:"profile_id"`
	TaskID            int64              `xorm:"task_id"`
	Title             string             `xorm:"title"`
	DueDate           time.Time          `xorm:"due_date"`
	DoneAt            time.Time          `xorm:"done_at"`
	OrgUnitID         int64              `xorm:"org_unit_id"`
	Category          RetailTaskCategory `xorm:"category"`
	Status            RetailTaskStatus   `xorm:"status"`
	PrimaryAssigneeID int64              `xorm:"primary_assignee_id"`
	EstimatedMinutes  int                `xorm:"estimated_minutes"`
}

func validateRetailDashboardRange(from, to time.Time) (time.Time, time.Time, error) {
	from = retailDay(from)
	to = retailDay(to)
	if from.IsZero() || to.IsZero() || to.Before(from) || to.Sub(from) > 92*24*time.Hour {
		return time.Time{}, time.Time{}, ErrInvalidData{Message: "Dashboard date range must contain 1 to 93 days."}
	}
	return from.In(config.GetTimeZone()), to.In(config.GetTimeZone()), nil
}
