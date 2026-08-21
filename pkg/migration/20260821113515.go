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

type RetailTaskProfiles20260821113515 struct {
	ID                int64     `xorm:"bigint autoincr not null unique pk"`
	TaskID            int64     `xorm:"bigint not null unique INDEX"`
	OrgUnitID         int64     `xorm:"bigint not null INDEX"`
	Category          string    `xorm:"varchar(40) not null INDEX"`
	PrimaryAssigneeID int64     `xorm:"bigint null INDEX"`
	ReviewerID        int64     `xorm:"bigint null INDEX"`
	EstimatedMinutes  int       `xorm:"int not null default 0 INDEX"`
	Status            string    `xorm:"varchar(30) not null INDEX"`
	Source            string    `xorm:"varchar(30) not null INDEX"`
	SourceID          int64     `xorm:"bigint null INDEX"`
	EvidenceRequired  bool      `xorm:"not null default false INDEX"`
	CreatedByID       int64     `xorm:"bigint not null INDEX"`
	Created           time.Time `xorm:"created not null"`
	Updated           time.Time `xorm:"updated not null"`
}

func (RetailTaskProfiles20260821113515) TableName() string {
	return "retail_task_profiles"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260821113515",
		Description: "Create retail task profiles",
		Migrate: func(tx *xorm.Engine) error {
			return tx.Sync(RetailTaskProfiles20260821113515{}) //nolint:forbidigo // brand-new table, nothing to preserve
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(RetailTaskProfiles20260821113515{})
		},
	})
}
