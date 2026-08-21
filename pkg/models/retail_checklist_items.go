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

// RetailChecklistItem is one verifiable step of a retail task.
type RetailChecklistItem struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk" json:"id" param:"id" readOnly:"true" doc:"The unique numeric ID of this checklist item."`
	ProfileID   int64     `xorm:"bigint not null INDEX" json:"profile_id" doc:"The retail task profile this item belongs to. Immutable after creation."`
	Title       string    `xorm:"varchar(500) not null" json:"title" valid:"required,runelength(1|500)" minLength:"1" maxLength:"500" doc:"The action or condition staff must verify."`
	Required    bool      `xorm:"not null default true INDEX" json:"required" doc:"Whether this item must be checked before task submission."`
	Position    int       `xorm:"int not null default 0 INDEX" json:"position" minimum:"0" doc:"Display order within the checklist."`
	Done        bool      `xorm:"not null default false INDEX" json:"done" readOnly:"true" doc:"Whether staff marked this item complete."`
	DoneByID    int64     `xorm:"bigint null INDEX" json:"done_by_id" readOnly:"true" doc:"The user who last completed this item."`
	DoneAt      time.Time `xorm:"datetime null" json:"done_at,omitzero" readOnly:"true" doc:"When this item was last completed."`
	CreatedByID int64     `xorm:"bigint not null INDEX" json:"created_by_id" readOnly:"true" doc:"The user who created this checklist item."`
	Created     time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When this checklist item was created."`
	Updated     time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When this checklist item was last updated."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

func (*RetailChecklistItem) TableName() string { return "retail_checklist_items" }

func GetRetailChecklistItemByID(s *xorm.Session, id int64) (*RetailChecklistItem, error) {
	item := &RetailChecklistItem{ID: id}
	if id < 1 {
		return item, ErrRetailChecklistItemDoesNotExist{ID: id}
	}
	exists, err := s.Get(item)
	if err != nil {
		return item, err
	}
	if !exists {
		return item, ErrRetailChecklistItemDoesNotExist{ID: id}
	}
	return item, nil
}

func (r *RetailChecklistItem) Create(s *xorm.Session, a web.Auth) error {
	creator, err := user.GetFromAuth(a)
	if err != nil {
		return err
	}
	if _, err := GetRetailTaskProfileByID(s, r.ProfileID); err != nil {
		return err
	}
	r.Title = strings.TrimSpace(r.Title)
	if r.Title == "" {
		return ErrInvalidData{Message: "Checklist item title is required."}
	}
	r.ID = 0
	r.Done = false
	r.DoneByID = 0
	r.DoneAt = time.Time{}
	r.CreatedByID = creator.ID
	_, err = s.Insert(r)
	return err
}

func (r *RetailChecklistItem) ReadOne(s *xorm.Session, _ web.Auth) error {
	item, err := GetRetailChecklistItemByID(s, r.ID)
	if item != nil {
		*r = *item
	}
	return err
}

func (r *RetailChecklistItem) ReadAll(s *xorm.Session, a web.Auth, search string, page, perPage int) (interface{}, int, int64, error) {
	if r.ProfileID < 1 {
		return []*RetailChecklistItem{}, 0, 0, nil
	}
	profile, err := GetRetailTaskProfileByID(s, r.ProfileID)
	if err != nil {
		return nil, 0, 0, err
	}
	can, _, err := (&RetailOrgUnit{ID: profile.OrgUnitID}).CanRead(s, a)
	if err != nil {
		return nil, 0, 0, err
	}
	if !can {
		return nil, 0, 0, ErrGenericForbidden{}
	}
	items := []*RetailChecklistItem{}
	query := s.Where("profile_id = ?", r.ProfileID).Asc("position", "id")
	if search != "" {
		query = query.Where("title LIKE ?", "%"+strings.TrimSpace(search)+"%")
	}
	if err := query.Find(&items); err != nil {
		return nil, 0, 0, err
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Position == items[j].Position {
			return items[i].ID < items[j].ID
		}
		return items[i].Position < items[j].Position
	})
	total := int64(len(items))
	limit, start := getLimitFromPageIndex(page, perPage)
	if limit > 0 && start < len(items) {
		items = items[start:min(start+limit, len(items))]
	} else if limit > 0 {
		items = []*RetailChecklistItem{}
	}
	return items, len(items), total, nil
}

func (r *RetailChecklistItem) Update(s *xorm.Session, _ web.Auth) error {
	existing, err := GetRetailChecklistItemByID(s, r.ID)
	if err != nil {
		return err
	}
	r.ProfileID = existing.ProfileID
	r.Title = strings.TrimSpace(r.Title)
	if r.Title == "" {
		return ErrInvalidData{Message: "Checklist item title is required."}
	}
	_, err = s.ID(r.ID).Cols("title", "required", "position").Update(r)
	if err != nil {
		return err
	}
	return r.ReadOne(s, nil)
}

func (r *RetailChecklistItem) Delete(s *xorm.Session, _ web.Auth) error {
	_, err := s.ID(r.ID).Delete(&RetailChecklistItem{})
	return err
}
