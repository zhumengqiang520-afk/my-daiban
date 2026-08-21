<template>
	<div class="retail-operations">
		<header class="retail-header">
			<div>
				<p class="retail-eyebrow">
					寝务看板
				</p>
				<h1>零售任务运营</h1>
				<p class="retail-subtitle">
					安排门店人员、跟进执行、复核结果与逾期风险。
				</p>
			</div>
			<label class="retail-org-picker">
				<span>当前组织</span>
				<select
					v-model.number="selectedOrgId"
					@change="loadCurrentTab"
				>
					<option
						v-for="org in orgUnits"
						:key="org.id"
						:value="org.id"
					>{{ org.name }}（{{ org.code }}）</option>
				</select>
			</label>
		</header>

		<nav
			class="retail-tabs"
			aria-label="零售任务功能"
		>
			<button
				v-for="item in tabs"
				:key="item.id"
				type="button"
				:class="{'is-active': activeTab === item.id}"
				@click="selectTab(item.id)"
			>
				<Icon :icon="item.icon" /> {{ item.label }}
			</button>
		</nav>

		<p
			v-if="loading"
			class="retail-empty"
		>
			正在加载…
		</p>
		<template v-else-if="selectedOrgId || activeTab === 'staff'">
			<section
				v-if="activeTab === 'overview' && dashboard"
				class="retail-section"
			>
				<div class="metric-grid">
					<div class="metric metric--primary">
						<span>任务总数</span><strong>{{ dashboard.total }}</strong>
					</div>
					<div class="metric metric--success">
						<span>完成率</span><strong>{{ dashboard.completionRatePercent }}%</strong>
					</div>
					<div class="metric">
						<span>按时率</span><strong>{{ dashboard.onTimeRatePercent }}%</strong>
					</div>
					<div class="metric metric--warning">
						<span>待复核</span><strong>{{ dashboard.pendingReview }}</strong>
					</div>
					<div class="metric metric--danger">
						<span>已逾期</span><strong>{{ dashboard.overdue }}</strong>
					</div>
					<div class="metric">
						<span>驳回率</span><strong>{{ dashboard.rejectionRatePercent }}%</strong>
					</div>
				</div>
				<Card
					title="逾期任务队列"
					class="retail-card"
				>
					<div
						v-if="dashboard.overdueTasks.length === 0"
						class="retail-empty"
					>
						当前筛选范围没有逾期任务。
					</div>
					<div
						v-for="task in dashboard.overdueTasks"
						v-else
						:key="task.taskId"
						class="queue-row"
					>
						<div><strong>{{ task.title }}</strong><small>{{ task.orgUnitName }} · {{ statusLabel(task.status) }}</small></div>
						<div class="queue-row__actions">
							<time>{{ formatDateTime(task.dueDate) }}</time><XButton
								variant="secondary"
								size="small"
								@click="openWorkflow(task.taskId)"
							>
								跟进
							</XButton>
						</div>
					</div>
				</Card>
			</section>

			<section
				v-if="activeTab === 'tasks'"
				class="retail-section task-columns"
			>
				<Card
					title="我的待办"
					class="retail-card"
				>
					<div
						v-if="myTasks.length === 0"
						class="retail-empty"
					>
						暂无待办。
					</div>
					<button
						v-for="task in myTasks"
						:key="task.id"
						type="button"
						class="task-row"
						@click="openWorkflow(task.taskId)"
					>
						<span><strong>{{ task.taskTitle }}</strong><small>{{ categoryLabel(task.category) }} · {{ task.estimatedMinutes }} 分钟</small></span>
						<span
							class="status-pill"
							:class="[`status-${task.status}`]"
						>{{ statusLabel(task.status) }}</span>
					</button>
				</Card>
				<Card
					title="待我复核"
					class="retail-card"
				>
					<div
						v-if="reviewTasks.length === 0"
						class="retail-empty"
					>
						暂无待复核任务。
					</div>
					<button
						v-for="task in reviewTasks"
						:key="task.id"
						type="button"
						class="task-row"
						@click="openWorkflow(task.taskId)"
					>
						<span><strong>{{ task.taskTitle }}</strong><small>{{ task.primaryAssignee || '未显示负责人' }}</small></span>
						<span class="status-pill status-pending_review">待复核</span>
					</button>
				</Card>
			</section>

			<section
				v-if="activeTab === 'staff'"
				class="retail-section staff-layout"
			>
				<Card
					title="组织与门店"
					class="retail-card"
				>
					<div class="org-list">
						<button
							v-for="org in orgUnits"
							:key="org.id"
							type="button"
							:class="{'is-selected': org.id === selectedOrgId}"
							@click="selectOrganization(org.id)"
						>
							<span>{{ org.name }}</span><small>{{ orgTypeLabel(org.type) }} · {{ org.code }}</small>
						</button>
					</div>
					<form
						class="retail-form divided-form"
						@submit.prevent="createOrgUnit"
					>
						<h3>新增组织节点</h3>
						<div class="form-grid">
							<label>类型<select
								v-model="orgForm.type"
								@change="resetOrgParent"
							><option value="company">公司</option><option value="region">区域</option><option value="store">门店</option><option value="warehouse">仓库</option></select></label><label>上级<select
								v-model.number="orgForm.parentId"
								:disabled="orgForm.type === 'company'"
							><option :value="0">无</option><option
								v-for="org in parentOrgOptions"
								:key="org.id"
								:value="org.id"
							>{{ org.name }}</option></select></label>
						</div>
						<div class="form-grid">
							<label>名称<input
								v-model.trim="orgForm.name"
								required
								maxlength="250"
							></label><label>业务编码<input
								v-model.trim="orgForm.code"
								required
								maxlength="64"
								placeholder="例如 SH-PD-01"
							></label>
						</div>
						<XButton
							type="submit"
							:loading="saving"
						>
							创建组织
						</XButton>
					</form>
				</Card>

				<Card
					v-if="selectedOrgId"
					title="人员与岗位"
					class="retail-card"
				>
					<div
						v-if="members.length === 0"
						class="retail-empty"
					>
						当前组织尚无员工。
					</div>
					<div
						v-for="member in members"
						v-else
						:key="member.id"
						class="staff-row"
					>
						<div><strong>{{ member.userName || member.username }}</strong><small>{{ member.jobTitle || '未设置岗位' }}<template v-if="member.managerName"> · 主管：{{ member.managerName }}</template></small></div>
						<div class="staff-row__actions">
							<span
								v-if="member.admin"
								class="status-pill"
							>管理员</span><span
								class="status-pill"
								:class="[member.active ? 'status-completed' : 'status-cancelled']"
							>{{ member.active ? '在岗' : '停用' }}</span><XButton
								variant="secondary"
								size="small"
								@click="toggleMemberActive(member)"
							>
								{{ member.active ? '停用' : '启用' }}
							</XButton>
						</div>
					</div>
					<form
						class="retail-form divided-form"
						@submit.prevent="createMember"
					>
						<h3>添加员工</h3>
						<div class="form-grid">
							<label>Vikunja 用户名<input
								v-model.trim="staffForm.username"
								required
							></label><label>岗位<input
								v-model.trim="staffForm.jobTitle"
								maxlength="100"
								placeholder="店长 / 导购 / 库管"
							></label>
						</div>
						<label>直属主管<select v-model.number="staffForm.managerUserId"><option :value="0">无</option><option
							v-for="member in activeMembers"
							:key="member.id"
							:value="member.userId"
						>{{ member.userName || member.username }}</option></select></label>
						<div class="check-row">
							<label class="checkbox"><input
								v-model="staffForm.primary"
								type="checkbox"
							> 主归属组织</label><label class="checkbox"><input
								v-model="staffForm.admin"
								type="checkbox"
							> 组织管理员</label><label class="checkbox"><input
								v-model="staffForm.temporary"
								type="checkbox"
							> 临时调配</label>
						</div>
						<label v-if="staffForm.temporary">调配结束时间<input
							v-model="staffForm.endsAt"
							type="datetime-local"
							required
						></label>
						<XButton
							type="submit"
							:loading="saving"
						>
							添加员工
						</XButton>
					</form>
				</Card>

				<Card
					v-if="selectedOrgId"
					title="设置单日工时容量"
					class="retail-card staff-wide"
				>
					<form
						class="retail-form capacity-form"
						@submit.prevent="saveCapacity"
					>
						<label>员工<select
							v-model.number="capacityForm.userId"
							required
						><option
							v-for="member in activeMembers"
							:key="member.id"
							:value="member.userId"
						>{{ member.userName || member.username }}</option></select></label>
						<label>日期<input
							v-model="capacityForm.day"
							type="date"
							required
						></label>
						<label>可用分钟<input
							v-model.number="capacityForm.minutes"
							type="number"
							min="0"
							max="1440"
							required
						><small>休息日填写 0；标准 8 小时填写 480。</small></label>
						<label>原因<input
							v-model.trim="capacityForm.reason"
							maxlength="500"
							placeholder="请假 / 培训 / 临时支援"
						></label>
						<XButton
							type="submit"
							:loading="saving"
						>
							保存容量
						</XButton>
					</form>
				</Card>
			</section>

			<section
				v-if="activeTab === 'workload'"
				class="retail-section"
			>
				<div class="filter-bar">
					<label>开始日期<input
						v-model="dateFrom"
						type="date"
					></label>
					<label>结束日期<input
						v-model="dateTo"
						type="date"
					></label>
					<XButton
						variant="secondary"
						@click="loadWorkload"
					>
						刷新负荷
					</XButton>
				</div>
				<Card
					title="人员负荷（分钟）"
					class="retail-card"
				>
					<div class="table-wrap">
						<table class="retail-table">
							<thead><tr><th>日期</th><th>门店 / 人员</th><th>岗位</th><th>任务</th><th>已分配 / 容量</th><th>利用率</th></tr></thead>
							<tbody>
								<tr
									v-for="row in workload"
									:key="`${row.orgUnitId}-${row.userId}-${row.capacityDay}`"
									:class="{'is-overloaded': row.overloaded}"
								>
									<td>{{ formatDay(row.capacityDay) }}</td><td><strong>{{ row.userName }}</strong><small>{{ row.orgUnitName }}</small></td><td>{{ row.jobTitle || '—' }}</td><td>{{ row.taskCount }}</td><td>{{ row.assignedMinutes }} / {{ row.capacityMinutes }}</td><td>
										<span
											class="load-pill"
											:class="[{warning: row.warning}]"
										>{{ row.utilizationPercent }}%</span>
									</td>
								</tr>
							</tbody>
						</table>
					</div>
				</Card>
			</section>

			<section
				v-if="activeTab === 'templates'"
				class="retail-section template-layout"
			>
				<Card
					title="新建标准任务模板"
					class="retail-card"
				>
					<form
						class="retail-form"
						@submit.prevent="createTemplate"
					>
						<label>模板名称<input
							v-model.trim="templateForm.name"
							required
							maxlength="250"
						></label>
						<label>任务标题<input
							v-model.trim="templateForm.title"
							required
							maxlength="500"
						></label>
						<div class="form-grid">
							<label>类别<select v-model="templateForm.category"><option
								v-for="item in categories"
								:key="item.id"
								:value="item.id"
							>{{ item.label }}</option></select></label><label>预计分钟<input
								v-model.number="templateForm.estimatedMinutes"
								type="number"
								min="0"
								max="1440"
							></label>
						</div>
						<label>验收清单（每行一项）<textarea
							v-model="templateForm.checklistText"
							rows="4"
							placeholder="检查样床整洁&#10;核对价签&#10;打开灯光"
						/></label>
						<label class="checkbox"><input
							v-model="templateForm.evidenceRequired"
							type="checkbox"
						> 必须上传照片或附件凭证</label>
						<XButton
							type="submit"
							:loading="saving"
						>
							保存模板
						</XButton>
					</form>
				</Card>
				<Card
					title="模板库与快速派发"
					class="retail-card"
				>
					<div
						v-if="templates.length === 0"
						class="retail-empty"
					>
						尚未创建模板。
					</div>
					<div
						v-for="template in templates"
						v-else
						:key="template.id"
						class="template-row"
					>
						<div><strong>{{ template.name }}</strong><small>v{{ template.currentVersion }} · {{ categoryLabel(template.category) }} · {{ template.estimatedMinutes }} 分钟</small></div>
						<XButton
							variant="secondary"
							size="small"
							@click="prepareDispatch(template)"
						>
							派发
						</XButton>
					</div>
				</Card>
				<Card
					title="自动派发计划"
					class="retail-card template-wide"
				>
					<div
						v-if="schedules.length === 0"
						class="retail-empty"
					>
						暂无自动计划。先选择模板“派发”，再设置重复规则。
					</div>
					<div
						v-for="schedule in schedules"
						v-else
						:key="schedule.id"
						class="queue-row"
					>
						<div><strong>{{ schedule.templateName }}</strong><small>{{ schedule.targetOrgUnitName }} · 每 {{ schedule.interval }} {{ frequencyLabel(schedule.frequency) }}</small></div><time>{{ formatDateTime(schedule.nextRunAt) }}</time>
					</div>
				</Card>
			</section>
		</template>
		<p
			v-else-if="!loading"
			class="retail-empty"
		>
			尚无可访问的零售组织，请先由管理员创建公司或门店。
		</p>

		<Modal
			:enabled="workflowOpen"
			:overflow="true"
			@close="closeWorkflow"
		>
			<Card
				v-if="workflow"
				:title="workflow.profile.taskTitle"
				show-close
				class="workflow-card"
				@close="closeWorkflow"
			>
				<div class="workflow-meta">
					<span
						class="status-pill"
						:class="[`status-${workflow.profile.status}`]"
					>{{ statusLabel(workflow.profile.status) }}</span><span>{{ workflow.profile.orgUnitName }}</span><span>{{ workflow.profile.estimatedMinutes }} 分钟</span>
				</div>
				<XButton
					v-if="['assigned', 'rejected'].includes(workflow.profile.status)"
					class="mbe-3"
					@click="startSelectedTask"
				>
					开始执行
				</XButton>
				<div
					v-if="workflow.checklist.length"
					class="workflow-list"
				>
					<h3>验收清单</h3><label
						v-for="item in workflow.checklist"
						:key="item.id"
					><input
						type="checkbox"
						:checked="item.done"
						:disabled="workflow.profile.status !== 'in_progress'"
						@change="toggleChecklist(item, $event)"
					><span>{{ item.title }}<em v-if="item.required">必做</em></span></label>
				</div>
				<div
					v-if="workflow.profile.status === 'in_progress'"
					class="retail-form workflow-submit"
				>
					<label>完成说明<textarea
						v-model="submissionNote"
						rows="3"
					/></label><label>凭证附件 ID（逗号分隔）<input
						v-model="evidenceIds"
						placeholder="例如 12, 13"
					><small>请先在任务详情上传照片，再填写显示的附件 ID。</small></label><div class="form-actions">
						<RouterLink :to="{name: 'task.detail', params: {id: workflow.profile.taskId}}">
							打开任务详情上传附件
						</RouterLink><XButton @click="submitSelectedTask">
							提交完成
						</XButton>
					</div>
				</div>
				<div
					v-if="workflow.profile.status === 'pending_review'"
					class="retail-form workflow-submit"
				>
					<label>复核意见<textarea
						v-model="reviewComment"
						rows="3"
						placeholder="驳回时必须填写原因"
					/></label><div class="form-actions">
						<XButton
							variant="secondary"
							@click="reviewSelectedTask('rejected')"
						>
							驳回返工
						</XButton><XButton @click="reviewSelectedTask('approved')">
							通过
						</XButton>
					</div>
				</div>
				<div
					v-if="workflow.transitions.length"
					class="timeline"
				>
					<h3>流转记录</h3><div
						v-for="event in workflow.transitions"
						:key="event.id"
					>
						<time>{{ formatDateTime(event.created) }}</time><span>{{ statusLabel(event.from) }} → {{ statusLabel(event.to) }}<small v-if="event.reason">{{ event.reason }}</small></span>
					</div>
				</div>
			</Card>
		</Modal>

		<Modal
			:enabled="dispatchOpen"
			:overflow="true"
			@close="dispatchOpen = false"
		>
			<Card
				v-if="dispatchTemplate"
				:title="`派发：${dispatchTemplate.name}`"
				show-close
				@close="dispatchOpen = false"
			>
				<form
					class="retail-form"
					@submit.prevent="dispatchNow"
				>
					<label>目标组织<select
						v-model.number="dispatchForm.orgUnitId"
						@change="loadMembers"
					><option
						v-for="org in orgUnits"
						:key="org.id"
						:value="org.id"
					>{{ org.name }}</option></select></label>
					<label>Vikunja 项目 ID<input
						v-model.number="dispatchForm.projectId"
						type="number"
						min="1"
						required
					><small>选择专用于该门店的项目 ID。</small></label>
					<div class="form-grid">
						<label>主负责人<select
							v-model.number="dispatchForm.assigneeId"
							required
						><option
							v-for="member in members"
							:key="member.id"
							:value="member.userId"
						>{{ member.userName || member.username }} · {{ member.jobTitle }}</option></select></label><label>复核人<select v-model.number="dispatchForm.reviewerId"><option :value="0">无需复核</option><option
							v-for="member in adminMembers"
							:key="member.id"
							:value="member.userId"
						>{{ member.userName || member.username }}</option></select></label>
					</div>
					<div class="form-grid">
						<label>开始时间<input
							v-model="dispatchForm.scheduledFor"
							type="datetime-local"
							required
						></label><label>截止时间<input
							v-model="dispatchForm.dueDate"
							type="datetime-local"
							required
						></label>
					</div>
					<label class="checkbox"><input
						v-model="dispatchForm.recurring"
						type="checkbox"
					> 同时建立自动重复计划</label>
					<div
						v-if="dispatchForm.recurring"
						class="form-grid"
					>
						<label>频率<select v-model="dispatchForm.frequency"><option value="daily">每天</option><option value="weekly">每周</option><option value="monthly">每月</option></select></label><label>间隔<input
							v-model.number="dispatchForm.interval"
							type="number"
							min="1"
							max="365"
						></label>
					</div>
					<XButton
						type="submit"
						:loading="saving"
					>
						确认派发
					</XButton>
				</form>
			</Card>
		</Modal>
	</div>
