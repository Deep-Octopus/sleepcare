<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-6">
    <button
      type="button"
      class="mb-6 inline-flex min-h-10 items-center gap-2 rounded-lg px-1 text-sm text-primary focus-visible:outline-2 focus-visible:outline-primary"
      @click="router.push({ name: 'ClientConsultations' })"
    >
      <span aria-hidden="true">←</span>
      返回咨询记录
    </button>

    <div
      v-if="loading"
      class="rounded-2xl border border-border bg-muted p-5 text-sm text-muted-foreground"
    >
      正在读取咨询详情…
    </div>

    <div
      v-else-if="errorMessage"
      class="rounded-2xl border border-error/30 bg-error/8 p-5"
    >
      <p class="font-medium text-error">暂时无法读取咨询详情</p>
      <p class="mt-2 text-sm leading-6 text-error">{{ errorMessage }}</p>
      <el-button
        class="!mt-4 !h-11 !rounded-xl"
        @click="loadDetail"
      >
        重试
      </el-button>
    </div>

    <template v-else-if="detail">
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <p class="text-sm text-muted-foreground">咨询编号 {{ detail.id }}</p>
          <h1 class="mt-1 break-words text-[1.75rem] font-semibold leading-tight tracking-[-0.035em]">
            {{ detail.subject }}
          </h1>
        </div>
        <span
          class="shrink-0 rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary"
        >
          {{ statusLabel(detail.status) }}
        </span>
      </div>

      <div class="mt-6 grid grid-cols-2 gap-3">
        <div class="rounded-2xl bg-muted p-4">
          <p class="text-xs text-muted-foreground">提交时间</p>
          <p class="mt-1 text-sm font-medium">{{ formatTaskTime(detail.openedAt) }}</p>
        </div>
        <div class="rounded-2xl bg-muted p-4">
          <p class="text-xs text-muted-foreground">联系顺序</p>
          <p class="mt-1 text-sm font-medium">{{ urgencyLabel(detail.urgency) }}</p>
        </div>
      </div>

      <section
        v-if="detail.resolution"
        class="mt-6 rounded-2xl border border-success/30 bg-success/8 p-5"
      >
        <p class="text-xs font-semibold tracking-[0.12em] text-success">处理结果</p>
        <p class="mt-2 whitespace-pre-wrap text-sm leading-6">{{ detail.resolution }}</p>
        <p
          v-if="detail.followUpPlan"
          class="mt-3 whitespace-pre-wrap border-t border-success/20 pt-3 text-sm leading-6 text-muted-foreground"
        >
          后续安排：{{ detail.followUpPlan }}
        </p>
      </section>

      <section class="mt-7">
        <h2 class="text-lg font-semibold">沟通记录</h2>
        <div class="mt-4 space-y-4">
          <article
            v-for="interaction in detail.interactions"
            :key="interaction.id"
            class="flex"
            :class="interaction.senderType === 'CLIENT' ? 'justify-end' : 'justify-start'"
          >
            <div
              class="max-w-[86%] rounded-2xl px-4 py-3"
              :class="messageClass(interaction.senderType)"
            >
              <p class="text-xs font-medium opacity-70">{{ senderLabel(interaction.senderType) }}</p>
              <p class="mt-1 whitespace-pre-wrap break-words text-sm leading-6">{{ interaction.content }}</p>
              <p class="mt-2 text-[11px] opacity-60">{{ formatTaskTime(interaction.occurredAt) }}</p>
            </div>
          </article>
        </div>
      </section>

      <section
        v-if="detail.status !== 'CLOSED'"
        class="mt-8 rounded-2xl border border-border bg-container p-4"
      >
        <h2 class="text-base font-semibold">补充信息</h2>
        <p class="mt-1 text-xs leading-5 text-muted-foreground">补充后，服务团队会在当前咨询中继续处理。</p>
        <el-input
          v-model="message"
          class="mt-4"
          maxlength="4000"
          placeholder="请输入需要补充的内容"
          :rows="4"
          show-word-limit
          type="textarea"
        />
        <p
          v-if="submitError"
          class="mt-3 text-sm text-error"
        >
          {{ submitError }}
        </p>
        <el-button
          type="primary"
          class="!mt-4 !h-11 !w-full !rounded-xl"
          :disabled="!message.trim()"
          :loading="submitting"
          @click="submitMessage"
        >
          提交补充
        </el-button>
      </section>

      <section
        v-else
        class="mt-8 rounded-2xl border border-border bg-muted p-5"
      >
        <p class="text-sm leading-6 text-muted-foreground">本次咨询已经关闭。如有新的事项，请重新发起咨询。</p>
        <el-button
          type="primary"
          class="!mt-4 !h-11 !rounded-xl"
          @click="router.push({ name: 'ClientConsultationNew' })"
        >
          发起新咨询
        </el-button>
      </section>

      <p class="mt-7 text-xs leading-5 text-muted-foreground">
        人工回复时间以服务安排为准；如遇紧急情况，请使用当地正式急救或就医渠道。
      </p>
    </template>
  </section>
</template>

<script setup>
  import { onMounted, ref } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import {
    addClientConsultationMessage,
    getClientConsultation
  } from '@/api/sleep-care/client-access'
  import {
    formatTaskTime,
    newIdempotencyKey,
    unwrapClientResponse
  } from './state'

  defineOptions({
    name: 'ClientConsultationDetail'
  })

  const route = useRoute()
  const router = useRouter()
  const consultationId = Number(route.params.id)
  const loading = ref(true)
  const submitting = ref(false)
  const detail = ref(null)
  const message = ref('')
  const errorMessage = ref('')
  const submitError = ref('')

  const statusLabels = {
    NEW: '新建',
    WAITING_ASSIGNMENT: '待分配',
    ASSIGNED: '已受理',
    HANDLING: '处理中',
    WAITING_CLIENT: '等待你补充',
    WAITING_COLLABORATION: '协同处理中',
    RESOLVED: '已解决',
    CLOSED: '已关闭'
  }

  const statusLabel = (value) => statusLabels[value] || '处理中'
  const urgencyLabel = (value) => value === 'EXPEDITED' ? '优先联系' : '常规联系'
  const senderLabel = (value) => ({
    CLIENT: '你',
    SERVICE_TEAM: '服务团队',
    SYSTEM: '系统'
  }[value] || '服务团队')
  const messageClass = (value) => ({
    CLIENT: 'bg-primary text-white',
    SERVICE_TEAM: 'border border-border bg-muted text-base-text',
    SYSTEM: 'border border-border bg-container text-muted-foreground'
  }[value] || 'border border-border bg-muted text-base-text')

  const loadDetail = async () => {
    loading.value = true
    errorMessage.value = ''
    try {
      detail.value = unwrapClientResponse(await getClientConsultation(consultationId))
    } catch (error) {
      errorMessage.value = error.message || '请检查网络后重试。'
    } finally {
      loading.value = false
    }
  }

  const submitMessage = async () => {
    const value = message.value.trim()
    if (!value || !detail.value) {
      return
    }
    submitting.value = true
    submitError.value = ''
    try {
      unwrapClientResponse(await addClientConsultationMessage(
        consultationId,
        newIdempotencyKey(),
        {
          expectedVersion: detail.value.version,
          message: value
        }
      ))
      message.value = ''
      await loadDetail()
    } catch (error) {
      submitError.value = error.message || '暂时无法提交补充，请稍后重试。'
    } finally {
      submitting.value = false
    }
  }

  onMounted(loadDetail)
</script>
