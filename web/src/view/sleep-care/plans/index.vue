<template>
  <div>
    <div class="mb-4 rounded-lg border border-border bg-container px-4 py-3">
      <div class="font-semibold">服务计划模板</div>
      <div class="mt-1 text-sm leading-6">
        查看每套计划包含的任务次数、开放时间和截止时间。系统目前不会自动发送通知。
      </div>
    </div>

    <div class="gva-search-box">
      <el-form
        :inline="true"
        :model="searchForm"
      >
        <el-form-item label="计划名称">
          <el-input
            v-model="searchForm.keyword"
            clearable
            placeholder="请输入计划名称"
            @keyup.enter="search"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-select
            v-model="searchForm.status"
            clearable
            class="w-40"
          >
            <el-option label="已发布" value="PUBLISHED" />
            <el-option label="已禁用" value="DISABLED" />
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
        empty-text="暂无可用的计划模板"
      >
        <el-table-column label="计划名称" min-width="240">
          <template #default="scope">{{ readablePlanTitle(scope.row.title) }}</template>
        </el-table-column>
        <el-table-column label="任务安排" min-width="300">
          <template #default="scope">
            <div class="flex flex-wrap gap-1">
              <el-tag effect="plain">共 {{ scope.row.taskCount }} 次</el-tag>
              <el-tag type="info" effect="plain">
                {{ pauseStrategyLabel(scope.row.pauseStrategy) }}
              </el-tag>
              <el-tag type="warning" effect="plain">
                {{ latePolicyLabel(scope.row.lateSubmissionPolicy) }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="120">
          <template #default="scope">
            <el-tag :type="scope.row.status === 'PUBLISHED' ? 'success' : 'info'">
              {{ scope.row.status === 'PUBLISHED' ? '可使用' : '已停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="发布时间" min-width="170">
          <template #default="scope">
            {{ formatDateTime(scope.row.publishedAt) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="100">
          <template #default="scope">
            <el-button
              v-if="btnAuth.preview"
              link
              type="primary"
              @click="openDetail(scope.row.id)"
            >
              预览
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
      title="服务计划详情"
      size="860px"
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
            title="这里展示任务时间安排，不提供诊断或治疗建议。"
          />

          <el-descriptions
            :column="2"
            border
          >
            <el-descriptions-item label="计划名称" :span="2">
              {{ readablePlanTitle(detail.title) }}
            </el-descriptions-item>
            <el-descriptions-item label="开始时间">
              以启动计划时选择的时间为准
            </el-descriptions-item>
            <el-descriptions-item label="暂停后安排">
              {{ pauseStrategyLabel(detail.pauseStrategy) }}
            </el-descriptions-item>
            <el-descriptions-item label="超过截止时间">
              {{ latePolicyLabel(detail.lateSubmissionPolicy) }}
            </el-descriptions-item>
            <el-descriptions-item label="用途" :span="2">
              {{ readablePlanPurpose(detail.purpose) }}
            </el-descriptions-item>
            <el-descriptions-item label="确认记录" :span="2">
              {{ formatDateTime(detail.reviewRecord?.reviewedAt) }} ·
              {{ readableReviewNote(detail.reviewRecord?.note) }}
            </el-descriptions-item>
          </el-descriptions>

          <div class="mb-3 mt-7 flex items-end justify-between gap-4">
            <div>
              <h3 class="text-lg font-semibold">任务时间安排</h3>
            </div>
            <div class="text-right text-xs leading-5 text-gray-500">
              时间按计划开始时间计算<br>
              系统目前不会自动发送通知
            </div>
          </div>

          <div class="grid grid-cols-1 gap-3 md:grid-cols-5">
            <div
              v-for="task in detail.tasks"
              :key="task.id"
              class="relative overflow-hidden rounded-lg border border-slate-200 bg-slate-50 px-3 pb-3 pt-9"
            >
              <div class="absolute left-0 top-0 flex w-full items-center justify-between bg-slate-900 px-3 py-2 text-white">
                <span class="text-sm font-semibold">{{ dayLabel(task.dayCode) }}</span>
                <span class="text-[10px] text-slate-300">{{ executionRoleLabel(task.executionRole) }}</span>
              </div>
              <div class="mt-3 min-h-12 text-sm font-medium leading-5">
                {{ readableTaskTitle(task.title, task.dayCode) }}
              </div>
              <div class="mt-3 border-t border-slate-200 pt-2 text-xs leading-5 text-slate-600">
                <div>开放 {{ formatOffset(task.openOffsetSeconds) }}</div>
                <div>截止 {{ formatOffset(task.dueOffsetSeconds) }}</div>
              </div>
              <div class="mt-2 flex flex-wrap gap-1">
                <el-tag
                  v-if="task.questionnaireVersionId"
                  size="small"
                  type="warning"
                  effect="plain"
                >
                  需要填写问卷
                </el-tag>
                <el-tag
                  v-if="task.boundRuleVersionIds?.length"
                  size="small"
                  type="warning"
                  effect="plain"
                >
                  已设置关注规则
                </el-tag>
                <el-tag
                  v-if="task.reviewRequired"
                  size="small"
                  type="info"
                  effect="plain"
                >
                  {{ reviewRoleLabel(task.reviewRole) }}复核
                </el-tag>
                <el-tag
                  v-if="!task.questionnaireVersionId"
                  size="small"
                  type="info"
                  effect="plain"
                >
                  无需填写问卷
                </el-tag>
                <el-tag
                  v-if="!task.boundRuleVersionIds?.length"
                  size="small"
                  type="info"
                  effect="plain"
                >
                  没有额外关注规则
                </el-tag>
              </div>
            </div>
          </div>
        </template>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
  import { onMounted, reactive, ref } from 'vue'
  import { useBtnAuth } from '@/utils/btnAuth'
  import {
    readablePlanPurpose,
    readablePlanTitle,
    readableReviewNote,
    readableTaskTitle
  } from '@/utils/sleep-care-display'
  import {
    getPlanVersion,
    getPlanVersions
  } from '@/api/sleep-care/care-path'

  defineOptions({ name: 'CarePlans' })

  const btnAuth = useBtnAuth()
  const searchForm = reactive({
    keyword: '',
    status: ''
  })
  const page = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const tableData = ref([])
  const loading = ref(false)
  const detailVisible = ref(false)
  const detailLoading = ref(false)
  const detail = ref(null)

  const loadTable = async () => {
    loading.value = true
    try {
      const res = await getPlanVersions({
        page: page.value,
        pageSize: pageSize.value,
        synthetic: true,
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
    searchForm.keyword = ''
    searchForm.status = ''
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
      const res = await getPlanVersion(id)
      if (res.code === 0) {
        detail.value = res.data
      }
    } finally {
      detailLoading.value = false
    }
  }

  const formatOffset = (seconds) => {
    const day = Math.floor(seconds / 86400) + 1
    const hour = Math.floor((seconds % 86400) / 3600)
    return `第${day}天 ${String(hour).padStart(2, '0')}:00`
  }

  const dayLabel = (value) => `第${String(value || '').replace(/^D/, '') || '-'}次`
  const pauseStrategyLabel = (value) => ({
    KEEP_WINDOWS: '暂停后保留原任务时间'
  }[value] || '按原任务时间执行')
  const latePolicyLabel = (value) => ({
    DENY: '截止后不能提交'
  }[value] || '按页面提示处理')
  const executionRoleLabel = (value) => ({
    CARE_CLIENT: '用户填写',
    CARE_STEWARD: '健康管家处理',
    CLINICIAN: '一线医护处理'
  }[value] || '按安排处理')
  const reviewRoleLabel = (value) => ({
    CARE_STEWARD: '健康管家',
    CLINICIAN: '一线医护',
    SUPERVISOR: '上级医师'
  }[value] || '工作人员')

  const formatDateTime = (value) => value
    ? new Date(value).toLocaleString('zh-CN', { hour12: false })
    : '-'

  onMounted(loadTable)
</script>