</template>

<script setup lang="ts">
import {computed, onMounted, reactive, ref} from 'vue'
import type {IconProp} from '@fortawesome/fontawesome-svg-core'
import {useTitle} from '@/composables/useTitle'
import {useAuthStore} from '@/stores/auth'
import {error, success} from '@/message'
import {retailService, type RetailCategory, type RetailChecklistItem, type RetailDashboard, type RetailMembership, type RetailOrgUnit, type RetailTaskProfile, type RetailTemplate, type RetailTemplateSchedule, type RetailWorkflow, type RetailWorkload} from '@/services/retail'

type Tab = 'overview' | 'tasks' | 'staff' | 'workload' | 'templates'

useTitle(() => '零售任务运营')
const authStore = useAuthStore()
const tabs: Array<{id: Tab, label: string, icon: IconProp}> = [
	{id: 'overview', label: '运营总览', icon: 'tachometer-alt'},
	{id: 'tasks', label: '执行与复核', icon: 'list-check'},
	{id: 'staff', label: '组织与人员', icon: 'user-edit'},
	{id: 'workload', label: '人员负荷', icon: 'users'},
	{id: 'templates', label: '模板派发', icon: 'copy'},
]
const categories: Array<{id: RetailCategory, label: string}> = [
	{id: 'opening', label: '开店'}, {id: 'closing', label: '闭店'}, {id: 'display', label: '陈列'}, {id: 'inventory', label: '盘点'},
	{id: 'customer_followup', label: '客户回访'}, {id: 'delivery', label: '配送'}, {id: 'after_sales', label: '售后'}, {id: 'other', label: '其他'},
]
const activeTab = ref<Tab>('overview')
const loading = ref(true)
const saving = ref(false)
const orgUnits = ref<RetailOrgUnit[]>([])
const selectedOrgId = ref(0)
const dashboard = ref<RetailDashboard | null>(null)
const profiles = ref<RetailTaskProfile[]>([])
const workload = ref<RetailWorkload[]>([])
const templates = ref<RetailTemplate[]>([])
const schedules = ref<RetailTemplateSchedule[]>([])
const members = ref<RetailMembership[]>([])
const dateFrom = ref(toDateInput(new Date()))
const weekEnd = new Date()
weekEnd.setDate(weekEnd.getDate() + 6)
const dateTo = ref(toDateInput(weekEnd))

