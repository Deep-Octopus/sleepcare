<template>
  <div>
    <div class="mb-4 rounded-lg border border-border bg-container px-4 py-3">
      <div class="font-semibold">问卷内容管理</div>
      <div class="mt-1 text-sm leading-6">
        查看已配置的题目和关注规则。本页不显示用户答卷，也不提供诊断或治疗建议。
      </div>
    </div>

    <div class="gva-search-box">
      <el-form
        :inline="true"
        :model="searchForm"
      >
        <el-form-item label="问卷名称">
          <el-input
            v-model="searchForm.keyword"
            clearable
            placeholder="请输入问卷名称"
            @keyup.enter="search"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-select
            v-model="searchForm.status"
            clearable
            class="w-40"
          >
            <el-option
              v-for="item in lifecycleOptions"
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
        empty-text="暂无可用问卷"
      >
        <el-table-column label="问卷名称" min-width="260">
          <template #default="scope">{{ readableQuestionnaireTitle(scope.row.title) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="120">
          <template #default="scope">
            <el-tag :type="lifecycleTagType(scope.row.lifecycleStatus)">
              {{ lifecycleLabel(scope.row.lifecycleStatus) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="内容规模" width="130">
          <template #default="scope">
            {{ scope.row.questionCount }} 题 / {{ scope.row.ruleCount }} 条规则
          </template>
        </el-table-column>
        <el-table-column label="内容确认时间" min-width="180">
          <template #default="scope">
            {{ formatTimestamp(scope.row.reviewRecord?.reviewedAt) }}
          </template>
        </el-table-column>
        <el-table-column label="发布时间" min-width="170">
          <template #default="scope">
            {{ formatTimestamp(scope.row.publishedAt) }}
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
      title="问卷内容预览"
      size="760px"
    >
      <div
        v-if="detailLoading"
        class="flex min-h-48 items-center justify-center"
      >
        <el-icon class="is-loading text-xl"><Loading /></el-icon>
      </div>
      <template v-else-if="detail">
        <el-alert
          class="mb-4"
          type="info"
          :closable="false"
          title="请按页面内容核对题目与关注流程。本页不会展示用户作答结果。"
        />

        <el-descriptions
          :column="2"
          border
        >
          <el-descriptions-item label="状态">
            {{ lifecycleLabel(detail.lifecycleStatus) }}
          </el-descriptions-item>
          <el-descriptions-item label="预计耗时">
            {{ detail.expectedMinutes }} 分钟
          </el-descriptions-item>
          <el-descriptions-item label="问卷名称" :span="2">
              {{ readableQuestionnaireTitle(detail.title) }}
          </el-descriptions-item>
          <el-descriptions-item label="用途" :span="2">
              {{ readableQuestionnairePurpose(detail.purpose) }}
          </el-descriptions-item>
          <el-descriptions-item label="内容确认记录" :span="2">
            {{ formatTimestamp(detail.reviewRecord?.reviewedAt) }} ·
            {{ readableReviewNote(detail.reviewRecord?.note) }}
          </el-descriptions-item>
        </el-descriptions>

        <div class="mb-2 mt-6 flex items-center justify-between">
          <h3 class="font-semibold">题目预览</h3>
          <span class="text-sm text-gray-500">{{ detail.questions?.length || 0 }} 题</span>
        </div>
        <div class="flex flex-col gap-3">
          <div
            v-for="question in detail.questions"
            :key="question.id"
            class="rounded-lg border border-gray-200 bg-gray-50 px-4 py-3"
          >
            <div class="flex flex-wrap items-center gap-2">
              <span class="font-semibold">{{ question.order }}. {{ readableQuestionTitle(question.title) }}</span>
              <el-tag size="small" effect="plain">{{ questionTypeLabel(question.type) }}</el-tag>
              <el-tag
                v-if="question.required"
                size="small"
                type="danger"
                effect="plain"
              >
                必答
              </el-tag>
            </div>
            <div
              v-if="question.options?.length"
              class="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2"
            >
              <div
                v-for="option in question.options"
                :key="option.id"
                class="rounded border border-gray-200 bg-white px-3 py-2 text-sm"
              >
                <span>{{ readableOptionLabel(option.label) }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="mb-2 mt-6 flex items-center justify-between">
          <h3 class="font-semibold">关注规则</h3>
          <span class="text-sm text-gray-500">{{ detail.rules?.length || 0 }} 条</span>
        </div>
        <el-empty
          v-if="!detail.rules?.length"
          description="当前问卷没有关注规则"
          :image-size="72"
        />
        <div
          v-else
          class="flex flex-col gap-3"
        >
          <div
            v-for="rule in detail.rules"
            :key="rule.id"
            class="rounded-lg border border-gray-200 px-4 py-3"
          >
            <div class="flex flex-wrap items-center gap-2">
              <span class="font-semibold">{{ readableRuleTitle(rule.title) }}</span>
              <el-tag :type="lifecycleTagType(rule.lifecycleStatus)" size="small">
                {{ lifecycleLabel(rule.lifecycleStatus) }}
              </el-tag>
            </div>
            <el-descriptions
              class="mt-3"
              :column="1"
              size="small"
              border
            >
              <el-descriptions-item label="关注程度">
                需要工作人员关注
              </el-descriptions-item>
              <el-descriptions-item label="触发说明">
                {{ readableAttentionReason(rule.reasonSnapshot) }}
              </el-descriptions-item>
              <el-descriptions-item label="由谁处理">
                {{ recipientLabels(rule.recipients) }}
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
  import { onMounted, reactive, ref } from 'vue'
  import { Loading } from '@element-plus/icons-vue'
  import { getQuestionnaireVersion, getQuestionnaireVersions } from '@/api/sleep-care/questionnaires'
  import { useBtnAuth } from '@/utils/btnAuth'
  import { formatDate } from '@/utils/format'
  import {
    readableAttentionReason,
    readableOptionLabel,
    readableQuestionnairePurpose,
    readableQuestionnaireTitle,
    readableQuestionTitle,
    readableReviewNote,
    readableRuleTitle
  } from '@/utils/sleep-care-display'

  defineOptions({ name: 'CareQuestionnaires' })

  const btnAuth = useBtnAuth()
  const lifecycleOptions = [
    { label: '草稿', value: 'DRAFT' },
    { label: '复核中', value: 'IN_REVIEW' },
    { label: '已批准', value: 'APPROVED' },
    { label: '已发布', value: 'PUBLISHED' },
    { label: '已禁用', value: 'DISABLED' }
  ]
  const searchForm = reactive({
    keyword: '',
    status: '',
    usageScope: ''
  })
  const page = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const tableData = ref([])
  const loading = ref(false)
  const detailLoading = ref(false)
  const detailVisible = ref(false)
  const detail = ref(null)

  const loadTable = async () => {
    loading.value = true
    try {
      const response = await getQuestionnaireVersions({
        page: page.value,
        pageSize: pageSize.value,
        synthetic: true,
        ...searchForm
      })
      if (response.code === 0) {
        tableData.value = response.data.list || []
        total.value = response.data.total || 0
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
    searchForm.usageScope = ''
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
      const response = await getQuestionnaireVersion(id)
      if (response.code === 0) {
        detail.value = response.data
      }
    } finally {
      detailLoading.value = false
    }
  }

  const lifecycleLabel = (value) => lifecycleOptions.find((item) => item.value === value)?.label || '未说明'

  const lifecycleTagType = (value) => ({
    DRAFT: 'info',
    IN_REVIEW: 'warning',
    APPROVED: 'primary',
    PUBLISHED: 'success',
    DISABLED: 'danger'
  })[value] || 'info'

  const questionTypeLabel = (value) => ({
    SINGLE_CHOICE: '单选',
    MULTIPLE_CHOICE: '多选',
    TEXT: '文本',
    NUMBER: '数字',
    DATE: '日期',
    BOOLEAN: '是或否'
  })[value] || '其他题型'

  const recipientLabels = (values = []) => values.map((value) => ({
    CARE_STEWARD: '健康管家',
    CLINICIAN: '一线医护',
    SUPERVISOR: '上级医师'
  })[value] || '工作人员').join('、') || '未指定'

  const formatTimestamp = (value) => value ? formatDate(value) : '-'

  onMounted(loadTable)
</script>
