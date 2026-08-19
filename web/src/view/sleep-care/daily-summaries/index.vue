<template>
  <main class="space-y-4 text-base-text">
    <section class="rounded-xl border border-border bg-container p-5 shadow-card">
      <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div>
          <div class="mb-2 flex items-center gap-2 text-sm font-medium text-primary">
            <svg-icon icon="lucide:chart-no-axes-combined" />
            <span>管理范围总览</span>
          </div>
          <h1 class="text-2xl font-semibold tracking-tight">运营概览与每日汇总</h1>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
            今日数据实时复算，历史记录按版本保存。当前结果只用于固定流程验证，不作为正式经营或人员评价口径。
          </p>
        </div>
        <el-button
          :loading="loading || dashboardLoading"
          @click="refreshAll"
        >
          <svg-icon
            class="mr-1"
            icon="lucide:refresh-cw"
          />
          刷新
        </el-button>
      </div>
    </section>

    <section
      v-if="dashboard"
      class="rounded-xl border border-border bg-container p-5 shadow-card"
    >
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div class="max-w-3xl">
          <div class="flex flex-wrap items-center gap-2">
            <el-tag
              type="warning"
              effect="plain"
            >
              正式口径保持关闭
            </el-tag>
            <el-tag effect="plain">Asia/Shanghai 自然日</el-tag>
          </div>
          <p class="mt-3 text-sm leading-6 text-muted-foreground">
            看板按机构汇总，不计算人员排行或团队转交归属。历史修正只会追加新版本，不会覆盖旧记录。
          </p>
          <p
            v-if="missingDateSummary"
            class="mt-2 text-xs leading-5 text-warning"
          >
            近 7 日待补齐：{{ missingDateSummary }}
          </p>
        </div>
        <dl class="grid min-w-72 grid-cols-3 gap-2">
          <div class="rounded-lg border border-border bg-muted p-3">
            <dt class="text-xs text-muted-foreground">已保存日期</dt>
            <dd class="mt-1 text-lg font-semibold tabular-nums">
              {{ dashboard.coverage.snapshotDays }}
            </dd>
          </div>
          <div class="rounded-lg border border-border bg-muted p-3">
            <dt class="text-xs text-muted-foreground">待补日期</dt>
            <dd class="mt-1 text-lg font-semibold tabular-nums">
              {{ dashboard.coverage.missingDates.length }}
            </dd>
          </div>
          <div class="rounded-lg border border-border bg-muted p-3">
            <dt class="text-xs text-muted-foreground">已有修正</dt>
            <dd class="mt-1 text-lg font-semibold tabular-nums">
              {{ dashboard.coverage.revisedDates }}
            </dd>
          </div>
        </dl>
      </div>
    </section>

    <section
      v-if="realtimeSummary"
      class="grid grid-cols-2 gap-3 lg:grid-cols-4 xl:grid-cols-6"
    >
      <article
        v-for="metric in realtimeMetrics"
        :key="metric.key"
        class="rounded-xl border border-border bg-container p-4 shadow-card"
      >
        <div class="flex items-center justify-between gap-2">
          <span class="text-xs font-medium text-muted-foreground">{{ metric.label }}</span>
          <svg-icon
            class="text-primary"
            :icon="metric.icon"
          />
        </div>
        <p class="mt-3 text-2xl font-semibold tabular-nums">{{ metric.value }}</p>
      </article>
    </section>

    <section class="gva-search-box">
      <el-form
        :inline="true"
        :model="searchForm"
      >
        <el-form-item label="业务日期">
          <el-date-picker
            v-model="searchForm.businessDate"
            :disabled-date="disableFutureDate"
            placeholder="全部日期"
            type="date"
            value-format="YYYY-MM-DD"
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
        empty-text="当前管理范围内暂无汇总记录"
        row-key="summaryKey"
      >
        <el-table-column
          label="业务日期"
          min-width="130"
          prop="businessDate"
        />
        <el-table-column
          label="记录类型"
          min-width="150"
        >
          <template #default="scope">
            <el-tag
              :type="scope.row.summaryType === 'REALTIME_PREVIEW' ? 'primary' : 'info'"
              effect="plain"
            >
              {{ summaryTypeLabel(scope.row.summaryType) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column
          label="版本"
          min-width="90"
        >
          <template #default="scope">
            {{ scope.row.version ? `v${scope.row.version}` : '-' }}
          </template>
        </el-table-column>
        <el-table-column
          label="生成方式"
          min-width="120"
        >
          <template #default="scope">
            {{ generationTypeLabel(scope.row.generationType) }}
          </template>
        </el-table-column>
        <el-table-column
          label="服务人数"
          min-width="100"
          prop="servedClients"
        />
        <el-table-column
          label="应执行任务"
          min-width="110"
          prop="dueTasks"
        />
        <el-table-column
          label="已提交任务"
          min-width="110"
          prop="submittedTasks"
        />
        <el-table-column
          label="逾期任务"
          min-width="100"
          prop="overdueTasks"
        />
        <el-table-column
          label="通知异常"
          min-width="100"
          prop="deliveryIssues"
        />
        <el-table-column
          label="未关闭事项"
          min-width="110"
          prop="openAttentionCases"
        />
        <el-table-column
          label="未结咨询"
          min-width="100"
          prop="openConsultations"
        />
        <el-table-column
          label="活动待办"
          min-width="100"
          prop="openTodos"
        />
        <el-table-column
          label="待上级复核"
          min-width="120"
          prop="reviewRequired"
        />
        <el-table-column
          fixed="right"
          label="操作"
          width="90"
        >
          <template #default="scope">
            <el-button
              v-if="btnAuth.viewDetail && scope.row.id"
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
      size="min(960px, 100%)"
      title="历史汇总详情"
    >
      <div
        v-loading="detailLoading"
        class="min-h-56"
      >
        <template v-if="detail">
          <section class="rounded-xl border border-border bg-muted p-4">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <p class="text-xs font-medium text-muted-foreground">每日汇总</p>
                <h2 class="mt-2 text-xl font-semibold">
                  {{ detail.businessDate }} · v{{ detail.version }}
                </h2>
                <p class="mt-2 text-sm text-muted-foreground">
                  {{ generationTypeLabel(detail.generationType) }} · 统计截止 {{ formatTimestamp(detail.sourceCutoffAt) }}
                </p>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <el-tag
                  :type="detail.isLatest ? 'success' : 'info'"
                  effect="plain"
                >
                  {{ detail.isLatest ? '当前最新版本' : '历史旧版本' }}
                </el-tag>
                <el-button
                  v-if="btnAuth.revise && detail.isLatest"
                  type="primary"
                  @click="openRevisionDialog"
                >
                  追加修正版
                </el-button>
              </div>
            </div>
          </section>

          <section
            v-if="detail.generationType === 'CORRECTION'"
            class="mt-5 rounded-xl border border-border bg-container p-4"
          >
            <h3 class="font-semibold">本版修正说明</h3>
            <p class="mt-2 text-sm leading-6 text-muted-foreground">
              {{ detail.correctionReason }}
            </p>
            <div
              v-if="detail.revisionChanges?.length"
              class="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2"
            >
              <div
                v-for="change in detail.revisionChanges"
                :key="change.key"
                class="flex items-center justify-between gap-3 rounded-lg bg-muted px-3 py-2 text-sm"
              >
                <span>{{ metricLabel(change.key) }}</span>
                <span class="font-medium tabular-nums">{{ change.before }} → {{ change.after }}</span>
              </div>
            </div>
            <p
              v-if="detail.focusCasesChanged"
              class="mt-3 text-xs text-muted-foreground"
            >
              本版重点事项快照也发生了变化。
            </p>
          </section>

          <section class="mt-5 grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-4">
            <article
              v-for="metric in detailMetrics"
              :key="metric.key"
              class="rounded-lg border border-border bg-container p-4"
            >
              <p class="text-xs text-muted-foreground">{{ metric.label }}</p>
              <p class="mt-2 text-xl font-semibold tabular-nums">{{ metric.value }}</p>
            </article>
          </section>

          <section class="mt-7">
            <div class="mb-3">
              <h3 class="text-lg font-semibold">当日重点事项</h3>
            </div>
            <el-table
              :data="detail.focusCases || []"
              empty-text="该记录暂无重点事项"
              row-key="id"
            >
              <el-table-column
                label="事项"
                min-width="100"
              >
                <template #default="scope">编号 {{ scope.row.id }}</template>
              </el-table-column>
              <el-table-column
                label="关联对象"
                min-width="130"
              >
                <template #default="scope">用户 {{ scope.row.careClientId }}</template>
              </el-table-column>
              <el-table-column
                label="状态"
                min-width="150"
              >
                <template #default="scope">
                  <el-tag effect="plain">{{ caseStatusLabel(scope.row.status) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column
                label="关注程度"
                min-width="140"
              >
                <template #default>需要关注</template>
              </el-table-column>
              <el-table-column
                label="触发摘要"
                min-width="280"
              >
                <template #default="scope">{{ readableAttentionReason(scope.row.reasonSummary) }}</template>
              </el-table-column>
              <el-table-column
                label="打开时间"
                min-width="180"
              >
                <template #default="scope">{{ formatTimestamp(scope.row.openedAt) }}</template>
              </el-table-column>
            </el-table>
          </section>
        </template>
      </div>
    </el-drawer>

    <el-dialog
      v-model="revisionVisible"
      title="追加历史修正版"
      width="min(520px, 92vw)"
    >
      <el-alert
        :closable="false"
        show-icon
        title="系统会重新读取原始记录并生成新版本，旧版本不会被覆盖。"
        type="info"
      />
      <el-form
        class="mt-4"
        label-position="top"
      >
        <el-form-item label="事实性修正原因">
          <el-input
            v-model="revisionForm.reason"
            maxlength="1000"
            placeholder="请说明已补录或更正的原始记录"
            show-word-limit
            type="textarea"
            :rows="4"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="revisionVisible = false">取消</el-button>
        <el-button
          :loading="revisionSubmitting"
          type="primary"
          @click="submitRevision"
        >
          重新复算并追加
        </el-button>
      </template>
    </el-dialog>
  </main>
</template>

<script setup>
  import { computed, onMounted, reactive, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    getDailySummaries,
    getDailySummary,
    getOperationsDashboard,
    reviseDailySummary
  } from '@/api/sleep-care/supervision'
  import { useBtnAuth } from '@/utils/btnAuth'
  import { formatDate } from '@/utils/format'
  import { readableAttentionReason } from '@/utils/sleep-care-display'

  defineOptions({ name: 'CareDailySummaries' })

  const btnAuth = useBtnAuth()
  const searchForm = reactive({
    businessDate: ''
  })
  const revisionForm = reactive({
    reason: ''
  })
  const page = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const tableData = ref([])
  const loading = ref(false)
  const dashboardLoading = ref(false)
  const dashboard = ref(null)
  const detailVisible = ref(false)
  const detailLoading = ref(false)
  const detail = ref(null)
  const revisionVisible = ref(false)
  const revisionSubmitting = ref(false)
  const revisionKey = ref('')

  const metricDefinitions = [
    { key: 'servedClients', label: '服务人数', icon: 'lucide:users' },
    { key: 'dueTasks', label: '应执行任务', icon: 'lucide:list-checks' },
    { key: 'submittedTasks', label: '已提交任务', icon: 'lucide:clipboard-check' },
    { key: 'overdueTasks', label: '逾期任务', icon: 'lucide:clock-alert' },
    { key: 'deliveryIssues', label: '通知异常', icon: 'lucide:mail-warning' },
    { key: 'openAttentionCases', label: '未关闭事项', icon: 'lucide:circle-alert' },
    { key: 'resolvedAttentionCases', label: '当日解决', icon: 'lucide:circle-check' },
    { key: 'consultationsOpened', label: '新增咨询', icon: 'lucide:message-square-plus' },
    { key: 'consultationsClosed', label: '关闭咨询', icon: 'lucide:message-square-check' },
    { key: 'openConsultations', label: '未结咨询', icon: 'lucide:messages-square' },
    { key: 'openTodos', label: '活动待办', icon: 'lucide:list-todo' },
    { key: 'reviewRequired', label: '待上级复核', icon: 'lucide:user-round-check' }
  ]
  const realtimeSummary = computed(() => dashboard.value?.current || null)
  const realtimeMetrics = computed(() => buildMetrics(realtimeSummary.value))
  const detailMetrics = computed(() => buildMetrics(detail.value))
  const missingDateSummary = computed(() => {
    const dates = dashboard.value?.coverage?.missingDates || []
    if (!dates.length) return ''
    const visible = dates.slice(0, 3).join('、')
    return dates.length > 3 ? `${visible} 等 ${dates.length} 天` : visible
  })

  const buildMetrics = (summary) => metricDefinitions.map((item) => ({
    ...item,
    value: summary?.[item.key] ?? 0
  }))

  const loadDashboard = async () => {
    dashboardLoading.value = true
    try {
      const res = await getOperationsDashboard({ days: 7 })
      if (res.code === 0) {
        dashboard.value = res.data
      }
    } finally {
      dashboardLoading.value = false
    }
  }

  const loadTable = async () => {
    loading.value = true
    try {
      const res = await getDailySummaries({
        page: page.value,
        pageSize: pageSize.value,
        businessDate: searchForm.businessDate || undefined
      })
      if (res.code === 0) {
        tableData.value = (res.data.list || []).map((item) => ({
          ...item,
          summaryKey: `${item.summaryType}-${item.id}-${item.businessDate}-${item.version || 0}`
        }))
        total.value = res.data.total || 0
      }
    } finally {
      loading.value = false
    }
  }

  const refreshAll = () => Promise.all([loadDashboard(), loadTable()])

  const search = () => {
    page.value = 1
    loadTable()
  }

  const resetSearch = () => {
    searchForm.businessDate = ''
    search()
  }

  const handleSizeChange = () => {
    page.value = 1
    loadTable()
  }

  const openDetail = async (id) => {
    detail.value = null
    detailVisible.value = true
    detailLoading.value = true
    try {
      const res = await getDailySummary(id)
      if (res.code === 0) {
        detail.value = res.data
      }
    } finally {
      detailLoading.value = false
    }
  }

  const openRevisionDialog = () => {
    revisionForm.reason = ''
    revisionKey.value = crypto.randomUUID()
    revisionVisible.value = true
  }

  const submitRevision = async () => {
    const reason = revisionForm.reason.trim()
    if (!reason) {
      ElMessage.warning('请填写事实性修正原因')
      return
    }
    revisionSubmitting.value = true
    try {
      const res = await reviseDailySummary(
        detail.value.id,
        {
          expectedVersion: detail.value.version,
          reason
        },
        revisionKey.value
      )
      if (res.code === 0) {
        detail.value = res.data
        revisionVisible.value = false
        ElMessage.success('历史修正版已追加')
        await refreshAll()
      }
    } finally {
      revisionSubmitting.value = false
    }
  }

  const disableFutureDate = (date) => {
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    return date.getTime() > today.getTime()
  }
  const summaryTypeLabel = (value) => value === 'REALTIME_PREVIEW' ? '实时数据' : '已保存记录'
  const generationTypeLabel = (value) => ({
    SCHEDULED: '每日自动生成',
    CORRECTION: '历史修正版',
    SYSTEM_RECOMPUTE: '系统复算',
    LEGACY: '早期版本'
  }[value] || (value ? '已保存' : '-'))
  const metricLabel = (key) => metricDefinitions.find((item) => item.key === key)?.label || '汇总项'
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
  const formatTimestamp = (value) => value ? formatDate(value) : '-'

  onMounted(refreshAll)
</script>