const currentUserId = computed(() => Number(authStore.info?.id ?? 0))
const myTasks = computed(() => profiles.value.filter(task => task.primaryAssigneeId === currentUserId.value && !['completed', 'cancelled'].includes(task.status)))
const reviewTasks = computed(() => profiles.value.filter(task => task.status === 'pending_review' && task.reviewerId === currentUserId.value))
const adminMembers = computed(() => members.value.filter(member => member.admin))
const activeMembers = computed(() => members.value.filter(member => member.active))
const parentOrgOptions = computed(() => orgUnits.value.filter((org) => {
	if (orgForm.type === 'region') return org.type === 'company'
	if (orgForm.type === 'store') return org.type === 'region'
	if (orgForm.type === 'warehouse') return org.type === 'company' || org.type === 'region'
	return false
}))

const workflowOpen = ref(false)
const workflow = ref<RetailWorkflow | null>(null)
const submissionNote = ref('')
const evidenceIds = ref('')
const reviewComment = ref('')

const orgForm = reactive({type: 'company' as RetailOrgUnit['type'], parentId: 0, name: '', code: ''})
const staffForm = reactive({username: '', jobTitle: '', managerUserId: 0, admin: false, primary: true, temporary: false, endsAt: ''})
const capacityForm = reactive({userId: 0, day: toDateInput(new Date()), minutes: 480, reason: ''})
const templateForm = reactive({name: '', title: '', category: 'opening' as RetailCategory, estimatedMinutes: 30, evidenceRequired: false, checklistText: ''})
const dispatchOpen = ref(false)
const dispatchTemplate = ref<RetailTemplate | null>(null)
const dispatchForm = reactive({orgUnitId: 0, projectId: 1, assigneeId: 0, reviewerId: 0, scheduledFor: toDateTimeInput(new Date()), dueDate: toDateTimeInput(new Date(Date.now() + 2 * 60 * 60 * 1000)), recurring: false, frequency: 'daily' as 'daily' | 'weekly' | 'monthly', interval: 1})

