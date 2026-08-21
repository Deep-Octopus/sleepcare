<template>
  <section class="px-5 pb-8 pt-8">
    <ClientStatePanel
      v-if="loading"
      title="正在准备你的服务首页"
      description="随访安排和服务消息马上就好。"
      tone="muted"
      icon="lucide:loader-circle"
    />

    <ClientStatePanel
      v-else-if="errorMessage"
      title="暂时无法打开首页"
      :description="errorMessage"
      tone="danger"
      icon="lucide:circle-alert"
    >
      <el-button
        class="!h-11 !rounded-xl"
        @click="loadHome"
      >
        重新加载
      </el-button>
    </ClientStatePanel>

    <template v-else>
      <div class="flex items-center justify-between gap-5">
        <div class="min-w-0">
          <p class="text-sm text-muted-foreground">{{ todayLabel }}</p>
          <h1 class="mt-1.5 truncate text-[2rem] font-semibold leading-tight tracking-[-0.04em]">
            你好，{{ profile.displayName }}
          </h1>
        </div>
        <span class="inline-flex h-12 w-12 shrink-0 items-center justify-center rounded-full border border-border bg-muted text-lg font-semibold text-primary">
          {{ profileInitial }}
        </span>
      </div>

      <section class="mt-9 border-y border-border py-6">
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-2 text-sm font-medium text-muted-foreground">
            <svg-icon
              icon="lucide:calendar-days"
              class="text-primary"
            />
            <span>接下来</span>
          </div>
          <ClientStatusBadge
            v-if="nextTask"
            :label="taskStateCopy(nextTask).label"
            :tone="nextTaskTone"
          />
        </div>

        <template v-if="nextTask">
          <h2 class="mt-5 max-w-[20rem] text-[1.65rem] font-semibold leading-[1.2] tracking-[-0.035em]">
            {{ readableTaskTitle(nextTask.title, nextTask.dayCode) }}
          </h2>
          <p class="mt-3 text-sm leading-6 text-muted-foreground">
            {{ nextTaskHint }}
          </p>
          <button
            type="button"
            class="mt-6 inline-flex min-h-12 items-center gap-2 rounded-xl bg-primary px-5 font-semibold text-white transition-transform focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary active:scale-[0.98]"
            @click="openNextTask"
          >
            {{ nextTask.executionStatus === 'SUBMITTED' ? '查看提交结果' : '继续处理' }}
            <svg-icon icon="lucide:arrow-right" />
          </button>
        </template>

        <template v-else>
          <h2 class="mt-5 text-[1.65rem] font-semibold leading-[1.2] tracking-[-0.035em]">
            当前没有待处理随访
          </h2>
          <p class="mt-3 text-sm leading-6 text-muted-foreground">
            新的安排会显示在这里，你也可以查看全部随访记录。
          </p>
          <button
            type="button"
            class="mt-6 inline-flex min-h-12 items-center gap-2 rounded-xl border border-border px-5 font-semibold transition-colors hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary active:scale-[0.98]"
            @click="router.push({ name: 'ClientTasks' })"
          >
            查看全部随访
            <svg-icon icon="lucide:arrow-right" />
          </button>
        </template>
      </section>

      <section class="mt-9">
        <div class="flex items-center justify-between gap-4">
          <h2 class="text-xl font-semibold tracking-[-0.025em]">随访概览</h2>
          <button
            type="button"
            class="min-h-10 rounded-xl px-2 text-sm font-medium text-primary focus-visible:outline-2 focus-visible:outline-primary"
            @click="router.push({ name: 'ClientTasks' })"
          >
            查看全部
          </button>
        </div>

        <div class="mt-4 grid grid-cols-[1.1fr_1fr] border-y border-border">
          <div class="border-r border-border py-5 pr-5">
            <p class="text-sm text-muted-foreground">现在可填写</p>
            <p class="mt-2 text-[2.5rem] font-semibold leading-none tracking-[-0.04em] tabular-nums text-primary">
              {{ taskSummary.available }}
            </p>
            <p class="mt-3 text-xs leading-5 text-muted-foreground">需要你处理的随访</p>
          </div>
          <dl class="divide-y divide-border pl-5">
            <div class="flex items-center justify-between gap-3 py-3">
              <dt class="text-sm text-muted-foreground">之后开放</dt>
              <dd class="font-semibold tabular-nums">{{ taskSummary.upcoming }}</dd>
            </div>
            <div class="flex items-center justify-between gap-3 py-3">
              <dt class="text-sm text-muted-foreground">已经提交</dt>
              <dd class="font-semibold tabular-nums">{{ taskSummary.submitted }}</dd>
            </div>
            <div class="flex items-center justify-between gap-3 py-3">
              <dt class="text-sm text-muted-foreground">已经结束</dt>
              <dd class="font-semibold tabular-nums">{{ taskSummary.expired }}</dd>
            </div>
          </dl>
        </div>
      </section>

      <section class="mt-10">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-xl font-semibold tracking-[-0.025em]">服务与反馈</h2>
            <p class="mt-1 text-sm text-muted-foreground">需要协助时，可在这里留言</p>
          </div>
          <button
            type="button"
            class="min-h-10 rounded-xl px-2 text-sm font-medium text-primary focus-visible:outline-2 focus-visible:outline-primary"
            @click="router.push({ name: 'ClientConsultationNew' })"
          >
            发起咨询
          </button>
        </div>

        <div class="mt-4 divide-y divide-border border-y border-border">
          <button
            type="button"
            class="flex min-h-20 w-full items-center gap-4 py-4 text-left transition-colors hover:text-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary active:scale-[0.99]"
            @click="router.push({ name: 'ClientConsultations' })"
          >
            <span class="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-muted text-xl text-primary">
              <svg-icon icon="lucide:messages-square" />
            </span>
            <span class="min-w-0 flex-1">
              <span class="block font-medium text-base-text">在线咨询</span>
              <span class="mt-1 block truncate text-sm text-muted-foreground">{{ consultationHint }}</span>
            </span>
            <svg-icon
              icon="lucide:chevron-right"
              class="text-muted-foreground"
            />
          </button>

          <button
            type="button"
            class="flex min-h-20 w-full items-center gap-4 py-4 text-left transition-colors hover:text-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary active:scale-[0.99]"
            @click="router.push({ name: 'ClientSatisfaction' })"
          >
            <span class="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-muted text-xl text-primary">
              <svg-icon icon="lucide:star" />
            </span>
            <span class="min-w-0 flex-1">
              <span class="block font-medium text-base-text">服务评价</span>
              <span class="mt-1 block text-sm text-muted-foreground">{{ satisfactionHint }}</span>
            </span>
            <svg-icon
              icon="lucide:chevron-right"
              class="text-muted-foreground"
            />
          </button>
        </div>
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
  import ClientStatePanel from '@/components/client-mobile/client-state-panel.vue'
  import ClientStatusBadge from '@/components/client-mobile/client-status-badge.vue'
  import {
    clearClientAuthMode,
    clearClientDraftState
  } from '@/utils/client-session'
  import { formatTaskTime, taskStateCopy, unwrapClientResponse } from './state'

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
  const taskSummary = computed(() => ({
    available: tasks.value.filter((task) => (
        task.accessible && task.executionStatus !== 'SUBMITTED'
      )).length,
    upcoming: tasks.value.filter((task) => (
        task.executionStatus !== 'SUBMITTED' && task.timingStatus === 'NOT_OPEN'
      )).length,
    expired: tasks.value.filter((task) => (
        task.executionStatus !== 'SUBMITTED' && task.timingStatus === 'EXPIRED'
      )).length,
    submitted: tasks.value.filter((task) => task.executionStatus === 'SUBMITTED').length
  }))
  const nextTaskTone = computed(() => {
    if (!nextTask.value) {
      return 'muted'
    }
    return ({
      active: 'primary',
      success: 'success',
      danger: 'danger',
      muted: 'muted'
    })[taskStateCopy(nextTask.value).tone] || 'muted'
  })
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
