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

type RetailOrgUnitType string

const (
	RetailOrgUnitCompany   RetailOrgUnitType = "company"
	RetailOrgUnitRegion    RetailOrgUnitType = "region"
	RetailOrgUnitStore     RetailOrgUnitType = "store"
	RetailOrgUnitWarehouse RetailOrgUnitType = "warehouse"
)

// RetailOrgUnit represents one node in the retail company hierarchy. Each
// unit owns a Vikunja team; team membership is the source of truth for access.
type RetailOrgUnit struct {
	ID          int64             `xorm:"bigint autoincr not null unique pk" json:"id" param:"id" readOnly:"true" doc:"The unique numeric ID of this organization unit."`
	ParentID    int64             `xorm:"bigint not null default 0 INDEX" json:"parent_id" doc:"The parent organization unit ID. Zero is only valid for a company. Immutable after creation."`
	Type        RetailOrgUnitType `xorm:"varchar(20) not null INDEX" json:"type" doc:"The organization unit type: company, region, store, or warehouse. Immutable after creation."`
	Name        string            `xorm:"varchar(250) not null" json:"name" valid:"required,runelength(1|250)" minLength:"1" maxLength:"250" doc:"The display name of this organization unit."`
	Code        string            `xorm:"varchar(64) not null unique" json:"code" valid:"required,runelength(1|64)" minLength:"1" maxLength:"64" doc:"A stable, unique business code for this organization unit."`
	TeamID      int64             `xorm:"bigint not null unique INDEX" json:"team_id" readOnly:"true" doc:"The Vikunja team that controls membership for this organization unit."`
	Active      bool              `xorm:"not null default true INDEX" json:"active" doc:"Whether this organization unit is active."`
	CreatedByID int64             `xorm:"bigint not null INDEX" json:"created_by_id" readOnly:"true" doc:"The user ID that created this organization unit."`
	Created     time.Time         `xorm:"created not null" json:"created" readOnly:"true" doc:"When this organization unit was created."`
	Updated     time.Time         `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When this organization unit was last updated."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

func (*RetailOrgUnit) TableName() string { return "retail_org_units" }

func GetRetailOrgUnitByID(s *xorm.Session, id int64) (*RetailOrgUnit, error) {
	unit := &RetailOrgUnit{ID: id}
	if id < 1 {
		return unit, ErrRetailOrgUnitDoesNotExist{ID: id}
	}
	exists, err := s.Get(unit)
	if err != nil {
		return unit, err
	}
	if !exists {
		return unit, ErrRetailOrgUnitDoesNotExist{ID: id}
	}
	return unit, nil
}

func (r *RetailOrgUnit) validateHierarchy(s *xorm.Session) error {
	switch r.Type {
	case RetailOrgUnitCompany:
		if r.ParentID != 0 {
			return ErrRetailOrgUnitInvalidParent{Type: r.Type}
		}
		return nil
	case RetailOrgUnitRegion, RetailOrgUnitStore, RetailOrgUnitWarehouse:
	default:
		return ErrRetailOrgUnitInvalidType{Type: r.Type}
	}

	parent, err := GetRetailOrgUnitByID(s, r.ParentID)
	if err != nil {
		return err
	}
	valid := r.Type == RetailOrgUnitRegion && parent.Type == RetailOrgUnitCompany ||
		r.Type == RetailOrgUnitStore && parent.Type == RetailOrgUnitRegion ||
		r.Type == RetailOrgUnitWarehouse && (parent.Type == RetailOrgUnitCompany || parent.Type == RetailOrgUnitRegion)
	if !valid {
		return ErrRetailOrgUnitInvalidParent{Type: r.Type, ParentType: parent.Type}
	}
	return nil
}

func (r *RetailOrgUnit) Create(s *xorm.Session, a web.Auth) error {
	creator, err := user.GetFromAuth(a)
	if err != nil {
		return err
	}
	r.Name = strings.TrimSpace(r.Name)
	r.Code = strings.TrimSpace(r.Code)
	if r.Name == "" || r.Code == "" {
		return ErrInvalidData{Message: "Organization unit name and code are required."}
	}
	if err := r.validateHierarchy(s); err != nil {
		return err
	}

	team := &Team{
		Name:        retailOrgTeamName(r.Code, r.Name),
		Description: "Managed by the retail organization hierarchy.",
	}
	if err := team.CreateNewTeam(s, creator, true); err != nil {
		return err
	}

	r.ID = 0
	r.TeamID = team.ID
	r.CreatedByID = creator.ID
	if !r.Active {
		// New units are active by default. They can be deactivated afterwards.
		r.Active = true
	}
	_, err = s.Insert(r)
	return err
}

func (r *RetailOrgUnit) ReadOne(s *xorm.Session, _ web.Auth) error {
	unit, err := GetRetailOrgUnitByID(s, r.ID)
	if unit != nil {
		*r = *unit
	}
	return err
}

func (r *RetailOrgUnit) ReadAll(s *xorm.Session, a web.Auth, search string, page, perPage int) (interface{}, int, int64, error) {
	if _, is := a.(*LinkSharing); is {
		return nil, 0, 0, ErrGenericForbidden{}
	}

	all := []*RetailOrgUnit{}
	if err := s.Asc("id").Find(&all); err != nil {
		return nil, 0, 0, err
	}
	teamIDs := []int64{}
	if err := s.Table("team_members").Cols("team_id").Where("user_id = ?", a.GetID()).Find(&teamIDs); err != nil {
		return nil, 0, 0, err
	}
	memberOf := make(map[int64]bool, len(teamIDs))
	for _, teamID := range teamIDs {
		memberOf[teamID] = true
	}

	visible := make(map[int64]bool)
	for changed := true; changed; {
		changed = false
		for _, unit := range all {
			if visible[unit.ID] || (!memberOf[unit.TeamID] && !visible[unit.ParentID]) {
				continue
			}
			visible[unit.ID] = true
			changed = true
		}
	}

	needle := strings.ToLower(strings.TrimSpace(search))
	result := make([]*RetailOrgUnit, 0, len(visible))
	for _, unit := range all {
		if !visible[unit.ID] {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(unit.Name), needle) && !strings.Contains(strings.ToLower(unit.Code), needle) {
			continue
		}
		result = append(result, unit)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	total := int64(len(result))
	limit, start := getLimitFromPageIndex(page, perPage)
	if limit > 0 && start < len(result) {
		end := start + limit
		if end > len(result) {
			end = len(result)
		}
		result = result[start:end]
	} else if limit > 0 {
		result = []*RetailOrgUnit{}
	}
	return result, len(result), total, nil
}

func (r *RetailOrgUnit) Update(s *xorm.Session, _ web.Auth) error {
	existing, err := GetRetailOrgUnitByID(s, r.ID)
	if err != nil {
		return err
	}
	r.Name = strings.TrimSpace(r.Name)
	r.Code = strings.TrimSpace(r.Code)
	if r.Name == "" || r.Code == "" {
		return ErrInvalidData{Message: "Organization unit name and code are required."}
	}
	_, err = s.ID(r.ID).Cols("name", "code", "active").Update(&RetailOrgUnit{Name: r.Name, Code: r.Code, Active: r.Active})
	if err != nil {
		return err
	}
	_, err = s.ID(existing.TeamID).Cols("name").Update(&Team{Name: retailOrgTeamName(r.Code, r.Name)})
	if err != nil {
		return err
	}
	return r.ReadOne(s, nil)
}

func (r *RetailOrgUnit) Delete(s *xorm.Session, a web.Auth) error {
	existing, err := GetRetailOrgUnitByID(s, r.ID)
	if err != nil {
		return err
	}
	hasChildren, err := s.Where("parent_id = ?", r.ID).Exist(&RetailOrgUnit{})
	if err != nil {
		return err
	}
	if hasChildren {
		return ErrRetailOrgUnitHasChildren{ID: r.ID}
	}
	if _, err = s.ID(r.ID).Delete(&RetailOrgUnit{}); err != nil {
		return err
	}
	return (&Team{ID: existing.TeamID}).Delete(s, a)
}

func retailOrgTeamName(code, name string) string {
	return "[Retail " + code + "] " + name
}
