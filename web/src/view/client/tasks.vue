<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-7">
    <div class="flex items-end justify-between gap-4">
      <div>
        <p class="text-sm text-[#60766f] dark:text-slate-400">
          本次可访问范围
        </p>
        <h1 class="mt-1 text-[1.8rem] font-semibold tracking-[-0.035em]">
          我的任务
        </h1>
      </div>
      <span class="mb-1 text-sm tabular-nums text-[#47766b] dark:text-emerald-300">
        {{ tasks.length }} 项
      </span>
    </div>

    <div
      v-if="loading"
      class="mt-10 rounded-2xl border border-[#dce8e3] bg-[#f7faf8] p-5 text-sm text-[#60766f] dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300"
    >
      正在读取任务…
    </div>

    <div
      v-else-if="errorMessage"
      class="mt-10 rounded-2xl border border-red-200 bg-red-50 p-5 dark:border-red-950 dark:bg-red-950/40"
    >
      <p class="font-medium text-red-700 dark:text-red-200">
        暂时无法读取任务
      </p>
      <p class="mt-2 text-sm leading-6 text-red-600 dark:text-red-300">
        {{ errorMessage }}
      </p>
      <el-button class="!mt-4 !h-11 !rounded-xl" @click="loadTasks">
        重试
      </el-button>
    </div>

    <div
      v-else-if="tasks.length === 0"
      class="mt-10 rounded-2xl border border-dashed border-[#bfd4cc] p-7 text-center dark:border-slate-700"
    >
      <p class="text-base font-medium">当前没有可展示的任务</p>
      <p class="mt-2 text-sm text-[#6e827c] dark:text-slate-400">可以稍后从原入口再次查看。</p>
    </div>

    <div v-else class="mt-7 space-y-3">
      <button
        v-for="task in tasks"
        :key="task.id"
        type="button"
        class="group w-full rounded-2xl border border-[#dce8e3] bg-white p-4 text-left transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#2c806c] dark:border-slate-800 dark:bg-slate-900"
        :class="task.accessible || task.executionStatus === 'SUBMITTED' ? 'cursor-pointer hover:border-[#8db9aa] hover:bg-[#fbfdfc]' : 'cursor-default opacity-72'"
        @click="openTask(task)"
      >
        <div class="flex gap-4">
          <div class="flex h-14 w-12 shrink-0 flex-col items-center justify-center rounded-xl bg-[#e5f0ec] text-[#276b5b] dark:bg-emerald-950 dark:text-emerald-200">
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
            <p class="mt-2 text-sm text-[#71847e] dark:text-slate-400">
              开放 {{ formatTaskTime(task.openAt) }} · 截止 {{ formatTaskTime(task.dueAt) }}
            </p>
          </div>
        </div>
      </button>
    </div>

    <section class="mt-8 rounded-2xl border border-border bg-muted p-5">
      <div class="flex items-start justify-between gap-4">
        <div>
          <p class="text-xs font-semibold tracking-[0.12em] text-primary">联系服务</p>
          <h2 class="mt-2 text-lg font-semibold">有问题需要工作人员协助？</h2>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">
            可以随时提交在线咨询，并在本页查看服务团队的回复。
          </p>
        </div>
        <span
          class="mt-1 text-xl text-primary"
          aria-hidden="true"
        >
          ↗
        </span>
      </div>
      <div class="mt-4 grid grid-cols-2 gap-3">
        <el-button
          class="!m-0 !h-11 !rounded-xl"
          @click="router.push({ name: 'ClientConsultations' })"
        >
          咨询记录
        </el-button>
        <el-button
          type="primary"
          class="!m-0 !h-11 !rounded-xl"
          @click="router.push({ name: 'ClientConsultationNew' })"
        >
          发起咨询
        </el-button>
      </div>
    </section>

    <section class="mt-4 rounded-2xl border border-border bg-container p-5">
      <div class="flex items-start justify-between gap-4">
        <div>
          <p class="text-xs font-semibold tracking-[0.12em] text-primary">服务评价</p>
          <h2 class="mt-2 text-lg font-semibold">分享本次服务体验</h2>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">
            已关闭的服务会在这里生成评价邀请，可填写 1 至 5 星和补充意见。
          </p>
        </div>
        <span class="mt-1 text-xl text-primary" aria-hidden="true">☆</span>
      </div>
      <el-button
        class="!mt-4 !h-11 !w-full !rounded-xl"
        @click="router.push({ name: 'ClientSatisfaction' })"
      >
        查看服务评价
      </el-button>
    </section>

    <p class="mt-8 border-t border-[#e5ece9] pt-5 text-xs leading-5 text-[#7b8e89] dark:border-slate-800 dark:text-slate-500">
      为保护信息安全，你只能查看本页列出的任务。
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
    success: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-200',
    active: 'bg-[#e5f0ec] text-[#276b5b] dark:bg-emerald-950 dark:text-emerald-200',
    danger: 'bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-200',
    muted: 'bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400'
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
