<template>
  <main class="space-y-4 text-base-text">
    <section class="rounded-xl border border-border bg-container p-5 shadow-card">
      <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div>
          <div class="mb-2 flex items-center gap-2 text-sm font-medium text-primary">
            <svg-icon icon="lucide:user-round-check" />
            <span>上级督导</span>
          </div>
          <h1 class="text-2xl font-semibold tracking-tight">待复核事项</h1>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
            这里展示需要上级确认的事项，并完整记录处理过程和责任人变化。
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

    <section class="gva-table-box">
      <el-table
        v-loading="loading"
        :data="tableData"
        empty-text="当前管理范围内暂无待复核事项"
        row-key="id"
      >
        <el-table-column
          label="复核事项"
          min-width="130"
        >
          <template #default="scope">
            <div class="font-semibold">编号 {{ scope.row.id }}</div>
            <div class="mt-1 text-xs text-muted-foreground">关注事项 {{ scope.row.attentionCaseId }}</div>
          </template>
        </el-table-column>
        <el-table-column
          label="复核状态"
          min-width="140"
        >
          <template #default="scope">
            <el-tag
              :type="reviewStatusTag(scope.row.status)"
              effect="plain"
            >
              {{ reviewStatusLabel(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column
          label="请求时间"
          min-width="180"
        >
          <template #default="scope">{{ formatTimestamp(scope.row.requestedAt) }}</template>
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
              @click="openDetail(scope.row)"
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
      title="复核事项详情"
    >
      <div
        v-loading="detailLoading"
        class="min-h-56"
      >
        <template v-if="detail">
          <section class="rounded-xl border border-border bg-muted p-4">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <p class="text-xs font-medium text-muted-foreground">复核编号 {{ selectedReview?.id }}</p>
                <h2 class="mt-2 text-xl font-semibold">关注事项 {{ detail.id }}</h2>
                <p class="mt-2 text-sm leading-6">{{ readableAttentionReason(detail.reasonSummary) }}</p>
              </div>
              <el-tag
                :type="reviewStatusTag(selectedReview?.status)"
                effect="plain"
              >
                {{ reviewStatusLabel(selectedReview?.status) }}
              </el-tag>
            </div>
          </section>

          <div class="my-5 flex flex-wrap gap-2">
            <el-button
              v-if="btnAuth.guide && canAct"
              type="primary"
              @click="openAction('guidance')"
            >
              给出处理意见
            </el-button>
            <el-button
              v-if="btnAuth.discuss && canAct"
              @click="openAction('discussion')"
            >
              安排讨论
            </el-button>
            <el-button
              v-if="btnAuth.intervene && canAct"
              type="warning"
              @click="openAction('intervene')"
            >
              直接介入
            </el-button>
          </div>

          <el-alert
            v-if="!canAct"
            :closable="false"
            class="mb-5"
            title="该事项当前已离开待上级复核状态，仅保留历史查看。"
            type="info"
          />

          <el-descriptions
            :column="2"
            border
          >
            <el-descriptions-item label="康养用户">编号 {{ detail.careClientId }}</el-descriptions-item>
            <el-descriptions-item label="计划任务">编号 {{ detail.taskId }}</el-descriptions-item>
            <el-descriptions-item label="事项状态">{{ caseStatusLabel(detail.status) }}</el-descriptions-item>
            <el-descriptions-item label="当前责任人">
              {{ detail.assigneeId ? '已分配' : '待分配' }}
            </el-descriptions-item>
            <el-descriptions-item label="目标时间">{{ formatTimestamp(detail.dueAt) }}</el-descriptions-item>
            <el-descriptions-item
              label="最近处理结果"
              :span="2"
            >
              {{ detail.handlingResult || '尚未记录' }}
            </el-descriptions-item>
          </el-descriptions>

          <section class="mt-7">
            <div class="mb-3">
              <h3 class="text-lg font-semibold">处理记录</h3>
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
      v-model="actionVisible"
      :title="actionTitle"
      width="min(580px, 92vw)"
    >
      <div v-loading="optionsLoading">
        <el-alert
          :closable="false"
          class="mb-4"
          :title="actionHint"
          type="info"
        />
        <el-form
          label-position="top"
          :model="actionForm"
        >
          <el-form-item
            :label="actionContentLabel"
            required
          >
            <el-input
              v-model="actionForm.content"
              :maxlength="actionMode === 'discussion' ? 3995 : 4000"
              :placeholder="actionContentPlaceholder"
              :rows="4"
              show-word-limit
              type="textarea"
            />
          </el-form-item>
          <el-form-item label="后续责任医护" required>
            <el-select
              v-model="actionForm.responsibleAssigneeId"
              class="w-full"
              placeholder="选择当前有效责任医护"
            >
              <el-option
                v-for="item in clinicianOptions"
                :key="item.assigneeId"
                :label="item.assigneeDisplayName"
                :value="item.assigneeId"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="完成时限" required>
            <el-date-picker
              v-model="actionForm.dueAt"
              class="w-full"
              :disabled-date="disablePastDate"
              placeholder="选择未来时间"
              type="datetime"
            />
          </el-form-item>
        </el-form>
        <el-empty
          v-if="!optionsLoading && !clinicianOptions.length"
          :image-size="56"
          description="当前责任链中没有可选责任医护"
        />
      </div>
      <template #footer>
        <el-button @click="actionVisible = false">取消</el-button>
        <el-button
          :disabled="!clinicianOptions.length"
          :loading="actionSubmitting"
          :type="actionMode === 'intervene' ? 'warning' : 'primary'"
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
  import { getCareClient } from '@/api/sleep-care/care-clients'
  import { getAttentionCase } from '@/api/sleep-care/case-work'
  import {
    addSupervisorGuidance,
    getSupervisionReviews,
    interveneSupervisionReview
  } from '@/api/sleep-care/supervision'
  import { useBtnAuth } from '@/utils/btnAuth'
  import { formatDate } from '@/utils/format'
  import { readableAttentionReason } from '@/utils/sleep-care-display'

  defineOptions({ name: 'CareReviewQueue' })

  const props = defineProps({
    initialAttentionCaseId: {
      type: [String, Number],
      default: ''
    }
  })

  const btnAuth = useBtnAuth()
  const page = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const tableData = ref([])
  const loading = ref(false)
  const detailVisible = ref(false)
  const detailLoading = ref(false)
  const detail = ref(null)
  const selectedReview = ref(null)
  const actionVisible = ref(false)
  const actionSubmitting = ref(false)
  const optionsLoading = ref(false)
  const actionMode = ref('guidance')
  const commandKey = ref('')
  const clinicianOptions = ref([])
  const actionForm = reactive({
    content: '',
    responsibleAssigneeId: undefined,
    dueAt: null
  })

  const canAct = computed(() => detail.value?.status === 'WAITING_SUPERVISOR')
  const actionTitle = computed(() => ({
    guidance: '给出处理意见',
    discussion: '安排流程讨论',
    intervene: '记录上级直接介入'
  }[actionMode.value]))
  const actionHint = computed(() => actionMode.value === 'intervene'
    ? '介入后事项回到处理中，并由所选责任医护继续处理。'
    : '提交后事项继续等待上级确认，并由所选责任医护跟进。'
  )
  const actionContentLabel = computed(() => ({
    guidance: '指导内容',
    discussion: '讨论安排',
    intervene: '介入结果'
  }[actionMode.value]))
  const actionContentPlaceholder = computed(() => ({
    guidance: '请记录明确、可执行的流程指导',
    discussion: '请记录讨论主题、参与方式或后续安排',
    intervene: '请记录本次介入完成的流程动作与后续安排'
  }[actionMode.value]))

  const loadTable = async () => {
    loading.value = true
    try {
      const res = await getSupervisionReviews({
        page: page.value,
        pageSize: pageSize.value
      })
      if (res.code === 0) {
        tableData.value = res.data.list || []
        total.value = res.data.total || 0
        if (selectedReview.value) {
          const current = tableData.value.find((item) => item.id === selectedReview.value.id)
          if (current) {
            selectedReview.value = current
          }
        }
      }
    } finally {
      loading.value = false
    }
  }

  const handleSizeChange = () => {
    page.value = 1
    loadTable()
  }

  const openDetail = async (review) => {
    selectedReview.value = review
    detail.value = null
    detailVisible.value = true
    await loadDetail(review.attentionCaseId)
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

  const openAction = async (mode) => {
    actionMode.value = mode
    actionForm.content = ''
    actionForm.responsibleAssigneeId = detail.value.assigneeId || undefined
    actionForm.dueAt = null
    commandKey.value = crypto.randomUUID()
    clinicianOptions.value = []
    actionVisible.value = true
    optionsLoading.value = true
    try {
      const res = await getCareClient(detail.value.careClientId)
      if (res.code === 0) {
        clinicianOptions.value = (res.data.currentAssignments || []).filter((item) =>
          item.roleType === 'CLINICIAN' && item.status === 'ACTIVE'
        )
        if (!clinicianOptions.value.some((item) => item.assigneeId === actionForm.responsibleAssigneeId)) {
          actionForm.responsibleAssigneeId = clinicianOptions.value.length === 1
            ? clinicianOptions.value[0].assigneeId
            : undefined
        }
      }
    } finally {
      optionsLoading.value = false
    }
  }

  const submitAction = async () => {
    const content = actionForm.content.trim()
    const dueAt = actionForm.dueAt ? new Date(actionForm.dueAt) : null
    if (!content || !actionForm.responsibleAssigneeId || !dueAt) {
      ElMessage.warning('请完整填写内容、责任医护和完成时限')
      return
    }
    if (dueAt.getTime() <= Date.now()) {
      ElMessage.warning('完成时限必须晚于当前时间')
      return
    }
    actionSubmitting.value = true
    try {
      const common = {
        expectedVersion: detail.value.version,
        responsibleAssigneeId: actionForm.responsibleAssigneeId,
        dueAt
      }
      const res = actionMode.value === 'intervene'
        ? await interveneSupervisionReview(detail.value.id, {
          ...common,
          result: content
        }, commandKey.value)
        : await addSupervisorGuidance(detail.value.id, {
          ...common,
          guidance: actionMode.value === 'discussion' ? `讨论安排：${content}` : content
        }, commandKey.value)
      if (res.code === 0) {
        ElMessage.success(actionMode.value === 'intervene' ? '介入已记录' : '指导已追加')
        actionVisible.value = false
        await Promise.all([loadDetail(detail.value.id), loadTable()])
      }
    } finally {
      actionSubmitting.value = false
    }
  }

  const disablePastDate = (date) => {
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    return date.getTime() < today.getTime()
  }
  const reviewStatusLabel = (value) => ({
    PENDING: '待处理',
    GUIDED: '已指导',
    INTERVENED: '已介入',
    COMPLETED: '已完成'
  }[value] || '未说明')
  const reviewStatusTag = (value) => ({
    PENDING: 'warning',
    GUIDED: 'primary',
    INTERVENED: 'success',
    COMPLETED: 'info'
  }[value] || 'info')
  const caseStatusLabel = (value) => ({
    PENDING_ACK: '待确认',
    ACKNOWLEDGED: '已确认',
    HANDLING: '处理中',
    WAITING_CLIENT: '等待用户',
    WAITING_COLLABORATION: '等待责任医护处理',
    WAITING_SUPERVISOR: '等待上级复核',
    RESOLVED: '已解决',
    CLOSED: '已关闭'
  }[value] || '未说明')
  const actionTypeLabel = (value) => ({
    ACKNOWLEDGE: '确认',
    CONTACT: '联系',
    HANDLING: '处置',
    ESCALATE: '转交责任医护',
    GUIDANCE: '上级指导',
    INTERVENE: '上级介入',
    RESOLVE: '解决',
    CLOSE: '关闭',
    REOPEN: '重开'
  }[value] || '其他操作')
  const actorRoleLabel = (value) => ({
    CARE_STEWARD: '健康管家',
    CLINICIAN: '一线医护',
    SUPERVISOR: '上级医师'
  }[value] || '工作人员')
  const formatTimestamp = (value) => value ? formatDate(value) : '-'

  onMounted(async () => {
    await loadTable()
    const attentionCaseId = Number(props.initialAttentionCaseId)
    if (Number.isInteger(attentionCaseId) && attentionCaseId > 0) {
      const review = tableData.value.find((item) => item.attentionCaseId === attentionCaseId)
      await openDetail(review || { attentionCaseId })
    }
  })
</script>
