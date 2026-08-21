// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package migration

import (
	"time"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

type RetailMemberships20260821112334 struct {
	ID            int64     `xorm:"bigint autoincr not null unique pk"`
	OrgUnitID     int64     `xorm:"bigint not null INDEX unique(retail_org_user)"`
	UserID        int64     `xorm:"bigint not null INDEX unique(retail_org_user)"`
	JobTitle      string    `xorm:"varchar(100) null"`
	ManagerUserID int64     `xorm:"bigint null INDEX"`
	IsPrimary     bool      `xorm:"not null default false INDEX"`
	Temporary     bool      `xorm:"not null default false INDEX"`
	StartsAt      time.Time `xorm:"datetime null"`
	EndsAt        time.Time `xorm:"datetime null INDEX"`
	Active        bool      `xorm:"not null default true INDEX"`
	CreatedByID   int64     `xorm:"bigint not null INDEX"`
	Created       time.Time `xorm:"created not null"`
	Updated       time.Time `xorm:"updated not null"`
}

func (RetailMemberships20260821112334) TableName() string {
	return "retail_memberships"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260821112334",
		Description: "Create retail staff memberships",
		Migrate: func(tx *xorm.Engine) error {
			return tx.Sync(RetailMemberships20260821112334{}) //nolint:forbidigo // brand-new table, nothing to preserve
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(RetailMemberships20260821112334{})
		},
	})
}
