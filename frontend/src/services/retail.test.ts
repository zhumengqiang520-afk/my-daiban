import {beforeEach, describe, expect, it, vi} from 'vitest'

const http = vi.hoisted(() => ({
	get: vi.fn(),
	request: vi.fn(),
}))

vi.mock('@/helpers/fetcher', () => ({
	apiV2Url: (path: string) => `/api/v2/${path}`,
	AuthenticatedHTTPFactory: () => http,
}))

import {retailService} from './retail'

describe('retailService', () => {
	beforeEach(() => {
		http.get.mockReset()
		http.request.mockReset()
	})

	it('converts paginated profile responses and query parameters', async () => {
		http.get.mockResolvedValue({
			data: {
				items: [{id: 3, task_id: 9, task_title: '整理样床', org_unit_id: 2, primary_assignee_id: 7}],
				total: 1,
			},
		})

		const profiles = await retailService.getProfiles({orgUnitId: 2})

		expect(http.get).toHaveBeenCalledWith('/api/v2/retail/task-profiles', {
			params: {per_page: 100, org_unit_id: 2},
		})
		expect(profiles[0]).toMatchObject({taskId: 9, taskTitle: '整理样床', orgUnitId: 2, primaryAssigneeId: 7})
	})

	it('preserves top-level arrays while converting workload fields', async () => {
		http.get.mockResolvedValue({
			data: [{org_unit_id: 2, user_id: 7, assigned_minutes: 360, capacity_minutes: 420}],
		})

		const workload = await retailService.getWorkload(2, '2026-08-21', '2026-08-27')

		expect(Array.isArray(workload)).toBe(true)
		expect(workload[0]).toMatchObject({orgUnitId: 2, userId: 7, assignedMinutes: 360, capacityMinutes: 420})
	})

	it('sends dispatch payloads to v2 using snake_case fields', async () => {
		http.request.mockResolvedValue({data: []})

		await retailService.dispatchTemplate(5, {
			targetOrgUnitId: 2,
			primaryAssigneeId: 7,
			dueDate: '2026-08-21T12:00:00Z',
		})

		expect(http.request).toHaveBeenCalledWith({
			method: 'post',
			url: '/api/v2/retail/templates/5/dispatch',
			data: {
				targets: [{
					target_org_unit_id: 2,
					primary_assignee_id: 7,
					due_date: '2026-08-21T12:00:00Z',
				}],
			},
		})
	})

	it('updates staff membership through the partial-update endpoint', async () => {
		http.request.mockResolvedValue({data: {id: 12, job_title: '店长', active: false}})

		const membership = await retailService.updateMembership(12, {jobTitle: '店长', active: false})

		expect(http.request).toHaveBeenCalledWith({
			method: 'patch',
			url: '/api/v2/retail/memberships/12',
			data: {job_title: '店长', active: false},
		})
		expect(membership).toMatchObject({id: 12, jobTitle: '店长', active: false})
	})
})
