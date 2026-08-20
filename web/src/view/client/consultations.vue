<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-6">
    <div class="flex items-start justify-between gap-4">
      <div>
        <p class="text-sm text-muted-foreground">服务沟通记录</p>
        <h1 class="mt-1 text-[1.8rem] font-semibold tracking-[-0.035em]">联系服务</h1>
      </div>
      <el-button
        type="primary"
        class="!h-10 !rounded-xl"
        @click="router.push({ name: 'ClientConsultationNew' })"
      >
        发起咨询
      </el-button>
    </div>

    <div class="mt-6 rounded-2xl border border-warning/30 bg-warning/8 p-4 text-sm leading-6 text-base-text">
      系统可随时接收；人工回复时间以服务安排为准。如遇紧急情况，请立即联系当地急救或前往正规医疗机构，本页面不提供急救服务。
    </div>

    <div
      v-if="loading"
      class="mt-7 rounded-2xl border border-border bg-muted p-5 text-sm text-muted-foreground"
    >
      正在读取咨询记录…
    </div>

    <div
      v-else-if="errorMessage"
      class="mt-7 rounded-2xl border border-error/30 bg-error/8 p-5"
    >
      <p class="font-medium text-error">暂时无法读取咨询记录</p>
      <p class="mt-2 text-sm leading-6 text-error">{{ errorMessage }}</p>
      <el-button
        class="!mt-4 !h-11 !rounded-xl"
        @click="loadConsultations"
      >
        重试
      </el-button>
    </div>

    <div
      v-else-if="consultations.length === 0"
      class="mt-7 rounded-2xl border border-dashed border-border p-7 text-center"
    >
      <p class="text-base font-medium">还没有咨询记录</p>
      <p class="mt-2 text-sm leading-6 text-muted-foreground">如需工作人员协助，可以发起一条在线咨询。</p>
      <el-button
        type="primary"
        class="!mt-5 !h-11 !rounded-xl"
        @click="router.push({ name: 'ClientConsultationNew' })"
      >
        发起咨询
      </el-button>
    </div>

    <div
      v-else
      class="mt-7 space-y-3"
    >
      <button
        v-for="item in consultations"
        :key="item.id"
        type="button"
        class="w-full rounded-2xl border border-border bg-container p-4 text-left transition-colors hover:border-primary/45 hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
        @click="openConsultation(item.id)"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <p class="truncate text-base font-semibold">{{ item.subject }}</p>
            <p class="mt-2 text-sm text-muted-foreground">提交于 {{ formatTaskTime(item.openedAt) }}</p>
          </div>
          <span
            class="shrink-0 rounded-full px-2.5 py-1 text-xs font-medium"
            :class="statusTone(item.status)"
          >
            {{ statusLabel(item.status) }}
          </span>
        </div>
        <div class="mt-3 flex items-center justify-between gap-3 text-xs text-muted-foreground">
          <span>{{ urgencyLabel(item.urgency) }}</span>
          <span aria-hidden="true">查看详情 →</span>
        </div>
      </button>
    </div>
  </section>
</template>

<script setup>
  import { onMounted, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { getClientConsultations } from '@/api/sleep-care/client-access'
  import { formatTaskTime, unwrapClientResponse } from './state'

  defineOptions({
    name: 'ClientConsultations'
  })

  const router = useRouter()
  const loading = ref(true)
  const consultations = ref([])
  const errorMessage = ref('')

  const statusLabels = {
    NEW: '新建',
    WAITING_ASSIGNMENT: '待分配',
    ASSIGNED: '已受理',
    HANDLING: '处理中',
    WAITING_CLIENT: '等待你补充',
    WAITING_COLLABORATION: '协同处理中',
    RESOLVED: '已解决',
    CLOSED: '已关闭'
  }

  const statusLabel = (value) => statusLabels[value] || '处理中'

  const statusTone = (value) => {
    if (value === 'RESOLVED') {
      return 'bg-success/10 text-success'
    }
    if (value === 'CLOSED') {
      return 'bg-muted text-muted-foreground'
    }
    if (value === 'WAITING_CLIENT') {
      return 'bg-warning/12 text-warning'
    }
    return 'bg-primary/10 text-primary'
  }

  const urgencyLabel = (value) => value === 'EXPEDITED' ? '优先联系' : '常规联系'

  const loadConsultations = async () => {
    loading.value = true
    errorMessage.value = ''
    try {
      const data = unwrapClientResponse(await getClientConsultations({
        page: 1,
        pageSize: 100
      }))
      consultations.value = data.list || []
    } catch (error) {
      errorMessage.value = error.message || '请检查网络后重试。'
    } finally {
      loading.value = false
    }
  }

  const openConsultation = (id) => {
    router.push({
      name: 'ClientConsultationDetail',
      params: { id }
    })
  }

  onMounted(loadConsultations)
</script>
