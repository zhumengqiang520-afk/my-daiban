// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/cron"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/user"
)

// RegisterRetailMembershipExpiryCron revokes access for ended temporary assignments.
func RegisterRetailMembershipExpiryCron() {
	if !config.RetailEnabled.GetBool() {
		return
	}
	if err := cron.Schedule("* * * * *", func() { expireRetailMembershipsAt(time.Now()) }); err != nil {
		log.Errorf("Could not register retail membership expiry cron: %s", err)
	}
}

func expireRetailMembershipsAt(now time.Time) {
	s := db.NewSession()
	defer s.Close()
	memberships := []*RetailMembership{}
	err := s.Where("temporary = ?", true).
		And("active = ?", true).
		And("ends_at <= ?", now).
		Find(&memberships)
	if err != nil {
		log.Errorf("Could not load expired retail memberships: %s", err)
		return
	}
	for _, membership := range memberships {
		org, orgErr := GetRetailOrgUnitByID(s, membership.OrgUnitID)
		if orgErr != nil {
			log.Errorf("Could not load organization unit %d for expired membership %d: %s", membership.OrgUnitID, membership.ID, orgErr)
			continue
		}
		member, userErr := user.GetUserByID(s, membership.UserID)
		if userErr != nil && !user.IsErrUserStatusError(userErr) {
			log.Errorf("Could not load user %d for expired membership %d: %s", membership.UserID, membership.ID, userErr)
			continue
		}
		if adminErr := ensureRetailOrgKeepsAdmin(s, org.TeamID, org.ID, membership.UserID, false, false); adminErr != nil {
			log.Errorf("Could not expire retail membership %d: %s", membership.ID, adminErr)
			continue
		}
		if syncErr := syncRetailTeamMember(s, org.TeamID, member, false, false, member); syncErr != nil {
			log.Errorf("Could not revoke team access for expired retail membership %d: %s", membership.ID, syncErr)
			continue
		}
		if _, updateErr := s.ID(membership.ID).Cols("active").Update(&RetailMembership{Active: false}); updateErr != nil {
			log.Errorf("Could not deactivate expired retail membership %d: %s", membership.ID, updateErr)
			continue
		}
	}
	if err := s.Commit(); err != nil {
		log.Errorf("Could not commit expired retail membership cleanup: %s", err)
	}
}
