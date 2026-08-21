<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-4">
    <ClientBackButton
      class="mb-7"
      label="返回咨询记录"
      @click="router.push({ name: 'ClientConsultations' })"
    />

    <ClientStatePanel
      v-if="loading"
      title="正在读取咨询详情"
      tone="muted"
      icon="lucide:loader-circle"
    />

    <ClientStatePanel
      v-else-if="errorMessage"
      title="暂时无法读取咨询详情"
      :description="errorMessage"
      tone="danger"
      icon="lucide:circle-alert"
    >
      <el-button
        class="!h-11 !rounded-xl"
        @click="loadDetail"
      >
        重试
      </el-button>
    </ClientStatePanel>

    <template v-else-if="detail">
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <p class="text-sm font-medium text-primary">在线咨询</p>
          <h1 class="mt-3 break-words text-[2rem] font-semibold leading-[1.15] tracking-[-0.04em]">
            {{ detail.subject }}
          </h1>
        </div>
        <ClientStatusBadge
          :label="statusLabel(detail.status)"
          :tone="statusTone(detail.status)"
        />
      </div>

      <dl class="mt-7 grid grid-cols-2 border-y border-border">
        <div class="border-r border-border py-4 pr-4">
          <dt class="text-xs text-muted-foreground">提交时间</dt>
          <dd class="mt-1.5 text-sm font-medium">{{ formatTaskTime(detail.openedAt) }}</dd>
        </div>
        <div class="py-4 pl-4">
          <dt class="text-xs text-muted-foreground">联系顺序</dt>
          <dd class="mt-1.5 text-sm font-medium">{{ urgencyLabel(detail.urgency) }}</dd>
        </div>
      </dl>

      <section
        v-if="detail.resolution"
        class="mt-7 border-l-2 border-success bg-success-50 px-4 py-4"
      >
        <p class="text-sm font-semibold text-success-700">处理结果</p>
        <p class="mt-2 whitespace-pre-wrap text-sm leading-6">{{ detail.resolution }}</p>
        <p
          v-if="detail.followUpPlan"
          class="mt-3 whitespace-pre-wrap border-t border-success-200 pt-3 text-sm leading-6 text-muted-foreground"
        >
          后续安排：{{ detail.followUpPlan }}
        </p>
      </section>

      <section class="mt-9">
        <h2 class="text-xl font-semibold tracking-[-0.025em]">沟通记录</h2>
        <div class="mt-5 space-y-4">
          <article
            v-for="interaction in detail.interactions"
            :key="interaction.id"
            class="flex"
            :class="interaction.senderType === 'CLIENT' ? 'justify-end' : 'justify-start'"
          >
            <div
              class="max-w-[86%] rounded-xl px-4 py-3"
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
        class="mt-9 border-y border-border py-5"
      >
        <h2 class="text-lg font-semibold">补充信息</h2>
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
          class="!mt-5 !h-12 !w-full !rounded-xl !font-semibold"
          :disabled="!message.trim()"
          :loading="submitting"
          @click="submitMessage"
        >
          提交补充
        </el-button>
      </section>

      <ClientStatePanel
        v-else
        class="mt-9"
        title="本次咨询已经关闭"
        description="如有新的事项，请重新发起咨询。"
        tone="muted"
        icon="lucide:message-circle-off"
      >
        <el-button
          type="primary"
          class="!h-11 !rounded-xl"
          @click="router.push({ name: 'ClientConsultationNew' })"
        >
          发起新咨询
        </el-button>
      </ClientStatePanel>

      <p class="mt-8 border-t border-border pt-5 text-xs leading-5 text-muted-foreground">
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
  import ClientBackButton from '@/components/client-mobile/client-back-button.vue'
  import ClientStatePanel from '@/components/client-mobile/client-state-panel.vue'
  import ClientStatusBadge from '@/components/client-mobile/client-status-badge.vue'

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
  const statusTone = (value) => {
    if (value === 'RESOLVED') {
      return 'success'
    }
    if (value === 'CLOSED') {
      return 'muted'
    }
    if (value === 'WAITING_CLIENT') {
      return 'warning'
    }
    return 'primary'
  }
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
