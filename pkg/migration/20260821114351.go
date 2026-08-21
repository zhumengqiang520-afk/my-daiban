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

type RetailChecklistItem20260821114351 struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk"`
	ProfileID   int64     `xorm:"bigint not null INDEX"`
	Title       string    `xorm:"varchar(500) not null"`
	Required    bool      `xorm:"not null default true INDEX"`
	Position    int       `xorm:"int not null default 0 INDEX"`
	Done        bool      `xorm:"not null default false INDEX"`
	DoneByID    int64     `xorm:"bigint null INDEX"`
	DoneAt      time.Time `xorm:"datetime null"`
	CreatedByID int64     `xorm:"bigint not null INDEX"`
	Created     time.Time `xorm:"created not null"`
	Updated     time.Time `xorm:"updated not null"`
}

func (RetailChecklistItem20260821114351) TableName() string { return "retail_checklist_items" }

type RetailSubmission20260821114351 struct {
	ID            int64     `xorm:"bigint autoincr not null unique pk"`
	ProfileID     int64     `xorm:"bigint not null INDEX"`
	SubmittedByID int64     `xorm:"bigint not null INDEX"`
	Note          string    `xorm:"longtext null"`
	Created       time.Time `xorm:"created not null INDEX"`
}

func (RetailSubmission20260821114351) TableName() string { return "retail_submissions" }

type RetailSubmissionFile20260821114351 struct {
	ID           int64 `xorm:"bigint autoincr not null unique pk"`
	SubmissionID int64 `xorm:"bigint not null INDEX unique(retail_submission_attachment)"`
	AttachmentID int64 `xorm:"bigint not null INDEX unique(retail_submission_attachment)"`
}

func (RetailSubmissionFile20260821114351) TableName() string { return "retail_submission_files" }

type RetailReview20260821114351 struct {
	ID           int64     `xorm:"bigint autoincr not null unique pk"`
	ProfileID    int64     `xorm:"bigint not null INDEX"`
	SubmissionID int64     `xorm:"bigint not null INDEX"`
	ReviewerID   int64     `xorm:"bigint not null INDEX"`
	Decision     string    `xorm:"varchar(20) not null INDEX"`
	Comment      string    `xorm:"longtext null"`
	Created      time.Time `xorm:"created not null INDEX"`
}

func (RetailReview20260821114351) TableName() string { return "retail_reviews" }

type RetailTaskTransition20260821114351 struct {
	ID        int64     `xorm:"bigint autoincr not null unique pk"`
	ProfileID int64     `xorm:"bigint not null INDEX"`
	From      string    `xorm:"varchar(30) not null INDEX"`
	To        string    `xorm:"varchar(30) not null INDEX"`
	ActorID   int64     `xorm:"bigint not null INDEX"`
	Reason    string    `xorm:"longtext null"`
	Created   time.Time `xorm:"created not null INDEX"`
}

func (RetailTaskTransition20260821114351) TableName() string {
	return "retail_task_transitions"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260821114351",
		Description: "Create retail checklist and review workflow",
		Migrate: func(tx *xorm.Engine) error {
			return tx.Sync( //nolint:forbidigo // all tables in this migration are brand new
				RetailChecklistItem20260821114351{},
				RetailSubmission20260821114351{},
				RetailSubmissionFile20260821114351{},
				RetailReview20260821114351{},
				RetailTaskTransition20260821114351{},
			)
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(
				RetailTaskTransition20260821114351{},
				RetailReview20260821114351{},
				RetailSubmissionFile20260821114351{},
				RetailSubmission20260821114351{},
				RetailChecklistItem20260821114351{},
			)
		},
	})
}
