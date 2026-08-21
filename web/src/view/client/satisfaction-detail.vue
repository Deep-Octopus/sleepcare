<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-4">
    <ClientBackButton
      class="mb-7"
      label="返回服务评价"
      @click="router.push({ name: 'ClientSatisfaction' })"
    />

    <ClientStatePanel
      v-if="loading"
      title="正在读取评价详情"
      tone="muted"
      icon="lucide:loader-circle"
    />

    <ClientStatePanel
      v-else-if="errorMessage"
      title="暂时无法读取评价详情"
      :description="errorMessage"
      tone="danger"
      icon="lucide:circle-alert"
    >
      <el-button class="!h-11 !rounded-xl" @click="loadDetail">
        重试
      </el-button>
    </ClientStatePanel>

    <template v-else-if="detail">
      <div class="flex items-start justify-between gap-3">
        <div>
          <p class="text-sm font-medium text-primary">服务反馈</p>
          <h1 class="mt-3 text-[2rem] font-semibold tracking-[-0.04em]">本次在线服务</h1>
        </div>
        <ClientStatusBadge
          :label="statusLabel(detail.status)"
          :tone="statusTone(detail.status)"
        />
      </div>

      <ClientStatePanel
        class="mt-7"
        title="你的身份信息会被隐去"
        description="工作人员查看反馈时不会看到你的身份或本次服务责任人，系统关联仅供授权质量核查使用。"
        tone="primary"
        icon="lucide:shield-check"
      />

      <section
        v-if="detail.status === 'PENDING'"
        class="mt-8 border-y border-border py-7"
      >
        <div class="text-center">
          <p class="text-base font-semibold">你对本次服务的整体感受如何？</p>
          <p class="mt-2 text-sm text-muted-foreground">
            请在 {{ formatTaskTime(detail.expiresAt) }} 前提交
          </p>
          <el-rate
            v-model="form.rating"
            class="!mt-5 !h-auto justify-center"
            :colors="rateColors"
            size="large"
          />
          <p class="mt-2 min-h-5 text-sm font-medium text-primary">
            {{ ratingHint(form.rating) }}
          </p>
        </div>

        <el-input
          v-model="form.comment"
          class="mt-5"
          maxlength="2000"
          placeholder="可补充说明本次服务中做得好或希望改进的地方（选填）"
          :rows="5"
          show-word-limit
          type="textarea"
        />
        <p v-if="submitError" class="mt-3 text-sm text-error">{{ submitError }}</p>
        <el-button
          type="primary"
          class="!mt-6 !h-13 !w-full !rounded-xl !font-semibold"
          :disabled="form.rating === 0"
          :loading="submitting"
          @click="submitResponse"
        >
          提交评价
        </el-button>
      </section>

      <section
        v-else-if="detail.status === 'SUBMITTED' && detail.response"
        class="mt-8 border-l-2 border-success bg-success-50 px-4 py-5"
      >
        <p class="text-sm font-semibold text-success-700">感谢你的反馈</p>
        <el-rate
          class="!mt-4"
          :model-value="detail.response.rating"
          disabled
          show-score
        />
        <p
          v-if="detail.response.comment"
          class="mt-4 whitespace-pre-wrap break-words border-t border-success-200 pt-4 text-sm leading-6"
        >
          {{ detail.response.comment }}
        </p>
        <p v-else class="mt-4 text-sm text-muted-foreground">本次未填写补充意见。</p>
        <p class="mt-4 text-xs text-muted-foreground">
          提交于 {{ formatTaskTime(detail.response.submittedAt) }}
        </p>
      </section>

      <ClientStatePanel
        v-else
        class="mt-8"
        title="本次评价邀请已结束"
        description="有效期结束后不能再提交；如有新的服务事项，可以通过联系服务发起咨询。"
        tone="muted"
        icon="lucide:clock"
      />

      <p class="mt-8 border-t border-border pt-5 text-xs leading-5 text-muted-foreground">
        单次评价用于发现服务流程中的改进线索，不会单独形成对工作人员的结论。
      </p>
    </template>
  </section>
</template>

<script setup>
  import { onMounted, reactive, ref } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import {
    getClientSatisfactionRequest,
    submitClientSatisfactionResponse
  } from '@/api/sleep-care/client-access'
  import {
    clearSatisfactionSubmitKey,
    formatTaskTime,
    getOrCreateSatisfactionSubmitKey,
    unwrapClientResponse
  } from './state'
  import ClientBackButton from '@/components/client-mobile/client-back-button.vue'
  import ClientStatePanel from '@/components/client-mobile/client-state-panel.vue'
  import ClientStatusBadge from '@/components/client-mobile/client-status-badge.vue'

  defineOptions({
    name: 'ClientSatisfactionDetail'
  })

  const route = useRoute()
  const router = useRouter()
  const requestId = Number(route.params.id)
  const loading = ref(true)
  const submitting = ref(false)
  const errorMessage = ref('')
  const submitError = ref('')
  const detail = ref(null)
  const form = reactive({
    rating: 0,
    comment: ''
  })
  const rateColors = [
    'var(--el-color-primary)',
    'var(--el-color-primary)',
    'var(--el-color-primary)'
  ]

  const statusLabel = (value) => ({
    PENDING: '待评价',
    SUBMITTED: '已提交',
    EXPIRED: '已结束'
  }[value] || '未说明')

  const statusTone = (value) => ({
    PENDING: 'primary',
    SUBMITTED: 'success',
    EXPIRED: 'muted'
  }[value] || 'muted')

  const ratingHint = (value) => ({
    1: '体验未达到预期',
    2: '仍有较多改进空间',
    3: '整体基本满意',
    4: '服务体验良好',
    5: '服务体验很好'
  }[value] || '请选择 1 至 5 星')

  const loadDetail = async () => {
    loading.value = true
    errorMessage.value = ''
    try {
      detail.value = unwrapClientResponse(await getClientSatisfactionRequest(requestId))
      if (detail.value.response) {
        form.rating = detail.value.response.rating
        form.comment = detail.value.response.comment || ''
      }
    } catch (error) {
      errorMessage.value = error.message || '请检查网络后重试。'
    } finally {
      loading.value = false
    }
  }

  const submitResponse = async () => {
    if (!detail.value || form.rating < 1 || form.rating > 5) {
      return
    }
    submitting.value = true
    submitError.value = ''
    try {
      unwrapClientResponse(await submitClientSatisfactionResponse(
        requestId,
        getOrCreateSatisfactionSubmitKey(requestId),
        {
          expectedVersion: detail.value.version,
          rating: form.rating,
          comment: form.comment.trim()
        }
      ))
      clearSatisfactionSubmitKey(requestId)
      await loadDetail()
    } catch (error) {
      submitError.value = error.message || '暂时无法提交评价，请稍后重试。'
    } finally {
      submitting.value = false
    }
  }

  onMounted(loadDetail)
</script>
