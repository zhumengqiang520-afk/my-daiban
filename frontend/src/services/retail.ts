import {apiV2Url, AuthenticatedHTTPFactory} from '@/helpers/fetcher'
import {objectToCamelCase, objectToSnakeCase} from '@/helpers/case'

export type RetailStatus = 'draft' | 'assigned' | 'in_progress' | 'pending_review' | 'rejected' | 'completed' | 'cancelled'
export type RetailCategory = 'opening' | 'closing' | 'display' | 'inventory' | 'customer_followup' | 'delivery' | 'after_sales' | 'other'

export interface RetailOrgUnit {
	id: number
	parentId: number
	type: 'company' | 'region' | 'store' | 'warehouse'
	name: string
	code: string
	active: boolean
}

export interface RetailTaskProfile {
	id: number
	taskId: number
	taskTitle: string
	dueDate?: string
	orgUnitId: number
	orgUnitName: string
	category: RetailCategory
	primaryAssigneeId: number
	primaryAssignee: string
	reviewerId: number
	reviewer: string
	estimatedMinutes: number
	status: RetailStatus
	evidenceRequired: boolean
}

export interface RetailMembership {
	id: number
	orgUnitId: number
	userId: number
	username: string
	userName: string
	jobTitle: string
	managerUserId: number
	managerName: string
	admin: boolean
	primary: boolean
	temporary: boolean
	startsAt?: string
	endsAt?: string
	active: boolean
}

export interface RetailChecklistItem {
	id: number
	profileId: number
	title: string
	required: boolean
	position: number
	done: boolean
	doneById: number
	doneAt?: string
}

export interface RetailSubmission {
	id: number
	note: string
	evidenceAttachmentIds: number[]
	created: string
}

export interface RetailReview {
	id: number
	submissionId: number
	decision: 'approved' | 'rejected'
	comment: string
	created: string
}

export interface RetailWorkflow {
	profile: RetailTaskProfile
	checklist: RetailChecklistItem[]
	submissions: RetailSubmission[]
	reviews: RetailReview[]
	transitions: Array<{id: number, from: RetailStatus, to: RetailStatus, actorId: number, reason: string, created: string}>
}

export interface RetailDashboard {
	total: number
	completed: number
	overdue: number
	pendingReview: number
	rejected: number
	cancelled: number
	onTimeCompleted: number
	completionRatePercent: number
	onTimeRatePercent: number
	rejectionRatePercent: number
	statusCounts: Record<string, number>
	categoryCounts: Record<string, number>
	overdueTasks: Array<{profileId: number, taskId: number, title: string, orgUnitName: string, status: RetailStatus, dueDate: string}>
}

export interface RetailWorkload {
	orgUnitId: number
	orgUnitName: string
	userId: number
	userName: string
	jobTitle: string
	capacityDay: string
	capacityMinutes: number
	assignedMinutes: number
	taskCount: number
	utilizationPercent: number
	warning: boolean
	overloaded: boolean
}

export interface RetailTemplate {
	id: number
	orgUnitId: number
	orgUnitName: string
	name: string
	title: string
	description: string
	category: RetailCategory
	estimatedMinutes: number
	evidenceRequired: boolean
	active: boolean
	currentVersion: number
	checklist: Array<{title: string, required: boolean, position: number}>
}

export interface RetailTemplateSchedule {
	id: number
	templateId: number
	templateName: string
	targetOrgUnitId: number
	targetOrgUnitName: string
	projectId: number
	primaryAssigneeId: number
	reviewerId: number
	frequency: 'daily' | 'weekly' | 'monthly'
	interval: number
	dueOffsetMinutes: number
	nextRunAt: string
	active: boolean
}

interface Paginated<T> {
	items: T[]
	total: number
}

const http = AuthenticatedHTTPFactory()

function camel<T>(value: unknown): T {
	if (Array.isArray(value)) {
		return value.map(item => camel(item)) as T
	}
	if (typeof value === 'object' && value !== null) {
		return objectToCamelCase(value as Record<string, unknown>) as T
	}
	return value as T
}

