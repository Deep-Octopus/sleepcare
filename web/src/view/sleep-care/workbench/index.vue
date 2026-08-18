<template>
  <main class="space-y-4 text-base-text">
    <section class="rounded-xl border border-border bg-container p-5 shadow-card">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div class="max-w-3xl">
          <div class="mb-3 flex items-center gap-2 text-sm font-medium text-primary">
            <svg-icon icon="lucide:radio-tower" />
            <span>实时责任范围</span>
          </div>
          <h1 class="text-2xl font-semibold tracking-tight">今日工作台</h1>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">
            聚合今日任务、未结事项和待复核工作。所有数字都按当前角色、部门和有效责任关系实时计算。
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <span
            class="text-xs text-muted-foreground"
            aria-live="polite"
          >
            {{ refreshedAt ? `更新于 ${refreshedAt}` : '尚未完成首次刷新' }}
          </span>
          <el-button
            :loading="loading"
            @click="loadWorkbench"
          >
            <svg-icon
              class="mr-1"
              icon="lucide:refresh-cw"
            />
            刷新
          </el-button>
        </div>
      </div>
    </section>

    <section
      v-loading="loading"
      class="grid min-h-44 grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
      aria-label="工作台指标"
    >
      <article
        v-for="card in metricCards"
        :key="card.key"
        class="group rounded-xl border border-border bg-container p-5 shadow-card"
      >
        <div class="flex items-start justify-between gap-4">
          <div>
            <p class="text-sm font-medium text-muted-foreground">{{ card.label }}</p>
            <p class="mt-3 text-3xl font-semibold tabular-nums tracking-tight">
              {{ card.value }}
            </p>
          </div>
          <div class="grid h-10 w-10 place-items-center rounded-lg bg-muted text-primary">
            <svg-icon
              :icon="card.icon"
              class="text-lg"
            />
          </div>
        </div>
        <p class="mt-4 min-h-10 text-sm leading-5 text-muted-foreground">
          {{ card.description }}
        </p>
        <button
          v-if="btnAuth.viewDetail && card.routeName"
          type="button"
          class="mt-4 inline-flex appearance-none items-center gap-1 bg-transparent text-sm font-medium text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-300"
          @click="navigate(card.routeName)"
        >
          查看明细
          <svg-icon icon="lucide:arrow-right" />
        </button>
      </article>
    </section>

    <section class="rounded-xl border border-border bg-container p-5 shadow-card">
      <div class="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h2 class="text-lg font-semibold">执行入口</h2>
          <p class="mt-1 text-sm text-muted-foreground">
            用户资料、任务与事项沿用同一责任范围，页面入口不会扩大后端授权。
          </p>
        </div>
        <div
          v-if="btnAuth.viewDetail"
          class="flex flex-wrap gap-2"
        >
          <el-button @click="navigate('CareClients')">
            <svg-icon
              class="mr-1"
              icon="lucide:users"
            />
            康养用户
          </el-button>
          <el-button @click="navigate('CareTasks')">
            <svg-icon
              class="mr-1"
              icon="lucide:list-checks"
            />
            计划任务
          </el-button>
          <el-button
            type="primary"
            @click="navigate('CareAttentionCases')"
          >
            <svg-icon
              class="mr-1"
              icon="lucide:clipboard-clock"
            />
            关注事项
          </el-button>
        </div>
      </div>
    </section>
  </main>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { getCareWorkbench } from '@/api/sleep-care/case-work'
  import { formatDate } from '@/utils/format'
  import { useBtnAuth } from '@/utils/btnAuth'

  defineOptions({ name: 'CareWorkbench' })

  const router = useRouter()
  const btnAuth = useBtnAuth()
  const loading = ref(false)
  const refreshedAt = ref('')
  const metrics = ref({
    dueToday: 0,
    waitingClient: 0,
    deliveryIssues: 0,
    attentionCases: 0,
    assignedToMe: 0,
    reviewRequired: 0
  })

  const metricCards = computed(() => [
    {
      key: 'dueToday',
      label: '今日应随访',
      value: metrics.value.dueToday,
      description: '今日截止且尚未提交或取消的计划任务。',
      icon: 'lucide:calendar-clock',
      routeName: 'CareTasks'
    },
    {
      key: 'waitingClient',
      label: '等待用户完成',
      value: metrics.value.waitingClient,
      description: '已开放或正在填写的用户侧任务。',
      icon: 'lucide:user-round-clock',
      routeName: 'CareTasks'
    },
    {
      key: 'deliveryIssues',
      label: '通知异常',
      value: metrics.value.deliveryIssues,
      description: '统一待办中仍未完成的通知异常记录。',
      icon: 'lucide:message-square-warning',
      routeName: ''
    },
    {
      key: 'attentionCases',
      label: '问卷关注',
      value: metrics.value.attentionCases,
      description: '已生成但尚未关闭的关注事项。',
      icon: 'lucide:clipboard-list',
      routeName: 'CareAttentionCases'
    },
    {
      key: 'assignedToMe',
      label: '待本人处理',
      value: metrics.value.assignedToMe,
      description: '当前分配给本人且仍开放的统一待办。',
      icon: 'lucide:inbox',
      routeName: 'CareAttentionCases'
    },
    {
      key: 'reviewRequired',
      label: '待专业复核',
      value: metrics.value.reviewRequired,
      description: '已提交并进入待复核状态的任务。',
      icon: 'lucide:scan-search',
      routeName: 'CareTasks'
    }
  ])

  const loadWorkbench = async () => {
    loading.value = true
    try {
      const res = await getCareWorkbench()
      if (res.code === 0) {
        metrics.value = {
          ...metrics.value,
          ...res.data
        }
        refreshedAt.value = formatDate(new Date())
      }
    } finally {
      loading.value = false
    }
  }

  const navigate = (name) => {
    router.push({ name })
  }

  onMounted(loadWorkbench)
</script>
