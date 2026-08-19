<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-6">
    <button
      type="button"
      class="mb-6 inline-flex min-h-10 items-center gap-2 rounded-lg px-1 text-sm text-primary focus-visible:outline-2 focus-visible:outline-primary"
      @click="router.push({ name: 'ClientSatisfaction' })"
    >
      <span aria-hidden="true">←</span>
      返回服务评价
    </button>

    <div
      v-if="loading"
      class="rounded-2xl border border-border bg-muted p-5 text-sm text-muted-foreground"
    >
      正在读取评价详情…
    </div>

    <div
      v-else-if="errorMessage"
      class="rounded-2xl border border-error/30 bg-error/8 p-5"
    >
      <p class="font-medium text-error">暂时无法读取评价详情</p>
      <p class="mt-2 text-sm leading-6 text-error">{{ errorMessage }}</p>
      <el-button class="!mt-4 !h-11 !rounded-xl" @click="loadDetail">
        重试
      </el-button>
    </div>

    <template v-else-if="detail">
      <div class="flex items-start justify-between gap-3">
        <div>
          <p class="text-sm text-muted-foreground">匿名编号 {{ detail.publicCode }}</p>
          <h1 class="mt-1 text-[1.75rem] font-semibold tracking-[-0.035em]">本次在线服务</h1>
        </div>
        <span
          class="shrink-0 rounded-full px-2.5 py-1 text-xs font-medium"
          :class="statusTone(detail.status)"
        >
          {{ statusLabel(detail.status) }}
        </span>
      </div>

      <div class="mt-6 rounded-2xl border border-primary/20 bg-primary/6 p-4 text-sm leading-6">
        工作人员查看时不显示你的身份或本次服务责任人。系统关联仅供授权质量核查使用。
      </div>

      <section
        v-if="detail.status === 'PENDING'"
        class="mt-6 rounded-2xl border border-border bg-container p-5"
      >
        <div class="text-center">
          <p class="text-base font-semibold">你对本次服务的整体感受如何？</p>
          <p class="mt-2 text-sm text-muted-foreground">
            请在 {{ formatTaskTime(detail.expiresAt) }} 前提交
          </p>
          <el-rate
            v-model="form.rating"
            class="!mt-5 !h-auto justify-center"
            :colors="['#d09128', '#d09128', '#2c806c']"
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
          class="!mt-5 !h-12 !w-full !rounded-xl"
          :disabled="form.rating === 0"
          :loading="submitting"
          @click="submitResponse"
        >
          提交评价
        </el-button>
      </section>

      <section
        v-else-if="detail.status === 'SUBMITTED' && detail.response"
        class="mt-6 rounded-2xl border border-success/25 bg-success/6 p-5"
      >
        <p class="text-xs font-semibold tracking-[0.12em] text-success">感谢你的反馈</p>
        <el-rate
          class="!mt-4"
          :model-value="detail.response.rating"
          disabled
          show-score
        />
        <p
          v-if="detail.response.comment"
          class="mt-4 whitespace-pre-wrap break-words border-t border-success/20 pt-4 text-sm leading-6"
        >
          {{ detail.response.comment }}
        </p>
        <p v-else class="mt-4 text-sm text-muted-foreground">本次未填写补充意见。</p>
        <p class="mt-4 text-xs text-muted-foreground">
          提交于 {{ formatTaskTime(detail.response.submittedAt) }}
        </p>
      </section>

      <section
        v-else
        class="mt-6 rounded-2xl border border-border bg-muted p-5"
      >
        <p class="font-medium">本次评价邀请已结束</p>
        <p class="mt-2 text-sm leading-6 text-muted-foreground">
          有效期结束后不能再提交；如有新的服务事项，可以通过联系服务发起咨询。
        </p>
      </section>

      <p class="mt-7 text-xs leading-5 text-muted-foreground">
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

  const statusLabel = (value) => ({
    PENDING: '待评价',
    SUBMITTED: '已提交',
    EXPIRED: '已结束'
  }[value] || '未说明')

  const statusTone = (value) => ({
    PENDING: 'bg-primary/10 text-primary',
    SUBMITTED: 'bg-success/10 text-success',
    EXPIRED: 'bg-muted text-muted-foreground'
  }[value] || 'bg-muted text-muted-foreground')

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