onMounted(async () => {
	try {
		orgUnits.value = await retailService.getOrgUnits()
		selectedOrgId.value = orgUnits.value[0]?.id ?? 0
		if (!selectedOrgId.value) activeTab.value = 'staff'
		await loadCurrentTab()
	} catch (e) {
		error(e)
	} finally {
		loading.value = false
	}
})

async function selectTab(tab: Tab) {
	activeTab.value = tab
	await loadCurrentTab()
}

async function loadCurrentTab() {
	if (!selectedOrgId.value) return
	loading.value = true
	try {
		if (activeTab.value === 'overview') dashboard.value = await retailService.getDashboard(selectedOrgId.value, dateFrom.value, dateTo.value)
		if (activeTab.value === 'tasks') profiles.value = await retailService.getProfiles({orgUnitId: selectedOrgId.value})
		if (activeTab.value === 'staff') await loadStaff()
		if (activeTab.value === 'workload') await loadWorkload()
		if (activeTab.value === 'templates') [templates.value, schedules.value] = await Promise.all([retailService.getTemplates(), retailService.getSchedules()])
	} catch (e) {
		error(e)
	} finally {
		loading.value = false
	}
}

async function selectOrganization(orgUnitId: number) {
	selectedOrgId.value = orgUnitId
	await loadCurrentTab()
}

