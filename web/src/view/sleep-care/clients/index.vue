<template>
  <div>
    <div class="mb-4 rounded-lg border border-border bg-container px-4 py-3">
      <div class="font-semibold">康养用户管理</div>
      <div class="mt-1 text-sm">
        在这里维护用户资料、责任人和服务计划。页面只记录服务流程，不提供诊断或治疗建议。
      </div>
    </div>

    <div class="gva-search-box">
      <el-form :inline="true" :model="searchForm">
        <el-form-item label="编号或姓名">
          <el-input v-model="searchForm.keyword" clearable placeholder="请输入用户编号或姓名" @keyup.enter="search" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchForm.status" clearable class="w-36">
            <el-option label="服务中" value="ACTIVE" />
            <el-option label="已停用" value="INACTIVE" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="search">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button v-if="btnAuth.createClient" type="primary" @click="openCreate">新建康养用户</el-button>
      </div>
      <el-table v-loading="loading" :data="tableData" row-key="id" empty-text="当前责任或组织范围内暂无康养用户">
        <el-table-column prop="displayCode" label="用户编号" min-width="150" />
        <el-table-column prop="displayName" label="姓名" min-width="150" />
        <el-table-column prop="organizationName" label="机构" min-width="170" />
        <el-table-column prop="teamName" label="团队" min-width="150" />
        <el-table-column prop="contactMobile" label="联系电话" min-width="130" />
        <el-table-column label="责任关系" min-width="220">
          <template #default="scope">
            <div v-if="scope.row.currentAssignments?.length" class="flex flex-col gap-1">
              <span v-for="item in scope.row.currentAssignments" :key="item.id">
                {{ roleLabel(item.roleType) }}：{{ item.assigneeDisplayName }}
              </span>
            </div>
            <span v-else class="text-gray-400">未分配</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.status === 'ACTIVE' ? 'success' : 'info'">
              {{ scope.row.status === 'ACTIVE' ? '服务中' : '已停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" min-width="290">
          <template #default="scope">
            <el-button v-if="btnAuth.viewDetail" link type="primary" @click="openDetail(scope.row.id)">详情</el-button>
            <el-button v-if="btnAuth.maintainClient" link type="primary" @click="openEdit(scope.row)">维护</el-button>
            <el-button v-if="btnAuth.assignCare" link type="primary" @click="openAssignment(scope.row.id)">分配责任</el-button>
            <el-button v-if="btnAuth.recordConsent" link type="primary" @click="openConsent(scope.row.id)">参与状态</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="gva-pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[10, 30, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="loadTable"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <el-drawer v-model="detailVisible" title="康养用户与计划时间线" size="900px">
      <template v-if="detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="用户编号">{{ detail.displayCode }}</el-descriptions-item>
          <el-descriptions-item label="姓名">{{ detail.displayName }}</el-descriptions-item>
          <el-descriptions-item label="联系电话">{{ detail.contactMobile || '-' }}</el-descriptions-item>
          <el-descriptions-item label="服务项目">{{ detail.servicePackageCode || '-' }}</el-descriptions-item>
          <el-descriptions-item label="机构">{{ detail.organizationName }}</el-descriptions-item>
          <el-descriptions-item label="团队">{{ detail.teamName || '-' }}</el-descriptions-item>
          <el-descriptions-item label="服务说明" :span="2">{{ detail.serviceReason || '-' }}</el-descriptions-item>
        </el-descriptions>

        <h3 class="mt-6 mb-2 font-semibold">责任关系历史</h3>
        <el-table :data="detail.assignments" size="small" empty-text="暂无责任关系">
          <el-table-column label="角色" width="110"><template #default="scope">{{ roleLabel(scope.row.roleType) }}</template></el-table-column>
          <el-table-column prop="assigneeDisplayName" label="责任人" min-width="120" />
          <el-table-column prop="teamName" label="团队" min-width="130" />
          <el-table-column label="状态" width="90"><template #default="scope">{{ assignmentStatusLabel(scope.row.status) }}</template></el-table-column>
          <el-table-column label="生效时间" min-width="170"><template #default="scope">{{ formatDateTime(scope.row.validFrom) }}</template></el-table-column>
        </el-table>

        <h3 class="mt-6 mb-2 font-semibold">参与状态记录</h3>
        <el-table :data="detail.consentRecords" size="small" empty-text="暂无参与状态记录">
          <el-table-column label="状态变化" width="110"><template #default="scope">{{ scope.row.action === 'GRANT' ? '确认参与' : '停止参与' }}</template></el-table-column>
          <el-table-column label="发生时间" min-width="170"><template #default="scope">{{ formatDateTime(scope.row.occurredAt) }}</template></el-table-column>
          <el-table-column prop="reason" label="备注" min-width="180" />
        </el-table>

        <DataGovernancePanel
          v-if="btnAuth.recordLifecycleRequest"
          :client-id="detail.id"
          :client-version="detail.version"
          :can-record="Boolean(btnAuth.recordLifecycleRequest)"
          @recorded="handleLifecycleRecorded"
        />

        <div class="mb-3 mt-7 flex items-end justify-between gap-4">
          <div>
            <h3 class="text-lg font-semibold">服务计划时间线</h3>
          </div>
          <el-button
            v-if="btnAuth.startPlan && !planLoading && !activePlan"
            type="primary"
            @click="openPlanStart"
          >
            预览并启动计划
          </el-button>
        </div>
        <el-alert
          class="mb-3"
          type="warning"
          :closable="false"
          title="系统目前不会自动发送通知。暂停或恢复计划时，原定任务时间不会改变。"
        />
        <div
          v-loading="planLoading"
          class="min-h-24"
        >
          <el-empty
            v-if="!planLoading && !clientPlans.length"
            description="尚未启动服务计划"
            :image-size="72"
          />
          <div
            v-for="plan in clientPlans"
            :key="plan.id"
            class="mb-4 overflow-hidden rounded-lg border border-slate-200"
          >
            <div class="flex flex-wrap items-center justify-between gap-3 bg-slate-900 px-4 py-3 text-white">
              <div>
                <div class="flex items-center gap-2">
                  <span class="font-semibold">{{ readablePlanTitle(plan.templateTitle) }}</span>
                  <el-tag
                    :type="plan.status === 'PAUSED' ? 'warning' : 'success'"
                    size="small"
                  >
                    {{ planStatusLabel(plan.status) }}
                  </el-tag>
                </div>
                <div class="mt-1 text-xs text-slate-400">
                  开始时间 {{ formatDateTime(plan.anchorAt) }}
                </div>
              </div>
              <div class="flex gap-2">
                <el-button
                  v-if="btnAuth.pausePlan && plan.status === 'ACTIVE'"
                  size="small"
                  :loading="planStateActionId === plan.id"
                  :disabled="planStateActionId !== 0 && planStateActionId !== plan.id"
                  @click="changePlanState(plan, 'pause')"
                >
                  暂停
                </el-button>
                <el-button
                  v-if="btnAuth.resumePlan && plan.status === 'PAUSED'"
                  size="small"
                  type="primary"
                  :loading="planStateActionId === plan.id"
                  :disabled="planStateActionId !== 0 && planStateActionId !== plan.id"
                  @click="changePlanState(plan, 'resume')"
                >
                  恢复
                </el-button>
              </div>
            </div>
            <div class="grid grid-cols-1 gap-px bg-slate-200 md:grid-cols-5">
              <div
                v-for="task in plan.tasks"
                :key="task.id"
                class="bg-white px-3 py-3"
              >
                <div class="flex items-center justify-between gap-2">
                  <span class="text-base font-semibold">{{ dayLabel(task.dayCode) }}</span>
                  <el-tag
                    :type="taskExecutionTagType(task.executionStatus)"
                    size="small"
                    effect="plain"
                  >
                    {{ taskStatusLabel(task.executionStatus) }}
                  </el-tag>
                </div>
                <div class="mt-2 min-h-10 text-xs font-medium leading-5 text-slate-700">
                  {{ readableTaskTitle(task.title, task.dayCode) }}
                </div>
                <div class="mt-2 border-t border-slate-100 pt-2 text-[11px] leading-5 text-slate-500">
                  <div>开放 {{ formatDateTime(task.openAt) }}</div>
                  <div>截止 {{ formatDateTime(task.dueAt) }}</div>
                </div>
                <div class="mt-2 flex flex-wrap gap-1">
                  <el-tag
                    :type="taskTimingTagType(task.timingStatus)"
                    size="small"
                  >
                    {{ taskStatusLabel(task.timingStatus) }}
                  </el-tag>
                  <el-tag
                    size="small"
                    type="info"
                    effect="plain"
                  >
                    {{ taskStatusLabel(task.reviewStatus) }}
                  </el-tag>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </el-drawer>

    <el-dialog v-model="clientDialogVisible" :title="editingId ? '维护康养用户' : '新建康养用户'" width="620px">
      <el-alert class="mb-4" type="info" :closable="false" title="请只填写完成服务流程所需的信息，不要在说明中填写诊断或治疗内容。" />
      <el-form :model="clientForm" label-width="110px">
        <el-form-item v-if="!editingId" label="用户编号" required>
          <el-input v-model="clientForm.displayCode" placeholder="例如 KH-003" />
        </el-form-item>
        <el-form-item label="姓名" required>
          <el-input v-model="clientForm.displayName" placeholder="请输入姓名" />
        </el-form-item>
        <el-form-item label="联系电话">
          <el-input v-model="clientForm.contactMobile" placeholder="仅用于页面展示，不会发送短信" />
        </el-form-item>
        <el-form-item label="服务项目">
          <el-input v-model="clientForm.servicePackageCode" placeholder="请输入服务项目" />
        </el-form-item>
        <el-form-item label="服务原因">
          <el-input v-model="clientForm.serviceReason" type="textarea" :rows="3" placeholder="请输入服务说明" />
        </el-form-item>
        <template v-if="!editingId">
          <el-form-item label="机构" required>
            <el-select v-model="clientForm.organizationId" class="w-full" @change="clientForm.teamId = undefined">
              <el-option v-for="item in organizationOptions" :key="item.departmentId" :label="item.name" :value="item.departmentId" />
            </el-select>
          </el-form-item>
        </template>
        <el-form-item label="团队">
          <el-select v-model="clientForm.teamId" clearable class="w-full">
            <el-option v-for="item in teamOptions" :key="item.departmentId" :label="item.name" :value="item.departmentId" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="editingId" label="状态">
          <el-radio-group v-model="clientForm.status"><el-radio value="ACTIVE">服务中</el-radio><el-radio value="INACTIVE">已停用</el-radio></el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer><el-button @click="clientDialogVisible = false">取消</el-button><el-button type="primary" @click="saveClient">保存</el-button></template>
    </el-dialog>

    <el-dialog v-model="assignmentVisible" title="记录责任关系" width="580px">
      <el-alert class="mb-4" type="info" :closable="false" title="每个康养用户同一时间最多一名管家和一名医护；转交必须明确替代旧关系。" />
      <el-form :model="assignmentForm" label-width="110px">
        <el-form-item label="责任类型" required>
          <el-select v-model="assignmentForm.roleType" class="w-full" @change="assignmentForm.assigneeId = undefined">
            <el-option label="健康管家" value="CARE_STEWARD" /><el-option label="一线医护" value="CLINICIAN" />
          </el-select>
        </el-form-item>
        <el-form-item label="责任人" required>
          <el-select v-model="assignmentForm.assigneeId" class="w-full">
            <el-option v-for="item in assignmentAssignees" :key="item.id" :label="`${item.displayName}（${teamName(item.teamId)}）`" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="生效时间" required><el-date-picker v-model="assignmentForm.validFrom" type="datetime" class="w-full" /></el-form-item>
        <el-form-item label="调整原因" required><el-input v-model="assignmentForm.reason" type="textarea" :rows="3" placeholder="请说明本次责任人调整原因" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="assignmentVisible = false">取消</el-button><el-button type="primary" @click="saveAssignment">确认记录</el-button></template>
    </el-dialog>

    <el-dialog v-model="consentVisible" title="记录参与状态" width="560px">
      <el-alert class="mb-4" type="info" :closable="false" title="请根据用户已表达的意愿如实记录；本操作不会发送通知。" />
      <el-form :model="consentForm" label-width="110px">
        <el-form-item label="参与状态" required><el-radio-group v-model="consentForm.action"><el-radio value="GRANT">确认参与</el-radio><el-radio value="WITHDRAW">停止参与</el-radio></el-radio-group></el-form-item>
        <el-form-item label="发生时间" required><el-date-picker v-model="consentForm.occurredAt" type="datetime" class="w-full" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="consentForm.reason" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="consentVisible = false">取消</el-button><el-button type="primary" @click="saveConsent">确认记录</el-button></template>
    </el-dialog>

    <el-dialog
      v-model="planStartVisible"
      title="预览并启动服务计划"
      width="820px"
      :close-on-click-modal="false"
    >
      <el-alert
        class="mb-4"
        type="warning"
        :closable="false"
        title="请先选择计划开始时间。启动后会生成五次任务，系统目前不会自动发送通知。"
      />
      <el-form
        :model="planStartForm"
        label-width="130px"
      >
        <el-form-item label="计划模板" required>
          <el-select
            v-model="planStartForm.planTemplateVersionId"
            class="w-full"
            @change="resetPlanPreview"
          >
            <el-option
              v-for="item in planVersionOptions"
              :key="item.id"
              :label="readablePlanTitle(item.title)"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="计划开始时间" required>
          <el-date-picker
            v-model="planStartForm.anchorAt"
            type="datetime"
            class="w-full"
            placeholder="请选择计划开始时间"
            @change="resetPlanPreview"
          />
        </el-form-item>
      </el-form>

      <div
        v-if="planPreview"
        class="mt-5"
      >
        <div class="mb-3 flex items-center justify-between">
          <div>
            <div class="font-semibold">五次任务时间预览</div>
          </div>
          <div class="text-right font-mono text-xs text-gray-400">
            预览有效至 {{ formatDateTime(planPreview.expiresAt) }}
          </div>
        </div>
        <div class="grid grid-cols-1 gap-2 md:grid-cols-5">
          <div
            v-for="task in planPreview.tasks"
            :key="task.id"
            class="rounded-lg border border-slate-200 bg-slate-50 px-3 py-3"
          >
            <div class="text-base font-semibold">{{ dayLabel(task.dayCode) }}</div>
            <div class="mt-2 min-h-10 text-xs leading-5">{{ readableTaskTitle(task.title, task.dayCode) }}</div>
            <div class="mt-2 border-t border-slate-200 pt-2 text-[11px] leading-5 text-slate-500">
              <div>开放 {{ formatDateTime(task.openAt) }}</div>
              <div>截止 {{ formatDateTime(task.dueAt) }}</div>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="planStartVisible = false">取消</el-button>
        <el-button
          :loading="planActionLoading"
          @click="previewPlan"
        >
          生成预览
        </el-button>
        <el-button
          type="primary"
          :disabled="!planPreview"
          :loading="planActionLoading"
          @click="confirmStartPlan"
        >
          确认启动
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
  import { computed, onMounted, reactive, ref } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { useBtnAuth } from '@/utils/btnAuth'
  import { readablePlanTitle, readableTaskTitle } from '@/utils/sleep-care-display'
  import DataGovernancePanel from './components/DataGovernancePanel.vue'
  import {
    createCareAssignment,
    createCareClient,
    createCareConsent,
    getCareClient,
    getCareClientOptions,
    getCareClients,
    updateCareClient
  } from '@/api/sleep-care/care-clients'
  import {
    getClientPlans,
    getPlanVersions,
    pauseCarePlan,
    previewCarePlan,
    resumeCarePlan,
    startCarePlan
  } from '@/api/sleep-care/care-path'

  defineOptions({ name: 'CareClients' })

  const props = defineProps({
    initialDetailId: {
      type: [String, Number],
      default: ''
    }
  })

  const btnAuth = useBtnAuth()
  const searchForm = reactive({ keyword: '', status: '' })
  const page = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const tableData = ref([])
  const loading = ref(false)
  const detail = ref(null)
  const detailVisible = ref(false)
  const options = ref({ orgUnits: [], assignees: [] })
  const optionsLoaded = ref(false)
  const editingId = ref(0)
  const editingVersion = ref(0)
  const clientDialogVisible = ref(false)
  const assignmentVisible = ref(false)
  const consentVisible = ref(false)
  const actionClient = ref(null)
  const clientPlans = ref([])
  const planLoading = ref(false)
  const planStartVisible = ref(false)
  const planVersionOptions = ref([])
  const planPreview = ref(null)
  const planActionLoading = ref(false)
  const planStateActionId = ref(0)
  const planPreviewKey = ref('')
  const planStartKey = ref('')

  const emptyClientForm = () => ({
    displayCode: '',
    displayName: '',
    contactMobile: '',
    serviceReason: '',
    servicePackageCode: '',
    organizationId: undefined,
    teamId: undefined,
    status: 'ACTIVE'
  })
  const clientForm = reactive(emptyClientForm())
  const assignmentForm = reactive({ roleType: 'CARE_STEWARD', assigneeId: undefined, validFrom: new Date(), reason: '' })
  const consentForm = reactive({ action: 'GRANT', textVersion: 'SYNTHETIC-V1', occurredAt: new Date(), reason: '' })
  const planStartForm = reactive({ planTemplateVersionId: undefined, anchorAt: undefined })

  const organizationOptions = computed(() => options.value.orgUnits.filter((item) => item.unitType === 'ORGANIZATION'))
  const teamOptions = computed(() => options.value.orgUnits.filter((item) => item.unitType === 'TEAM' && (!clientForm.organizationId || item.organizationId === clientForm.organizationId)))
  const assignmentAssignees = computed(() => options.value.assignees.filter((item) => item.roleType === assignmentForm.roleType && item.teamId === actionClient.value?.teamId))
  const activePlan = computed(() => clientPlans.value.find((item) => ['ACTIVE', 'PAUSED'].includes(item.status)))

  const loadTable = async () => {
    loading.value = true
    try {
      const res = await getCareClients({ page: page.value, pageSize: pageSize.value, ...searchForm })
      if (res.code === 0) {
        tableData.value = res.data.list || []
        total.value = res.data.total
      }
    } finally {
      loading.value = false
    }
  }
  const search = () => { page.value = 1; loadTable() }
  const resetSearch = () => { searchForm.keyword = ''; searchForm.status = ''; search() }
  const handleSizeChange = () => { page.value = 1; loadTable() }
  const ensureOptions = async () => {
    if (optionsLoaded.value) return true
    const res = await getCareClientOptions()
    if (res.code !== 0) return false
    options.value = res.data
    optionsLoaded.value = true
    return true
  }
  const loadDetail = async (id) => {
    const res = await getCareClient(id)
    if (res.code !== 0) return null
    detail.value = res.data
    return res.data
  }
  const loadPlans = async (id) => {
    clientPlans.value = []
    planLoading.value = true
    try {
      const res = await getClientPlans(id)
      if (res.code === 0) {
        clientPlans.value = res.data || []
      }
    } finally {
      planLoading.value = false
    }
  }
  const openDetail = async (id) => {
    detail.value = null
    clientPlans.value = []
    if (await loadDetail(id)) {
      detailVisible.value = true
      await loadPlans(id)
    }
  }
  const openCreate = async () => {
    if (!(await ensureOptions())) return
    Object.assign(clientForm, emptyClientForm())
    editingId.value = 0
    editingVersion.value = 0
    clientDialogVisible.value = true
  }
  const openEdit = async (row) => {
    if (!(await ensureOptions())) return
    Object.assign(clientForm, { displayName: row.displayName, contactMobile: row.contactMobile, serviceReason: row.serviceReason, servicePackageCode: row.servicePackageCode, organizationId: row.organizationId, teamId: row.teamId, status: row.status })
    editingId.value = row.id
    editingVersion.value = row.version
    clientDialogVisible.value = true
  }
  const saveClient = async () => {
    if (!clientForm.displayName?.trim()) {
      ElMessage.warning('显示名称必填')
      return
    }
    let res
    if (editingId.value) {
      res = await updateCareClient(editingId.value, { expectedVersion: editingVersion.value, displayName: clientForm.displayName, contactMobile: clientForm.contactMobile, serviceReason: clientForm.serviceReason, servicePackageCode: clientForm.servicePackageCode, teamId: clientForm.teamId, status: clientForm.status })
    } else {
      if (!clientForm.displayCode || !clientForm.organizationId) return ElMessage.warning('用户编号和机构必填')
      res = await createCareClient({ ...clientForm, synthetic: true })
    }
    if (res.code === 0) { ElMessage.success(res.msg); clientDialogVisible.value = false; await loadTable() }
  }
  const openAssignment = async (id) => {
    if (!(await ensureOptions())) return
    actionClient.value = await loadDetail(id)
    if (!actionClient.value) return
    Object.assign(assignmentForm, { roleType: 'CARE_STEWARD', assigneeId: undefined, validFrom: new Date(), reason: '' })
    assignmentVisible.value = true
  }
  const saveAssignment = async () => {
    if (!assignmentForm.assigneeId || !assignmentForm.reason) return ElMessage.warning('责任人和原因必填')
    const assignee = options.value.assignees.find((item) => item.id === assignmentForm.assigneeId)
    const current = actionClient.value.assignments.find((item) => item.roleType === assignmentForm.roleType && item.status === 'ACTIVE')
    const res = await createCareAssignment(actionClient.value.id, { expectedVersion: actionClient.value.version, roleType: assignmentForm.roleType, assigneeId: assignmentForm.assigneeId, teamId: assignee.teamId, validFrom: assignmentForm.validFrom.toISOString(), replacesAssignmentId: current?.id, reason: assignmentForm.reason })
    if (res.code === 0) { ElMessage.success(res.msg); assignmentVisible.value = false; await loadTable() }
  }
  const openConsent = async (id) => {
    actionClient.value = await loadDetail(id)
    if (!actionClient.value) return
    Object.assign(consentForm, { action: actionClient.value.consentStatus === 'GRANTED' ? 'WITHDRAW' : 'GRANT', textVersion: 'SYNTHETIC-V1', occurredAt: new Date(), reason: '' })
    consentVisible.value = true
  }
  const saveConsent = async () => {
    const res = await createCareConsent(actionClient.value.id, { expectedVersion: actionClient.value.version, consentType: 'SYNTHETIC_TEST_PARTICIPATION', action: consentForm.action, textVersion: consentForm.textVersion, occurredAt: consentForm.occurredAt.toISOString(), source: 'STAFF_RECORDED', reason: consentForm.reason })
    if (res.code === 0) { ElMessage.success(res.msg); consentVisible.value = false; await loadTable() }
  }
  const handleLifecycleRecorded = async () => {
    if (!detail.value) return
    const clientId = detail.value.id
    await loadDetail(clientId)
    await loadTable()
  }
  const resetPlanPreview = () => {
    planPreview.value = null
    planPreviewKey.value = crypto.randomUUID()
    planStartKey.value = crypto.randomUUID()
  }
  const openPlanStart = async () => {
    const res = await getPlanVersions({
      page: 1,
      pageSize: 100,
      status: 'PUBLISHED',
      synthetic: true
    })
    if (res.code !== 0) return
    planVersionOptions.value = res.data.list || []
    if (!planVersionOptions.value.length) {
      ElMessage.warning('当前没有可用的计划模板')
      return
    }
    planStartForm.planTemplateVersionId = planVersionOptions.value[0].id
    planStartForm.anchorAt = undefined
    resetPlanPreview()
    planStartVisible.value = true
  }
  const previewPlan = async () => {
    if (!planStartForm.planTemplateVersionId || !planStartForm.anchorAt) {
      ElMessage.warning('请选择计划模板和计划开始时间')
      return
    }
    planActionLoading.value = true
    try {
      const res = await previewCarePlan(detail.value.id, {
        planTemplateVersionId: planStartForm.planTemplateVersionId,
        anchorAt: new Date(planStartForm.anchorAt).toISOString()
      }, planPreviewKey.value)
      if (res.code === 0) {
        planPreview.value = res.data
      }
    } finally {
      planActionLoading.value = false
    }
  }
  const confirmStartPlan = async () => {
    if (!planPreview.value) return
    planActionLoading.value = true
    try {
      const res = await startCarePlan(detail.value.id, {
        expectedClientVersion: detail.value.version,
        previewId: planPreview.value.previewId
      }, planStartKey.value)
      if (res.code === 0) {
        ElMessage.success('服务计划已启动')
        planStartVisible.value = false
        await loadDetail(detail.value.id)
        await loadPlans(detail.value.id)
        await loadTable()
      }
    } finally {
      planActionLoading.value = false
    }
  }
  const changePlanState = async (plan, action) => {
    if (planStateActionId.value) return
    planStateActionId.value = plan.id
    const verb = action === 'pause' ? '暂停' : '恢复'
    const commandKey = crypto.randomUUID()
    try {
      const { value } = await ElMessageBox.prompt(
        `请输入${verb}原因。${verb}后，已经安排的任务时间不会改变。`,
        `${verb}服务计划`,
        {
          confirmButtonText: `确认${verb}`,
          cancelButtonText: '取消',
          inputType: 'textarea',
          inputValidator: (input) => {
            if (!input?.trim()) return '原因必填'
            return input.length <= 1000 || '原因不能超过 1000 字符'
          }
        }
      )
      const request = {
        expectedVersion: plan.version,
        reason: value.trim()
      }
      const res = action === 'pause'
        ? await pauseCarePlan(plan.id, request, commandKey)
        : await resumeCarePlan(plan.id, request, commandKey)
      if (res.code === 0) {
        ElMessage.success(`计划已${verb}`)
        await loadPlans(detail.value.id)
      }
    } catch (error) {
      const actionName = typeof error === 'string' ? error : error?.message
      if (!['cancel', 'close'].includes(actionName)) {
        ElMessage.error(`计划${verb}失败，请重试`)
      }
    } finally {
      planStateActionId.value = 0
    }
  }
  const roleLabel = (value) => ({ CARE_STEWARD: '健康管家', CLINICIAN: '一线医护' }[value] || '未说明')
  const assignmentStatusLabel = (value) => ({ ACTIVE: '生效中', SCHEDULED: '待生效', ENDED: '已结束', CANCELLED: '已取消' }[value] || '未说明')
  const planStatusLabel = (value) => ({ ACTIVE: '进行中', PAUSED: '已暂停', COMPLETED: '已完成', TERMINATED: '已终止' }[value] || '未说明')
  const taskStatusLabel = (value) => ({ SCHEDULED: '待开放', OPEN: '已开放', IN_PROGRESS: '进行中', SUBMITTED: '已提交', CANCELLED: '已取消', NOT_OPEN: '未开放', WITHIN_WINDOW: '可按时完成', OVERDUE: '已超过截止时间', EXPIRED: '已结束', NOT_READY: '尚不可复核', NOT_REQUIRED: '无需复核', PENDING: '待复核', REVIEWING: '复核中', REVIEWED: '已复核', RETURNED: '已退回' }[value] || '未说明')
  const taskExecutionTagType = (value) => ({ OPEN: 'success', SUBMITTED: 'primary', CANCELLED: 'info' }[value] || 'info')
  const taskTimingTagType = (value) => ({ WITHIN_WINDOW: 'success', OVERDUE: 'warning', EXPIRED: 'danger' }[value] || 'info')
  const formatDateTime = (value) => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
  const dayLabel = (value) => `第${String(value || '').replace(/^D/, '') || '-'}次`
  const teamName = (id) => options.value.orgUnits.find((item) => item.departmentId === id)?.name || '未分配团队'

  onMounted(async () => {
    await loadTable()
    const detailId = Number(props.initialDetailId)
    if (Number.isInteger(detailId) && detailId > 0) {
      await openDetail(detailId)
    }
  })
</script>
