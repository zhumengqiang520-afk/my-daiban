// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"fmt"
	"time"

	"code.vikunja.io/api/pkg/notifications"
)

const (
	RetailEscalationAssignee = "assignee_overdue"
	RetailEscalationManager  = "manager_30m"
	RetailEscalationArea     = "area_120m"
)

type RetailNotificationDelivery struct {
	ID             int64     `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true"`
	ProfileID      int64     `xorm:"bigint not null INDEX" json:"profile_id" readOnly:"true"`
	TaskID         int64     `xorm:"bigint not null INDEX" json:"task_id" readOnly:"true"`
	Level          string    `xorm:"varchar(30) not null INDEX" json:"level" readOnly:"true"`
	RecipientID    int64     `xorm:"bigint not null INDEX" json:"recipient_id" readOnly:"true"`
	IdempotencyKey string    `xorm:"varchar(250) not null unique INDEX" json:"idempotency_key" readOnly:"true"`
	SentAt         time.Time `xorm:"datetime not null INDEX" json:"sent_at" readOnly:"true"`
	Created        time.Time `xorm:"created not null" json:"created" readOnly:"true"`
}

func (*RetailNotificationDelivery) TableName() string { return "retail_notification_deliveries" }

type RetailTaskEscalationNotification struct {
	ProfileID int64     `json:"profile_id"`
	TaskID    int64     `json:"task_id"`
	TaskTitle string    `json:"task_title"`
	OrgUnitID int64     `json:"org_unit_id"`
	OrgName   string    `json:"org_name"`
	Level     string    `json:"level"`
	DueDate   time.Time `json:"due_date"`
}

func init() {
	notifications.Register(func() notifications.PersistedNotification { return &RetailTaskEscalationNotification{} })
}

func (n *RetailTaskEscalationNotification) ToTitle(_ string) string {
	return fmt.Sprintf("Retail task overdue: %s", n.TaskTitle)
}

func (n *RetailTaskEscalationNotification) ToMail(_ string) *notifications.Mail { return nil }

func (n *RetailTaskEscalationNotification) ToDB() interface{} { return n }

func (n *RetailTaskEscalationNotification) Name() string { return "retail.task.escalation" }

func (n *RetailTaskEscalationNotification) SubjectID() int64 { return n.TaskID }

// Retail notifications are recipient-specific and account-scoped because
// ancestor managers inherit retail scope without necessarily having a direct
// Vikunja project share.
func (n *RetailTaskEscalationNotification) ProjectID() int64 { return 0 }