async function loadStaff() {
	members.value = selectedOrgId.value ? await retailService.getMemberships(selectedOrgId.value) : []
	capacityForm.userId = activeMembers.value[0]?.userId ?? 0
}

function resetOrgParent() {
	orgForm.parentId = orgForm.type === 'company' ? 0 : (parentOrgOptions.value[0]?.id ?? 0)
}

async function createOrgUnit() {
	saving.value = true
	try {
		const created = await retailService.createOrgUnit({...orgForm, active: true})
		orgUnits.value = await retailService.getOrgUnits()
		selectedOrgId.value = created.id
		orgForm.name = ''
		orgForm.code = ''
		await loadStaff()
		success('组织已创建')
	} catch (e) {
		error(e)
	} finally {
		saving.value = false
	}
}

async function createMember() {
	saving.value = true
	try {
		await retailService.createMembership({
			orgUnitId: selectedOrgId.value,
			username: staffForm.username,
			jobTitle: staffForm.jobTitle,
			managerUserId: staffForm.managerUserId,
			admin: staffForm.admin,
			primary: staffForm.primary,
			temporary: staffForm.temporary,
			endsAt: staffForm.temporary ? new Date(staffForm.endsAt).toISOString() : undefined,
			active: true,
		})
		staffForm.username = ''
		staffForm.jobTitle = ''
		staffForm.managerUserId = 0
		staffForm.admin = false
		staffForm.temporary = false
		staffForm.endsAt = ''
		await loadStaff()
		success('员工已加入当前组织')
	} catch (e) {
		error(e)
	} finally {
		saving.value = false
	}
}

