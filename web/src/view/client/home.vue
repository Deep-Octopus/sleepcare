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
      <section class="client-home-scene relative isolate min-h-52 overflow-hidden rounded-2xl border border-border shadow-card">
        <img
          :src="wellnessHorizon"
          alt=""
          width="960"
          height="640"
          class="pointer-events-none absolute inset-0 h-full w-full select-none object-cover object-center"
          aria-hidden="true"
          draggable="false"
          fetchpriority="high"
        >
        <span
          class="client-home-scene__scrim absolute inset-0"
          aria-hidden="true"
        />
        <div class="relative max-w-[14rem] p-5">
          <p class="text-sm font-medium text-primary-700">{{ todayLabel }}</p>
          <h1 class="mt-2 text-[1.9rem] font-semibold leading-tight tracking-[-0.04em] text-base-text">
            你好，{{ profile.displayName }}
          </h1>
          <p class="mt-3 text-sm leading-6 text-muted-foreground">
            今天的服务安排已经为你整理好了。
          </p>
        </div>
      </section>

      <section
        class="relative mt-5 overflow-hidden rounded-2xl border p-5 shadow-card"
        :class="nextTask ? 'border-primary-100 bg-primary-50' : 'border-border bg-muted'"
      >
        <span
          class="pointer-events-none absolute -bottom-18 -right-12 h-40 w-40 rounded-full border border-primary-200 opacity-60"
          aria-hidden="true"
        />
        <div class="relative flex items-center justify-between gap-3">
          <span class="inline-flex h-10 w-10 items-center justify-center rounded-xl bg-container text-lg text-primary shadow-card">
            <svg-icon icon="lucide:calendar-days" />
          </span>
          <ClientStatusBadge
            v-if="nextTask"
            :label="taskStateCopy(nextTask).label"
            :tone="nextTaskTone"
          />
        </div>

        <template v-if="nextTask">
          <h2 class="relative mt-5 max-w-[20rem] text-[1.65rem] font-semibold leading-[1.2] tracking-[-0.035em]">
            {{ readableTaskTitle(nextTask.title, nextTask.dayCode) }}
          </h2>
          <p class="relative mt-3 text-sm leading-6 text-muted-foreground">
            {{ nextTaskHint }}
          </p>
          <button
            type="button"
            class="client-primary-action relative mt-6 flex min-h-13 w-full items-center justify-between gap-3 rounded-xl bg-primary py-2 pl-5 pr-2 font-semibold text-white transition-transform focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary active:scale-[0.98]"
            @click="openNextTask"
          >
            {{ nextTask.executionStatus === 'SUBMITTED' ? '查看提交结果' : '继续处理' }}
            <span class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-container text-primary shadow-card">
              <svg-icon icon="lucide:arrow-right" />
            </span>
          </button>
        </template>

        <template v-else>
          <h2 class="relative mt-5 text-[1.65rem] font-semibold leading-[1.2] tracking-[-0.035em]">
            当前没有待处理随访
          </h2>
          <p class="relative mt-3 text-sm leading-6 text-muted-foreground">
            新的安排会显示在这里，你也可以查看全部随访记录。
          </p>
          <button
            type="button"
            class="relative mt-6 flex min-h-13 w-full items-center justify-between gap-3 rounded-xl border border-border bg-container py-2 pl-5 pr-2 font-semibold transition-colors hover:border-primary-200 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary active:scale-[0.98]"
            @click="router.push({ name: 'ClientTasks' })"
          >
            查看全部随访
            <span class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-muted text-primary">
              <svg-icon icon="lucide:arrow-right" />
            </span>
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

        <div class="client-summary mt-4 grid grid-cols-[1.1fr_1fr] overflow-hidden rounded-2xl border border-border bg-container shadow-card">
          <div class="relative overflow-hidden border-r border-border bg-primary-50 p-5">
            <svg-icon
              icon="lucide:chart-no-axes-column-increasing"
              class="pointer-events-none absolute -bottom-4 -right-3 text-[5rem] text-primary-200 opacity-60"
              aria-hidden="true"
            />
            <p class="text-sm text-muted-foreground">现在可填写</p>
            <p class="relative mt-2 text-[2.5rem] font-semibold leading-none tracking-[-0.04em] tabular-nums text-primary">
              {{ taskSummary.available }}
            </p>
            <p class="relative mt-3 text-xs leading-5 text-muted-foreground">需要你处理的随访</p>
          </div>
          <dl class="divide-y divide-border px-4">
            <div class="flex items-center justify-between gap-3 py-3.5">
              <dt class="text-sm text-muted-foreground">之后开放</dt>
              <dd class="font-semibold tabular-nums">{{ taskSummary.upcoming }}</dd>
            </div>
            <div class="flex items-center justify-between gap-3 py-3.5">
              <dt class="text-sm text-muted-foreground">已经提交</dt>
              <dd class="font-semibold tabular-nums">{{ taskSummary.submitted }}</dd>
            </div>
            <div class="flex items-center justify-between gap-3 py-3.5">
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

        <div class="mt-4 grid grid-cols-2 gap-3">
          <button
            type="button"
            class="client-service-card group relative flex min-h-52 overflow-hidden rounded-2xl border border-primary-100 bg-primary-50 p-4 text-left shadow-card transition-[transform,border-color] hover:-translate-y-0.5 hover:border-primary-300 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary active:scale-[0.99]"
            @click="router.push({ name: 'ClientConsultations' })"
          >
            <img
              :src="serviceRibbons"
              alt=""
              width="640"
              height="640"
              class="pointer-events-none absolute -right-7 -top-5 h-30 w-30 select-none object-contain opacity-80"
              aria-hidden="true"
              draggable="false"
              loading="lazy"
            >
            <span class="relative flex h-full min-w-0 flex-1 flex-col">
              <span class="inline-flex h-10 w-10 items-center justify-center rounded-xl border border-primary-100 bg-container text-lg text-primary-700 shadow-card">
                <svg-icon icon="lucide:messages-square" />
              </span>
              <span class="mt-auto block font-semibold text-base-text">在线咨询</span>
              <span class="mt-1 block text-sm leading-5 text-muted-foreground">
                {{ consultationHint }}
              </span>
              <span class="mt-3 inline-flex items-center gap-1 text-sm font-medium text-primary">
                查看记录
                <svg-icon
                  icon="lucide:arrow-right"
                  class="transition-transform group-hover:translate-x-0.5"
                />
              </span>
            </span>
          </button>

          <button
            type="button"
            class="client-service-card group relative flex min-h-52 overflow-hidden rounded-2xl border border-border bg-muted p-4 text-left shadow-card transition-[transform,border-color] hover:-translate-y-0.5 hover:border-primary-200 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary active:scale-[0.99]"
            @click="router.push({ name: 'ClientSatisfaction' })"
          >
            <img
              :src="feedbackBloom"
              alt=""
              width="640"
              height="640"
              class="pointer-events-none absolute -right-6 -top-5 h-30 w-30 select-none object-contain opacity-80"
              aria-hidden="true"
              draggable="false"
              loading="lazy"
            >
            <span class="relative flex h-full min-w-0 flex-1 flex-col">
              <span class="inline-flex h-10 w-10 items-center justify-center rounded-xl border border-border bg-container text-lg text-primary-700 shadow-card">
                <svg-icon icon="lucide:star" />
              </span>
              <span class="mt-auto block font-semibold text-base-text">服务评价</span>
              <span class="mt-1 block text-sm leading-5 text-muted-foreground">
                {{ satisfactionHint }}
              </span>
              <span class="mt-3 inline-flex items-center gap-1 text-sm font-medium text-primary">
                查看邀请
                <svg-icon
                  icon="lucide:arrow-right"
                  class="transition-transform group-hover:translate-x-0.5"
                />
              </span>
            </span>
          </button>
        </div>
      </section>
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
    getClientTasks
  } from '@/api/sleep-care/client-access'
  import { readableTaskTitle } from '@/utils/sleep-care-display'
  import ClientStatePanel from '@/components/client-mobile/client-state-panel.vue'
  import ClientStatusBadge from '@/components/client-mobile/client-status-badge.vue'
  import feedbackBloom from '@/assets/client/feedback-bloom.webp'
  import serviceRibbons from '@/assets/client/service-ribbons.webp'
  import wellnessHorizon from '@/assets/client/wellness-horizon.webp'
  import { formatTaskTime, taskStateCopy, unwrapClientResponse } from './state'

  defineOptions({
    name: 'ClientHome'
  })

  const router = useRouter()
  const loading = ref(true)
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

  onMounted(loadHome)
</script>

<style scoped>
  .client-home-scene {
    background-color: rgb(var(--primary-50-color));
    box-shadow:
      inset 0 1px 0 rgb(var(--container-bg-color) / 0.7),
      var(--card-box-shadow);
  }

  .client-home-scene__scrim {
    background: linear-gradient(
      90deg,
      rgb(var(--container-bg-color)) 0%,
      rgb(var(--container-bg-color) / 0.94) 45%,
      rgb(var(--container-bg-color) / 0.34) 72%,
      transparent 100%
    );
  }

  .client-primary-action {
    box-shadow:
      inset 0 1px 0 rgb(var(--primary-300-color)),
      0 10px 24px rgb(var(--primary-950-color) / 0.16);
  }

  .client-summary,
  .client-service-card {
    box-shadow:
      inset 0 1px 0 rgb(var(--container-bg-color) / 0.72),
      var(--card-box-shadow);
  }

  :global(.dark) .client-home-scene > img,
  :global(.dark) .client-service-card > img {
    filter: brightness(0.72) saturate(0.72);
  }
</style>
