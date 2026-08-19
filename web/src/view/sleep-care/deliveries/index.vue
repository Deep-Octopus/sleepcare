<template>
  <main class="space-y-4 text-base-text">
    <section class="rounded-xl border border-border bg-container p-5 shadow-card">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <div class="mb-2 flex items-center gap-2 text-sm font-medium text-primary">
            <svg-icon icon="lucide:mail-check" />
            <span>发送记录与结果</span>
          </div>
          <h1 class="text-2xl font-semibold tracking-tight">通知记录</h1>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
            当前不会发送真实短信。系统会记录每次通知处理结果，补发后仍保留之前的记录。
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
            <p class="text-xs text-muted-foreground">结果未知</p>
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
        <el-form-item label="发送状态">
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
            <p class="mt-1 text-xs text-muted-foreground">
              用户编号 {{ scope.row.careClientDisplayCode }}
            </p>
          </template>
        </el-table-column>
        <el-table-column
          label="任务 / 发送次数"
          min-width="150"
        >
          <template #default="scope">
            <p class="text-sm">任务编号 {{ scope.row.taskId }}</p>
            <p class="mt-1 text-xs text-muted-foreground">
              第 {{ scope.row.attemptNo }} 次发送
              <span v-if="scope.row.retryOfAttemptId">· 由之前记录补发</span>
            </p>
          </template>
        </el-table-column>
        <el-table-column
          label="发送方式"
          width="90"
        >
          <template #default="scope">{{ channelLabel(scope.row.channel) }}</template>
        </el-table-column>
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
          label="开始时间"
          min-width="180"
        >
          <template #default="scope">{{ formatDateTime(scope.row.requestedAt) }}</template>
        </el-table-column>
        <el-table-column
          label="受理 / 完成"
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
          label="失败原因"
          min-width="150"
        >
          <template #default="scope">
            <span class="text-xs">{{ failureCodeLabel(scope.row.failureCode) }}</span>
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
              发送详情
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
      title="通知发送详情"
    >
      <template v-if="selected">
        <section class="rounded-xl border border-border bg-muted p-4">
          <div class="flex items-start justify-between gap-3">
            <div>
              <p class="text-xs font-medium text-muted-foreground">发送记录</p>
              <h2 class="mt-2 text-xl font-semibold">第 {{ selected.attemptNo }} 次发送</h2>
              <p class="mt-1 text-sm text-muted-foreground">
                任务编号 {{ selected.taskId }}
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
          title="当前只能确认发送流程已受理，还不能确认已经送达。"
          type="warning"
        />
        <div class="mb-3 mt-6">
          <h3 class="text-lg font-semibold">发送状态记录</h3>
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
              <span v-if="event.failureCode"> · {{ failureCodeLabel(event.failureCode) }}</span>
            </p>
          </el-timeline-item>
        </el-timeline>
      </template>
    </el-drawer>

    <el-dialog
      v-model="resendVisible"
      title="再次发送通知"
      width="min(520px, 94vw)"
    >
      <el-alert
        class="mb-4"
        :closable="false"
        title="补发会新增一条发送记录，当前记录仍会保留。"
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
          确认补发
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
    { label: '待处理', value: 'PENDING' },
    { label: '已进入发送流程', value: 'SUBMITTED_TO_PROVIDER' },
    { label: '发送流程已受理', value: 'ACCEPTED' },
    { label: '已送达', value: 'DELIVERED' },
    { label: '发送失败', value: 'FAILED' },
    { label: '结果未知', value: 'UNKNOWN' }
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

  const statusLabel = (value) => statusOptions.find((item) => item.value === value)?.label || '未说明'
  const statusTagType = (value) => ({
    ACCEPTED: 'primary',
    DELIVERED: 'success',
    FAILED: 'danger',
    UNKNOWN: 'warning'
  }[value] || 'info')
  const eventLabel = (value) => ({
    NotificationRequested: '通知请求已建立',
    NotificationSubmittedToProvider: '已进入发送流程',
    NotificationAccepted: '发送流程已受理',
    NotificationDelivered: '已确认送达',
    NotificationFailed: '本次尝试失败',
    NotificationUnknown: '最终回执未知'
  }[value] || '状态已更新')
  const transitionLabel = (event) => event.fromStatus
    ? `${statusLabel(event.fromStatus)}变为${statusLabel(event.toStatus)}`
    : statusLabel(event.toStatus)
  const channelLabel = (value) => ({
    DEMO: '系统记录'
  }[value] || '未启用外部发送')
  const failureCodeLabel = (value) => ({
    DEMO_REJECTED: '发送流程未受理',
    DEMO_UNKNOWN: '未收到最终结果'
  }[value] || (value ? '原因未说明' : '-'))
  const formatDateTime = (value) => value
    ? new Date(value).toLocaleString('zh-CN', { hour12: false })
    : '-'

  onMounted(loadTable)
</script>
