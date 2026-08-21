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

type RetailScheduleFrequency string

const (
	RetailScheduleDaily   RetailScheduleFrequency = "daily"
	RetailScheduleWeekly  RetailScheduleFrequency = "weekly"
	RetailScheduleMonthly RetailScheduleFrequency = "monthly"
)

type RetailTemplateSchedule struct {
	ID                int64                   `xorm:"bigint autoincr not null unique pk" json:"id" param:"id" readOnly:"true"`
	TemplateID        int64                   `xorm:"bigint not null INDEX" json:"template_id" doc:"The template dispatched by this schedule. Immutable after creation."`
	TemplateName      string                  `xorm:"-" json:"template_name" readOnly:"true"`
	TargetOrgUnitID   int64                   `xorm:"bigint not null INDEX" json:"target_org_unit_id" doc:"The organization receiving generated tasks."`
	TargetOrgUnitName string                  `xorm:"-" json:"target_org_unit_name" readOnly:"true"`
	ProjectID         int64                   `xorm:"bigint not null INDEX" json:"project_id" doc:"The Vikunja project receiving generated tasks."`
	PrimaryAssigneeID int64                   `xorm:"bigint not null INDEX" json:"primary_assignee_id" doc:"The default accountable staff member."`
	ReviewerID        int64                   `xorm:"bigint null INDEX" json:"reviewer_id" doc:"The default reviewer; zero enables automatic completion."`
	Frequency         RetailScheduleFrequency `xorm:"varchar(20) not null INDEX" json:"frequency" doc:"Recurrence frequency: daily, weekly or monthly."`
	Interval          int                     `xorm:"int not null default 1" json:"interval" minimum:"1" maximum:"365" doc:"Run every N frequency periods."`
	DueOffsetMinutes  int                     `xorm:"int not null default 0" json:"due_offset_minutes" minimum:"0" maximum:"44640" doc:"Minutes from occurrence time until the generated task is due."`
	AnchorDay         int                     `xorm:"int not null default 0" json:"anchor_day" readOnly:"true" doc:"Original business month day retained when short months are clamped."`
	NextRunAt         time.Time               `xorm:"datetime not null INDEX" json:"next_run_at" doc:"Next occurrence time. This fixes weekday, month day and local wall-clock time."`
	LastRunAt         time.Time               `xorm:"datetime null INDEX" json:"last_run_at,omitzero" readOnly:"true"`
	Active            bool                    `xorm:"not null default true INDEX" json:"active" doc:"Whether the scheduler may generate tasks."`
	CreatedByID       int64                   `xorm:"bigint not null INDEX" json:"created_by_id" readOnly:"true"`
	Created           time.Time               `xorm:"created not null" json:"created" readOnly:"true"`
	Updated           time.Time               `xorm:"updated not null" json:"updated" readOnly:"true"`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

func (*RetailTemplateSchedule) TableName() string { return "retail_template_schedules" }

func GetRetailTemplateScheduleByID(s *xorm.Session, id int64) (*RetailTemplateSchedule, error) {
	schedule := &RetailTemplateSchedule{ID: id}
	exists, err := s.Get(schedule)
	if err != nil {
		return schedule, err
	}
	if !exists {
		return schedule, ErrRetailTemplateScheduleDoesNotExist{ID: id}
	}
	return schedule, addRetailTemplateScheduleInfo(s, schedule)
}

func (r *RetailTemplateSchedule) validate(s *xorm.Session, a web.Auth) error {
	if r.Frequency != RetailScheduleDaily && r.Frequency != RetailScheduleWeekly && r.Frequency != RetailScheduleMonthly {
		return ErrInvalidData{Message: "Schedule frequency must be daily, weekly or monthly."}
	}
	if r.Interval < 1 || r.Interval > 365 || r.DueOffsetMinutes < 0 || r.DueOffsetMinutes > 44640 || r.NextRunAt.IsZero() {
		return ErrInvalidData{Message: "Schedule interval, due offset and next run time are invalid."}
	}
	_, err := PreviewRetailTaskTemplateDispatch(s, r.TemplateID, RetailTemplateDispatchInput{
		TargetOrgUnitID: r.TargetOrgUnitID, ProjectID: r.ProjectID, PrimaryAssigneeID: r.PrimaryAssigneeID,
		ReviewerID: r.ReviewerID, ScheduledFor: r.NextRunAt, DueDate: r.NextRunAt.Add(time.Duration(r.DueOffsetMinutes) * time.Minute),
	}, a)
	return err
}

func (r *RetailTemplateSchedule) Create(s *xorm.Session, a web.Auth) error {
	if err := r.validate(s, a); err != nil {
		return err
	}
	r.ID = 0
	r.AnchorDay = r.NextRunAt.In(config.GetTimeZone()).Day()
	r.CreatedByID = a.GetID()
	if _, err := s.Insert(r); err != nil {
		return err
	}
	return addRetailTemplateScheduleInfo(s, r)
}

func (r *RetailTemplateSchedule) ReadOne(s *xorm.Session, _ web.Auth) error {
	schedule, err := GetRetailTemplateScheduleByID(s, r.ID)
	if schedule != nil {
		*r = *schedule
	}
	return err
}

func (r *RetailTemplateSchedule) ReadAll(s *xorm.Session, a web.Auth, search string, page, perPage int) (interface{}, int, int64, error) {
	all := []*RetailTemplateSchedule{}
	query := s.Asc("next_run_at", "id")
	if r.TemplateID > 0 {
		query = query.Where("template_id = ?", r.TemplateID)
	}
	if r.TargetOrgUnitID > 0 {
		query = query.Where("target_org_unit_id = ?", r.TargetOrgUnitID)
	}
	if err := query.Find(&all); err != nil {
		return nil, 0, 0, err
	}
	needle := strings.ToLower(strings.TrimSpace(search))
	result := make([]*RetailTemplateSchedule, 0, len(all))
	for _, schedule := range all {
		can, _, err := (&RetailOrgUnit{ID: schedule.TargetOrgUnitID}).CanRead(s, a)
		if err != nil {
			return nil, 0, 0, err
		}
		if !can {
			continue
		}
		if err := addRetailTemplateScheduleInfo(s, schedule); err != nil {
			return nil, 0, 0, err
		}
		if needle == "" || strings.Contains(strings.ToLower(schedule.TemplateName+" "+schedule.TargetOrgUnitName), needle) {
			result = append(result, schedule)
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].NextRunAt.Before(result[j].NextRunAt) })
	total := int64(len(result))
	limit, start := getLimitFromPageIndex(page, perPage)
	if limit > 0 && start < len(result) {
		result = result[start:min(start+limit, len(result))]
	} else if limit > 0 {
		result = []*RetailTemplateSchedule{}
	}
	return result, len(result), total, nil
}

