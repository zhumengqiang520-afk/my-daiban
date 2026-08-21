// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"strconv"
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/cron"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/notifications"
	"code.vikunja.io/api/pkg/user"

	"xorm.io/xorm"
)

func RegisterRetailEscalationCron() {
	if !config.RetailEnabled.GetBool() {
		return
	}
	if err := cron.Schedule("* * * * *", func() { sendRetailEscalationsAt(time.Now()) }); err != nil {
		log.Errorf("Could not register retail escalation cron: %s", err)
	}
}

func sendRetailEscalationsAt(now time.Time) {
	s := db.NewSession()
	defer s.Close()
	type overdueRetailTask struct {
		ProfileID         int64            `xorm:"profile_id"`
		TaskID            int64            `xorm:"task_id"`
		TaskTitle         string           `xorm:"task_title"`
		ProjectID         int64            `xorm:"project_id"`
		DueDate           time.Time        `xorm:"due_date"`
		OrgUnitID         int64            `xorm:"org_unit_id"`
		PrimaryAssigneeID int64            `xorm:"primary_assignee_id"`
		ReviewerID        int64            `xorm:"reviewer_id"`
		Status            RetailTaskStatus `xorm:"status"`
	}
	overdue := []*overdueRetailTask{}
	err := s.Table("retail_task_profiles").
		Select("retail_task_profiles.id AS profile_id, tasks.id AS task_id, tasks.title AS task_title, tasks.project_id, tasks.due_date, retail_task_profiles.org_unit_id, retail_task_profiles.primary_assignee_id, retail_task_profiles.reviewer_id, retail_task_profiles.status").
		Join("INNER", "tasks", "tasks.id = retail_task_profiles.task_id").
		NotIn("retail_task_profiles.status", []RetailTaskStatus{RetailTaskStatusCompleted, RetailTaskStatusCancelled}).
		And("tasks.due_date IS NOT NULL").And("tasks.due_date < ?", now).
		Find(&overdue)
	if err != nil {
		log.Errorf("Could not load overdue retail tasks: %s", err)
		_ = s.Rollback()
		return
	}
	for _, task := range overdue {
		org, err := GetRetailOrgUnitByID(s, task.OrgUnitID)
		if err != nil {
			log.Errorf("Could not load organization for overdue retail task %d: %s", task.TaskID, err)
			continue
		}
		levels := retailEscalationRecipients(s, task.OrgUnitID, task.PrimaryAssigneeID, task.ReviewerID, now.Sub(task.DueDate))
		for level, recipients := range levels {
			for _, recipientID := range recipients {
				if err := sendOneRetailEscalation(s, task.ProfileID, task.TaskID, task.TaskTitle, org, task.DueDate, level, recipientID, now); err != nil {
					log.Errorf("Could not send retail escalation for task %d to user %d: %s", task.TaskID, recipientID, err)
				}
			}
		}
	}
	if err := s.Commit(); err != nil {
		log.Errorf("Could not commit retail escalation deliveries: %s", err)
	}
}

func retailEscalationRecipients(s *xorm.Session, orgUnitID, assigneeID, reviewerID int64, overdueFor time.Duration) map[string][]int64 {
	result := map[string][]int64{}
	if assigneeID > 0 {
		result[RetailEscalationAssignee] = []int64{assigneeID}
	}
	if overdueFor >= 30*time.Minute {
		managerID := reviewerID
		if managerID == 0 && assigneeID > 0 {
			membership := &RetailMembership{}
			if exists, _ := s.Where("org_unit_id = ?", orgUnitID).And("user_id = ?", assigneeID).Get(membership); exists {
				managerID = membership.ManagerUserID
			}
		}
		if managerID > 0 {
			result[RetailEscalationManager] = []int64{managerID}
		}
	}
	if overdueFor >= 120*time.Minute {
		result[RetailEscalationArea] = retailAncestorAdminIDs(s, orgUnitID)
	}
	return result
}

func retailAncestorAdminIDs(s *xorm.Session, orgUnitID int64) []int64 {
	org, err := GetRetailOrgUnitByID(s, orgUnitID)
	if err != nil {
		return nil
	}
	if org.ParentID > 0 {
		org, err = GetRetailOrgUnitByID(s, org.ParentID)
		if err != nil {
			return nil
		}
	}
	members := []*TeamMember{}
	if err := s.Where("team_id = ?", org.TeamID).And("admin = ?", true).Find(&members); err != nil {
		return nil
	}
	ids := make([]int64, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.UserID)
	}
	return uniqueInt64s(ids)
}

func sendOneRetailEscalation(s *xorm.Session, profileID, taskID int64, taskTitle string, org *RetailOrgUnit, dueDate time.Time, level string, recipientID int64, now time.Time) error {
	key := "retail-escalation:" + strconv.FormatInt(profileID, 10) + ":" + level + ":" + strconv.FormatInt(recipientID, 10)
	exists, err := s.Where("idempotency_key = ?", key).Exist(&RetailNotificationDelivery{})
	if err != nil || exists {
		return err
	}
	recipient, err := user.GetUserByID(s, recipientID)
	if err != nil {
		return err
	}
	notification := &RetailTaskEscalationNotification{
		ProfileID: profileID, TaskID: taskID, TaskTitle: taskTitle, OrgUnitID: org.ID, OrgName: org.Name, Level: level, DueDate: dueDate,
	}
	if err := notifications.Notify(recipient, notification, s); err != nil {
		return err
	}
	_, err = s.Insert(&RetailNotificationDelivery{
		ProfileID: profileID, TaskID: taskID, Level: level, RecipientID: recipientID, IdempotencyKey: key, SentAt: now,
	})
	return err
}
