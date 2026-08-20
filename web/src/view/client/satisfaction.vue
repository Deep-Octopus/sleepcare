<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-6">
    <div class="flex items-end justify-between gap-4">
      <div>
        <p class="text-sm text-muted-foreground">服务完成后的反馈</p>
        <h1 class="mt-1 text-[1.8rem] font-semibold tracking-[-0.035em]">服务评价</h1>
      </div>
      <span class="mb-1 text-sm tabular-nums text-primary">{{ requests.length }} 项</span>
    </div>

    <div class="mt-6 rounded-2xl border border-primary/20 bg-primary/6 p-4 text-sm leading-6">
      工作人员查看时使用匿名编号；系统会保留服务关联，仅供授权的质量核查使用。
    </div>

    <div
      v-if="loading"
      class="mt-7 rounded-2xl border border-border bg-muted p-5 text-sm text-muted-foreground"
    >
      正在读取评价邀请…
    </div>

    <div
      v-else-if="errorMessage"
      class="mt-7 rounded-2xl border border-error/30 bg-error/8 p-5"
    >
      <p class="font-medium text-error">暂时无法读取服务评价</p>
      <p class="mt-2 text-sm leading-6 text-error">{{ errorMessage }}</p>
      <el-button class="!mt-4 !h-11 !rounded-xl" @click="loadRequests">
        重试
      </el-button>
    </div>

    <div
      v-else-if="requests.length === 0"
      class="mt-7 rounded-2xl border border-dashed border-border p-7 text-center"
    >
      <p class="text-base font-medium">当前没有评价邀请</p>
      <p class="mt-2 text-sm leading-6 text-muted-foreground">
        服务关闭后，符合条件的评价邀请会显示在这里。
      </p>
    </div>

    <div v-else class="mt-7 space-y-3">
      <button
        v-for="item in requests"
        :key="item.id"
        type="button"
        class="w-full rounded-2xl border border-border bg-container p-4 text-left transition-colors hover:border-primary/45 hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
        @click="openRequest(item.id)"
      >
        <div class="flex items-start justify-between gap-3">
          <div>
            <p class="text-xs font-medium tracking-[0.08em] text-muted-foreground">
              匿名编号 {{ item.publicCode }}
            </p>
            <h2 class="mt-2 text-base font-semibold">本次在线服务</h2>
          </div>
          <span
            class="shrink-0 rounded-full px-2.5 py-1 text-xs font-medium"
            :class="statusTone(item.status)"
          >
            {{ statusLabel(item.status) }}
          </span>
        </div>
        <div class="mt-4 flex items-center justify-between gap-3 text-xs text-muted-foreground">
          <span>{{ timeHint(item) }}</span>
          <span aria-hidden="true">{{ item.status === 'PENDING' ? '去评价' : '查看详情' }} →</span>
        </div>
      </button>
    </div>
  </section>
</template>

<script setup>
  import { onMounted, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { getClientSatisfactionRequests } from '@/api/sleep-care/client-access'
  import { formatTaskTime, unwrapClientResponse } from './state'

  defineOptions({
    name: 'ClientSatisfaction'
  })

  const router = useRouter()
  const loading = ref(true)
  const errorMessage = ref('')
  const requests = ref([])

  const statusLabel = (value) => ({
    PENDING: '待评价',
    SUBMITTED: '已提交',
    EXPIRED: '已结束'
  }[value] || '未说明')

  const statusTone = (value) => ({
    PENDING: 'bg-primary/10 text-primary',
    SUBMITTED: 'bg-success/10 text-success',
    EXPIRED: 'bg-muted text-muted-foreground'
  }[value] || 'bg-muted text-muted-foreground')

  const timeHint = (item) => {
    if (item.status === 'SUBMITTED') {
      return `提交于 ${formatTaskTime(item.submittedAt)}`
    }
    if (item.status === 'EXPIRED') {
      return `结束于 ${formatTaskTime(item.expiresAt)}`
    }
    return `请在 ${formatTaskTime(item.expiresAt)} 前完成`
  }

  const loadRequests = async () => {
    loading.value = true
    errorMessage.value = ''
    try {
      const data = unwrapClientResponse(await getClientSatisfactionRequests({
        page: 1,
        pageSize: 100
      }))
      requests.value = data.list || []
    } catch (error) {
      errorMessage.value = error.message || '请检查网络后重试。'
    } finally {
      loading.value = false
    }
  }

  const openRequest = (id) => {
    router.push({
      name: 'ClientSatisfactionDetail',
      params: { id }
    })
  }

  onMounted(loadRequests)
</script>