async function toggleMemberActive(member: RetailMembership) {
	try {
		await retailService.updateMembership(member.id, {
			jobTitle: member.jobTitle,
			managerUserId: member.managerUserId,
			admin: member.admin,
			primary: member.primary,
			temporary: member.temporary,
			startsAt: member.startsAt,
			endsAt: member.endsAt,
			active: !member.active,
		})
		await loadStaff()
		success(member.active ? '员工已停用' : '员工已启用')
	} catch (e) {
		error(e)
	}
}

async function saveCapacity() {
	saving.value = true
	try {
		await retailService.setCapacity(capacityForm.userId, selectedOrgId.value, capacityForm.day, capacityForm.minutes, capacityForm.reason)
		success('工时容量已保存')
	} catch (e) {
		error(e)
	} finally {
		saving.value = false
	}
}

async function loadWorkload() {
	workload.value = await retailService.getWorkload(selectedOrgId.value, dateFrom.value, dateTo.value)
}

async function openWorkflow(taskId: number) {
	try {
		workflow.value = await retailService.getWorkflow(taskId)
		workflowOpen.value = true
	} catch (e) { error(e) }
}

function closeWorkflow() { workflowOpen.value = false; workflow.value = null }

async function startSelectedTask() {
	if (!workflow.value) return
	try { workflow.value = await retailService.startTask(workflow.value.profile.taskId); await refreshTasks() } catch (e) { error(e) }
}

async function toggleChecklist(item: RetailChecklistItem, event: Event) {
	try {
		const target = event.target as HTMLInputElement
		const updated = await retailService.setChecklistDone(item.id, target.checked)
		if (workflow.value) workflow.value.checklist = workflow.value.checklist.map(value => value.id === updated.id ? updated : value)
	} catch (e) { error(e); await openWorkflow(workflow.value?.profile.taskId ?? 0) }
}

async function submitSelectedTask() {
	if (!workflow.value) return
	const ids = evidenceIds.value.split(',').map(value => Number(value.trim())).filter(value => value > 0)
	try { workflow.value = await retailService.submitTask(workflow.value.profile.taskId, submissionNote.value, ids); submissionNote.value = ''; evidenceIds.value = ''; success('任务已提交'); await refreshTasks() } catch (e) { error(e) }
}

async function reviewSelectedTask(decision: 'approved' | 'rejected') {
	if (!workflow.value) return
	const submission = workflow.value.submissions[workflow.value.submissions.length - 1]
	if (!submission) return
	try { workflow.value = await retailService.reviewTask(workflow.value.profile.taskId, submission.id, decision, reviewComment.value); reviewComment.value = ''; success(decision === 'approved' ? '复核已通过' : '已驳回返工'); await refreshTasks() } catch (e) { error(e) }
}

async function refreshTasks() {
	profiles.value = await retailService.getProfiles({orgUnitId: selectedOrgId.value})
}

async function createTemplate() {
	saving.value = true
	try {
		await retailService.createTemplate({orgUnitId: selectedOrgId.value, name: templateForm.name, title: templateForm.title, description: '', category: templateForm.category, estimatedMinutes: templateForm.estimatedMinutes, evidenceRequired: templateForm.evidenceRequired, active: true, checklist: templateForm.checklistText.split('\n').map(value => value.trim()).filter(Boolean).map((title, index) => ({title, required: true, position: index}))})
		templateForm.name = ''; templateForm.title = ''; templateForm.checklistText = ''
		templates.value = await retailService.getTemplates(); success('模板已保存')
	} catch (e) { error(e) } finally { saving.value = false }
}

async function prepareDispatch(template: RetailTemplate) {
	dispatchTemplate.value = template
	dispatchForm.orgUnitId = selectedOrgId.value
	dispatchOpen.value = true
	await loadMembers()
}

async function loadMembers() {
	members.value = await retailService.getMemberships(dispatchForm.orgUnitId)
	dispatchForm.assigneeId = members.value.find(member => !member.admin)?.userId ?? members.value[0]?.userId ?? 0
	dispatchForm.reviewerId = adminMembers.value[0]?.userId ?? 0
}

