<template>
  <main class="space-y-4 text-base-text">
    <section class="rounded-xl border border-border bg-container p-5 shadow-card">
      <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div>
          <div class="mb-2 flex items-center gap-2 text-sm font-medium text-primary">
            <svg-icon icon="lucide:messages-square" />
            <span>服务沟通队列</span>
          </div>
          <h1 class="text-2xl font-semibold tracking-tight">主动咨询</h1>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
            接收康养用户主动提出的服务问题，按责任关系完成回复、转交、升级、解决和关闭。咨询入口不提供急救服务。
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
        <el-form-item label="咨询状态">
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
        <el-form-item label="联系顺序">
          <el-select
            v-model="searchForm.urgency"
            class="w-44"
            clearable
            placeholder="全部"
          >
            <el-option label="常规联系" value="ROUTINE" />
            <el-option label="优先联系" value="EXPEDITED" />
          </el-select>
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
        empty-text="当前责任或组织范围内暂无咨询"
        row-key="id"
      >
        <el-table-column
          label="咨询"
          min-width="250"
        >
          <template #default="scope">
            <div class="font-semibold">{{ scope.row.subject }}</div>
            <div class="mt-1 text-xs text-muted-foreground">编号 {{ scope.row.id }}</div>
          </template>
        </el-table-column>
        <el-table-column
          label="康养用户"
          min-width="180"
        >
          <template #default="scope">
            <div>{{ scope.row.clientDisplayName || '未命名用户' }}</div>
            <div class="mt-1 text-xs text-muted-foreground">{{ scope.row.clientDisplayCode }}</div>
          </template>
        </el-table-column>
        <el-table-column
          label="状态"
          min-width="140"
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
          label="联系顺序"
          min-width="110"
        >
          <template #default="scope">
            {{ urgencyLabel(scope.row.urgency) }}
          </template>
        </el-table-column>
        <el-table-column
          label="当前责任人"
          min-width="150"
        >
          <template #default="scope">
            <div>{{ scope.row.assigneeName || '待分配' }}</div>
            <div
              v-if="scope.row.assigneeRole"
              class="mt-1 text-xs text-muted-foreground"
            >
              {{ actorRoleLabel(scope.row.assigneeRole) }}
            </div>
          </template>
        </el-table-column>
        <el-table-column
          label="提交时间"
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
      size="min(820px, 100%)"
      title="咨询详情"
    >
      <div
        v-loading="detailLoading"
        class="min-h-56"
      >
        <template v-if="detail">
          <section class="mb-5 rounded-xl border border-border bg-muted p-4">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <p class="text-xs font-medium text-muted-foreground">
                  咨询编号 {{ detail.id }} · {{ detail.clientDisplayName }}（{{ detail.clientDisplayCode }}）
                </p>
                <h2 class="mt-2 text-xl font-semibold">{{ detail.subject }}</h2>
                <p class="mt-2 whitespace-pre-wrap text-sm leading-6">{{ detail.initialQuestion }}</p>
              </div>
              <el-tag
                :type="statusTagType(detail.status)"
                effect="plain"
              >
                {{ statusLabel(detail.status) }}
              </el-tag>
            </div>
          </section>

          <div class="mb-5 flex flex-wrap gap-2">
            <el-button
              v-if="btnAuth.assign && detail.status === 'WAITING_ASSIGNMENT'"
              type="primary"
              @click="openAction('assign')"
            >
              分配咨询
            </el-button>
            <el-button
              v-if="btnAuth.reply && isCurrentAssignee && canHandle"
              type="primary"
              @click="openAction('reply')"
            >
              公开回复
            </el-button>
            <el-button
              v-if="btnAuth.transfer && isCurrentAssignee && canHandle"
              @click="openAction('transfer')"
            >
              转交
            </el-button>
            <el-button
              v-if="btnAuth.escalate && isCurrentAssignee && canHandle"
              @click="openAction('escalate')"
            >
              升级
            </el-button>
            <el-button
              v-if="btnAuth.resolve && isCurrentAssignee && canHandle"
              type="success"
              @click="openAction('resolve')"
            >
              记录解决结果
            </el-button>
            <el-button
              v-if="btnAuth.close && detail.status === 'RESOLVED' && (isCurrentAssignee || btnAuth.reopen)"
              type="success"
              @click="openAction('close')"
            >
              关闭咨询
            </el-button>
            <el-button
              v-if="btnAuth.reopen && detail.status === 'CLOSED'"
              type="warning"
              @click="openAction('reopen')"
            >
              重开咨询
            </el-button>
          </div>

          <el-descriptions
            :column="2"
            border
          >
            <el-descriptions-item label="康养用户">
              {{ detail.clientDisplayName }}（{{ detail.clientDisplayCode }}）
            </el-descriptions-item>
            <el-descriptions-item label="来源">在线咨询</el-descriptions-item>
            <el-descriptions-item label="联系顺序">{{ urgencyLabel(detail.urgency) }}</el-descriptions-item>
            <el-descriptions-item label="当前责任人">
              {{ detail.assigneeName || '待分配' }}
              <span
                v-if="detail.assigneeRole"
                class="text-muted-foreground"
              >
                · {{ actorRoleLabel(detail.assigneeRole) }}
              </span>
            </el-descriptions-item>
            <el-descriptions-item label="提交时间">{{ formatTimestamp(detail.openedAt) }}</el-descriptions-item>
            <el-descriptions-item label="首次回复">{{ formatTimestamp(detail.firstRespondedAt) }}</el-descriptions-item>
            <el-descriptions-item
              label="解决结果"
              :span="2"
            >
              {{ detail.resolution || '尚未记录' }}
            </el-descriptions-item>
            <el-descriptions-item
              label="后续安排"
              :span="2"
            >
              {{ detail.followUpPlan || '尚未记录' }}
            </el-descriptions-item>
            <el-descriptions-item
              label="关闭理由"
              :span="2"
            >
              {{ detail.closeReason || '尚未关闭' }}
            </el-descriptions-item>
          </el-descriptions>

          <section class="mt-7">
            <h3 class="text-lg font-semibold">互动时间线</h3>
            <el-empty
              v-if="!detail.interactions?.length"
              :image-size="64"
              description="尚无互动记录"
            />
            <el-timeline
              v-else
              class="mt-4"
            >
              <el-timeline-item
                v-for="interaction in detail.interactions"
                :key="interaction.id"
                :timestamp="formatTimestamp(interaction.occurredAt)"
                placement="top"
              >
                <div class="flex flex-wrap items-center gap-2">
                  <span class="font-medium">{{ actionTypeLabel(interaction.actionType) }}</span>
                  <el-tag
                    effect="plain"
                    size="small"
                  >
                    {{ interaction.actorName || actorTypeLabel(interaction.actorType) }}
                  </el-tag>
                  <span class="text-xs text-muted-foreground">
                    {{ statusLabel(interaction.fromStatus) }} → {{ statusLabel(interaction.toStatus) }}
                  </span>
                </div>
                <p
                  v-if="interaction.content"
                  class="mt-2 whitespace-pre-wrap text-sm leading-6"
                >
                  {{ interaction.content }}
                </p>
                <p
                  v-if="interaction.reason"
                  class="mt-1 whitespace-pre-wrap text-sm leading-6 text-muted-foreground"
                >
                  说明：{{ interaction.reason }}
                </p>
              </el-timeline-item>
            </el-timeline>
          </section>
        </template>
      </div>
    </el-drawer>

    <el-dialog
      v-model="actionVisible"
      :title="actionTitle"
      width="min(580px, 92vw)"
    >
      <el-form
        label-position="top"
        :model="actionForm"
      >
        <template v-if="needsAssignee">
          <el-form-item
            label="目标责任人"
            required
          >
            <el-select
              v-model="actionForm.targetAssigneeId"
              class="w-full"
              :loading="optionsLoading"
              placeholder="请选择符合当前责任关系的人员"
            >
              <el-option
                v-for="option in filteredAssigneeOptions"
                :key="`${option.roleType}-${option.id}`"
                :label="`${option.displayName} · ${actorRoleLabel(option.roleType)}`"
                :value="option.id"
              />
            </el-select>
          </el-form-item>
          <el-alert
            v-if="!optionsLoading && !filteredAssigneeOptions.length"
            :closable="false"
            class="mb-4"
            title="当前没有符合责任关系的可选人员，请先维护责任关系。"
            type="warning"
          />
          <el-form-item
            :label="actionMode === 'assign' ? '分配原因' : actionMode === 'transfer' ? '转交原因' : '升级原因'"
            required
          >
            <el-input
              v-model="actionForm.reason"
              maxlength="2000"
              placeholder="请说明本次责任变化原因"
              :rows="4"
              show-word-limit
              type="textarea"
            />
          </el-form-item>
        </template>

        <template v-else-if="actionMode === 'reply'">
          <el-form-item
            label="公开回复"
            required
          >
            <el-input
              v-model="actionForm.message"
              maxlength="4000"
              placeholder="该内容会展示给康养用户"
              :rows="5"
              show-word-limit
              type="textarea"
            />
          </el-form-item>
          <el-form-item
            label="回复后状态"
            required
          >
            <el-select
              v-model="actionForm.nextStatus"
              class="w-full"
            >
              <el-option label="继续处理" value="HANDLING" />
              <el-option label="等待用户补充" value="WAITING_CLIENT" />
            </el-select>
          </el-form-item>
        </template>

        <template v-else-if="actionMode === 'resolve'">
          <el-form-item
            label="解决结果"
            required
          >
            <el-input
              v-model="actionForm.resolution"
              maxlength="4000"
              placeholder="请记录本次咨询的明确处理结果"
              :rows="5"
              show-word-limit
              type="textarea"
            />
          </el-form-item>
          <el-form-item label="后续安排（可选）">
            <el-input
              v-model="actionForm.followUpPlan"
              maxlength="2000"
              placeholder="可记录后续联系或服务安排"
              :rows="3"
              show-word-limit
              type="textarea"
            />
          </el-form-item>
        </template>

        <el-form-item
          v-else
          :label="actionMode === 'close' ? '关闭理由' : '重开理由'"
          required
        >
          <el-input
            v-model="actionForm.reason"
            maxlength="2000"
            :placeholder="actionMode === 'close' ? '请说明本次关闭理由' : '请说明重开原因'"
            :rows="4"
            show-word-limit
            type="textarea"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="actionVisible = false">取消</el-button>
        <el-button
          type="primary"
          :disabled="needsAssignee && !filteredAssigneeOptions.length"
          :loading="actionSubmitting"
          @click="submitAction"
        >
          确认提交
        </el-button>
      </template>
    </el-dialog>
  </main>
