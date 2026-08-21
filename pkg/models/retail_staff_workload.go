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

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

type RetailStaffCapacity struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true"`
	OrgUnitID   int64     `xorm:"bigint not null INDEX unique(retail_capacity_day)" json:"org_unit_id" readOnly:"true"`
	UserID      int64     `xorm:"bigint not null INDEX unique(retail_capacity_day)" json:"user_id" readOnly:"true"`
	CapacityDay time.Time `xorm:"date not null INDEX unique(retail_capacity_day)" json:"capacity_day" readOnly:"true"`
	Minutes     int       `xorm:"int not null" json:"minutes" readOnly:"true"`
	Reason      string    `xorm:"varchar(500) null" json:"reason" readOnly:"true"`
	CreatedByID int64     `xorm:"bigint not null INDEX" json:"created_by_id" readOnly:"true"`
	Created     time.Time `xorm:"created not null" json:"created" readOnly:"true"`
	Updated     time.Time `xorm:"updated not null" json:"updated" readOnly:"true"`
}

func (*RetailStaffCapacity) TableName() string { return "retail_staff_capacities" }

type RetailStaffWorkload struct {
	OrgUnitID       int64     `json:"org_unit_id" readOnly:"true"`
	OrgUnitName     string    `json:"org_unit_name" readOnly:"true"`
	UserID          int64     `json:"user_id" readOnly:"true"`
	UserName        string    `json:"user_name" readOnly:"true"`
	JobTitle        string    `json:"job_title" readOnly:"true"`
	CapacityDay     time.Time `json:"capacity_day" readOnly:"true"`
	CapacityMinutes int       `json:"capacity_minutes" readOnly:"true"`
	AssignedMinutes int       `json:"assigned_minutes" readOnly:"true"`
	TaskCount       int       `json:"task_count" readOnly:"true"`
	Utilization     int       `json:"utilization_percent" readOnly:"true"`
	Warning         bool      `json:"warning" readOnly:"true"`
	Overloaded      bool      `json:"overloaded" readOnly:"true"`
}

func SetRetailStaffCapacity(s *xorm.Session, orgUnitID, userID int64, day time.Time, minutes int, reason string, a web.Auth) (*RetailStaffCapacity, error) {
	admin, err := (&RetailOrgUnit{ID: orgUnitID}).hasInheritedAdminAccess(s, a)
	if err != nil {
		return nil, err
	}
	if !admin {
		return nil, ErrGenericForbidden{}
	}
	active, err := isActiveRetailMember(s, orgUnitID, userID)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, ErrRetailTaskInvalidStaff{OrgUnitID: orgUnitID, UserID: userID, Role: "capacity owner"}
	}
	if minutes < 0 || minutes > 1440 {
		return nil, ErrInvalidData{Message: "Capacity minutes must be between 0 and 1440."}
	}
	day = retailCapacityStorageDay(day)
	if day.IsZero() {
		return nil, ErrInvalidData{Message: "Capacity day is required."}
	}
	reason = strings.TrimSpace(reason)
	capacity := &RetailStaffCapacity{}
	exists, err := s.Where("org_unit_id = ?", orgUnitID).And("user_id = ?", userID).And("capacity_day = ?", day).Get(capacity)
	if err != nil {
		return nil, err
	}
	if exists {
		capacity.Minutes = minutes
		capacity.Reason = reason
		_, err = s.ID(capacity.ID).Cols("minutes", "reason").Update(capacity)
		return capacity, err
	}
	capacity = &RetailStaffCapacity{OrgUnitID: orgUnitID, UserID: userID, CapacityDay: day, Minutes: minutes, Reason: reason, CreatedByID: a.GetID()}
	_, err = s.Insert(capacity)
	return capacity, err
}

