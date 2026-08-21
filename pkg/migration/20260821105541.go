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

type RetailOrgUnits20260821105541 struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk"`
	ParentID    int64     `xorm:"bigint not null default 0 INDEX"`
	Type        string    `xorm:"varchar(20) not null INDEX"`
	Name        string    `xorm:"varchar(250) not null"`
	Code        string    `xorm:"varchar(64) not null unique"`
	TeamID      int64     `xorm:"bigint not null unique INDEX"`
	Active      bool      `xorm:"not null default true INDEX"`
	CreatedByID int64     `xorm:"bigint not null INDEX"`
	Created     time.Time `xorm:"created not null"`
	Updated     time.Time `xorm:"updated not null"`
}

func (RetailOrgUnits20260821105541) TableName() string {
	return "retail_org_units"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260821105541",
		Description: "Create retail organization units",
		Migrate: func(tx *xorm.Engine) error {
			return tx.Sync(RetailOrgUnits20260821105541{}) //nolint:forbidigo // brand-new table, nothing to preserve
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(RetailOrgUnits20260821105541{})
		},
	})
}
