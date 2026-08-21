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

// RetailMembership adds retail employment metadata to a Vikunja team member.
type RetailMembership struct {
	ID            int64     `xorm:"bigint autoincr not null unique pk" json:"id" param:"id" readOnly:"true" doc:"The unique numeric ID of this staff membership."`
	OrgUnitID     int64     `xorm:"bigint not null INDEX unique(retail_org_user)" json:"org_unit_id" doc:"The organization unit this staff assignment belongs to. Immutable after creation."`
	UserID        int64     `xorm:"bigint not null INDEX unique(retail_org_user)" json:"user_id" readOnly:"true" doc:"The assigned user's numeric ID, resolved from username by the server."`
	Username      string    `xorm:"-" json:"username" valid:"required,runelength(1|250)" minLength:"1" maxLength:"250" doc:"The username to assign. Immutable after creation."`
	UserName      string    `xorm:"-" json:"user_name" readOnly:"true" doc:"The user's display name."`
	JobTitle      string    `xorm:"varchar(100) null" json:"job_title" maxLength:"100" doc:"The staff member's business job title, such as store manager or sales associate."`
	ManagerUserID int64     `xorm:"bigint null INDEX" json:"manager_user_id" doc:"The numeric ID of the staff member's direct manager in the same organization unit. Zero means none."`
	ManagerName   string    `xorm:"-" json:"manager_name" readOnly:"true" doc:"The direct manager's display name."`
	Admin         bool      `xorm:"-" json:"admin" doc:"Whether this member manages the organization unit and its descendants."`
	IsPrimary     bool      `xorm:"not null default false INDEX" json:"primary" doc:"Whether this is the user's primary organization assignment."`
	Temporary     bool      `xorm:"not null default false INDEX" json:"temporary" doc:"Whether this assignment is a temporary secondment."`
	StartsAt      time.Time `xorm:"datetime null" json:"starts_at,omitzero" doc:"When the assignment becomes effective. Future starts are not supported in the MVP."`
	EndsAt        time.Time `xorm:"datetime null INDEX" json:"ends_at,omitzero" doc:"When a temporary assignment expires. Required for temporary assignments."`
	Active        bool      `xorm:"not null default true INDEX" json:"active" doc:"Whether this assignment currently grants access."`
	CreatedByID   int64     `xorm:"bigint not null INDEX" json:"created_by_id" readOnly:"true" doc:"The user ID that created this membership."`
	Created       time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When this membership was created."`
	Updated       time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When this membership was last updated."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

func (*RetailMembership) TableName() string { return "retail_memberships" }

func GetRetailMembershipByID(s *xorm.Session, id int64) (*RetailMembership, error) {
	membership := &RetailMembership{ID: id}
	if id < 1 {
		return membership, ErrRetailMembershipDoesNotExist{ID: id}
	}
	exists, err := s.Get(membership)
	if err != nil {
		return membership, err
	}
	if !exists {
		return membership, ErrRetailMembershipDoesNotExist{ID: id}
	}
	if err := addRetailMembershipInfo(s, []*RetailMembership{membership}); err != nil {
		return membership, err
	}
	return membership, nil
}

func (r *RetailMembership) validate(s *xorm.Session) error {
	if _, err := GetRetailOrgUnitByID(s, r.OrgUnitID); err != nil {
		return err
	}
	if r.Temporary {
		if r.EndsAt.IsZero() || (!r.StartsAt.IsZero() && !r.EndsAt.After(r.StartsAt)) || r.StartsAt.After(time.Now()) {
			return ErrRetailMembershipInvalidPeriod{}
		}
	} else {
		r.EndsAt = time.Time{}
	}
	if r.ManagerUserID == 0 {
		return nil
	}
	if r.ManagerUserID == r.UserID {
		return ErrRetailMembershipInvalidManager{ManagerUserID: r.ManagerUserID}
	}
	manager := &RetailMembership{}
	exists, err := s.Where("org_unit_id = ?", r.OrgUnitID).And("user_id = ?", r.ManagerUserID).And("active = ?", true).Get(manager)
	if err != nil {
		return err
	}
	if !exists || (!manager.EndsAt.IsZero() && !manager.EndsAt.After(time.Now())) {
		return ErrRetailMembershipInvalidManager{ManagerUserID: r.ManagerUserID}
	}
	return nil
}

func (r *RetailMembership) Create(s *xorm.Session, a web.Auth) error {
	creator, err := user.GetFromAuth(a)
	if err != nil {
		return err
	}
	member, err := user.GetUserByUsername(s, strings.TrimSpace(r.Username))
	if err != nil {
		return err
	}
	r.UserID = member.ID
	r.Username = member.Username
	r.JobTitle = strings.TrimSpace(r.JobTitle)
	if err := r.validate(s); err != nil {
		return err
	}
	exists, err := s.Where("org_unit_id = ?", r.OrgUnitID).And("user_id = ?", r.UserID).Exist(&RetailMembership{})
	if err != nil {
		return err
	}
	if exists {
		return ErrRetailMembershipAlreadyExists{OrgUnitID: r.OrgUnitID, UserID: r.UserID}
	}
	if !r.Active {
		r.Active = true
	}
	if r.StartsAt.IsZero() {
		r.StartsAt = time.Now()
	}
	if r.IsPrimary {
		if err := clearOtherPrimaryRetailMemberships(s, r.UserID, 0); err != nil {
			return err
		}
	}
	org, err := GetRetailOrgUnitByID(s, r.OrgUnitID)
	if err != nil {
		return err
	}
	if err := syncRetailTeamMember(s, org.TeamID, member, r.Admin, true, a); err != nil {
		return err
	}
	r.ID = 0
	r.CreatedByID = creator.ID
	if _, err = s.Insert(r); err != nil {
		return err
	}
	return addRetailMembershipInfo(s, []*RetailMembership{r})
}

func (r *RetailMembership) ReadOne(s *xorm.Session, _ web.Auth) error {
	membership, err := GetRetailMembershipByID(s, r.ID)
	if membership != nil {
		*r = *membership
	}
	return err
}

func (r *RetailMembership) ReadAll(s *xorm.Session, a web.Auth, search string, page, perPage int) (interface{}, int, int64, error) {
	if _, is := a.(*LinkSharing); is {
		return nil, 0, 0, ErrGenericForbidden{}
	}
	all := []*RetailMembership{}
	query := s.Asc("org_unit_id", "id")
	if r.OrgUnitID > 0 {
		query = query.Where("org_unit_id = ?", r.OrgUnitID)
	}
	if err := query.Find(&all); err != nil {
		return nil, 0, 0, err
	}
	if err := addRetailMembershipInfo(s, all); err != nil {
		return nil, 0, 0, err
	}
	needle := strings.ToLower(strings.TrimSpace(search))
	result := make([]*RetailMembership, 0, len(all))
	access := map[int64]bool{}
	for _, membership := range all {
		can, known := access[membership.OrgUnitID]
		if !known {
			var accessErr error
			can, _, accessErr = (&RetailOrgUnit{ID: membership.OrgUnitID}).CanRead(s, a)
			if accessErr != nil {
				return nil, 0, 0, accessErr
			}
			access[membership.OrgUnitID] = can
		}
		if !can {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(membership.Username), needle) &&
			!strings.Contains(strings.ToLower(membership.UserName), needle) &&
			!strings.Contains(strings.ToLower(membership.JobTitle), needle) {
			continue
		}
		result = append(result, membership)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	total := int64(len(result))
	limit, start := getLimitFromPageIndex(page, perPage)
	if limit > 0 && start < len(result) {
		end := min(start+limit, len(result))
		result = result[start:end]
	} else if limit > 0 {
		result = []*RetailMembership{}
	}
	return result, len(result), total, nil
}

func (r *RetailMembership) Update(s *xorm.Session, a web.Auth) error {
	existing, err := GetRetailMembershipByID(s, r.ID)
	if err != nil {
		return err
	}
	r.OrgUnitID = existing.OrgUnitID
	r.UserID = existing.UserID
	r.Username = existing.Username
	r.JobTitle = strings.TrimSpace(r.JobTitle)
	if err := r.validate(s); err != nil {
		return err
	}
	org, err := GetRetailOrgUnitByID(s, r.OrgUnitID)
	if err != nil {
		return err
	}
	member, err := user.GetUserByID(s, r.UserID)
	if err != nil && !user.IsErrUserStatusError(err) {
		return err
	}
	if err := ensureRetailOrgKeepsAdmin(s, org.TeamID, r.OrgUnitID, r.UserID, r.Admin, r.Active); err != nil {
		return err
	}
	if err := syncRetailTeamMember(s, org.TeamID, member, r.Admin, r.Active, a); err != nil {
		return err
	}
	if r.IsPrimary {
		if err := clearOtherPrimaryRetailMemberships(s, r.UserID, r.ID); err != nil {
			return err
		}
	}
	_, err = s.ID(r.ID).Cols("job_title", "manager_user_id", "is_primary", "temporary", "starts_at", "ends_at", "active").Update(r)
	if err != nil {
		return err
	}
	return r.ReadOne(s, a)
}

func (r *RetailMembership) Delete(s *xorm.Session, a web.Auth) error {
	existing, err := GetRetailMembershipByID(s, r.ID)
	if err != nil {
		return err
	}
	org, err := GetRetailOrgUnitByID(s, existing.OrgUnitID)
	if err != nil {
		return err
	}
	if err := ensureRetailOrgKeepsAdmin(s, org.TeamID, existing.OrgUnitID, existing.UserID, false, false); err != nil {
		return err
	}
	member, err := user.GetUserByID(s, existing.UserID)
	if err != nil && !user.IsErrUserStatusError(err) {
		return err
	}
	if err := syncRetailTeamMember(s, org.TeamID, member, false, false, a); err != nil {
		return err
	}
	if _, err = s.Where("org_unit_id = ?", existing.OrgUnitID).And("manager_user_id = ?", existing.UserID).Cols("manager_user_id").Update(&RetailMembership{}); err != nil {
		return err
	}
	_, err = s.ID(existing.ID).Delete(&RetailMembership{})
	return err
}

func clearOtherPrimaryRetailMemberships(s *xorm.Session, userID, exceptID int64) error {
	query := s.Where("user_id = ?", userID).And("is_primary = ?", true)
	if exceptID > 0 {
		query = query.And("id != ?", exceptID)
	}
	_, err := query.Cols("is_primary").Update(&RetailMembership{IsPrimary: false})
	return err
}

func syncRetailTeamMember(s *xorm.Session, teamID int64, member *user.User, admin, active bool, a web.Auth) error {
	tm := &TeamMember{}
	exists, err := s.Where("team_id = ?", teamID).And("user_id = ?", member.ID).Get(tm)
	if err != nil {
		return err
	}
	if active {
		if !exists {
			return (&TeamMember{TeamID: teamID, Username: member.Username, Admin: admin}).Create(s, a)
		}
		_, err = s.ID(tm.ID).Cols("admin").Update(&TeamMember{Admin: admin})
		return err
	}
	if !exists {
		return nil
	}
	return (&TeamMember{TeamID: teamID, Username: member.Username, UserID: member.ID}).Delete(s, a)
}

func ensureRetailOrgKeepsAdmin(s *xorm.Session, teamID, orgUnitID, userID int64, willBeAdmin, willBeActive bool) error {
	current := &TeamMember{}
	exists, err := s.Where("team_id = ?", teamID).And("user_id = ?", userID).Get(current)
	if err != nil || !exists || !current.Admin || willBeAdmin && willBeActive {
		return err
	}
	admins, err := s.Where("team_id = ?", teamID).And("admin = ?", true).Count(&TeamMember{})
	if err != nil {
		return err
	}
	if admins <= 1 {
		return ErrRetailOrgUnitNeedsAdmin{OrgUnitID: orgUnitID}
	}
	return nil
}

func addRetailMembershipInfo(s *xorm.Session, memberships []*RetailMembership) error {
	if len(memberships) == 0 {
		return nil
	}
	userIDs := make([]int64, 0, len(memberships)*2)
	orgIDs := make([]int64, 0, len(memberships))
	for _, membership := range memberships {
		userIDs = append(userIDs, membership.UserID)
		if membership.ManagerUserID > 0 {
			userIDs = append(userIDs, membership.ManagerUserID)
		}
		orgIDs = append(orgIDs, membership.OrgUnitID)
	}
	users, err := user.GetUsersByIDs(s, userIDs)
	if err != nil {
		return err
	}
	orgs := []*RetailOrgUnit{}
	if err := s.In("id", orgIDs).Find(&orgs); err != nil {
		return err
	}
	teamByOrg := make(map[int64]int64, len(orgs))
	for _, org := range orgs {
		teamByOrg[org.ID] = org.TeamID
	}
	teamMembers := []*TeamMember{}
	if err := s.In("team_id", teamByOrgValues(teamByOrg)).Find(&teamMembers); err != nil {
		return err
	}
	adminByTeamUser := make(map[[2]int64]bool, len(teamMembers))
	for _, tm := range teamMembers {
		adminByTeamUser[[2]int64{tm.TeamID, tm.UserID}] = tm.Admin
	}
	for _, membership := range memberships {
		if u := users[membership.UserID]; u != nil {
			membership.Username = u.Username
			membership.UserName = u.GetName()
		}
		if manager := users[membership.ManagerUserID]; manager != nil {
			membership.ManagerName = manager.GetName()
		}
		membership.Admin = adminByTeamUser[[2]int64{teamByOrg[membership.OrgUnitID], membership.UserID}]
	}
	return nil
}

func teamByOrgValues(teamByOrg map[int64]int64) []int64 {
	values := make([]int64, 0, len(teamByOrg))
	for _, teamID := range teamByOrg {
		values = append(values, teamID)
	}
	return values
}