func GetRetailStaffWorkload(s *xorm.Session, orgUnitID int64, from, to time.Time, a web.Auth) ([]*RetailStaffWorkload, error) {
	can, _, err := (&RetailOrgUnit{ID: orgUnitID}).CanRead(s, a)
	if err != nil {
		return nil, err
	}
	if !can {
		return nil, ErrGenericForbidden{}
	}
	from, to, err = validateRetailWorkloadRange(from, to)
	if err != nil {
		return nil, err
	}
	orgs := []*RetailOrgUnit{}
	if err := s.Find(&orgs); err != nil {
		return nil, err
	}
	orgIDs := make([]int64, 0, len(orgs))
	orgByID := make(map[int64]*RetailOrgUnit, len(orgs))
	for _, org := range orgs {
		orgByID[org.ID] = org
		if retailOrgIsWithinScope(s, org, orgUnitID) {
			orgIDs = append(orgIDs, org.ID)
		}
	}
	memberships := []*RetailMembership{}
	if err := s.In("org_unit_id", orgIDs).And("active = ?", true).Asc("org_unit_id", "user_id").Find(&memberships); err != nil {
		return nil, err
	}
	today := time.Now()
	activeMemberships := make([]*RetailMembership, 0, len(memberships))
	userIDs := make([]int64, 0, len(memberships))
	for _, membership := range memberships {
		if !membership.EndsAt.IsZero() && !membership.EndsAt.After(today) {
			continue
		}
		activeMemberships = append(activeMemberships, membership)
		userIDs = append(userIDs, membership.UserID)
	}
	users, err := user.GetUsersByIDs(s, userIDs)
	if err != nil {
		return nil, err
	}
	endExclusive := to.AddDate(0, 0, 1)
	allCapacities := []*RetailStaffCapacity{}
	if err := s.In("org_unit_id", orgIDs).Find(&allCapacities); err != nil {
		return nil, err
	}
	capacities := make([]*RetailStaffCapacity, 0, len(allCapacities))
	for _, capacity := range allCapacities {
		day := retailDay(capacity.CapacityDay)
		if !day.Before(from) && day.Before(endExclusive) {
			capacities = append(capacities, capacity)
		}
	}
	type assignment struct {
		OrgUnitID         int64     `xorm:"org_unit_id"`
		PrimaryAssigneeID int64     `xorm:"primary_assignee_id"`
		EstimatedMinutes  int       `xorm:"estimated_minutes"`
		DueDate           time.Time `xorm:"due_date"`
	}
	assignments := []*assignment{}
	if err := s.Table("retail_task_profiles").
		Select("retail_task_profiles.org_unit_id, retail_task_profiles.primary_assignee_id, retail_task_profiles.estimated_minutes, tasks.due_date").
		Join("INNER", "tasks", "tasks.id = retail_task_profiles.task_id").
		In("retail_task_profiles.org_unit_id", orgIDs).
		NotIn("retail_task_profiles.status", []RetailTaskStatus{RetailTaskStatusCompleted, RetailTaskStatusCancelled}).
		And("retail_task_profiles.primary_assignee_id > 0").
		And("tasks.due_date >= ?", from).And("tasks.due_date < ?", endExclusive).
		Find(&assignments); err != nil {
		return nil, err
	}
	type workloadKey struct {
		orgID  int64
		userID int64
		day    string
	}
	capacityByKey := make(map[workloadKey]int, len(capacities))
	for _, capacity := range capacities {
		capacityByKey[workloadKey{orgID: capacity.OrgUnitID, userID: capacity.UserID, day: retailDayKey(capacity.CapacityDay)}] = capacity.Minutes
	}
	assignedByKey := map[workloadKey]int{}
	countByKey := map[workloadKey]int{}
	for _, value := range assignments {
		key := workloadKey{orgID: value.OrgUnitID, userID: value.PrimaryAssigneeID, day: retailDayKey(value.DueDate)}
		assignedByKey[key] += value.EstimatedMinutes
		countByKey[key]++
	}
	defaultCapacity := config.RetailDefaultDailyCapacityMinutes.GetInt()
	warningPercent := config.RetailOverloadWarningPercent.GetInt()
	result := make([]*RetailStaffWorkload, 0, len(activeMemberships)*int(to.Sub(from)/(24*time.Hour)+1))
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		for _, membership := range activeMemberships {
			key := workloadKey{orgID: membership.OrgUnitID, userID: membership.UserID, day: retailDayKey(day)}
			capacity := defaultCapacity
			if override, ok := capacityByKey[key]; ok {
				capacity = override
			}
			assigned := assignedByKey[key]
			utilization := 0
			if capacity > 0 {
				utilization = assigned * 100 / capacity
			} else if assigned > 0 {
				utilization = 100
			}
			name := ""
			if memberUser := users[membership.UserID]; memberUser != nil {
				name = memberUser.GetName()
			}
			result = append(result, &RetailStaffWorkload{
				OrgUnitID: membership.OrgUnitID, OrgUnitName: orgByID[membership.OrgUnitID].Name, UserID: membership.UserID,
				UserName: name, JobTitle: membership.JobTitle, CapacityDay: day, CapacityMinutes: capacity,
				AssignedMinutes: assigned, TaskCount: countByKey[key], Utilization: utilization,
				Warning: assigned > 0 && (capacity == 0 || utilization >= warningPercent), Overloaded: assigned > capacity,
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CapacityDay.Equal(result[j].CapacityDay) {
			if result[i].OrgUnitID == result[j].OrgUnitID {
				return result[i].UserID < result[j].UserID
			}
			return result[i].OrgUnitID < result[j].OrgUnitID
		}
		return result[i].CapacityDay.Before(result[j].CapacityDay)
	})
	return result, nil
}

func retailDay(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	localized := value.In(config.GetTimeZone())
	return time.Date(localized.Year(), localized.Month(), localized.Day(), 0, 0, 0, 0, config.GetTimeZone())
}

func validateRetailWorkloadRange(from, to time.Time) (time.Time, time.Time, error) {
	from = retailDay(from)
	to = retailDay(to)
	if from.IsZero() || to.IsZero() || to.Before(from) || to.Sub(from) > 31*24*time.Hour {
		return time.Time{}, time.Time{}, ErrInvalidData{Message: "Workload date range must contain 1 to 32 days."}
	}
	return from, to, nil
}

func retailDayKey(value time.Time) string {
	return value.In(config.GetTimeZone()).Format("2006-01-02")
}

// DATE columns carry calendar values, not instants. Store the configured
// business-date components at UTC midnight so database timezone conversion
// cannot move an Asia/Shanghai date to the previous day.
func retailCapacityStorageDay(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	localized := value.In(config.GetTimeZone())
	return time.Date(localized.Year(), localized.Month(), localized.Day(), 0, 0, 0, 0, time.UTC)
}