async function dispatchNow() {
	if (!dispatchTemplate.value) return
	saving.value = true
	try {
		const scheduledFor = new Date(dispatchForm.scheduledFor).toISOString()
		const dueDate = new Date(dispatchForm.dueDate).toISOString()
		await retailService.dispatchTemplate(dispatchTemplate.value.id, {targetOrgUnitId: dispatchForm.orgUnitId, projectId: dispatchForm.projectId, primaryAssigneeId: dispatchForm.assigneeId, reviewerId: dispatchForm.reviewerId, scheduledFor, dueDate})
		if (dispatchForm.recurring) {
			await retailService.createSchedule({templateId: dispatchTemplate.value.id, targetOrgUnitId: dispatchForm.orgUnitId, projectId: dispatchForm.projectId, primaryAssigneeId: dispatchForm.assigneeId, reviewerId: dispatchForm.reviewerId, frequency: dispatchForm.frequency, interval: dispatchForm.interval, dueOffsetMinutes: Math.max(0, Math.round((new Date(dueDate).getTime() - new Date(scheduledFor).getTime()) / 60000)), nextRunAt: scheduledFor, active: true})
		}
		dispatchOpen.value = false; success('任务已派发'); schedules.value = await retailService.getSchedules()
	} catch (e) { error(e) } finally { saving.value = false }
}

