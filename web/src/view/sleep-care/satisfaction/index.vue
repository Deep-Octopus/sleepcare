<template>
  <main class="space-y-4 text-base-text">
    <section class="rounded-xl border border-border bg-container p-5 shadow-card">
      <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div>
          <div class="mb-2 flex items-center gap-2 text-sm font-medium text-primary">
            <svg-icon icon="lucide:star" />
            <span>质量改进</span>
          </div>
          <h1 class="text-2xl font-semibold tracking-tight">服务评价</h1>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
            查看匿名评价并跟进低分反馈。页面不展示用户、咨询或服务责任人的关联信息。
          </p>
        </div>
        <el-button :loading="activeLoading" @click="refreshActiveTab">
          <svg-icon class="mr-1" icon="lucide:refresh-cw" />
          刷新
        </el-button>
      </div>
    </section>

    <el-alert
      :closable="false"
      title="单条评价只能作为服务流程核查线索，不能直接形成对工作人员的结论。"
      type="warning"
    />

    <section class="rounded-xl border border-border bg-container p-5 shadow-card">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="匿名评价" name="responses">
          <div class="mb-4 flex flex-wrap items-center gap-3">
            <el-select
              v-model="responseFilters.rating"
              class="w-40"
              clearable
              placeholder="全部星级"
              @change="resetResponsePage"
            >
              <el-option
                v-for="rating in [1, 2, 3, 4, 5]"
                :key="rating"
                :label="`${rating} 星`"
                :value="rating"
              />
            </el-select>
            <el-select
              v-model="responseFilters.followUpStatus"
              class="w-44"
              clearable
              placeholder="全部跟进状态"
              @change="resetResponsePage"
            >
              <el-option label="未生成跟进" value="NONE" />
              <el-option label="待接收" value="OPEN" />
              <el-option label="核查中" value="IN_REVIEW" />
              <el-option label="已解决" value="RESOLVED" />
            </el-select>
          </div>

          <el-table
            v-loading="responseLoading"
            :data="responses"
            empty-text="当前管理范围内暂无匿名评价"
            row-key="id"
          >
            <el-table-column label="匿名编号" min-width="130" prop="publicCode" />
            <el-table-column label="评分" min-width="150">
              <template #default="scope">
                <el-rate
                  :model-value="scope.row.rating"
                  disabled
                  size="small"
                />
              </template>
            </el-table-column>
            <el-table-column label="补充意见" min-width="260">
              <template #default="scope">
                <p class="line-clamp-2 text-sm leading-6">
                  {{ scope.row.comment || '未填写' }}
                </p>
              </template>
            </el-table-column>
            <el-table-column label="跟进状态" min-width="120">
              <template #default="scope">
                <el-tag :type="followUpTag(scope.row.followUpStatus)" effect="plain">
                  {{ followUpLabel(scope.row.followUpStatus) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="提交时间" min-width="180">
              <template #default="scope">{{ formatTimestamp(scope.row.submittedAt) }}</template>
            </el-table-column>
            <el-table-column fixed="right" label="操作" width="90">
              <template #default="scope">
                <el-button
                  v-if="btnAuth.viewFollowUp && scope.row.followUpId"
                  link
                  type="primary"
                  @click="openFollowUp(scope.row.followUpId)"
                >
                  查看跟进
                </el-button>
                <span v-else class="text-xs text-muted-foreground">—</span>
              </template>
            </el-table-column>
          </el-table>

          <div class="gva-pagination">
            <el-pagination
              v-model:current-page="responsePage"
              v-model:page-size="responsePageSize"
              :page-sizes="[10, 30, 50, 100]"
              :total="responseTotal"
              layout="total, sizes, prev, pager, next, jumper"
              @current-change="loadResponses"
              @size-change="resetResponsePage"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane label="低分跟进" name="followUps">
          <div class="mb-4 flex flex-wrap items-center gap-3">
            <el-select
              v-model="followUpStatus"
              class="w-44"
              clearable
              placeholder="全部跟进状态"
              @change="resetFollowUpPage"
            >
              <el-option label="待接收" value="OPEN" />
              <el-option label="核查中" value="IN_REVIEW" />
              <el-option label="已解决" value="RESOLVED" />
            </el-select>
          </div>

          <el-table
            v-loading="followUpLoading"
            :data="followUps"
            empty-text="当前管理范围内暂无低分质量跟进"
            row-key="id"
          >
            <el-table-column label="匿名编号" min-width="130" prop="publicCode" />
            <el-table-column label="评分" min-width="150">
              <template #default="scope">
                <el-rate :model-value="scope.row.rating" disabled size="small" />
              </template>
            </el-table-column>
            <el-table-column label="状态" min-width="120">
              <template #default="scope">
                <el-tag :type="followUpTag(scope.row.status)" effect="plain">
                  {{ followUpLabel(scope.row.status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="质量跟进责任人" min-width="150">
              <template #default="scope">{{ scope.row.assigneeName || '待接收' }}</template>
            </el-table-column>
            <el-table-column label="创建时间" min-width="180">
              <template #default="scope">{{ formatTimestamp(scope.row.openedAt) }}</template>
            </el-table-column>
            <el-table-column fixed="right" label="操作" width="90">
              <template #default="scope">
                <el-button
                  v-if="btnAuth.viewFollowUp"
                  link
                  type="primary"
                  @click="openFollowUp(scope.row.id)"
                >
                  详情
                </el-button>
              </template>
            </el-table-column>
          </el-table>

          <div class="gva-pagination">
            <el-pagination
              v-model:current-page="followUpPage"
              v-model:page-size="followUpPageSize"
              :page-sizes="[10, 30, 50, 100]"
              :total="followUpTotal"
              layout="total, sizes, prev, pager, next, jumper"
              @current-change="loadFollowUps"
              @size-change="resetFollowUpPage"
            />
          </div>
        </el-tab-pane>
      </el-tabs>
    </section>

    <el-drawer
      v-model="detailVisible"
      size="min(760px, 100%)"
      title="质量跟进详情"
    >
      <div v-loading="detailLoading" class="min-h-56">
        <template v-if="detail">
          <section class="rounded-xl border border-border bg-muted p-4">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-xs font-medium tracking-[0.08em] text-muted-foreground">
                  匿名编号 {{ detail.publicCode }}
                </p>
                <el-rate
                  class="!mt-3"
                  :model-value="detail.rating"
                  disabled
                  show-score
                />
              </div>
              <el-tag :type="followUpTag(detail.status)" effect="plain">
                {{ followUpLabel(detail.status) }}
              </el-tag>
            </div>
            <p class="mt-4 whitespace-pre-wrap break-words text-sm leading-6">
              {{ detail.comment || '未填写补充意见' }}
            </p>
            <p class="mt-3 text-xs text-muted-foreground">
              提交于 {{ formatTimestamp(detail.submittedAt) }}
            </p>
          </section>

          <el-alert
            :closable="false"
            class="mt-4"
            title="请基于可核验的服务事实完成流程核查；当前详情不提供用户和服务责任人关联。"
            type="warning"
          />

          <div class="my-5 flex flex-wrap gap-2">
            <el-button
              v-if="btnAuth.acknowledgeFollowUp && detail.status === 'OPEN'"
              type="primary"
              @click="openAcknowledge"
            >
              接收跟进
            </el-button>
            <el-button
              v-if="btnAuth.resolveFollowUp && detail.status === 'IN_REVIEW'"
              type="success"
              @click="openResolve"
            >
              记录核查结果
            </el-button>
          </div>

          <el-descriptions :column="2" border>
            <el-descriptions-item label="当前责任人">
              {{ detail.assigneeName || '待接收' }}
            </el-descriptions-item>
            <el-descriptions-item label="创建时间">
              {{ formatTimestamp(detail.openedAt) }}
            </el-descriptions-item>
            <el-descriptions-item label="接收时间">
              {{ formatTimestamp(detail.acknowledgedAt) }}
            </el-descriptions-item>
            <el-descriptions-item label="解决时间">
              {{ formatTimestamp(detail.resolvedAt) }}
            </el-descriptions-item>
            <el-descriptions-item label="核查结果" :span="2">
              {{ detail.resolution || '尚未记录' }}
            </el-descriptions-item>
            <el-descriptions-item label="改进动作" :span="2">
              {{ detail.improvementAction || '尚未记录' }}
            </el-descriptions-item>
          </el-descriptions>

          <section class="mt-7">
            <h3 class="text-lg font-semibold">跟进记录</h3>
            <el-empty
              v-if="!detail.actions?.length"
              :image-size="64"
              description="尚无跟进记录"
            />
            <el-timeline v-else class="mt-4">
              <el-timeline-item
                v-for="action in detail.actions"
                :key="action.id"
                :timestamp="formatTimestamp(action.occurredAt)"
                placement="top"
              >
                <div class="flex flex-wrap items-center gap-2">
                  <span class="font-medium">{{ actionLabel(action.actionType) }}</span>
                  <el-tag effect="plain" size="small">
                    {{ action.actorName || '上级工作人员' }}
                  </el-tag>
                </div>
                <p class="mt-2 whitespace-pre-wrap text-sm leading-6 text-muted-foreground">
                  {{ action.content }}
                </p>
                <p
                  v-if="action.improvementAction"
                  class="mt-1 whitespace-pre-wrap text-sm leading-6 text-muted-foreground"
                >
                  改进动作：{{ action.improvementAction }}
                </p>
              </el-timeline-item>
            </el-timeline>
          </section>
        </template>
      </div>
    </el-drawer>

    <el-dialog v-model="acknowledgeVisible" title="接收质量跟进" width="min(560px, 92vw)">
      <el-form label-position="top">
        <el-form-item label="接收说明" required>
          <el-input
            v-model="acknowledgeNote"
            maxlength="2000"
            placeholder="请记录后续核查安排"
            :rows="4"
            show-word-limit
            type="textarea"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="acknowledgeVisible = false">取消</el-button>
        <el-button type="primary" :loading="actionSubmitting" @click="submitAcknowledge">
          确认接收
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="resolveVisible" title="记录质量核查结果" width="min(620px, 92vw)">
      <el-form label-position="top">
        <el-form-item label="核查结果" required>
          <el-input
            v-model="resolveForm.resolution"
            maxlength="4000"
            placeholder="请记录已经核验的服务流程事实和结论"
            :rows="4"
            show-word-limit
            type="textarea"
          />
        </el-form-item>
        <el-form-item label="改进动作">
          <el-input
            v-model="resolveForm.improvementAction"
            maxlength="2000"
            placeholder="可记录后续流程改进动作"
            :rows="3"
            show-word-limit
            type="textarea"
          />
        </el-form-item>
        <el-checkbox v-model="resolveForm.usageBoundaryConfirmed">
          我已确认：单条评价不能直接形成对工作人员的结论
        </el-checkbox>
      </el-form>
      <template #footer>
        <el-button @click="resolveVisible = false">取消</el-button>
        <el-button type="success" :loading="actionSubmitting" @click="submitResolve">
          确认解决
        </el-button>
      </template>
    </el-dialog>
  </main>
</template>

<script setup>
  import { computed, onMounted, reactive, ref, watch } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    acknowledgeSatisfactionFollowUp,
    getSatisfactionFollowUp,
    getSatisfactionFollowUps,
    getSatisfactionResponses,
    resolveSatisfactionFollowUp
  } from '@/api/sleep-care/satisfaction'
  import { useBtnAuth } from '@/utils/btnAuth'
  import { formatDate } from '@/utils/format'

  defineOptions({ name: 'CareSatisfaction' })

  const btnAuth = useBtnAuth()
  const activeTab = ref('responses')
  const responseLoading = ref(false)
  const responses = ref([])
  const responsePage = ref(1)
  const responsePageSize = ref(10)
  const responseTotal = ref(0)
  const responseFilters = reactive({
    rating: undefined,
    followUpStatus: ''
  })
  const followUpLoading = ref(false)
  const followUps = ref([])
  const followUpPage = ref(1)
  const followUpPageSize = ref(10)
  const followUpTotal = ref(0)
  const followUpStatus = ref('')
  const detailVisible = ref(false)
  const detailLoading = ref(false)
  const detail = ref(null)
  const acknowledgeVisible = ref(false)
  const acknowledgeNote = ref('')
  const resolveVisible = ref(false)
  const resolveForm = reactive({
    resolution: '',
    improvementAction: '',
    usageBoundaryConfirmed: false
  })
  const actionSubmitting = ref(false)
  const commandKey = ref('')

  const activeLoading = computed(() => activeTab.value === 'responses'
    ? responseLoading.value
    : followUpLoading.value
  )

  const loadResponses = async () => {
    responseLoading.value = true
    try {
      const res = await getSatisfactionResponses({
        page: responsePage.value,
        pageSize: responsePageSize.value,
        rating: responseFilters.rating,
        followUpStatus: responseFilters.followUpStatus
      })
      if (res.code === 0) {
        responses.value = res.data.list || []
        responseTotal.value = res.data.total || 0
      }
    } finally {
      responseLoading.value = false
    }
  }

  const loadFollowUps = async () => {
    followUpLoading.value = true
    try {
      const res = await getSatisfactionFollowUps({
        page: followUpPage.value,
        pageSize: followUpPageSize.value,
        status: followUpStatus.value
      })
      if (res.code === 0) {
        followUps.value = res.data.list || []
        followUpTotal.value = res.data.total || 0
      }
    } finally {
      followUpLoading.value = false
    }
  }

  const resetResponsePage = () => {
    responsePage.value = 1
    loadResponses()
  }

  const resetFollowUpPage = () => {
    followUpPage.value = 1
    loadFollowUps()
  }

  const refreshActiveTab = () => {
    if (activeTab.value === 'responses') {
      loadResponses()
      return
    }
    loadFollowUps()
  }

  const openFollowUp = async (id) => {
    detail.value = null
    detailVisible.value = true
    detailLoading.value = true
    try {
      const res = await getSatisfactionFollowUp(id)
      if (res.code === 0) {
        detail.value = res.data
      }
    } finally {
      detailLoading.value = false
    }
  }

  const refreshAfterAction = async () => {
    const followUpId = detail.value.id
    await Promise.all([
      openFollowUp(followUpId),
      loadResponses(),
      loadFollowUps()
    ])
  }

  const openAcknowledge = () => {
    acknowledgeNote.value = ''
    commandKey.value = crypto.randomUUID()
    acknowledgeVisible.value = true
  }

  const submitAcknowledge = async () => {
    const note = acknowledgeNote.value.trim()
    if (!note) {
      ElMessage.warning('请填写接收说明')
      return
    }
    actionSubmitting.value = true
    try {
      const res = await acknowledgeSatisfactionFollowUp(detail.value.id, {
        expectedVersion: detail.value.version,
        note
      }, commandKey.value)
      if (res.code === 0) {
        ElMessage.success('质量跟进已接收')
        acknowledgeVisible.value = false
        await refreshAfterAction()
      }
    } finally {
      actionSubmitting.value = false
    }
  }

  const openResolve = () => {
    resolveForm.resolution = ''
    resolveForm.improvementAction = ''
    resolveForm.usageBoundaryConfirmed = false
    commandKey.value = crypto.randomUUID()
    resolveVisible.value = true
  }

  const submitResolve = async () => {
    const resolution = resolveForm.resolution.trim()
    if (!resolution) {
      ElMessage.warning('请填写核查结果')
      return
    }
    if (!resolveForm.usageBoundaryConfirmed) {
      ElMessage.warning('请先确认单条评价的使用边界')
      return
    }
    actionSubmitting.value = true
    try {
      const res = await resolveSatisfactionFollowUp(detail.value.id, {
        expectedVersion: detail.value.version,
        resolution,
        improvementAction: resolveForm.improvementAction.trim(),
        usageBoundaryConfirmed: true
      }, commandKey.value)
      if (res.code === 0) {
        ElMessage.success('质量跟进已解决')
        resolveVisible.value = false
        await refreshAfterAction()
      }
    } finally {
      actionSubmitting.value = false
    }
  }

  const followUpLabel = (value) => ({
    OPEN: '待接收',
    IN_REVIEW: '核查中',
    RESOLVED: '已解决',
    NONE: '未生成'
  }[value] || '未生成')

  const followUpTag = (value) => ({
    OPEN: 'warning',
    IN_REVIEW: 'primary',
    RESOLVED: 'success',
    NONE: 'info'
  }[value] || 'info')

  const actionLabel = (value) => ({
    ACKNOWLEDGE: '接收跟进',
    RESOLVE: '解决跟进'
  }[value] || '跟进记录')

  const formatTimestamp = (value) => value ? formatDate(value) : '-'

  watch(activeTab, (value) => {
    if (value === 'followUps' && followUps.value.length === 0) {
      loadFollowUps()
    }
  })

  onMounted(loadResponses)
</script>
