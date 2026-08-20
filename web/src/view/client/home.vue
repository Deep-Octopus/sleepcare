<template>
  <section class="px-5 pb-8 pt-7">
    <div
      v-if="loading"
      class="rounded-2xl bg-muted p-5 text-sm text-muted-foreground"
    >
      正在准备你的服务首页…
    </div>

    <div
      v-else-if="errorMessage"
      class="rounded-2xl border border-error-200 bg-error-50 p-5"
    >
      <p class="font-medium text-error-700">暂时无法打开首页</p>
      <p class="mt-2 text-sm leading-6 text-error-600">{{ errorMessage }}</p>
      <el-button
        class="!mt-4 !h-11 !rounded-xl"
        @click="loadHome"
      >
        重新加载
      </el-button>
    </div>

    <template v-else>
      <div class="flex items-start justify-between gap-5">
        <div class="min-w-0">
          <p class="text-sm text-muted-foreground">{{ todayLabel }}</p>
          <h1 class="mt-1 truncate text-3xl font-semibold tracking-tight">
            你好，{{ profile.displayName }}
          </h1>
        </div>
        <span class="mt-1 inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-primary text-lg font-semibold text-white">
          {{ profileInitial }}
        </span>
      </div>

      <section class="relative mt-8 overflow-hidden rounded-3xl bg-primary p-5 text-white shadow-card">
        <div class="absolute -right-5 -top-7 h-28 w-28 rounded-full border border-white/20" />
        <div class="absolute -right-10 top-4 h-28 w-28 rounded-full border border-white/10" />
        <div class="relative">
          <div class="flex items-center gap-2 text-sm text-white/80">
            <svg-icon icon="lucide:calendar-days" />
            <span>接下来</span>
          </div>

          <template v-if="nextTask">
            <h2 class="mt-4 max-w-[18rem] text-2xl font-semibold leading-tight">
              {{ readableTaskTitle(nextTask.title, nextTask.dayCode) }}
            </h2>
            <p class="mt-3 text-sm leading-6 text-white/80">
              {{ nextTaskHint }}
            </p>
            <button
              type="button"
              class="mt-5 inline-flex min-h-11 items-center gap-2 rounded-xl bg-white px-4 font-medium text-primary transition-colors hover:bg-primary-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
              @click="openNextTask"
            >
              {{ nextTask.executionStatus === 'SUBMITTED' ? '查看提交结果' : '继续处理' }}
              <svg-icon icon="lucide:arrow-right" />
            </button>
          </template>

          <template v-else>
            <h2 class="mt-4 text-2xl font-semibold">当前没有待处理随访</h2>
            <p class="mt-3 text-sm leading-6 text-white/80">
              新的安排会显示在这里，你也可以查看全部随访记录。
            </p>
            <button
              type="button"
              class="mt-5 inline-flex min-h-11 items-center gap-2 rounded-xl bg-white px-4 font-medium text-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
              @click="router.push({ name: 'ClientTasks' })"
            >
              查看全部随访
              <svg-icon icon="lucide:arrow-right" />
            </button>
          </template>
        </div>
      </section>

      <div class="mt-4 grid grid-cols-4 divide-x divide-border rounded-2xl border border-border bg-container py-4">
        <div
          v-for="item in taskCounts"
          :key="item.label"
          class="text-center"
        >
          <p class="text-xl font-semibold tabular-nums">{{ item.value }}</p>
          <p class="mt-1 text-xs text-muted-foreground">{{ item.label }}</p>
        </div>
      </div>

      <section class="mt-8">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-muted-foreground">需要帮助时</p>
            <h2 class="mt-1 text-xl font-semibold">联系服务团队</h2>
          </div>
          <button
            type="button"
            class="min-h-10 rounded-lg px-2 text-sm font-medium text-primary focus-visible:outline-2 focus-visible:outline-primary"
            @click="router.push({ name: 'ClientConsultationNew' })"
          >
            发起咨询
          </button>
        </div>

        <button
          type="button"
          class="mt-4 flex w-full items-center gap-4 rounded-2xl border border-border bg-container p-4 text-left transition-colors hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
          @click="router.push({ name: 'ClientConsultations' })"
        >
          <span class="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-muted text-xl text-primary">
            <svg-icon icon="lucide:messages-square" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block font-medium">在线咨询</span>
            <span class="mt-1 block truncate text-sm text-muted-foreground">{{ consultationHint }}</span>
          </span>
          <svg-icon
            icon="lucide:chevron-right"
            class="text-muted-foreground"
          />
        </button>

        <button
          type="button"
          class="mt-3 flex w-full items-center gap-4 rounded-2xl border border-border bg-container p-4 text-left transition-colors hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
          @click="router.push({ name: 'ClientSatisfaction' })"
        >
          <span class="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-muted text-xl text-primary">
            <svg-icon icon="lucide:star" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block font-medium">服务评价</span>
            <span class="mt-1 block text-sm text-muted-foreground">{{ satisfactionHint }}</span>
          </span>
          <svg-icon
            icon="lucide:chevron-right"
            class="text-muted-foreground"
          />
        </button>
      </section>

      <button
        type="button"
        class="mx-auto mt-10 block min-h-10 rounded-lg px-3 text-sm text-muted-foreground hover:text-error focus-visible:outline-2 focus-visible:outline-error"
        :disabled="signingOut"
        @click="signOut"
      >
        {{ signingOut ? '正在退出…' : '退出当前账号' }}
      </button>
    </template>
  </section>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import {
    getClientConsultations,
    getClientProfile,
    getClientSatisfactionRequests,
    getClientTasks,
    logoutClient
  } from '@/api/sleep-care/client-access'
  import { readableTaskTitle } from '@/utils/sleep-care-display'
  import {
    clearClientAuthMode,
    clearClientDraftState
  } from '@/utils/client-session'
  import { formatTaskTime, unwrapClientResponse } from './state'

  defineOptions({
    name: 'ClientHome'
  })

  const router = useRouter()
  const loading = ref(true)
  const signingOut = ref(false)
  const errorMessage = ref('')
  const profile = ref({ displayName: '', displayCode: '' })
  const tasks = ref([])
  const consultations = ref([])
  const satisfactionRequests = ref([])

  const todayLabel = new Intl.DateTimeFormat('zh-CN', {
    month: 'long',
    day: 'numeric',
    weekday: 'long'
  }).format(new Date())

  const profileInitial = computed(() => profile.value.displayName?.slice(0, 1) || '我')
  const nextTask = computed(() => (
    tasks.value.find((task) => task.executionStatus === 'IN_PROGRESS') ||
    tasks.value.find((task) => task.accessible) ||
    null
  ))
  const nextTaskHint = computed(() => {
    if (!nextTask.value) {
      return ''
    }
    if (nextTask.value.executionStatus === 'SUBMITTED') {
      return `已于 ${formatTaskTime(nextTask.value.submittedAt)} 提交`
    }
    return `请在 ${formatTaskTime(nextTask.value.dueAt)} 前完成`
  })
  const taskCounts = computed(() => [
    {
      label: '可填写',
      value: tasks.value.filter((task) => (
        task.accessible && task.executionStatus !== 'SUBMITTED'
      )).length
    },
    {
      label: '未开放',
      value: tasks.value.filter((task) => (
        task.executionStatus !== 'SUBMITTED' && task.timingStatus === 'NOT_OPEN'
      )).length
    },
    {
      label: '已结束',
      value: tasks.value.filter((task) => (
        task.executionStatus !== 'SUBMITTED' && task.timingStatus === 'EXPIRED'
      )).length
    },
    {
      label: '已提交',
      value: tasks.value.filter((task) => task.executionStatus === 'SUBMITTED').length
    }
  ])
  const consultationHint = computed(() => {
    const waiting = consultations.value.filter((item) => item.status === 'WAITING_CLIENT').length
    if (waiting > 0) {
      return `${waiting} 条记录等待你补充`
    }
    return consultations.value.length > 0
      ? `共有 ${consultations.value.length} 条沟通记录`
      : '有问题可以随时留言'
  })
  const satisfactionHint = computed(() => {
    const pending = satisfactionRequests.value.filter((item) => item.status === 'PENDING').length
    return pending > 0 ? `${pending} 项服务等待评价` : '查看已收到的评价邀请'
  })

  const loadHome = async () => {
    loading.value = true
    errorMessage.value = ''
    try {
      const [profileData, taskData, consultationData, satisfactionData] = await Promise.all([
        getClientProfile(),
        getClientTasks({ page: 1, pageSize: 100 }),
        getClientConsultations({ page: 1, pageSize: 100 }),
        getClientSatisfactionRequests({ page: 1, pageSize: 100 })
      ])
      profile.value = unwrapClientResponse(profileData)
      tasks.value = unwrapClientResponse(taskData).list || []
      consultations.value = unwrapClientResponse(consultationData).list || []
      satisfactionRequests.value = unwrapClientResponse(satisfactionData).list || []
    } catch (error) {
      errorMessage.value = error.message || '请检查网络后重试。'
    } finally {
      loading.value = false
    }
  }

  const openNextTask = () => {
    if (!nextTask.value) {
      return
    }
    const routeName = nextTask.value.executionStatus === 'SUBMITTED'
      ? 'ClientTaskSuccess'
      : 'ClientTask'
    router.push({
      name: routeName,
      params: { taskId: nextTask.value.id }
    })
  }

  const signOut = async () => {
    signingOut.value = true
    try {
      unwrapClientResponse(await logoutClient())
      clearClientDraftState()
      clearClientAuthMode()
      await router.replace({ name: 'ClientLogin' })
    } catch (error) {
      errorMessage.value = error.message || '暂时无法退出，请稍后重试。'
    } finally {
      signingOut.value = false
    }
  }

  onMounted(loadHome)
</script>