function toDateInput(date: Date) { return date.toISOString().slice(0, 10) }
function toDateTimeInput(date: Date) { const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000); return local.toISOString().slice(0, 16) }
function formatDay(value: string) { return new Intl.DateTimeFormat('zh-CN', {month: '2-digit', day: '2-digit'}).format(new Date(value)) }
function formatDateTime(value: string) { return new Intl.DateTimeFormat('zh-CN', {month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit'}).format(new Date(value)) }
function statusLabel(status: string) { return ({draft: '草稿', assigned: '待开始', in_progress: '执行中', pending_review: '待复核', rejected: '已驳回', completed: '已完成', cancelled: '已取消'}[status] ?? status) }
function categoryLabel(category: RetailCategory) { return categories.find(item => item.id === category)?.label ?? category }
function frequencyLabel(frequency: string) { return ({daily: '天', weekly: '周', monthly: '月'}[frequency] ?? frequency) }
function orgTypeLabel(type: RetailOrgUnit['type']) { return ({company: '公司', region: '区域', store: '门店', warehouse: '仓库'}[type]) }
</script>

<style scoped lang="scss">
.retail-operations {
	max-inline-size: 1280px;
	margin: 0 auto;
	padding: 1rem;
}

.retail-header {
	display: flex;
	align-items: end;
	justify-content: space-between;
	gap: 1.5rem;
	padding: 1.5rem;
	border-radius: 1.25rem;
	color: #ffffff;
	background: linear-gradient(135deg, #173f3a, #267065 60%, #d89555 145%);
	box-shadow: 0 18px 50px rgb(23 63 58 / 18%);
}

.retail-header h1 {
	margin: .15rem 0 .25rem;
	color: inherit;
	font-size: clamp(1.65rem, 4vw, 2.45rem);
}

.retail-eyebrow {
	margin: 0;
	color: #f4c98e;
	font-weight: 700;
	letter-spacing: .16em;
}

.retail-subtitle {
	margin: 0;
	color: rgb(255 255 255 / 78%);
}

.retail-org-picker {
	display: grid;
	gap: .35rem;
	min-inline-size: 230px;
	font-size: .8rem;
	font-weight: 700;
}

select,
input,
textarea {
	inline-size: 100%;
	border: 1px solid var(--grey-300);
	border-radius: .55rem;
	padding: .65rem .75rem;
	color: var(--text);
	background: var(--white);
}

.retail-org-picker select {
	border-color: rgb(255 255 255 / 24%);
	color: #173f3a;
}

.retail-tabs {
	display: flex;
	gap: .5rem;
	padding: 1rem 0;
	overflow-x: auto;
}

.retail-tabs button {
	border: 0;
	border-radius: 999px;
	padding: .7rem 1rem;
	white-space: nowrap;
	color: var(--text);
	background: var(--white);
	box-shadow: var(--shadow-sm);
	cursor: pointer;
}

.retail-tabs button.is-active {
	color: #ffffff;
	background: #267065;
}

.retail-section,
.retail-form {
	display: grid;
	gap: 1rem;
}

.metric-grid {
	display: grid;
	grid-template-columns: repeat(6, minmax(0, 1fr));
	gap: .75rem;
}

.metric {
	padding: 1rem;
	border: 1px solid var(--grey-200);
	border-radius: .9rem;
	background: var(--white);
}

.metric span,
.queue-row small,
.template-row small,
.task-row small,
.retail-table small,
label small {
	display: block;
	color: var(--grey-600);
	font-size: .82rem;
}

.metric strong {
	display: block;
	margin-block-start: .2rem;
	font-size: 1.7rem;
}

.metric--primary strong { color: #267065; }
.metric--success strong { color: #23845d; }
.metric--warning strong { color: #bd6d17; }
.metric--danger strong { color: #bd3c3c; }

.retail-card { overflow: hidden; }

.queue-row,
.template-row {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 1rem;
	padding: .8rem 0;
	border-block-end: 1px solid var(--grey-200);
}

.queue-row:last-child,
.template-row:last-child {
	border-block-end: 0;
}

.queue-row__actions {
	display: flex;
	align-items: center;
	gap: .75rem;
}

.task-columns,
.template-layout,
.staff-layout {
	grid-template-columns: repeat(2, minmax(0, 1fr));
	align-items: start;
}

.staff-wide {
	grid-column: 1 / -1;
}

.org-list {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: .5rem;
}

.org-list button {
	display: grid;
	gap: .2rem;
	border: 1px solid var(--grey-200);
	border-radius: .65rem;
	padding: .7rem;
	text-align: start;
	background: var(--white);
	cursor: pointer;
}

.org-list button.is-selected {
	border-color: #267065;
	box-shadow: 0 0 0 2px rgb(38 112 101 / 12%);
}

.staff-row {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 1rem;
	padding: .8rem 0;
	border-block-end: 1px solid var(--grey-200);
}

.staff-row__actions,
.check-row {
	display: flex;
	align-items: center;
	gap: .55rem;
}

.divided-form {
	margin-block-start: 1rem;
	padding-block-start: 1rem;
	border-block-start: 1px solid var(--grey-200);
}

.divided-form h3 {
	margin: 0;
}

.capacity-form {
	grid-template-columns: repeat(5, minmax(0, 1fr));
	align-items: end;
}

.task-row {
	display: flex;
	inline-size: 100%;
	align-items: center;
	justify-content: space-between;
	gap: 1rem;
	border: 0;
	border-block-end: 1px solid var(--grey-200);
	padding: .9rem 0;
	text-align: start;
	background: transparent;
	cursor: pointer;
}

.status-pill,
.load-pill {
	display: inline-flex;
	border-radius: 999px;
	padding: .25rem .55rem;
	color: #31534f;
	background: #e5f2ef;
	font-size: .75rem;
	font-weight: 700;
	white-space: nowrap;
}

.status-rejected,
.status-cancelled,
.load-pill.warning {
	color: #8b3030;
	background: #f9e5e5;
}

.status-pending_review {
	color: #8a5d18;
	background: #faedce;
}

.status-completed {
	color: #1f6b4c;
	background: #dbf1e5;
}

.filter-bar,
.form-grid,
.form-actions {
	display: flex;
	align-items: end;
	gap: .75rem;
}

.filter-bar label,
.form-grid label {
	flex: 1;
}

.retail-form label {
	display: grid;
	gap: .35rem;
	font-weight: 600;
}

.retail-form .checkbox {
	display: flex;
	align-items: center;
	font-weight: 400;
}

.checkbox input { inline-size: auto; }
.template-wide { grid-column: 1 / -1; }
.table-wrap { overflow-x: auto; }

.retail-table {
	inline-size: 100%;
	border-collapse: collapse;
}

.retail-table th,
.retail-table td {
	padding: .75rem;
	border-block-end: 1px solid var(--grey-200);
	text-align: start;
}

.retail-table tr.is-overloaded { background: rgb(189 60 60 / 7%); }

.retail-empty {
	padding: 2rem;
	color: var(--grey-600);
	text-align: center;
}

.workflow-card { min-inline-size: min(720px, 90vw); }

.workflow-meta {
	display: flex;
	flex-wrap: wrap;
	gap: .6rem;
	margin-block-end: 1rem;
}

.workflow-list {
	display: grid;
	gap: .55rem;
	margin-block-end: 1rem;
}

.workflow-list label {
	display: flex;
	gap: .65rem;
	align-items: center;
}

.workflow-list input { inline-size: auto; }

.workflow-list em {
	margin-inline-start: .45rem;
	color: #b24a38;
	font-size: .72rem;
	font-style: normal;
}

.workflow-submit {
	padding: 1rem;
	border-radius: .75rem;
	background: var(--grey-100);
}

.form-actions { justify-content: flex-end; }
.timeline { margin-block-start: 1.25rem; }

.timeline > div {
	display: grid;
	grid-template-columns: 110px 1fr;
	gap: .75rem;
	padding: .45rem 0;
	border-block-start: 1px solid var(--grey-200);
}

.timeline time {
	color: var(--grey-600);
	font-size: .8rem;
}

.timeline small { display: block; }

@media (width <= 900px) {
	.metric-grid { grid-template-columns: repeat(3, 1fr); }

	.task-columns,
	.template-layout,
	.staff-layout {
		grid-template-columns: 1fr;
	}

	.template-wide,
	.staff-wide {
		grid-column: auto;
	}

	.capacity-form {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}
}

@media (width <= 600px) {
	.retail-operations { padding: .5rem; }

	.retail-header {
		align-items: stretch;
		flex-direction: column;
		padding: 1.15rem;
	}

	.retail-org-picker { min-inline-size: 0; }
	.metric-grid { grid-template-columns: repeat(2, 1fr); }

	.filter-bar,
	.form-grid,
	.form-actions,
	.check-row {
		align-items: stretch;
		flex-direction: column;
	}

	.org-list,
	.capacity-form {
		grid-template-columns: 1fr;
	}

	.queue-row {
		align-items: flex-start;
		flex-direction: column;
	}

	.staff-row {
		align-items: flex-start;
		flex-direction: column;
	}

	.queue-row__actions {
		inline-size: 100%;
		justify-content: space-between;
	}
}
</style>