</template>

<script setup>
  import { computed, onMounted, reactive, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import { useUserStore } from '@/pinia/modules/user'
  import {
    assignConsultation,
    closeConsultation,
    escalateConsultation,
    getConsultation,
    getConsultationAssigneeOptions,
    getConsultations,
    reopenConsultation,
    replyConsultation,
    resolveConsultation,
    transferConsultation
  } from '@/api/sleep-care/consultations'
  import { useBtnAuth } from '@/utils/btnAuth'
  import { formatDate } from '@/utils/format'

  defineOptions({
    name: 'CareConsultations'
  })

  const props = defineProps({
    initialDetailId: {
      type: [String, Number],
      default: ''
    }
  })

  const userStore = useUserStore()
  const btnAuth = useBtnAuth()
  const statusOptions = [
    { label: '新建', value: 'NEW' },
    { label: '待分配', value: 'WAITING_ASSIGNMENT' },
    { label: '已分配', value: 'ASSIGNED' },
    { label: '处理中', value: 'HANDLING' },
    { label: '等待用户补充', value: 'WAITING_CLIENT' },
    { label: '等待协作', value: 'WAITING_COLLABORATION' },
    { label: '已解决', value: 'RESOLVED' },
    { label: '已关闭', value: 'CLOSED' }
  ]
  const actionableStatuses = [
    'ASSIGNED',
    'HANDLING',
    'WAITING_CLIENT',
    'WAITING_COLLABORATION'
  ]
  const searchForm = reactive({
    status: '',
    urgency: ''
  })
  const page = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const tableData = ref([])
  const loading = ref(false)
  const detailVisible = ref(false)
  const detailLoading = ref(false)
  const detail = ref(null)
  const actionVisible = ref(false)
  const actionSubmitting = ref(false)
  const actionMode = ref('reply')
  const optionsLoading = ref(false)
  const assigneeOptions = ref([])
  const actionForm = reactive({
    targetAssigneeId: undefined,
    reason: '',
    message: '',
    nextStatus: 'WAITING_CLIENT',
    resolution: '',
    followUpPlan: ''
  })

  const isCurrentAssignee = computed(() => (
    Number(detail.value?.assigneeId) === Number(userStore.userInfo.ID)
  ))
  const canHandle = computed(() => actionableStatuses.includes(detail.value?.status))
  const needsAssignee = computed(() => ['assign', 'transfer', 'escalate'].includes(actionMode.value))
  const actionTitle = computed(() => ({
    assign: '分配咨询',
    reply: '公开回复',
    transfer: '转交咨询',
    escalate: '升级咨询',
    resolve: '记录解决结果',
    close: '关闭咨询',
    reopen: '重开咨询'
  }[actionMode.value]))
  const filteredAssigneeOptions = computed(() => {
    const currentID = Number(detail.value?.assigneeId)
    return assigneeOptions.value.filter((option) => {
      if (actionMode.value === 'escalate') {
        const targetRole = detail.value?.assigneeRole === 'CARE_STEWARD' ? 'CLINICIAN' : 'SUPERVISOR'
        return option.roleType === targetRole && option.id !== currentID
      }
      return ['CARE_STEWARD', 'CLINICIAN'].includes(option.roleType) && option.id !== currentID
    })
  })

  const loadTable = async () => {
    loading.value = true
    try {
      const res = await getConsultations({
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
    searchForm.urgency = ''
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
      const res = await getConsultation(id)
      if (res.code === 0) {
        detail.value = res.data
      }
    } finally {
      detailLoading.value = false
    }
  }

  const refreshAfterAction = async () => {
    const id = detail.value.id
    await Promise.all([
      loadDetail(id),
      loadTable()
    ])
  }

  const resetActionForm = () => {
    actionForm.targetAssigneeId = undefined
    actionForm.reason = ''
    actionForm.message = ''
    actionForm.nextStatus = 'WAITING_CLIENT'
    actionForm.resolution = ''
    actionForm.followUpPlan = ''
  }

  const openAction = async (mode) => {
    actionMode.value = mode
    resetActionForm()
    assigneeOptions.value = []
    actionVisible.value = true
    if (!['assign', 'transfer', 'escalate'].includes(mode)) {
      return
    }
    optionsLoading.value = true
    try {
      const res = await getConsultationAssigneeOptions(detail.value.id)
      if (res.code === 0) {
        assigneeOptions.value = res.data || []
        if (filteredAssigneeOptions.value.length === 1) {
          actionForm.targetAssigneeId = filteredAssigneeOptions.value[0].id
        }
      }
    } finally {
      optionsLoading.value = false
    }
  }

  const selectedAssignee = () => filteredAssigneeOptions.value.find((option) => (
    option.id === actionForm.targetAssigneeId
  ))

  const submitAction = async () => {
    let runCommand
    const expectedVersion = detail.value.version
    const reason = actionForm.reason.trim()
    if (actionMode.value === 'assign' || actionMode.value === 'transfer') {
      const target = selectedAssignee()
      if (!target || !reason) {
        ElMessage.warning('请选择目标责任人并填写原因')
        return
      }
      const payload = {
        expectedVersion,
        targetAssigneeId: target.id,
        targetRole: target.roleType,
        reason
      }
      runCommand = () => actionMode.value === 'assign'
        ? assignConsultation(detail.value.id, payload)
        : transferConsultation(detail.value.id, payload)
    } else if (actionMode.value === 'escalate') {
      const target = selectedAssignee()
      if (!target || !reason) {
        ElMessage.warning('请选择升级目标并填写原因')
        return
      }
      runCommand = () => escalateConsultation(detail.value.id, {
        expectedVersion,
        targetAssigneeId: target.id,
        reason
      })
    } else if (actionMode.value === 'reply') {
      const message = actionForm.message.trim()
      if (!message) {
        ElMessage.warning('请填写公开回复')
        return
      }
      runCommand = () => replyConsultation(detail.value.id, {
        expectedVersion,
        message,
        nextStatus: actionForm.nextStatus
      })
    } else if (actionMode.value === 'resolve') {
      const resolution = actionForm.resolution.trim()
      if (!resolution) {
        ElMessage.warning('请填写解决结果')
        return
      }
      runCommand = () => resolveConsultation(detail.value.id, {
        expectedVersion,
        resolution,
        followUpPlan: actionForm.followUpPlan.trim()
      })
    } else if (actionMode.value === 'close') {
      if (!reason) {
        ElMessage.warning('请填写关闭理由')
        return
      }
      runCommand = () => closeConsultation(detail.value.id, {
        expectedVersion,
        closeReason: reason
      })
    } else {
      if (!reason) {
        ElMessage.warning('请填写重开理由')
        return
      }
      runCommand = () => reopenConsultation(detail.value.id, {
        expectedVersion,
        reason
      })
    }

    actionSubmitting.value = true
    try {
      const res = await runCommand()
      if (res.code === 0) {
        ElMessage.success({
          assign: '咨询已分配',
          reply: '回复已记录',
          transfer: '咨询已转交',
          escalate: '咨询已升级',
          resolve: '解决结果已记录',
          close: '咨询已关闭',
          reopen: '咨询已重新打开'
        }[actionMode.value])
        actionVisible.value = false
        await refreshAfterAction()
      }
    } finally {
      actionSubmitting.value = false
    }
  }

  const statusLabel = (value) => statusOptions.find((item) => item.value === value)?.label || '未说明'
  const statusTagType = (value) => ({
    NEW: 'info',
    WAITING_ASSIGNMENT: 'warning',
    ASSIGNED: 'primary',
    HANDLING: 'primary',
    WAITING_CLIENT: 'warning',
    WAITING_COLLABORATION: 'warning',
    RESOLVED: 'success',
    CLOSED: 'info'
  }[value] || 'info')
  const urgencyLabel = (value) => value === 'EXPEDITED' ? '优先联系' : '常规联系'
  const actorRoleLabel = (value) => ({
    CARE_STEWARD: '健康管家',
    CLINICIAN: '一线医护',
    SUPERVISOR: '上级医师'
  }[value] || '工作人员')
  const actorTypeLabel = (value) => ({
    CLIENT: '康养用户',
    STAFF: '工作人员',
    SYSTEM: '系统'
  }[value] || '系统')
  const actionTypeLabel = (value) => ({
    CREATE: '发起咨询',
    ASSIGN: '分配',
    CLIENT_MESSAGE: '用户补充',
    REPLY: '公开回复',
    TRANSFER: '转交',
    ESCALATE: '升级',
    RESOLVE: '记录解决结果',
    CLOSE: '关闭',
    REOPEN: '重开'
  }[value] || '服务互动')
  const formatTimestamp = (value) => value ? formatDate(value) : '-'

  onMounted(async () => {
    await loadTable()
    const detailId = Number(props.initialDetailId)
    if (Number.isInteger(detailId) && detailId > 0) {
      await openDetail(detailId)
    }
  })
</script>
