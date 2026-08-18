<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-6">
    <button
      type="button"
      class="mb-7 inline-flex min-h-10 items-center gap-2 rounded-lg px-1 text-sm text-[#47766b] focus-visible:outline-2 focus-visible:outline-[#2c806c] dark:text-emerald-300"
      @click="router.push({ name: 'ClientTasks' })"
    >
      <span aria-hidden="true">←</span>
      返回任务列表
    </button>

    <div v-if="loading" class="rounded-2xl border border-[#dce8e3] p-5 text-sm dark:border-slate-800">
      正在读取任务说明…
    </div>

    <div v-else-if="errorMessage" class="rounded-2xl border border-red-200 bg-red-50 p-5 dark:border-red-950 dark:bg-red-950/40">
      <p class="font-medium text-red-700 dark:text-red-200">任务暂不可进入</p>
      <p class="mt-2 text-sm leading-6 text-red-600 dark:text-red-300">{{ errorMessage }}</p>
      <el-button class="!mt-4 !h-11 !rounded-xl" @click="loadTask">重试</el-button>
    </div>

    <template v-else-if="task">
      <div class="flex items-center gap-3">
        <div class="flex h-14 w-12 flex-col items-center justify-center rounded-xl bg-[#e5f0ec] text-[#276b5b] dark:bg-emerald-950 dark:text-emerald-200">
          <span class="text-[10px] font-semibold tracking-wider">DAY</span>
          <span class="text-lg font-bold leading-none">{{ task.dayCode.replace('D', '') }}</span>
        </div>
        <div>
          <p class="text-sm text-[#60766f] dark:text-slate-400">{{ taskStateCopy(task).label }}</p>
          <h1 class="mt-0.5 text-[1.75rem] font-semibold leading-tight tracking-[-0.035em]">{{ task.title }}</h1>
        </div>
      </div>

      <div class="mt-7 grid grid-cols-2 gap-3">
        <div class="rounded-2xl bg-[#f3f7f5] p-4 dark:bg-slate-800">
          <p class="text-xs text-[#71847e] dark:text-slate-400">开放时间</p>
          <p class="mt-1 text-sm font-medium">{{ formatTaskTime(task.openAt) }}</p>
        </div>
        <div class="rounded-2xl bg-[#f3f7f5] p-4 dark:bg-slate-800">
          <p class="text-xs text-[#71847e] dark:text-slate-400">截止时间</p>
          <p class="mt-1 text-sm font-medium">{{ formatTaskTime(task.dueAt) }}</p>
        </div>
      </div>

      <div class="mt-7 rounded-2xl border border-[#dce8e3] p-5 dark:border-slate-800">
        <h2 class="text-base font-semibold">开始前请确认</h2>
        <ul class="mt-4 space-y-3 text-sm leading-6 text-[#536c65] dark:text-slate-300">
          <li class="flex gap-3"><span class="text-[#2c806c]">01</span><span>按页面展示的题目逐项填写。</span></li>
          <li class="flex gap-3"><span class="text-[#2c806c]">02</span><span>填写过程中可以保存进度，提交前还能返回修改。</span></li>
          <li class="flex gap-3"><span class="text-[#2c806c]">03</span><span>提交成功只表示已收到，后续处理状态以工作人员记录为准。</span></li>
        </ul>
      </div>

      <label class="mt-6 flex min-h-14 cursor-pointer items-start gap-3 rounded-xl bg-[#edf5f1] p-4 dark:bg-emerald-950/50">
        <el-checkbox v-model="confirmed" size="large" />
        <span class="pt-0.5 text-sm leading-6">我已阅读以上说明，确认开始填写</span>
      </label>

      <p v-if="actionError" class="mt-4 text-sm text-red-600 dark:text-red-300">{{ actionError }}</p>

      <el-button
        type="primary"
        class="!mt-6 !h-13 !w-full !rounded-xl !text-base"
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
  import { onMounted, ref } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { getClientTask, recordClientInteraction } from '@/api/sleep-care/client-access'
  import { formatTaskTime, newIdempotencyKey, taskStateCopy, unwrapClientResponse } from './state'

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
