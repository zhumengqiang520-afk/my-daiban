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

type RetailTemplateSchedule20260821121720 struct {
	ID                int64     `xorm:"bigint autoincr not null unique pk"`
	TemplateID        int64     `xorm:"bigint not null INDEX"`
	TargetOrgUnitID   int64     `xorm:"bigint not null INDEX"`
	ProjectID         int64     `xorm:"bigint not null INDEX"`
	PrimaryAssigneeID int64     `xorm:"bigint not null INDEX"`
	ReviewerID        int64     `xorm:"bigint null INDEX"`
	Frequency         string    `xorm:"varchar(20) not null INDEX"`
	Interval          int       `xorm:"int not null default 1"`
	DueOffsetMinutes  int       `xorm:"int not null default 0"`
	AnchorDay         int       `xorm:"int not null default 0"`
	NextRunAt         time.Time `xorm:"datetime not null INDEX"`
	LastRunAt         time.Time `xorm:"datetime null INDEX"`
	Active            bool      `xorm:"not null default true INDEX"`
	CreatedByID       int64     `xorm:"bigint not null INDEX"`
	Created           time.Time `xorm:"created not null"`
	Updated           time.Time `xorm:"updated not null"`
}

func (RetailTemplateSchedule20260821121720) TableName() string { return "retail_template_schedules" }

type RetailNotificationDelivery20260821121720 struct {
	ID             int64     `xorm:"bigint autoincr not null unique pk"`
	ProfileID      int64     `xorm:"bigint not null INDEX"`
	TaskID         int64     `xorm:"bigint not null INDEX"`
	Level          string    `xorm:"varchar(30) not null INDEX"`
	RecipientID    int64     `xorm:"bigint not null INDEX"`
	IdempotencyKey string    `xorm:"varchar(250) not null unique INDEX"`
	SentAt         time.Time `xorm:"datetime not null INDEX"`
	Created        time.Time `xorm:"created not null"`
}

func (RetailNotificationDelivery20260821121720) TableName() string {
	return "retail_notification_deliveries"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260821121720",
		Description: "Create recurring retail template schedules",
		Migrate: func(tx *xorm.Engine) error {
			return tx.Sync( //nolint:forbidigo // both tables are brand new
				RetailTemplateSchedule20260821121720{},
				RetailNotificationDelivery20260821121720{},
			)
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(RetailNotificationDelivery20260821121720{}, RetailTemplateSchedule20260821121720{})
		},
	})
}
