<template>
  <main class="space-y-4 text-base-text">
    <section class="rounded-xl border border-border bg-container p-5 shadow-card">
      <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div>
          <div class="mb-2 flex items-center gap-2 text-sm font-medium text-primary">
            <svg-icon icon="lucide:clipboard-clock" />
            <span>人工处理队列</span>
          </div>
          <h1 class="text-2xl font-semibold tracking-tight">关注事项</h1>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
            列表仅展示处理所需摘要。规则命中表示需要人工关注，不代表诊断；原始答卷不会在这里展示。
          </p>
        </div>
        <el-button
          :loading="loading"
          @click="loadTable"
        >
          <svg-icon
            class="mr-1"
            icon="lucide:refresh-cw"
          />
          刷新
        </el-button>
      </div>
    </section>

    <section class="gva-search-box">
      <el-form
        :inline="true"
        :model="searchForm"
      >
        <el-form-item label="事项状态">
          <el-select
            v-model="searchForm.status"
            class="w-52"
            clearable
            placeholder="全部状态"
          >
            <el-option
              v-for="item in statusOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="责任人 ID">
          <el-input-number
            v-model="searchForm.assigneeId"
            :min="1"
            class="w-40"
            controls-position="right"
          />
        </el-form-item>
        <el-form-item>
          <el-button
            type="primary"
            @click="search"
          >
            查询
          </el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </section>

    <section class="gva-table-box">
      <el-table
        v-loading="loading"
        :data="tableData"
        empty-text="当前责任或组织范围内暂无关注事项"
        row-key="id"
      >
        <el-table-column
          label="事项"
          min-width="110"
        >
          <template #default="scope">
            <div class="font-mono font-semibold">#{{ scope.row.id }}</div>
            <div class="mt-1 text-xs text-muted-foreground">v{{ scope.row.version }}</div>
          </template>
        </el-table-column>
        <el-table-column
          label="关联对象"
          min-width="170"
        >
          <template #default="scope">
            <div>康养用户 #{{ scope.row.careClientId }}</div>
            <div class="mt-1 text-xs text-muted-foreground">任务 #{{ scope.row.taskId }}</div>
          </template>
        </el-table-column>
        <el-table-column
          label="状态"
          min-width="150"
        >
          <template #default="scope">
            <el-tag
              :type="statusTagType(scope.row.status)"
              effect="plain"
            >
              {{ statusLabel(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column
          label="关注等级"
          min-width="150"
          prop="attentionLevel"
        />
        <el-table-column
          label="触发摘要"
          min-width="300"
        >
          <template #default="scope">
            <p class="line-clamp-2 leading-6">{{ scope.row.reasonSummary }}</p>
          </template>
        </el-table-column>
        <el-table-column
          label="当前责任人"
          min-width="130"
        >
          <template #default="scope">
            {{ scope.row.assigneeId ? `#${scope.row.assigneeId}` : '待分配' }}
          </template>
        </el-table-column>
        <el-table-column
          label="打开时间"
          min-width="180"
        >
          <template #default="scope">
            {{ formatTimestamp(scope.row.openedAt) }}
          </template>
        </el-table-column>
        <el-table-column
          fixed="right"
          label="操作"
          width="90"
        >
          <template #default="scope">
            <el-button
              v-if="btnAuth.viewDetail"
              link
              type="primary"
              @click="openDetail(scope.row.id)"
            >
              详情
            </el-button>
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
    </section>

    <el-drawer
      v-model="detailVisible"
      size="min(780px, 100%)"
      title="关注事项详情"
    >
      <div
        v-loading="detailLoading"
        class="min-h-56"
      >
        <template v-if="detail">
          <section class="mb-5 rounded-xl border border-border bg-muted p-4">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <p class="text-xs font-medium tracking-widest text-muted-foreground">ATTENTION CASE #{{ detail.id }}</p>
                <h2 class="mt-2 text-xl font-semibold">{{ statusLabel(detail.status) }}</h2>
                <p class="mt-2 text-sm leading-6">{{ detail.reasonSummary }}</p>
              </div>
              <el-tag
                :type="statusTagType(detail.status)"
                effect="plain"
              >
                {{ detail.attentionLevel }}
              </el-tag>
            </div>
          </section>

          <div class="mb-5 flex flex-wrap gap-2">
            <el-button
              v-if="btnAuth.acknowledge && detail.status === 'PENDING_ACK'"
              type="primary"
              @click="openAction('acknowledge')"
            >
              确认事项
            </el-button>
            <el-button
              v-if="btnAuth.recordContact && canRecordHandling"
              @click="openAction('contact')"
            >
              记录联系
            </el-button>
            <el-button
              v-if="btnAuth.recordHandling && canRecordHandling"
              type="primary"
              @click="openAction('handling')"
            >
              处置 / 请求复核
            </el-button>
            <el-button
              v-if="btnAuth.escalate && canEscalate"
              @click="openEscalation"
            >
              升级责任医护
            </el-button>
            <el-button
              v-if="btnAuth.close && detail.status === 'RESOLVED'"
              type="success"
              @click="openAction('close')"
            >
              关闭事项
            </el-button>
            <el-button
              v-if="btnAuth.reopen && detail.status === 'CLOSED'"
              type="warning"
              @click="openAction('reopen')"
            >
              重开事项
            </el-button>
          </div>

          <el-descriptions
            :column="2"
            border
          >
            <el-descriptions-item label="康养用户">#{{ detail.careClientId }}</el-descriptions-item>
            <el-descriptions-item label="计划任务">#{{ detail.taskId }}</el-descriptions-item>
            <el-descriptions-item label="当前责任人">
              {{ detail.assigneeId ? `#${detail.assigneeId}` : '待分配' }}
            </el-descriptions-item>
            <el-descriptions-item label="版本">v{{ detail.version }}</el-descriptions-item>
            <el-descriptions-item label="打开时间">{{ formatTimestamp(detail.openedAt) }}</el-descriptions-item>
            <el-descriptions-item label="目标时间">{{ formatTimestamp(detail.dueAt) }}</el-descriptions-item>
            <el-descriptions-item
              label="最近处理结果"
              :span="2"
            >
              {{ detail.handlingResult || '尚未记录' }}
            </el-descriptions-item>
            <el-descriptions-item
              label="关闭理由"
              :span="2"
            >
              {{ detail.closeReason || '尚未关闭' }}
            </el-descriptions-item>
          </el-descriptions>

          <section class="mt-7">
            <div class="mb-3">
              <p class="text-xs font-medium tracking-widest text-muted-foreground">RULE HIT SNAPSHOT</p>
              <h3 class="mt-1 text-lg font-semibold">规则命中快照</h3>
            </div>
            <el-empty
              v-if="!detail.ruleHits?.length"
              :image-size="64"
              description="暂无规则命中快照"
            />
            <div
              v-else
              class="space-y-2"
            >
              <article
                v-for="hit in detail.ruleHits"
                :key="hit.id"
                class="rounded-lg border border-border bg-container p-4"
              >
                <div class="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
                  <span>规则版本 #{{ hit.ruleVersionId }}</span>
                  <span>{{ formatTimestamp(hit.occurredAt) }}</span>
                </div>
                <p class="mt-2 text-sm leading-6">{{ hit.reasonSnapshot }}</p>
              </article>
            </div>
          </section>

          <section class="mt-7">
            <div class="mb-3">
              <p class="text-xs font-medium tracking-widest text-muted-foreground">ACTION TRAIL</p>
              <h3 class="mt-1 text-lg font-semibold">行动时间线</h3>
            </div>
            <el-empty
              v-if="!detail.actions?.length"
              :image-size="64"
              description="尚无行动记录"
            />
            <el-timeline v-else>
              <el-timeline-item
                v-for="action in detail.actions"
                :key="action.id"
                :timestamp="formatTimestamp(action.occurredAt)"
                placement="top"
              >
                <div class="flex flex-wrap items-center gap-2">
                  <span class="font-medium">{{ actionTypeLabel(action.actionType) }}</span>
                  <el-tag
                    effect="plain"
                    size="small"
                  >
                    {{ actorRoleLabel(action.actorRole) }}
                  </el-tag>
                </div>
                <p class="mt-1 text-sm leading-6 text-muted-foreground">{{ action.result }}</p>
                <p
                  v-if="action.reason"
                  class="mt-1 text-sm leading-6 text-muted-foreground"
                >
                  后续 / 原因：{{ action.reason }}
                </p>
              </el-timeline-item>
            </el-timeline>
          </section>
        </template>
      </div>
    </el-drawer>

    <el-dialog
      v-model="actionDialogVisible"
      :title="actionTitle"
      width="min(560px, 92vw)"
    >
      <el-form
        label-position="top"
        :model="actionForm"
      >
        <el-form-item
          v-if="actionMode !== 'close' && actionMode !== 'reopen'"
          :label="actionResultLabel"
          required
        >
          <el-input
            v-model="actionForm.result"
            :maxlength="actionMode === 'acknowledge' ? 1000 : 4000"
            :placeholder="actionResultPlaceholder"
            :rows="4"
            show-word-limit
            type="textarea"
          />
        </el-form-item>
        <template v-if="actionMode === 'contact' || actionMode === 'handling'">
          <el-form-item label="下一状态" required>
            <el-select
              v-model="actionForm.nextStatus"
              class="w-full"
            >
              <el-option
                v-for="item in actionStatusOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="下一步安排">
            <el-input
              v-model="actionForm.nextAction"
              maxlength="1000"
              placeholder="可选，记录后续流程安排"
              show-word-limit
            />
          </el-form-item>
          <el-alert
            v-if="actionForm.nextStatus === 'WAITING_SUPERVISOR'"
            :closable="false"
            class="mb-3"
            title="该操作仅请求上级复核；事项保持未关闭，后续指导由督导功能追加。"
            type="info"
          />
        </template>
        <el-form-item
          v-if="actionMode === 'close' || actionMode === 'reopen'"
          :label="actionMode === 'close' ? '关闭理由' : '重开理由'"
          required
        >
          <el-input
            v-model="actionForm.reason"
            maxlength="2000"
            :placeholder="actionMode === 'close' ? '请说明关闭理由' : '请说明重开原因'"
            :rows="4"
            show-word-limit
            type="textarea"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="actionDialogVisible = false">取消</el-button>
        <el-button
          :loading="actionSubmitting"
          type="primary"
          @click="submitAction"
        >
          确认提交
        </el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="escalationVisible"
      title="升级至责任医护"
      width="min(560px, 92vw)"
    >
      <div v-loading="escalationOptionsLoading">
        <el-alert
          :closable="false"
          class="mb-4"
          title="升级会结束当前活动待办，并为所选责任医护创建新的活动待办；既有行动记录不会被覆盖。"
          type="info"
        />
        <el-form
          label-position="top"
          :model="escalationForm"
        >
          <el-form-item label="目标责任医护" required>
            <el-select
              v-model="escalationForm.targetAssigneeId"
              class="w-full"
              placeholder="选择当前有效责任医护"
            >
              <el-option
                v-for="item in clinicianOptions"
                :key="item.assigneeId"
                :label="`${item.assigneeDisplayName}（#${item.assigneeId}）`"
                :value="item.assigneeId"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="升级原因" required>
            <el-input
              v-model="escalationForm.reason"
              maxlength="2000"
              placeholder="请记录需要协作处理的流程原因"
              :rows="4"
              show-word-limit
              type="textarea"
            />
          </el-form-item>
          <el-form-item label="目标时间">
            <el-date-picker
              v-model="escalationForm.dueAt"
              class="w-full"
              placeholder="可选"
              type="datetime"
            />
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="escalationVisible = false">取消</el-button>
        <el-button
          :disabled="!clinicianOptions.length"
          :loading="escalationSubmitting"
          type="primary"
          @click="submitEscalation"
        >
          确认升级
        </el-button>
      </template>
    </el-dialog>
  </main>
</template>

<script setup>
  import { computed, onMounted, reactive, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import { getCareClient } from '@/api/sleep-care/care-clients'
  import {
    acknowledgeAttentionCase,
    closeAttentionCase,
    createAttentionHandlingRecord,
    escalateAttentionCase,
    getAttentionCase,
    getAttentionCases,
    reopenAttentionCase
  } from '@/api/sleep-care/case-work'
  import { useBtnAuth } from '@/utils/btnAuth'
  import { formatDate } from '@/utils/format'

  defineOptions({ name: 'CareAttentionCases' })

  const btnAuth = useBtnAuth()
  const statusOptions = [
    { label: '待确认', value: 'PENDING_ACK' },
    { label: '已确认', value: 'ACKNOWLEDGED' },
    { label: '处理中', value: 'HANDLING' },
    { label: '等待用户', value: 'WAITING_CLIENT' },
    { label: '等待协作', value: 'WAITING_COLLABORATION' },
    { label: '等待上级复核', value: 'WAITING_SUPERVISOR' },
    { label: '已解决', value: 'RESOLVED' },
    { label: '已关闭', value: 'CLOSED' }
  ]
  const contactStatusOptions = [
    { label: '进入处理中', value: 'HANDLING' },
    { label: '等待用户补充', value: 'WAITING_CLIENT' },
    { label: '等待协作', value: 'WAITING_COLLABORATION' }
  ]
  const handlingStatusOptions = [
    { label: '继续处理', value: 'HANDLING' },
    { label: '等待用户补充', value: 'WAITING_CLIENT' },
    { label: '等待协作', value: 'WAITING_COLLABORATION' },
    { label: '请求上级复核', value: 'WAITING_SUPERVISOR' },
    { label: '标记已解决', value: 'RESOLVED' }
  ]
  const actionableStatuses = [
    'ACKNOWLEDGED',
    'HANDLING',
    'WAITING_CLIENT',
    'WAITING_COLLABORATION',
    'WAITING_SUPERVISOR'
  ]
  const searchForm = reactive({
    status: '',
    assigneeId: undefined
  })
  const page = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const tableData = ref([])
  const loading = ref(false)
  const detailVisible = ref(false)
  const detailLoading = ref(false)
  const detail = ref(null)
  const actionDialogVisible = ref(false)
  const actionSubmitting = ref(false)
  const actionMode = ref('acknowledge')
  const actionForm = reactive({
    result: '',
    nextAction: '',
    nextStatus: 'HANDLING',
    reason: ''
  })
  const escalationVisible = ref(false)
  const escalationOptionsLoading = ref(false)
  const escalationSubmitting = ref(false)
  const clinicianOptions = ref([])
  const escalationForm = reactive({
    targetAssigneeId: undefined,
    reason: '',
    dueAt: null
  })

  const canRecordHandling = computed(() => actionableStatuses.includes(detail.value?.status))
  const canEscalate = computed(() => actionableStatuses.includes(detail.value?.status))
  const actionTitle = computed(() => ({
    acknowledge: '确认关注事项',
    contact: '记录联系结果',
    handling: '记录处置结果',
    close: '关闭关注事项',
    reopen: '重开关注事项'
  }[actionMode.value]))
  const actionResultLabel = computed(() => actionMode.value === 'acknowledge' ? '确认结果' : actionMode.value === 'contact' ? '联系结果' : '处置结果')
  const actionResultPlaceholder = computed(() => ({
    acknowledge: '请记录已接收并确认处理',
    contact: '请记录流程联系结果，不填写专业判断',
    handling: '请记录本次流程处理结果'
  }[actionMode.value] || '请输入处理结果'))
  const actionStatusOptions = computed(() => actionMode.value === 'contact' ? contactStatusOptions : handlingStatusOptions)

  const loadTable = async () => {
    loading.value = true
    try {
      const res = await getAttentionCases({
        page: page.value,
        pageSize: pageSize.value,
        ...searchForm
      })
      if (res.code === 0) {
        tableData.value = res.data.list || []
        total.value = res.data.total || 0
      }
    } finally {
      loading.value = false
    }
  }

  const search = () => {
    page.value = 1
    loadTable()
  }

  const resetSearch = () => {
    searchForm.status = ''
    searchForm.assigneeId = undefined
    search()
  }

  const handleSizeChange = () => {
    page.value = 1
    loadTable()
  }

  const openDetail = async (id) => {
    detailVisible.value = true
    await loadDetail(id)
  }

  const loadDetail = async (id) => {
    detailLoading.value = true
    try {
      const res = await getAttentionCase(id)
      if (res.code === 0) {
        detail.value = res.data
      }
    } finally {
      detailLoading.value = false
    }
  }

  const refreshAfterAction = async () => {
    const id = detail.value.id
    await Promise.all([loadDetail(id), loadTable()])
  }

  const openAction = (mode) => {
    actionMode.value = mode
    actionForm.result = ''
    actionForm.nextAction = ''
    actionForm.nextStatus = 'HANDLING'
    actionForm.reason = ''
    actionDialogVisible.value = true
  }

  const submitAction = async () => {
    const value = actionMode.value === 'close' || actionMode.value === 'reopen'
      ? actionForm.reason.trim()
      : actionForm.result.trim()
    if (!value) {
      ElMessage.warning('请填写必填内容')
      return
    }
    actionSubmitting.value = true
    try {
      let res
      if (actionMode.value === 'acknowledge') {
        res = await acknowledgeAttentionCase(detail.value.id, {
          expectedVersion: detail.value.version,
          result: value
        })
      } else if (actionMode.value === 'contact' || actionMode.value === 'handling') {
        res = await createAttentionHandlingRecord(detail.value.id, {
          expectedVersion: detail.value.version,
          actionType: actionMode.value === 'contact' ? 'CONTACT' : 'HANDLING',
          result: value,
          nextAction: actionForm.nextAction.trim(),
          nextStatus: actionForm.nextStatus
        })
      } else if (actionMode.value === 'close') {
        res = await closeAttentionCase(detail.value.id, {
          expectedVersion: detail.value.version,
          closeReason: value
        })
      } else {
        res = await reopenAttentionCase(detail.value.id, {
          expectedVersion: detail.value.version,
          reason: value
        })
      }
      if (res.code === 0) {
        ElMessage.success('操作已记录')
        actionDialogVisible.value = false
        await refreshAfterAction()
      }
    } finally {
      actionSubmitting.value = false
    }
  }

  const openEscalation = async () => {
    escalationForm.targetAssigneeId = undefined
    escalationForm.reason = ''
    escalationForm.dueAt = null
    clinicianOptions.value = []
    escalationVisible.value = true
    escalationOptionsLoading.value = true
    try {
      const res = await getCareClient(detail.value.careClientId)
      if (res.code === 0) {
        clinicianOptions.value = (res.data.currentAssignments || []).filter((item) =>
          item.roleType === 'CLINICIAN' && item.status === 'ACTIVE'
        )
        if (clinicianOptions.value.length === 1) {
          escalationForm.targetAssigneeId = clinicianOptions.value[0].assigneeId
        }
      }
    } finally {
      escalationOptionsLoading.value = false
    }
  }

  const submitEscalation = async () => {
    if (!escalationForm.targetAssigneeId || !escalationForm.reason.trim()) {
      ElMessage.warning('请选择责任医护并填写升级原因')
      return
    }
    escalationSubmitting.value = true
    try {
      const res = await escalateAttentionCase(detail.value.id, {
        expectedVersion: detail.value.version,
        targetAssigneeId: escalationForm.targetAssigneeId,
        reason: escalationForm.reason.trim(),
        dueAt: escalationForm.dueAt || undefined
      })
      if (res.code === 0) {
        ElMessage.success('事项已升级')
        escalationVisible.value = false
        await refreshAfterAction()
      }
    } finally {
      escalationSubmitting.value = false
    }
  }

  const statusLabel = (value) => statusOptions.find((item) => item.value === value)?.label || value
  const statusTagType = (value) => ({
    PENDING_ACK: 'warning',
    ACKNOWLEDGED: 'primary',
    HANDLING: 'primary',
    WAITING_CLIENT: 'info',
    WAITING_COLLABORATION: 'warning',
    WAITING_SUPERVISOR: 'warning',
    RESOLVED: 'success',
    CLOSED: 'info'
  }[value] || 'info')
  const actionTypeLabel = (value) => ({
    ACKNOWLEDGE: '确认',
    CONTACT: '联系',
    HANDLING: '处置',
    ESCALATE: '升级',
    GUIDANCE: '指导',
    INTERVENE: '介入',
    RESOLVE: '解决',
    CLOSE: '关闭',
    REOPEN: '重开'
  }[value] || value)
  const actorRoleLabel = (value) => ({
    CARE_STEWARD: '健康管家',
    CLINICIAN: '一线医护',
    SUPERVISOR: '上级医师'
  }[value] || value)
  const formatTimestamp = (value) => value ? formatDate(value) : '-'

  onMounted(loadTable)
</script>
