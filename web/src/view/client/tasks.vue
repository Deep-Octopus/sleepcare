<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-8">
    <div class="flex items-end justify-between gap-4">
      <div>
        <p class="text-sm text-muted-foreground">按时间查看服务安排</p>
        <h1 class="mt-1.5 text-[2rem] font-semibold tracking-[-0.04em]">
          我的随访
        </h1>
      </div>
      <span class="mb-1 text-sm tabular-nums text-muted-foreground">
        {{ tasks.length }} 项
      </span>
    </div>

    <ClientStatePanel
      v-if="loading"
      class="mt-9"
      title="正在读取随访安排"
      tone="muted"
      icon="lucide:loader-circle"
    />

    <ClientStatePanel
      v-else-if="errorMessage"
      class="mt-9"
      title="暂时无法读取随访安排"
      :description="errorMessage"
      tone="danger"
      icon="lucide:circle-alert"
    >
      <el-button class="!h-11 !rounded-xl" @click="loadTasks">
        重试
      </el-button>
    </ClientStatePanel>

    <ClientStatePanel
      v-else-if="tasks.length === 0"
      class="mt-9"
      title="当前没有随访安排"
      description="新的安排会显示在这里。"
      tone="muted"
      icon="lucide:calendar-check"
    />

    <div v-else class="mt-8 space-y-7">
      <section
        v-for="group in taskGroups"
        :key="group.key"
      >
        <div class="mb-3 flex items-center justify-between gap-3 px-1">
          <div class="flex items-center gap-2.5">
            <span
              class="inline-flex h-8 w-8 items-center justify-center rounded-lg"
              :class="group.iconClass"
              aria-hidden="true"
            >
              <svg-icon :icon="group.icon" />
            </span>
            <h2 class="font-semibold">{{ group.label }}</h2>
          </div>
          <span class="text-xs tabular-nums text-muted-foreground">
            {{ group.items.length }} 项
          </span>
        </div>

        <div
          class="overflow-hidden rounded-2xl border shadow-card"
          :class="group.key === 'active' ? 'border-primary-100 bg-primary-50' : 'border-border bg-container'"
        >
          <button
            v-for="(task, index) in group.items"
            :key="task.id"
            type="button"
            class="group flex w-full gap-3.5 p-4 text-left transition-[transform,background-color] focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-primary"
            :class="[
              index > 0 ? 'border-t border-border' : '',
              canOpenTask(task) ? 'hover:bg-muted active:scale-[0.99]' : 'cursor-default'
            ]"
            :disabled="!canOpenTask(task)"
            :aria-label="`${readableTaskTitle(task.title, task.dayCode)}，${taskStateCopy(task).label}`"
            @click="openTask(task)"
          >
            <span
              class="inline-flex h-11 w-11 shrink-0 flex-col items-center justify-center rounded-xl border bg-container leading-none"
              :class="task.accessible ? 'border-primary-200 text-primary' : 'border-border text-muted-foreground'"
              aria-hidden="true"
            >
              <span class="text-base font-semibold tabular-nums">{{ task.dayCode.replace('D', '') }}</span>
              <span class="mt-0.5 text-[10px]">次</span>
            </span>

            <span class="min-w-0 flex-1">
              <span class="flex items-start justify-between gap-2.5">
                <span class="min-w-0 break-words text-base font-semibold leading-6 text-base-text">
                  {{ readableTaskTitle(task.title, task.dayCode) }}
                </span>
                <ClientStatusBadge
                  :label="taskStateCopy(task).label"
                  :tone="stateTone(taskStateCopy(task).tone)"
                  :icon="taskStatusIcon(task)"
                />
              </span>

              <span class="mt-2.5 flex items-start gap-2 text-sm leading-5 text-muted-foreground">
                <svg-icon
                  icon="lucide:calendar-clock"
                  class="mt-0.5 shrink-0"
                  aria-hidden="true"
                />
                <span>{{ formatTaskWindow(task.openAt, task.dueAt) }}</span>
              </span>

              <span
                v-if="canOpenTask(task)"
                class="mt-3 flex items-center justify-end gap-1 text-sm font-medium text-primary"
              >
                {{ taskActionLabel(task) }}
                <svg-icon
                  icon="lucide:chevron-right"
                  class="transition-transform group-hover:translate-x-0.5"
                  aria-hidden="true"
                />
              </span>
            </span>
          </button>
        </div>
      </section>
    </div>

    <p class="mt-8 border-t border-border pt-5 text-xs leading-5 text-muted-foreground">
      页面只展示与你当前账号相关的随访安排。
    </p>
  </section>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { getClientTasks } from '@/api/sleep-care/client-access'
  import { formatTaskWindow, taskStateCopy, unwrapClientResponse } from './state'
  import { readableTaskTitle } from '@/utils/sleep-care-display'
  import ClientStatePanel from '@/components/client-mobile/client-state-panel.vue'
  import ClientStatusBadge from '@/components/client-mobile/client-status-badge.vue'

  defineOptions({
    name: 'ClientTasks'
  })

  const router = useRouter()
  const loading = ref(true)
  const tasks = ref([])
  const errorMessage = ref('')

  const taskGroups = computed(() => {
    const active = tasks.value.filter((task) => (
      task.accessible && task.executionStatus !== 'SUBMITTED'
    ))
    const upcoming = tasks.value.filter((task) => (
      task.executionStatus !== 'SUBMITTED' &&
      task.executionStatus !== 'CANCELLED' &&
      task.timingStatus === 'NOT_OPEN'
    ))
    const groupedIds = new Set([...active, ...upcoming].map((task) => task.id))
    const history = tasks.value.filter((task) => !groupedIds.has(task.id))

    return [
      {
        key: 'active',
        label: '现在可处理',
        icon: 'lucide:pen-line',
        iconClass: 'bg-primary-50 text-primary-700',
        items: active
      },
      {
        key: 'upcoming',
        label: '之后安排',
        icon: 'lucide:calendar-range',
        iconClass: 'bg-muted text-muted-foreground',
        items: upcoming
      },
      {
        key: 'history',
        label: '过往记录',
        icon: 'lucide:history',
        iconClass: 'bg-muted text-muted-foreground',
        items: history
      }
    ].filter((group) => group.items.length > 0)
  })

  const stateTone = (tone) => ({
    success: 'success',
    active: 'primary',
    danger: 'danger',
    muted: 'muted'
  })[tone] || 'muted'

  const canOpenTask = (task) => task.accessible || task.executionStatus === 'SUBMITTED'

  const taskStatusIcon = (task) => {
    if (task.executionStatus === 'SUBMITTED') {
      return 'lucide:circle-check'
    }
    if (task.timingStatus === 'EXPIRED') {
      return 'lucide:circle-x'
    }
    if (task.timingStatus === 'NOT_OPEN') {
      return 'lucide:clock-3'
    }
    return task.executionStatus === 'IN_PROGRESS'
      ? 'lucide:pen-line'
      : 'lucide:circle-play'
  }

  const taskActionLabel = (task) => {
    if (task.executionStatus === 'SUBMITTED') {
      return '查看提交结果'
    }
    if (task.executionStatus === 'IN_PROGRESS') {
      return '继续填写'
    }
    return '开始填写'
  }

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
