// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package migration

import (
	"time"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

type RetailTaskTemplate20260821115946 struct {
	ID               int64     `xorm:"bigint autoincr not null unique pk"`
	OrgUnitID        int64     `xorm:"bigint not null INDEX"`
	Name             string    `xorm:"varchar(250) not null INDEX"`
	Title            string    `xorm:"varchar(500) not null"`
	Description      string    `xorm:"longtext null"`
	Category         string    `xorm:"varchar(40) not null INDEX"`
	EstimatedMinutes int       `xorm:"int not null default 0 INDEX"`
	EvidenceRequired bool      `xorm:"not null default false INDEX"`
	Active           bool      `xorm:"not null default true INDEX"`
	CurrentVersion   int       `xorm:"int not null default 1"`
	CreatedByID      int64     `xorm:"bigint not null INDEX"`
	Created          time.Time `xorm:"created not null"`
	Updated          time.Time `xorm:"updated not null"`
}

func (RetailTaskTemplate20260821115946) TableName() string { return "retail_task_templates" }

type RetailTemplateVersion20260821115946 struct {
	ID               int64     `xorm:"bigint autoincr not null unique pk"`
	TemplateID       int64     `xorm:"bigint not null INDEX unique(retail_template_version)"`
	Version          int       `xorm:"int not null unique(retail_template_version)"`
	Title            string    `xorm:"varchar(500) not null"`
	Description      string    `xorm:"longtext null"`
	Category         string    `xorm:"varchar(40) not null INDEX"`
	EstimatedMinutes int       `xorm:"int not null default 0"`
	EvidenceRequired bool      `xorm:"not null default false"`
	ChecklistJSON    string    `xorm:"longtext null"`
	CreatedByID      int64     `xorm:"bigint not null INDEX"`
	Created          time.Time `xorm:"created not null"`
}

func (RetailTemplateVersion20260821115946) TableName() string { return "retail_template_versions" }

type RetailTemplateDispatch20260821115946 struct {
	ID                int64     `xorm:"bigint autoincr not null unique pk"`
	TemplateVersionID int64     `xorm:"bigint not null INDEX"`
	TargetOrgUnitID   int64     `xorm:"bigint not null INDEX"`
	ProjectID         int64     `xorm:"bigint not null INDEX"`
	TaskID            int64     `xorm:"bigint not null INDEX unique"`
	ScheduledFor      time.Time `xorm:"datetime not null INDEX"`
	IdempotencyKey    string    `xorm:"varchar(250) not null unique INDEX"`
	CreatedByID       int64     `xorm:"bigint not null INDEX"`
	Created           time.Time `xorm:"created not null"`
}

func (RetailTemplateDispatch20260821115946) TableName() string { return "retail_template_dispatches" }

type RetailStaffCapacity20260821115946 struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk"`
	OrgUnitID   int64     `xorm:"bigint not null INDEX unique(retail_capacity_day)"`
	UserID      int64     `xorm:"bigint not null INDEX unique(retail_capacity_day)"`
	CapacityDay time.Time `xorm:"date not null INDEX unique(retail_capacity_day)"`
	Minutes     int       `xorm:"int not null"`
	Reason      string    `xorm:"varchar(500) null"`
	CreatedByID int64     `xorm:"bigint not null INDEX"`
	Created     time.Time `xorm:"created not null"`
	Updated     time.Time `xorm:"updated not null"`
}

func (RetailStaffCapacity20260821115946) TableName() string { return "retail_staff_capacities" }

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260821115946",
		Description: "Create retail task templates, dispatches and staff capacities",
		Migrate: func(tx *xorm.Engine) error {
			return tx.Sync( //nolint:forbidigo // all tables in this migration are brand new
				RetailTaskTemplate20260821115946{},
				RetailTemplateVersion20260821115946{},
				RetailTemplateDispatch20260821115946{},
				RetailStaffCapacity20260821115946{},
			)
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(
				RetailStaffCapacity20260821115946{},
				RetailTemplateDispatch20260821115946{},
				RetailTemplateVersion20260821115946{},
				RetailTaskTemplate20260821115946{},
			)
		},
	})
}
