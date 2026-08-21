<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-8">
    <div class="flex items-end justify-between gap-4">
      <div>
        <p class="text-sm text-muted-foreground">服务完成后的反馈</p>
        <h1 class="mt-1.5 text-[2rem] font-semibold tracking-[-0.04em]">服务评价</h1>
      </div>
      <span class="mb-1 text-sm tabular-nums text-muted-foreground">{{ requests.length }} 项</span>
    </div>

    <ClientStatePanel
      class="mt-7"
      title="你的身份信息会被隐去"
      description="反馈仅供服务改进和授权的质量核查使用。"
      tone="primary"
      icon="lucide:shield-check"
    />

    <ClientStatePanel
      v-if="loading"
      class="mt-8"
      title="正在读取评价邀请"
      tone="muted"
      icon="lucide:loader-circle"
    />

    <ClientStatePanel
      v-else-if="errorMessage"
      class="mt-8"
      title="暂时无法读取服务评价"
      :description="errorMessage"
      tone="danger"
      icon="lucide:circle-alert"
    >
      <el-button class="!h-11 !rounded-xl" @click="loadRequests">
        重试
      </el-button>
    </ClientStatePanel>

    <ClientStatePanel
      v-else-if="requests.length === 0"
      class="mt-8"
      title="当前没有评价邀请"
      description="服务关闭后，符合条件的评价邀请会显示在这里。"
      tone="muted"
      icon="lucide:star"
    />

    <div v-else class="mt-8 border-t border-border">
      <button
        v-for="item in requests"
        :key="item.id"
        type="button"
        class="w-full border-b border-border py-5 text-left transition-colors hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary active:scale-[0.99]"
        @click="openRequest(item.id)"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <p class="text-xs font-medium text-muted-foreground">在线服务反馈</p>
            <h2 class="mt-2 text-base font-semibold">本次在线服务</h2>
          </div>
          <ClientStatusBadge
            :label="statusLabel(item.status)"
            :tone="statusTone(item.status)"
          />
        </div>
        <div class="mt-4 flex items-center justify-between gap-3 text-xs text-muted-foreground">
          <span>{{ timeHint(item) }}</span>
          <span class="inline-flex items-center gap-1" aria-hidden="true">
            {{ item.status === 'PENDING' ? '去评价' : '查看详情' }}
            <svg-icon icon="lucide:chevron-right" />
          </span>
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
  import ClientStatePanel from '@/components/client-mobile/client-state-panel.vue'
  import ClientStatusBadge from '@/components/client-mobile/client-status-badge.vue'

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
    PENDING: 'primary',
    SUBMITTED: 'success',
    EXPIRED: 'muted'
  }[value] || 'muted')

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
