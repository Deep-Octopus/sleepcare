<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-7">
    <div class="flex items-end justify-between gap-4">
      <div>
        <p class="text-sm text-muted-foreground">
          你的随访安排
        </p>
        <h1 class="mt-1 text-[1.8rem] font-semibold tracking-[-0.035em]">
          我的随访
        </h1>
      </div>
      <span class="mb-1 text-sm tabular-nums text-primary">
        {{ tasks.length }} 项
      </span>
    </div>

    <div
      v-if="loading"
      class="mt-10 rounded-2xl border border-border bg-muted p-5 text-sm text-muted-foreground"
    >
      正在读取任务…
    </div>

    <div
      v-else-if="errorMessage"
      class="mt-10 rounded-2xl border border-error-200 bg-error-50 p-5"
    >
      <p class="font-medium text-error-700">
        暂时无法读取任务
      </p>
      <p class="mt-2 text-sm leading-6 text-error-600">
        {{ errorMessage }}
      </p>
      <el-button class="!mt-4 !h-11 !rounded-xl" @click="loadTasks">
        重试
      </el-button>
    </div>

    <div
      v-else-if="tasks.length === 0"
      class="mt-10 rounded-2xl border border-dashed border-border p-7 text-center"
    >
      <p class="text-base font-medium">当前没有可展示的任务</p>
      <p class="mt-2 text-sm text-muted-foreground">新的随访安排会显示在这里。</p>
    </div>

    <div v-else class="mt-7 space-y-3">
      <button
        v-for="task in tasks"
        :key="task.id"
        type="button"
        class="group w-full rounded-2xl border border-border bg-container p-4 text-left transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
        :class="task.accessible || task.executionStatus === 'SUBMITTED' ? 'cursor-pointer hover:border-primary hover:bg-muted' : 'cursor-default opacity-72'"
        @click="openTask(task)"
      >
        <div class="flex gap-4">
          <div class="flex h-14 w-12 shrink-0 flex-col items-center justify-center rounded-xl bg-muted text-primary">
            <span class="text-[10px] font-semibold">第</span>
            <span class="text-lg font-bold leading-none">{{ task.dayCode.replace('D', '') }}</span>
            <span class="text-[10px] font-semibold">次</span>
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex items-start justify-between gap-3">
              <h2 class="text-base font-semibold leading-6">{{ readableTaskTitle(task.title, task.dayCode) }}</h2>
              <span
                class="shrink-0 rounded-full px-2.5 py-1 text-xs font-medium"
                :class="stateClass(taskStateCopy(task).tone)"
              >
                {{ taskStateCopy(task).label }}
              </span>
            </div>
            <p class="mt-2 text-sm text-muted-foreground">
              开放 {{ formatTaskTime(task.openAt) }} · 截止 {{ formatTaskTime(task.dueAt) }}
            </p>
          </div>
        </div>
      </button>
    </div>

    <p class="mt-8 border-t border-border pt-5 text-xs leading-5 text-muted-foreground">
      页面只展示与你当前账号相关的随访安排。
    </p>
  </section>
</template>

<script setup>
  import { onMounted, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { getClientTasks } from '@/api/sleep-care/client-access'
  import { formatTaskTime, taskStateCopy, unwrapClientResponse } from './state'
  import { readableTaskTitle } from '@/utils/sleep-care-display'

  defineOptions({
    name: 'ClientTasks'
  })

  const router = useRouter()
  const loading = ref(true)
  const tasks = ref([])
  const errorMessage = ref('')

  const stateClass = (tone) => ({
    success: 'bg-success-50 text-success-700',
    active: 'bg-primary-50 text-primary-700',
    danger: 'bg-error-50 text-error-700',
    muted: 'bg-muted text-muted-foreground'
  })[tone]

  const loadTasks = async () => {
    loading.value = true
    errorMessage.value = ''
    try {
      const data = unwrapClientResponse(await getClientTasks({ page: 1, pageSize: 100 }))
      tasks.value = data.list || []
    } catch (error) {
      errorMessage.value = error.message || '请检查网络后重试。'
    } finally {
      loading.value = false
    }
  }

  const openTask = (task) => {
    if (task.executionStatus === 'SUBMITTED') {
      router.push({ name: 'ClientTaskSuccess', params: { taskId: task.id } })
      return
    }
    if (task.accessible) {
      router.push({ name: 'ClientTask', params: { taskId: task.id } })
    }
  }

  onMounted(loadTasks)
</script>
