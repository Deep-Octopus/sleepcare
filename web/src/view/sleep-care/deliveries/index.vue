<template>
  <main class="space-y-4 text-base-text">
    <section class="rounded-xl border border-border bg-container p-5 shadow-card">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <div class="mb-2 flex items-center gap-2 text-sm font-medium text-primary">
            <svg-icon icon="lucide:mail-check" />
            <span>发送事实与回执证据</span>
          </div>
          <h1 class="text-2xl font-semibold tracking-tight">通知记录</h1>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
            通道受理不代表已送达。每次补发都会形成新的尝试，旧的失败或未知事实始终保留。
          </p>
        </div>
        <div class="grid min-w-72 grid-cols-3 gap-2">
          <article class="rounded-lg border border-border bg-muted p-3">
            <p class="text-xs text-muted-foreground">记录数</p>
            <p class="mt-1 text-xl font-semibold tabular-nums">{{ total }}</p>
          </article>
          <article class="rounded-lg border border-border bg-muted p-3">
            <p class="text-xs text-muted-foreground">本页失败</p>
            <p class="mt-1 text-xl font-semibold tabular-nums text-danger">{{ failedCount }}</p>
          </article>
          <article class="rounded-lg border border-border bg-muted p-3">
            <p class="text-xs text-muted-foreground">本页未知</p>
            <p class="mt-1 text-xl font-semibold tabular-nums text-warning">{{ unknownCount }}</p>
          </article>
        </div>
      </div>
    </section>

    <section class="gva-search-box">
      <el-form
        :inline="true"
        :model="searchForm"
      >
        <el-form-item label="尝试状态">
          <el-select
            v-model="searchForm.status"
            clearable
            class="w-52"
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
        <el-form-item>
          <el-button
            type="primary"
            @click="search"
          >
            查询
          </el-button>
          <el-button @click="resetSearch">重置</el-button>
          <el-button
            :loading="loading"
            @click="loadTable"
          >
            刷新
          </el-button>
        </el-form-item>
      </el-form>
    </section>

    <section class="gva-table-box">
      <el-table
        v-loading="loading"
        :data="tableData"
        empty-text="当前责任或管理范围内暂无通知记录"
        row-key="id"
      >
        <el-table-column
          label="康养用户"
          min-width="190"
        >
          <template #default="scope">
            <p class="font-medium">{{ scope.row.careClientDisplayName }}</p>
            <p class="mt-1 font-mono text-xs text-muted-foreground">
              {{ scope.row.careClientDisplayCode }} · #{{ scope.row.careClientId }}
            </p>
          </template>
        </el-table-column>
        <el-table-column
          label="任务 / 尝试"
          min-width="150"
        >
          <template #default="scope">
            <p class="font-mono text-sm">TASK #{{ scope.row.taskId }}</p>
            <p class="mt-1 text-xs text-muted-foreground">
              Attempt {{ scope.row.attemptNo }}
              <span v-if="scope.row.retryOfAttemptId">· 补发自 #{{ scope.row.retryOfAttemptId }}</span>
            </p>
          </template>
        </el-table-column>
        <el-table-column
          label="通道"
          width="90"
          prop="channel"
        />
        <el-table-column
          label="当前状态"
          min-width="150"
        >
          <template #default="scope">
            <el-tag
              :type="statusTagType(scope.row.status)"
              effect="plain"
            >
              {{ statusLabel(scope.row.status) }}
            </el-tag>
            <p
              v-if="scope.row.status === 'ACCEPTED'"
              class="mt-1 text-xs text-warning"
            >
              已受理，尚未确认送达
            </p>
          </template>
        </el-table-column>
        <el-table-column
          label="请求时间"
          min-width="180"
        >
          <template #default="scope">{{ formatDateTime(scope.row.requestedAt) }}</template>
        </el-table-column>
        <el-table-column
          label="受理 / 终结"
          min-width="220"
        >
          <template #default="scope">
            <p>受理：{{ formatDateTime(scope.row.acceptedAt) }}</p>
            <p class="mt-1 text-xs text-muted-foreground">
              终结：{{ formatDateTime(scope.row.finalizedAt) }}
            </p>
          </template>
        </el-table-column>
        <el-table-column
          label="失败码"
          min-width="150"
        >
          <template #default="scope">
            <span class="font-mono text-xs">{{ scope.row.failureCode || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column
          fixed="right"
          label="操作"
          width="160"
        >
          <template #default="scope">
            <el-button
              link
              type="primary"
              @click="openEvidence(scope.row)"
            >
              回执证据
            </el-button>
            <el-button
              v-if="btnAuth.resendNotice && canResend(scope.row)"
              link
              type="warning"
              @click="openResend(scope.row)"
            >
              补发
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
      v-model="evidenceVisible"
      size="min(680px, 100%)"
      title="通知回执证据"
    >
      <template v-if="selected">
        <section class="rounded-xl border border-border bg-muted p-4">
          <div class="flex items-start justify-between gap-3">
            <div>
              <p class="text-xs font-medium tracking-widest text-muted-foreground">DELIVERY ATTEMPT</p>
              <h2 class="mt-2 text-xl font-semibold">Attempt {{ selected.attemptNo }}</h2>
              <p class="mt-1 text-sm text-muted-foreground">
                任务 #{{ selected.taskId }} · 请求 #{{ selected.notificationRequestId }}
              </p>
            </div>
            <el-tag
              :type="statusTagType(selected.status)"
              effect="plain"
            >
              {{ statusLabel(selected.status) }}
            </el-tag>
          </div>
        </section>
        <el-alert
          v-if="selected.status === 'ACCEPTED'"
          class="mt-4"
          :closable="false"
          title="当前证据仅能证明通道已受理，不能标记为已送达。"
          type="warning"
        />
        <div class="mb-3 mt-6">
          <p class="text-xs font-medium tracking-widest text-muted-foreground">RECEIPT TIMELINE</p>
          <h3 class="mt-1 text-lg font-semibold">标准状态时间线</h3>
        </div>
        <el-timeline>
          <el-timeline-item
            v-for="event in selected.events || []"
            :key="event.id"
            :timestamp="formatDateTime(event.occurredAt)"
            placement="top"
          >
            <p class="font-medium">{{ eventLabel(event.eventType) }}</p>
            <p class="mt-1 text-sm text-muted-foreground">
              {{ transitionLabel(event) }}
              <span v-if="event.failureCode"> · {{ event.failureCode }}</span>
            </p>
          </el-timeline-item>
        </el-timeline>
      </template>
    </el-drawer>

    <el-dialog
      v-model="resendVisible"
      title="创建补发尝试"
      width="min(520px, 94vw)"
    >
      <el-alert
        class="mb-4"
        :closable="false"
        title="补发会创建新的 attempt；当前记录不会被覆盖。"
        type="warning"
      />
      <el-form
        label-position="top"
        :model="resendForm"
      >
        <el-form-item label="补发原因">
          <el-input
            v-model="resendForm.reason"
            maxlength="1000"
            placeholder="说明为什么需要再次尝试"
            show-word-limit
            type="textarea"
            :rows="4"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resendVisible = false">取消</el-button>
        <el-button
          :loading="submitting"
          type="primary"
          @click="submitResend"
        >
          创建新尝试
        </el-button>
      </template>
    </el-dialog>
  </main>
</template>

<script setup>
  import { computed, onMounted, reactive, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import { getCareDeliveries, resendCareDelivery } from '@/api/sleep-care/deliveries'
  import { useBtnAuth } from '@/utils/btnAuth'

  defineOptions({ name: 'CareDeliveries' })

  const btnAuth = useBtnAuth()
  const statusOptions = [
    { label: '待提交', value: 'PENDING' },
    { label: '已提交通道', value: 'SUBMITTED_TO_PROVIDER' },
    { label: '通道已受理', value: 'ACCEPTED' },
    { label: '已送达', value: 'DELIVERED' },
    { label: '失败', value: 'FAILED' },
    { label: '回执未知', value: 'UNKNOWN' }
  ]
  const searchForm = reactive({ status: '' })
  const page = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const tableData = ref([])
  const loading = ref(false)
  const evidenceVisible = ref(false)
  const resendVisible = ref(false)
  const submitting = ref(false)
  const selected = ref(null)
  const resendForm = reactive({ reason: '' })
  const failedCount = computed(() => tableData.value.filter((item) => item.status === 'FAILED').length)
  const unknownCount = computed(() => tableData.value.filter((item) => item.status === 'UNKNOWN').length)

  const loadTable = async () => {
    loading.value = true
    try {
      const res = await getCareDeliveries({
        page: page.value,
        pageSize: pageSize.value,
        status: searchForm.status || undefined
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
    search()
  }

  const handleSizeChange = () => {
    page.value = 1
    loadTable()
  }

  const openEvidence = (row) => {
    selected.value = row
    evidenceVisible.value = true
  }

  const canResend = (row) => ['FAILED', 'UNKNOWN'].includes(row.status)

  const openResend = (row) => {
    selected.value = row
    resendForm.reason = ''
    resendVisible.value = true
  }

  const submitResend = async () => {
    const reason = resendForm.reason.trim()
    if (!selected.value || !reason) {
      ElMessage.warning('请填写补发原因')
      return
    }
    submitting.value = true
    try {
      const res = await resendCareDelivery(selected.value.id, {
        expectedVersion: selected.value.version,
        reason
      })
      if (res.code === 0) {
        ElMessage.success('新的通知尝试已创建')
        resendVisible.value = false
        await loadTable()
      }
    } finally {
      submitting.value = false
    }
  }

  const statusLabel = (value) => statusOptions.find((item) => item.value === value)?.label || value
  const statusTagType = (value) => ({
    ACCEPTED: 'primary',
    DELIVERED: 'success',
    FAILED: 'danger',
    UNKNOWN: 'warning'
  }[value] || 'info')
  const eventLabel = (value) => ({
    NotificationRequested: '通知请求已建立',
    NotificationSubmittedToProvider: '已提交测试通道',
    NotificationAccepted: '测试通道已受理',
    NotificationDelivered: '已确认送达',
    NotificationFailed: '本次尝试失败',
    NotificationUnknown: '最终回执未知'
  }[value] || value)
  const transitionLabel = (event) => event.fromStatus
    ? `${event.fromStatus} → ${event.toStatus}`
    : event.toStatus
  const formatDateTime = (value) => value
    ? new Date(value).toLocaleString('zh-CN', { hour12: false })
    : '-'

  onMounted(loadTable)
</script>