async function get<T>(path: string, params?: Record<string, unknown>): Promise<T> {
	const {data} = await http.get(apiV2Url(path), {params: objectToSnakeCase(params ?? {})})
	return camel<T>(data)
}

async function send<T>(method: 'post' | 'put' | 'patch', path: string, body?: Record<string, unknown>): Promise<T> {
	const {data} = await http.request({method, url: apiV2Url(path), data: objectToSnakeCase(body ?? {})})
	return camel<T>(data)
}

export const retailService = {
	async getOrgUnits(): Promise<RetailOrgUnit[]> {
		return (await get<Paginated<RetailOrgUnit>>('retail/org-units', {perPage: 100})).items
	},
	createOrgUnit(orgUnit: Omit<RetailOrgUnit, 'id'>) {
		return send<RetailOrgUnit>('post', 'retail/org-units', orgUnit)
	},
	async getProfiles(params: Record<string, unknown> = {}): Promise<RetailTaskProfile[]> {
		return (await get<Paginated<RetailTaskProfile>>('retail/task-profiles', {perPage: 100, ...params})).items
	},
	async getMemberships(orgUnitId?: number): Promise<RetailMembership[]> {
		return (await get<Paginated<RetailMembership>>('retail/memberships', {perPage: 100, orgUnitId})).items
	},
	createMembership(membership: Record<string, unknown>) {
		return send<RetailMembership>('post', 'retail/memberships', membership)
	},
	updateMembership(id: number, membership: Record<string, unknown>) {
		return send<RetailMembership>('patch', `retail/memberships/${id}`, membership)
	},
	getWorkflow(taskId: number) {
		return get<RetailWorkflow>(`retail/tasks/${taskId}/workflow`)
	},
	startTask(taskId: number) {
		return send<RetailWorkflow>('post', `retail/tasks/${taskId}/start`)
	},
	setChecklistDone(itemId: number, done: boolean) {
		return send<RetailChecklistItem>('put', `retail/checklist-items/${itemId}/completion`, {done})
	},
	submitTask(taskId: number, note: string, evidenceAttachmentIds: number[]) {
		return send<RetailWorkflow>('post', `retail/tasks/${taskId}/submissions`, {note, evidenceAttachmentIds})
	},
	reviewTask(taskId: number, submissionId: number, decision: 'approved' | 'rejected', comment: string) {
		return send<RetailWorkflow>('post', `retail/tasks/${taskId}/reviews`, {submissionId, decision, comment})
	},
	getDashboard(orgUnitId: number, dateFrom: string, dateTo: string) {
		return get<RetailDashboard>('retail/dashboard/operations', {orgUnitId, dateFrom, dateTo})
	},
	getWorkload(orgUnitId: number, dateFrom: string, dateTo: string) {
		return get<RetailWorkload[]>('retail/staff/workload', {orgUnitId, dateFrom, dateTo})
	},
	setCapacity(userId: number, orgUnitId: number, capacityDay: string, minutes: number, reason: string) {
		return send('put', `retail/staff/${userId}/capacity`, {orgUnitId, capacityDay, minutes, reason})
	},
	async getTemplates(): Promise<RetailTemplate[]> {
		return (await get<Paginated<RetailTemplate>>('retail/templates', {perPage: 100})).items
	},
	createTemplate(template: Omit<RetailTemplate, 'id' | 'orgUnitName' | 'currentVersion'>) {
		return send<RetailTemplate>('post', 'retail/templates', template)
	},
	async getSchedules(): Promise<RetailTemplateSchedule[]> {
		return (await get<Paginated<RetailTemplateSchedule>>('retail/template-schedules', {perPage: 100})).items
	},
	createSchedule(schedule: Omit<RetailTemplateSchedule, 'id' | 'templateName' | 'targetOrgUnitName'>) {
		return send<RetailTemplateSchedule>('post', 'retail/template-schedules', schedule)
	},
	dispatchTemplate(templateId: number, target: Record<string, unknown>) {
		return send<Array<{reused: boolean, profile: RetailTaskProfile}>>('post', `retail/templates/${templateId}/dispatch`, {targets: [target]})
	},
}
