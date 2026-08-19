<template>
  <section
    v-loading="loading"
    class="mt-6 rounded-lg border border-amber-200 bg-amber-50/50 p-4"
  >
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h3 class="font-semibold text-slate-900">数据授权与生命周期门禁</h3>
        <p class="mt-1 text-sm leading-6 text-slate-600">
          这里只展示治理准备状态和待政策请求，不执行导出、更正、限制或删除。
        </p>
      </div>
      <div class="flex flex-wrap gap-2">
        <el-tag type="danger" effect="plain">真实数据未启用</el-tag>
        <el-tag type="warning" effect="plain">正式授权未启用</el-tag>
        <el-tag type="info" effect="plain">数据处置未启用</el-tag>
      </div>
    </div>

    <template v-if="readiness">
      <el-alert
        class="mt-4"
        :type="readiness.requestRecordingEnabled ? 'warning' : 'info'"
        :closable="false"
        :title="readinessMessage"
      />

      <div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-2">
        <div class="rounded-md border border-slate-200 bg-white p-3">
          <div class="mb-2 text-sm font-semibold text-slate-800">正式同意内容</div>
          <div class="flex flex-col gap-2">
            <div
              v-for="item in readiness.consentRequirements"
              :key="item.consentType"
              class="flex items-center justify-between gap-3 text-sm"
            >
              <span>{{ consentRequirementLabel(item.consentType) }}</span>
              <el-tag
                :type="item.contentReviewed ? 'success' : 'info'"
                size="small"
                effect="plain"
              >
                {{ consentRequirementStatus(item) }}
              </el-tag>
            </div>
          </div>
        </div>

        <div class="rounded-md border border-slate-200 bg-white p-3">
          <div class="mb-2 flex items-center justify-between gap-3 text-sm font-semibold text-slate-800">
            <span>配套治理规则</span>
            <span class="font-normal text-slate-500">
              {{ reviewedGateCount }}/{{ readiness.reviewGates.length }} 已评审
            </span>
          </div>
          <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
            <div
              v-for="gate in readiness.reviewGates"
              :key="gate.key"
              class="flex items-center justify-between gap-2 text-sm"
            >
              <span>{{ reviewGateLabel(gate.key) }}</span>
              <el-tag
                :type="gate.reviewed ? 'success' : 'info'"
                size="small"
                effect="plain"
              >
                {{ gate.reviewed ? '已评审' : '待确认' }}
              </el-tag>
            </div>
          </div>
        </div>
      </div>

      <div class="mb-3 mt-5 flex flex-wrap items-center justify-between gap-3">
        <div>
          <div class="font-semibold text-slate-900">生命周期请求台账</div>
          <div class="mt-1 text-xs text-slate-500">所有记录均保持“待政策确认”，且不可执行。</div>
        </div>
        <el-button
          v-if="canRecord && readiness.requestRecordingEnabled"
          type="primary"
          @click="openRequestDialog"
        >
          记录请求
        </el-button>
      </div>

      <el-table
        :data="requests"
        size="small"
        empty-text="暂无生命周期请求记录"
      >
        <el-table-column label="请求类型" min-width="120">
          <template #default="scope">
            {{ requestTypeLabel(scope.row.requestType) }}
          </template>
        </el-table-column>
        <el-table-column label="请求时间" min-width="170">
          <template #default="scope">
            {{ formatDateTime(scope.row.requestedAt) }}
          </template>
        </el-table-column>
        <el-table-column prop="reason" label="记录说明" min-width="200" />
        <el-table-column label="状态" width="120">
          <template #default="scope">
            <el-tag
              :type="requestStatusTagType(scope.row.status)"
              size="small"
              effect="plain"
            >
              {{ requestStatusLabel(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="身份核验" width="130">
          <template #default="scope">
            <el-tag type="info" size="small" effect="plain">
              {{ identityStatusLabel(scope.row.identityVerificationStatus) }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <el-dialog
      v-model="dialogVisible"
      title="记录生命周期请求"
      width="560px"
      append-to-body
      :close-on-click-modal="false"
    >
      <el-alert
        class="mb-4"
        type="warning"
        :closable="false"
        title="本操作只追加待政策请求，不会执行数据导出、更正、限制或删除。"
      />
      <el-form
        :model="form"
        label-width="110px"
      >
        <el-form-item label="请求类型" required>
          <el-select
            v-model="form.requestType"
            class="w-full"
          >
            <el-option
              v-for="item in requestTypeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="请求时间" required>
          <el-date-picker
            v-model="form.requestedAt"
            type="datetime"
            class="w-full"
          />
        </el-form-item>
        <el-form-item label="记录说明" required>
          <el-input
            v-model="form.reason"
            type="textarea"
            :rows="3"
            maxlength="1000"
            show-word-limit
            placeholder="只记录请求事实，不填写未确认的处置结论"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="submitting"
          @click="submitRequest"
        >
          确认记录
        </el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup>
  import { computed, onMounted, reactive, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    createDataLifecycleRequest,
    getDataGovernanceReadiness,
    getDataLifecycleRequests
  } from '@/api/sleep-care/care-clients'

  const props = defineProps({
    clientId: {
      type: Number,
      required: true
    },
    clientVersion: {
      type: Number,
      required: true
    },
    canRecord: {
      type: Boolean,
      default: false
    }
  })

  const emit = defineEmits(['recorded'])
  const readiness = ref(null)
  const requests = ref([])
  const loading = ref(false)
  const submitting = ref(false)
  const dialogVisible = ref(false)
  const requestKey = ref('')
  const form = reactive({
    requestType: 'ACCESS_COPY',
    requestedAt: new Date(),
    reason: ''
  })

  const requestTypeOptions = [
    { value: 'ACCESS_COPY', label: '查阅副本请求' },
    { value: 'CORRECTION', label: '更正请求' },
    { value: 'RESTRICTION', label: '限制处理请求' },
    { value: 'ERASURE', label: '删除请求' }
  ]

  const reviewedGateCount = computed(() =>
    readiness.value?.reviewGates?.filter((item) => item.reviewed).length || 0
  )

  const readinessMessage = computed(() => {
    if (readiness.value?.requestRecordingEnabled) {
      return '当前仅开放固定测试请求记录；正式授权和任何数据处置仍保持关闭。'
    }
    return '请求记录门禁处于关闭状态；需由业务与合规确认正式内容和处理规则后另行评审。'
  })

  const loadReadiness = async () => {
    const res = await getDataGovernanceReadiness()
    if (res.code === 0) {
      readiness.value = res.data
    }
  }

  const loadRequests = async () => {
    const res = await getDataLifecycleRequests(props.clientId, {
      page: 1,
      pageSize: 100
    })
    if (res.code === 0) {
      requests.value = res.data.list || []
    }
  }

  const openRequestDialog = () => {
    Object.assign(form, {
      requestType: 'ACCESS_COPY',
      requestedAt: new Date(),
      reason: ''
    })
    requestKey.value = crypto.randomUUID()
    dialogVisible.value = true
  }

  const submitRequest = async () => {
    if (!form.requestedAt || !form.reason.trim()) {
      ElMessage.warning('请求时间和记录说明必填')
      return
    }
    submitting.value = true
    try {
      const res = await createDataLifecycleRequest(
        props.clientId,
        {
          expectedVersion: props.clientVersion,
          requestType: form.requestType,
          requestedAt: new Date(form.requestedAt).toISOString(),
          source: 'STAFF_RECORDED',
          reason: form.reason.trim()
        },
        requestKey.value
      )
      if (res.code === 0) {
        ElMessage.success(res.msg)
        dialogVisible.value = false
        await loadRequests()
        emit('recorded', res.data)
      }
    } finally {
      submitting.value = false
    }
  }

  const consentRequirementLabel = (value) => ({
    SERVICE_NOTICE: '服务说明与确认',
    PRIVACY_NOTICE: '隐私说明与确认',
    NOTIFICATION_CONSENT: '通知联系同意',
    AI_PROCESSING_CONSENT: '智能处理同意（未启用）'
  }[value] || '未识别项目')

  const consentRequirementStatus = (item) => {
    if (!item.contentReviewed) return '待业务与合规确认'
    return `内容已评审 ${item.policyVersion}，记录仍未启用`
  }

  const reviewGateLabel = (value) => ({
    IDENTITY_VERIFICATION: '身份核验方案',
    CONSENT_EVIDENCE: '同意证据规则',
    WITHDRAWAL_POLICY: '撤回处理规则',
    MINIMUM_NECESSARY_FIELDS: '最小必要字段',
    RETENTION_POLICY: '保存期限',
    CORRECTION_POLICY: '更正规则',
    ERASURE_POLICY: '删除规则',
    EXPORT_POLICY: '导出规则',
    SENSITIVE_ACCESS_AUDIT: '敏感访问审计',
    BACKUP_RESTORE: '备份恢复'
  }[value] || '未识别规则')

  const requestTypeLabel = (value) =>
    requestTypeOptions.find((item) => item.value === value)?.label || '未识别请求'

  const requestStatusLabel = (value) => ({
    PENDING_POLICY: '待政策确认'
  }[value] || '未识别状态')

  const requestStatusTagType = (value) =>
    value === 'PENDING_POLICY' ? 'warning' : 'info'

  const identityStatusLabel = (value) => ({
    NOT_CONFIGURED: '尚未配置'
  }[value] || '未识别状态')

  const formatDateTime = (value) =>
    value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'

  onMounted(async () => {
    loading.value = true
    try {
      await Promise.all([
        loadReadiness(),
        loadRequests()
      ])
    } finally {
      loading.value = false
    }
  })
</script>