func (r *RetailTemplateSchedule) Update(s *xorm.Session, a web.Auth) error {
	existing, err := GetRetailTemplateScheduleByID(s, r.ID)
	if err != nil {
		return err
	}
	r.TemplateID = existing.TemplateID
	r.CreatedByID = existing.CreatedByID
	r.LastRunAt = existing.LastRunAt
	r.AnchorDay = r.NextRunAt.In(config.GetTimeZone()).Day()
	if err := r.validate(s, a); err != nil {
		return err
	}
	_, err = s.ID(r.ID).Cols("target_org_unit_id", "project_id", "primary_assignee_id", "reviewer_id", "frequency", "interval", "due_offset_minutes", "anchor_day", "next_run_at", "active").Update(r)
	if err != nil {
		return err
	}
	return r.ReadOne(s, a)
}

func (r *RetailTemplateSchedule) Delete(s *xorm.Session, _ web.Auth) error {
	_, err := s.ID(r.ID).Cols("active").Update(&RetailTemplateSchedule{Active: false})
	return err
}

func addRetailTemplateScheduleInfo(s *xorm.Session, schedule *RetailTemplateSchedule) error {
	template, err := GetRetailTaskTemplateByID(s, schedule.TemplateID)
	if err != nil {
		return err
	}
	org, err := GetRetailOrgUnitByID(s, schedule.TargetOrgUnitID)
	if err != nil {
		return err
	}
	schedule.TemplateName = template.Name
	schedule.TargetOrgUnitName = org.Name
	return nil
}

func nextRetailScheduleRun(schedule *RetailTemplateSchedule) time.Time {
	switch schedule.Frequency {
	case RetailScheduleWeekly:
		return schedule.NextRunAt.AddDate(0, 0, 7*schedule.Interval)
	case RetailScheduleMonthly:
		localized := schedule.NextRunAt.In(config.GetTimeZone())
		firstOfTarget := time.Date(localized.Year(), localized.Month()+time.Month(schedule.Interval), 1, localized.Hour(), localized.Minute(), localized.Second(), localized.Nanosecond(), localized.Location())
		lastDay := firstOfTarget.AddDate(0, 1, -1).Day()
		day := min(schedule.AnchorDay, lastDay)
		return time.Date(firstOfTarget.Year(), firstOfTarget.Month(), day, localized.Hour(), localized.Minute(), localized.Second(), localized.Nanosecond(), localized.Location())
	case RetailScheduleDaily:
		return schedule.NextRunAt.AddDate(0, 0, schedule.Interval)
	}
	return schedule.NextRunAt
}

func retailScheduleCreator(s *xorm.Session, schedule *RetailTemplateSchedule) (*user.User, error) {
	return user.GetUserByID(s, schedule.CreatedByID)
}
