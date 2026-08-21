<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-4">
    <ClientBackButton
      class="mb-7"
      label="返回随访列表"
      @click="router.push({ name: 'ClientTasks' })"
    />

    <ClientStatePanel
      v-if="loading"
      title="正在读取随访说明"
      tone="muted"
      icon="lucide:loader-circle"
    />

    <ClientStatePanel
      v-else-if="errorMessage"
      title="随访暂不可进入"
      :description="errorMessage"
      tone="danger"
      icon="lucide:circle-alert"
    >
      <el-button class="!h-11 !rounded-xl" @click="loadTask">重试</el-button>
    </ClientStatePanel>

    <template v-else-if="task">
      <div>
        <div class="flex items-center justify-between gap-3">
          <p class="text-sm font-medium text-primary">第 {{ task.dayCode.replace('D', '') }} 次随访</p>
          <ClientStatusBadge
            :label="taskStateCopy(task).label"
            :tone="stateTone"
          />
        </div>
        <h1 class="mt-4 max-w-[21rem] text-[2rem] font-semibold leading-[1.15] tracking-[-0.04em]">
          {{ readableTaskTitle(task.title, task.dayCode) }}
        </h1>
      </div>

      <dl class="mt-7 grid grid-cols-2 border-y border-border">
        <div class="border-r border-border py-4 pr-4">
          <dt class="text-xs text-muted-foreground">开放时间</dt>
          <dd class="mt-1.5 text-sm font-medium">{{ formatTaskTime(task.openAt) }}</dd>
        </div>
        <div class="py-4 pl-4">
          <dt class="text-xs text-muted-foreground">截止时间</dt>
          <dd class="mt-1.5 text-sm font-medium">{{ formatTaskTime(task.dueAt) }}</dd>
        </div>
      </dl>

      <div class="mt-8">
        <h2 class="text-lg font-semibold tracking-[-0.02em]">开始前请确认</h2>
        <ul class="mt-5 space-y-4 text-sm leading-6 text-muted-foreground">
          <li class="flex gap-3">
            <svg-icon icon="lucide:check" class="mt-1 shrink-0 text-primary" />
            <span>按页面展示的题目逐项填写。</span>
          </li>
          <li class="flex gap-3">
            <svg-icon icon="lucide:check" class="mt-1 shrink-0 text-primary" />
            <span>填写过程中可以保存进度，提交前还能返回修改。</span>
          </li>
          <li class="flex gap-3">
            <svg-icon icon="lucide:check" class="mt-1 shrink-0 text-primary" />
            <span>提交成功只表示已收到，后续处理状态以工作人员记录为准。</span>
          </li>
        </ul>
      </div>

      <label class="mt-8 flex min-h-16 cursor-pointer items-start gap-3 border-y border-border py-4">
        <el-checkbox v-model="confirmed" size="large" />
        <span class="pt-0.5 text-sm font-medium leading-6">我已阅读以上说明，确认开始填写</span>
      </label>

      <ClientStatePanel
        v-if="actionError"
        class="mt-5"
        title="暂时无法开始填写"
        :description="actionError"
        tone="danger"
        icon="lucide:circle-alert"
      />

      <el-button
        type="primary"
        class="!mt-7 !h-14 !w-full !rounded-xl !text-base !font-semibold"
        :disabled="!confirmed || actionLoading"
        :loading="actionLoading"
        @click="continueToForm"
      >
        {{ task.started ? '继续填写' : '确认并开始填写' }}
      </el-button>
    </template>
  </section>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { getClientTask, recordClientInteraction } from '@/api/sleep-care/client-access'
  import { formatTaskTime, newIdempotencyKey, taskStateCopy, unwrapClientResponse } from './state'
  import { readableTaskTitle } from '@/utils/sleep-care-display'
  import ClientBackButton from '@/components/client-mobile/client-back-button.vue'
  import ClientStatePanel from '@/components/client-mobile/client-state-panel.vue'
  import ClientStatusBadge from '@/components/client-mobile/client-status-badge.vue'

  defineOptions({
    name: 'ClientTask'
  })

  const route = useRoute()
  const router = useRouter()
  const taskId = Number(route.params.taskId)
  const loading = ref(true)
  const actionLoading = ref(false)
  const task = ref(null)
  const confirmed = ref(false)
  const errorMessage = ref('')
  const actionError = ref('')
  const stateTone = computed(() => {
    if (!task.value) {
      return 'muted'
    }
    return ({
      active: 'primary',
      success: 'success',
      danger: 'danger',
      muted: 'muted'
    })[taskStateCopy(task.value).tone] || 'muted'
  })

  const record = async (interactionType) => {
    const data = unwrapClientResponse(await recordClientInteraction(taskId, newIdempotencyKey(), {
      expectedVersion: task.value.version,
      interactionType
    }))
    task.value.version = data.taskVersion
    task.value.executionStatus = data.executionStatus
    task.value[interactionType.toLowerCase()] = true
  }

  const loadTask = async () => {
    loading.value = true
    errorMessage.value = ''
    try {
      task.value = unwrapClientResponse(await getClientTask(taskId))
      if (task.value.executionStatus === 'SUBMITTED') {
        await router.replace({ name: 'ClientTaskSuccess', params: { taskId } })
        return
      }
      if (!task.value.opened) {
        await record('OPENED')
      }
      confirmed.value = task.value.consented || task.value.started
    } catch (error) {
      errorMessage.value = error.message || '请稍后重试。'
    } finally {
      loading.value = false
    }
  }

  const continueToForm = async () => {
    actionLoading.value = true
    actionError.value = ''
    try {
      if (!task.value.consented) {
        await record('CONSENTED')
      }
      if (!task.value.started) {
        await record('STARTED')
      }
      await router.push({ name: 'ClientTaskForm', params: { taskId } })
    } catch (error) {
      actionError.value = error.message || '暂时无法开始填写，请重试。'
    } finally {
      actionLoading.value = false
    }
  }

  onMounted(loadTask)
</script>
