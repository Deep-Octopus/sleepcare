<template>
  <div>
    <div class="mb-4 rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 text-amber-900">
      <div class="font-semibold">合成 OSA 计划定义</div>
      <div class="mt-1 text-sm leading-6">
        本页只读展示 D1–D5 软件调度定义。所有任务均为合成流程验证内容，通知保持禁用，不提供医疗解释、真实短信或面向用户的 AI。
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
            placeholder="搜索合成计划定义"
            @keyup.enter="search"
          />
        </el-form-item>
        <el-form-item label="生命周期">
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
        empty-text="暂无可预览的计划模板版本"
      >
        <el-table-column prop="pathCode" label="路径" width="90" />
        <el-table-column prop="code" label="计划编码" min-width="170" />
        <el-table-column prop="version" label="版本" min-width="145" />
        <el-table-column prop="title" label="标题" min-width="240" />
        <el-table-column label="调度边界" min-width="230">
          <template #default="scope">
            <div class="flex flex-wrap gap-1">
              <el-tag effect="plain">D1–D{{ scope.row.taskCount }}</el-tag>
              <el-tag type="info" effect="plain">
                {{ scope.row.pauseStrategy }}
              </el-tag>
              <el-tag type="warning" effect="plain">
                {{ scope.row.lateSubmissionPolicy }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="门禁" min-width="200">
          <template #default="scope">
            <div class="flex flex-wrap gap-1">
              <el-tag type="warning">仅测试</el-tag>
              <el-tag
                v-if="scope.row.synthetic"
                type="warning"
                effect="plain"
              >
                合成
              </el-tag>
              <el-tag type="info" effect="plain">
                生产未启用
              </el-tag>
            </div>
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
      title="OSA 计划模板版本预览"
      size="860px"
    >
      <div
        v-loading="detailLoading"
        class="min-h-48"
      >
        <template v-if="detail">
          <el-alert
            class="mb-4"
            type="warning"
            :closable="false"
            title="该模板仅验证软件调度；不构成医疗路径、诊疗建议或真实服务安排。"
          />

          <el-descriptions
            :column="2"
            border
          >
            <el-descriptions-item label="路径 / 计划">
              {{ detail.pathCode }} / {{ detail.code }}
            </el-descriptions-item>
            <el-descriptions-item label="版本">
              {{ detail.version }}
            </el-descriptions-item>
            <el-descriptions-item label="锚点定义" :span="2">
              <span class="font-mono text-xs">{{ detail.anchorDefinition }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="暂停策略">
              {{ detail.pauseStrategy }}
            </el-descriptions-item>
            <el-descriptions-item label="逾期策略">
              {{ detail.lateSubmissionPolicy }}
            </el-descriptions-item>
            <el-descriptions-item label="标题" :span="2">
              {{ detail.title }}
            </el-descriptions-item>
            <el-descriptions-item label="用途" :span="2">
              {{ detail.purpose }}
            </el-descriptions-item>
            <el-descriptions-item label="工程复核" :span="2">
              {{ detail.reviewRecord?.reviewType }} ·
              {{ formatDateTime(detail.reviewRecord?.reviewedAt) }} ·
              {{ detail.reviewRecord?.note }}
            </el-descriptions-item>
            <el-descriptions-item label="定义哈希" :span="2">
              <span class="break-all font-mono text-xs">{{ detail.definitionHash }}</span>
            </el-descriptions-item>
          </el-descriptions>

          <div class="mb-3 mt-7 flex items-end justify-between gap-4">
            <div>
              <div class="text-xs font-semibold tracking-widest text-gray-400">ANCHOR → D5</div>
              <h3 class="mt-1 text-lg font-semibold">五日调度轨道</h3>
            </div>
            <div class="text-right text-xs leading-5 text-gray-500">
              相对时间由 anchorAt 冻结计算<br>
              通知策略全部为 DISABLED
            </div>
          </div>

          <div class="grid grid-cols-1 gap-3 md:grid-cols-5">
            <div
              v-for="task in detail.tasks"
              :key="task.id"
              class="relative overflow-hidden rounded-lg border border-slate-200 bg-slate-50 px-3 pb-3 pt-9"
            >
              <div class="absolute left-0 top-0 flex w-full items-center justify-between bg-slate-900 px-3 py-2 text-white">
                <span class="font-mono text-sm font-semibold tracking-wider">{{ task.dayCode }}</span>
                <span class="text-[10px] text-slate-300">{{ task.executionRole }}</span>
              </div>
              <div class="mt-3 min-h-12 text-sm font-medium leading-5">
                {{ task.title }}
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
                  问卷 {{ task.questionnaireVersionId }}
                </el-tag>
                <el-tag
                  v-if="task.boundRuleVersionIds?.length"
                  size="small"
                  type="warning"
                  effect="plain"
                >
                  规则 {{ task.boundRuleVersionIds.join(', ') }}
                </el-tag>
                <el-tag
                  v-if="task.reviewRequired"
                  size="small"
                  type="info"
                  effect="plain"
                >
                  复核 {{ task.reviewRole }}
                </el-tag>
                <el-tag
                  v-if="!task.questionnaireVersionId"
                  size="small"
                  type="info"
                  effect="plain"
                >
                  无问卷绑定
                </el-tag>
                <el-tag
                  v-if="!task.boundRuleVersionIds?.length"
                  size="small"
                  type="info"
                  effect="plain"
                >
                  无规则绑定
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
    return `D${day} ${String(hour).padStart(2, '0')}:00`
  }

  const formatDateTime = (value) => value
    ? new Date(value).toLocaleString('zh-CN', { hour12: false })
    : '-'

  onMounted(loadTable)
</script>
