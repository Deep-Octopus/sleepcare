<template>
  <div>
    <div class="mb-4 rounded-lg border border-border bg-container px-4 py-3">
      <div class="font-semibold">计划任务</div>
      <div class="mt-1 text-sm leading-6">
        查看责任范围内的任务进度、截止情况和复核状态。尚未开放的任务不能提前填写。
      </div>
    </div>

    <div class="gva-search-box">
      <el-form
        :inline="true"
        :model="searchForm"
      >
        <el-form-item label="用户编号">
          <el-input-number
            v-model="searchForm.careClientId"
            :min="1"
            controls-position="right"
            class="w-36"
          />
        </el-form-item>
        <el-form-item label="日序">
          <el-select
            v-model="searchForm.dayCode"
            clearable
            class="w-28"
          >
            <el-option
              v-for="day in dayOptions"
              :key="day"
              :label="dayLabel(day)"
              :value="day"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="执行状态">
          <el-select
            v-model="searchForm.executionStatus"
            clearable
            class="w-36"
          >
            <el-option
              v-for="item in executionOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="时效状态">
          <el-select
            v-model="searchForm.timingStatus"
            clearable
            class="w-36"
          >
            <el-option
              v-for="item in timingOptions"
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
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-table
        v-loading="loading"
        :data="tableData"
        row-key="id"
        empty-text="当前责任或组织范围内暂无计划任务"
      >
        <el-table-column label="任务次序" width="100">
          <template #default="scope">
            <span class="text-base font-semibold text-slate-800">
              {{ dayLabel(scope.row.dayCode) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="康养用户" min-width="180">
          <template #default="scope">
            <div class="font-medium">{{ scope.row.careClientDisplayName }}</div>
            <div class="mt-1 text-xs text-gray-400">
              用户编号 {{ scope.row.careClientDisplayCode }}
            </div>
          </template>
        </el-table-column>
        <el-table-column label="任务" min-width="220">
          <template #default="scope">{{ readableTaskTitle(scope.row.title, scope.row.dayCode) }}</template>
        </el-table-column>
        <el-table-column label="当前进度" min-width="290">
          <template #default="scope">
            <div class="flex flex-wrap gap-1">
              <el-tag :type="executionTagType(scope.row.executionStatus)">
                任务 · {{ statusLabel(scope.row.executionStatus) }}
              </el-tag>
              <el-tag
                :type="timingTagType(scope.row.timingStatus)"
                effect="plain"
              >
                时间 · {{ statusLabel(scope.row.timingStatus) }}
              </el-tag>
              <el-tag
                :type="reviewTagType(scope.row.reviewStatus)"
                effect="plain"
              >
                复核 · {{ statusLabel(scope.row.reviewStatus) }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="开放时间" min-width="170">
          <template #default="scope">
            {{ formatDateTime(scope.row.openAt) }}
          </template>
        </el-table-column>
        <el-table-column label="截止时间" min-width="170">
          <template #default="scope">
            {{ formatDateTime(scope.row.dueAt) }}
          </template>
        </el-table-column>
        <el-table-column label="计划" width="100">
          <template #default="scope">
            <el-tag
              :type="scope.row.planStatus === 'PAUSED' ? 'warning' : 'success'"
              effect="plain"
            >
              {{ statusLabel(scope.row.planStatus) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="160">
          <template #default="scope">
            <el-button
              v-if="btnAuth.viewDetail"
              link
              type="primary"
              @click="openDetail(scope.row.id)"
            >
              详情
            </el-button>
            <el-button
              v-if="btnAuth.recordContact"
              link
              type="warning"
              @click="openContact(scope.row)"
            >
              记录联系
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
    </div>

    <el-drawer
      v-model="detailVisible"
      title="计划任务详情"
      size="720px"
    >
      <div
        v-loading="detailLoading"
        class="min-h-48"
      >
        <template v-if="detail">
          <el-alert
            class="mb-4"
            type="info"
            :closable="false"
            title="这里用于查看任务安排和记录联系情况；填写与提交请从用户任务入口完成。"
          />

          <div class="mb-4 overflow-hidden rounded-lg border border-slate-200 bg-slate-50">
            <div class="flex items-center justify-between bg-slate-900 px-4 py-3 text-white">
              <div class="flex items-center gap-3">
                <span class="text-xl font-semibold">{{ dayLabel(detail.dayCode) }}</span>
                <span class="text-sm text-slate-300">{{ readableTaskTitle(detail.title, detail.dayCode) }}</span>
              </div>
              <span class="text-xs text-slate-400">任务编号 {{ detail.id }}</span>
            </div>
            <div class="grid grid-cols-1 gap-px bg-slate-200 md:grid-cols-3">
              <div class="bg-white px-4 py-3">
                <div class="text-xs text-gray-400">任务状态</div>
                <div class="mt-1 font-semibold">{{ statusLabel(detail.executionStatus) }}</div>
              </div>
              <div class="bg-white px-4 py-3">
                <div class="text-xs text-gray-400">时间状态</div>
                <div class="mt-1 font-semibold">{{ statusLabel(detail.timingStatus) }}</div>
              </div>
              <div class="bg-white px-4 py-3">
                <div class="text-xs text-gray-400">复核状态</div>
                <div class="mt-1 font-semibold">{{ statusLabel(detail.reviewStatus) }}</div>
              </div>
            </div>
          </div>

          <el-descriptions
            :column="2"
            border
          >
            <el-descriptions-item label="康养用户" :span="2">
              {{ detail.careClientDisplayName }}（{{ detail.careClientDisplayCode }}）
            </el-descriptions-item>
            <el-descriptions-item label="执行角色">
              {{ roleLabel(detail.executionRole) }}
            </el-descriptions-item>
            <el-descriptions-item label="计划状态">
              {{ statusLabel(detail.planStatus) }}
            </el-descriptions-item>
            <el-descriptions-item label="开放时间">
              {{ formatDateTime(detail.openAt) }}
            </el-descriptions-item>
            <el-descriptions-item label="截止时间">
              {{ formatDateTime(detail.dueAt) }}
            </el-descriptions-item>
            <el-descriptions-item label="超过截止时间">
              {{ latePolicyLabel(detail.lateSubmissionPolicy) }}
            </el-descriptions-item>
            <el-descriptions-item label="通知安排">
              {{ notificationPolicyLabel(detail.notificationPolicy) }}
            </el-descriptions-item>
            <el-descriptions-item label="填写内容">
              {{ detail.questionnaireVersionId ? '需要填写问卷' : '无需填写问卷' }}
            </el-descriptions-item>
            <el-descriptions-item label="关注规则">
              {{ detail.ruleVersionIds?.length ? '已设置' : '无' }}
            </el-descriptions-item>
            <el-descriptions-item label="复核角色" :span="2">
              {{ detail.reviewRole ? roleLabel(detail.reviewRole) : '无需复核' }}
            </el-descriptions-item>
          </el-descriptions>

          <div class="mb-3 mt-7">
            <h3 class="text-lg font-semibold">状态记录</h3>
          </div>
          <el-empty
            v-if="!detail.timeline?.length"
            description="暂无状态事件"
            :image-size="70"
          />
          <el-timeline v-else>
            <el-timeline-item
              v-for="event in detail.timeline"
              :key="`${event.eventType}-${event.occurredAt}`"
              :timestamp="formatDateTime(event.occurredAt)"
              placement="top"
            >
              <div class="font-medium">{{ eventLabel(event.eventType) }}</div>
              <div class="mt-1 text-sm text-gray-500">
                {{ event.summary }} · {{ sourceLabel(event.source) }}
              </div>
            </el-timeline-item>
          </el-timeline>
        </template>
      </div>
    </el-drawer>

    <el-dialog
      v-model="contactVisible"
      title="追加人工联系记录"
      width="min(560px, 94vw)"
    >
      <el-alert
        class="mb-4"
        :closable="false"
        title="仅记录沟通事实，不填写诊断或医疗结论。该动作不会改变任务执行状态。"
        type="info"
      />
      <el-form
        label-position="top"
        :model="contactForm"
      >
        <el-form-item label="联系渠道">
          <el-select
            v-model="contactForm.channel"
            class="w-full"
          >
            <el-option label="电话" value="PHONE" />
            <el-option label="线下" value="OFFLINE" />
            <el-option label="其它" value="OTHER" />
          </el-select>
        </el-form-item>
        <el-form-item label="发生时间">
          <el-date-picker
            v-model="contactForm.occurredAt"
            class="w-full"
            type="datetime"
          />
        </el-form-item>
        <el-form-item label="联系结果">
          <el-input
            v-model="contactForm.result"
            maxlength="2000"
            placeholder="填写已经发生的沟通结果"
            show-word-limit
            type="textarea"
            :rows="5"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="contactVisible = false">取消</el-button>
        <el-button
          :loading="contactSubmitting"
          type="primary"
          @click="submitContact"
        >
          保存记录
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
  import { onMounted, reactive, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import { useBtnAuth } from '@/utils/btnAuth'
  import { readableTaskTitle } from '@/utils/sleep-care-display'
  import {
    getCareTask,
    getCareTasks,
    recordCareTaskContact
  } from '@/api/sleep-care/care-path'

  defineOptions({ name: 'CareTasks' })

  const props = defineProps({
    initialDetailId: {
      type: [String, Number],
      default: ''
    }
  })

  const btnAuth = useBtnAuth()
  const dayOptions = ['D1', 'D2', 'D3', 'D4', 'D5']
  const executionOptions = [
    { label: '待开放', value: 'SCHEDULED' },
    { label: '已开放', value: 'OPEN' },
    { label: '进行中', value: 'IN_PROGRESS' },
    { label: '已提交', value: 'SUBMITTED' },
    { label: '已取消', value: 'CANCELLED' }
  ]
  const timingOptions = [
    { label: '未开放', value: 'NOT_OPEN' },
    { label: '可按时完成', value: 'WITHIN_WINDOW' },
    { label: '已超过截止时间', value: 'OVERDUE' },
    { label: '已结束', value: 'EXPIRED' }
  ]
  const searchForm = reactive({
    careClientId: undefined,
    dayCode: '',
    executionStatus: '',
    timingStatus: ''
  })
  const page = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const tableData = ref([])
  const loading = ref(false)
  const detailVisible = ref(false)
  const detailLoading = ref(false)
  const detail = ref(null)
  const contactVisible = ref(false)
  const contactSubmitting = ref(false)
  const contactTask = ref(null)
  const contactForm = reactive({
    channel: 'PHONE',
    result: '',
    occurredAt: new Date()
  })

  const loadTable = async () => {
    loading.value = true
    try {
      const res = await getCareTasks({
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
    searchForm.careClientId = undefined
    searchForm.dayCode = ''
    searchForm.executionStatus = ''
    searchForm.timingStatus = ''
    search()
  }

  const handleSizeChange = () => {
    page.value = 1
    loadTable()
  }

  const openDetail = async (id) => {
    detailVisible.value = true
    detailLoading.value = true
    detail.value = null
    try {
      const res = await getCareTask(id)
      if (res.code === 0) {
        detail.value = res.data
      }
    } finally {
      detailLoading.value = false
    }
  }

  const openContact = (task) => {
    contactTask.value = task
    contactForm.channel = 'PHONE'
    contactForm.result = ''
    contactForm.occurredAt = new Date()
    contactVisible.value = true
  }

  const submitContact = async () => {
    const result = contactForm.result.trim()
    if (!contactTask.value || !contactForm.occurredAt || !result) {
      ElMessage.warning('请完整填写联系渠道、时间和结果')
      return
    }
    contactSubmitting.value = true
    try {
      const res = await recordCareTaskContact(contactTask.value.id, {
        expectedVersion: contactTask.value.version,
        channel: contactForm.channel,
        result,
        occurredAt: new Date(contactForm.occurredAt).toISOString()
      })
      if (res.code === 0) {
        ElMessage.success('联系记录已追加')
        contactVisible.value = false
        await loadTable()
      }
    } finally {
      contactSubmitting.value = false
    }
  }

  const statusLabels = {
    SCHEDULED: '待开放',
    OPEN: '已开放',
    IN_PROGRESS: '进行中',
    SUBMITTED: '已提交',
    CANCELLED: '已取消',
    NOT_OPEN: '未开放',
    WITHIN_WINDOW: '可按时完成',
    OVERDUE: '已超过截止时间',
    EXPIRED: '已结束',
    NOT_READY: '尚不可复核',
    NOT_REQUIRED: '无需复核',
    PENDING: '待复核',
    REVIEWING: '复核中',
    REVIEWED: '已复核',
    RETURNED: '已退回',
    ACTIVE: '进行中',
    PAUSED: '已暂停'
  }

  const statusLabel = (value) => statusLabels[value] || '未说明'
  const executionTagType = (value) => ({
    OPEN: 'success',
    SUBMITTED: 'primary',
    CANCELLED: 'info'
  }[value] || 'info')
  const timingTagType = (value) => ({
    WITHIN_WINDOW: 'success',
    OVERDUE: 'warning',
    EXPIRED: 'danger'
  }[value] || 'info')
  const reviewTagType = (value) => ({
    PENDING: 'warning',
    REVIEWING: 'primary',
    REVIEWED: 'success',
    RETURNED: 'danger'
  }[value] || 'info')
  const roleLabel = (value) => ({
    CARE_CLIENT: '康养用户',
    CARE_STEWARD: '健康管家',
    CLINICIAN: '一线医护'
  }[value] || '未说明')
  const sourceLabel = (value) => ({
    SYSTEM: '系统调度',
    CARE_STEWARD: '健康管家',
    CLINICIAN: '一线医护',
    SUPERVISOR: '上级医师'
  }[value] || '工作人员')
  const eventLabel = (value) => ({
    CarePlanStarted: '计划已启动',
    CarePlanPaused: '计划已暂停',
    CarePlanResumed: '计划已恢复',
    TaskOpened: '任务已开放',
    TaskContactRecorded: '已记录人工联系'
  }[value] || '状态已更新')
  const dayLabel = (value) => `第${String(value || '').replace(/^D/, '') || '-'}次`
  const latePolicyLabel = (value) => ({
    DENY: '截止后不能提交'
  }[value] || '请按页面提示处理')
  const notificationPolicyLabel = (value) => ({
    DISABLED: '系统不会自动通知'
  }[value] || '按现有方式联系用户')
  const formatDateTime = (value) => value
    ? new Date(value).toLocaleString('zh-CN', { hour12: false })
    : '-'

  onMounted(async () => {
    await loadTable()
    const detailId = Number(props.initialDetailId)
    if (Number.isInteger(detailId) && detailId > 0) {
      await openDetail(detailId)
    }
  })
</script>
