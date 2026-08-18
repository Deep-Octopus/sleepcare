<template>
  <div>
    <div class="mb-4 rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 text-amber-900">
      <div class="font-semibold">测试定义只读预览</div>
      <div class="mt-1 text-sm leading-6">
        P1-03 仅展示软件流程验证用问卷与关注规则版本。页面不采集答卷、不表达医疗判断，也不启用短信或面向用户的 AI。
      </div>
    </div>

    <div class="gva-search-box">
      <el-form
        :inline="true"
        :model="searchForm"
      >
        <el-form-item label="编码、版本或标题">
          <el-input
            v-model="searchForm.keyword"
            clearable
            placeholder="搜索测试问卷定义"
            @keyup.enter="search"
          />
        </el-form-item>
        <el-form-item label="生命周期">
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
        <el-form-item label="使用范围">
          <el-select
            v-model="searchForm.usageScope"
            clearable
            class="w-36"
          >
            <el-option label="仅测试" value="TEST_ONLY" />
            <el-option label="正式" value="FORMAL" />
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
        empty-text="暂无可预览的问卷版本"
      >
        <el-table-column prop="code" label="问卷编码" min-width="180" />
        <el-table-column prop="version" label="版本" min-width="145" />
        <el-table-column prop="title" label="标题" min-width="220" />
        <el-table-column label="生命周期" width="120">
          <template #default="scope">
            <el-tag :type="lifecycleTagType(scope.row.lifecycleStatus)">
              {{ lifecycleLabel(scope.row.lifecycleStatus) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="门禁" min-width="190">
          <template #default="scope">
            <div class="flex flex-wrap gap-1">
              <el-tag type="warning">{{ usageScopeLabel(scope.row.usageScope) }}</el-tag>
              <el-tag
                v-if="scope.row.synthetic"
                type="warning"
                effect="plain"
              >
                测试
              </el-tag>
              <el-tag
                :type="scope.row.productionEnabled ? 'danger' : 'info'"
                effect="plain"
              >
                {{ scope.row.productionEnabled ? '生产已启用' : '生产未启用' }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="内容规模" width="130">
          <template #default="scope">
            {{ scope.row.questionCount }} 题 / {{ scope.row.ruleCount }} 条规则
          </template>
        </el-table-column>
        <el-table-column label="工程复核" min-width="180">
          <template #default="scope">
            <div>{{ reviewTypeLabel(scope.row.reviewRecord?.reviewType) }}</div>
            <div class="mt-1 text-xs text-gray-500">
              {{ formatTimestamp(scope.row.reviewRecord?.reviewedAt) }}
            </div>
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
      title="问卷与规则版本预览"
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
          type="warning"
          :closable="false"
          title="该定义只用于测试软件验收，不得解释为医疗问卷或关注建议。"
        />

        <el-descriptions
          :column="2"
          border
        >
          <el-descriptions-item label="醒目标识">
            <el-tag type="warning">测试定义</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="生命周期">
            {{ lifecycleLabel(detail.lifecycleStatus) }}
          </el-descriptions-item>
          <el-descriptions-item label="编码">{{ detail.code }}</el-descriptions-item>
          <el-descriptions-item label="版本">{{ detail.version }}</el-descriptions-item>
          <el-descriptions-item label="使用范围">
            {{ usageScopeLabel(detail.usageScope) }}
          </el-descriptions-item>
          <el-descriptions-item label="生产门禁">
            {{ detail.productionEnabled ? '已启用' : '未启用' }}
          </el-descriptions-item>
          <el-descriptions-item label="预计耗时">
            {{ detail.expectedMinutes }} 分钟
          </el-descriptions-item>
          <el-descriptions-item label="定义协议">
            {{ detail.definitionSchemaVersion }}
          </el-descriptions-item>
          <el-descriptions-item label="标题" :span="2">
            {{ detail.title }}
          </el-descriptions-item>
          <el-descriptions-item label="用途" :span="2">
            {{ detail.purpose }}
          </el-descriptions-item>
          <el-descriptions-item label="复核" :span="2">
            {{ reviewTypeLabel(detail.reviewRecord?.reviewType) }} ·
            {{ formatTimestamp(detail.reviewRecord?.reviewedAt) }} ·
            {{ detail.reviewRecord?.note || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="定义哈希" :span="2">
            <span class="break-all font-mono text-xs">{{ detail.definitionHash }}</span>
          </el-descriptions-item>
        </el-descriptions>

        <div class="mb-2 mt-6 flex items-center justify-between">
          <h3 class="font-semibold">题目定义</h3>
          <span class="text-sm text-gray-500">{{ detail.questions?.length || 0 }} 题</span>
        </div>
        <div class="flex flex-col gap-3">
          <div
            v-for="question in detail.questions"
            :key="question.id"
            class="rounded-lg border border-gray-200 bg-gray-50 px-4 py-3"
          >
            <div class="flex flex-wrap items-center gap-2">
              <span class="font-semibold">{{ question.order }}. {{ question.title }}</span>
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
            <div class="mt-2 text-xs text-gray-500">
              题目编码：<span class="font-mono">{{ question.code }}</span>
              · 校验协议：{{ question.validationSchemaVersion }}
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
                <span class="font-mono text-xs text-gray-500">{{ option.code }}</span>
                <span class="ml-2">{{ option.label }}</span>
              </div>
            </div>
            <div class="mt-3 text-xs text-gray-500">
              校验约束：<span class="font-mono">{{ formatObject(question.validation) }}</span>
            </div>
          </div>
        </div>

        <div class="mb-2 mt-6 flex items-center justify-between">
          <h3 class="font-semibold">绑定关注规则版本</h3>
          <span class="text-sm text-gray-500">{{ detail.rules?.length || 0 }} 条</span>
        </div>
        <el-empty
          v-if="!detail.rules?.length"
          description="当前版本没有规则定义"
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
              <span class="font-semibold">{{ rule.title }}</span>
              <el-tag :type="lifecycleTagType(rule.lifecycleStatus)" size="small">
                {{ lifecycleLabel(rule.lifecycleStatus) }}
              </el-tag>
              <el-tag type="warning" size="small" effect="plain">
                {{ usageScopeLabel(rule.usageScope) }}
              </el-tag>
              <el-tag
                :type="rule.productionEnabled ? 'danger' : 'info'"
                size="small"
                effect="plain"
              >
                {{ rule.productionEnabled ? '生产已启用' : '生产未启用' }}
              </el-tag>
            </div>
            <div class="mt-2 text-xs text-gray-500">
              {{ rule.code }} · {{ rule.version }} · {{ reviewTypeLabel(rule.reviewRecord?.reviewType) }}
            </div>
            <el-descriptions
              class="mt-3"
              :column="1"
              size="small"
              border
            >
              <el-descriptions-item label="测试关注级别">
                {{ rule.attentionLevel }}
              </el-descriptions-item>
              <el-descriptions-item label="触发条件">
                <span class="break-all font-mono text-xs">{{ formatObject(rule.condition) }}</span>
              </el-descriptions-item>
              <el-descriptions-item label="原因快照">
                {{ rule.reasonSnapshot }}
              </el-descriptions-item>
              <el-descriptions-item label="接收角色">
                {{ rule.recipients?.join('、') || '-' }}
              </el-descriptions-item>
              <el-descriptions-item label="去重模板">
                <span class="font-mono text-xs">{{ rule.dedupKeyTemplate }}</span>
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

  const lifecycleLabel = (value) => lifecycleOptions.find((item) => item.value === value)?.label || value || '-'

  const lifecycleTagType = (value) => ({
    DRAFT: 'info',
    IN_REVIEW: 'warning',
    APPROVED: 'primary',
    PUBLISHED: 'success',
    DISABLED: 'danger'
  })[value] || 'info'

  const usageScopeLabel = (value) => ({
    TEST_ONLY: '仅测试',
    FORMAL: '正式'
  })[value] || value || '-'

  const reviewTypeLabel = (value) => ({
    ENGINEERING_FIXTURE_REVIEW: '工程夹具复核',
    FORMAL_MEDICAL_REVIEW: '正式医疗复核'
  })[value] || value || '未复核'

  const questionTypeLabel = (value) => ({
    SINGLE_CHOICE: '单选',
    MULTIPLE_CHOICE: '多选',
    TEXT: '文本',
    NUMBER: '数字',
    DATE: '日期',
    BOOLEAN: '布尔'
  })[value] || value

  const formatObject = (value) => JSON.stringify(value || {})

  const formatTimestamp = (value) => value ? formatDate(value) : '-'

  onMounted(loadTable)
</script>
