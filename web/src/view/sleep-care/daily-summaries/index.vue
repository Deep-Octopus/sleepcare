<template>
  <main class="space-y-4 text-base-text">
    <section class="rounded-xl border border-border bg-container p-5 shadow-card">
      <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div>
          <div class="mb-2 flex items-center gap-2 text-sm font-medium text-primary">
            <svg-icon icon="lucide:chart-no-axes-combined" />
            <span>管理范围总览</span>
          </div>
          <h1 class="text-2xl font-semibold tracking-tight">每日汇总</h1>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
            今日数据会随处理进度更新；每天的数据会保存，方便后续查看。汇总结果仅用于流程管理。
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

    <section
      v-if="realtimeSummary"
      class="grid grid-cols-2 gap-3 lg:grid-cols-4 xl:grid-cols-7"
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
          label="送达异常"
          min-width="100"
          prop="deliveryIssues"
        />
        <el-table-column
          label="未关闭事项"
          min-width="110"
          prop="openAttentionCases"
        />
        <el-table-column
          label="当日解决"
          min-width="100"
          prop="resolvedAttentionCases"
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
      size="min(860px, 100%)"
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
                <h2 class="mt-2 text-xl font-semibold">{{ detail.businessDate }}</h2>
                <p class="mt-2 text-sm text-muted-foreground">已保存记录</p>
              </div>
              <el-tag effect="plain">已保存</el-tag>
            </div>
          </section>

          <section class="mt-5 grid grid-cols-2 gap-3 md:grid-cols-4">
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
              <el-table-column label="关注程度" min-width="140">
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
  </main>
</template>

<script setup>
  import { computed, onMounted, reactive, ref } from 'vue'
  import { getDailySummaries, getDailySummary } from '@/api/sleep-care/supervision'
  import { useBtnAuth } from '@/utils/btnAuth'
  import { formatDate } from '@/utils/format'
  import { readableAttentionReason } from '@/utils/sleep-care-display'

  defineOptions({ name: 'CareDailySummaries' })

  const btnAuth = useBtnAuth()
  const searchForm = reactive({
    businessDate: ''
  })
  const page = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const tableData = ref([])
  const loading = ref(false)
  const detailVisible = ref(false)
  const detailLoading = ref(false)
  const detail = ref(null)

  const metricDefinitions = [
    { key: 'servedClients', label: '服务人数', icon: 'lucide:users' },
    { key: 'dueTasks', label: '应执行任务', icon: 'lucide:list-checks' },
    { key: 'submittedTasks', label: '已提交任务', icon: 'lucide:clipboard-check' },
    { key: 'deliveryIssues', label: '通知异常', icon: 'lucide:mail-warning' },
    { key: 'openAttentionCases', label: '未关闭事项', icon: 'lucide:circle-alert' },
    { key: 'resolvedAttentionCases', label: '当日解决', icon: 'lucide:circle-check' },
    { key: 'reviewRequired', label: '待上级复核', icon: 'lucide:user-round-check' }
  ]
  const realtimeSummary = computed(() =>
    tableData.value.find((item) => item.summaryType === 'REALTIME_PREVIEW') || null
  )
  const realtimeMetrics = computed(() => buildMetrics(realtimeSummary.value))
  const detailMetrics = computed(() => buildMetrics(detail.value))

  const buildMetrics = (summary) => metricDefinitions.map((item) => ({
    ...item,
    value: summary?.[item.key] ?? 0
  }))

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

  const disableFutureDate = (date) => {
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    return date.getTime() > today.getTime()
  }
  const summaryTypeLabel = (value) => value === 'REALTIME_PREVIEW' ? '实时数据' : '已保存记录'
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

  onMounted(loadTable)
</script>
