<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-8">
    <div class="flex items-start justify-between gap-4">
      <div>
        <p class="text-sm text-muted-foreground">服务沟通记录</p>
        <h1 class="mt-1.5 text-[2rem] font-semibold tracking-[-0.04em]">联系服务</h1>
      </div>
      <el-button
        type="primary"
        class="!h-11 !rounded-xl !font-semibold"
        @click="router.push({ name: 'ClientConsultationNew' })"
      >
        发起咨询
      </el-button>
    </div>

    <ClientStatePanel
      class="mt-7"
      title="此处不提供急救服务"
      description="系统可随时接收留言，人工回复时间以服务安排为准。如遇紧急情况，请立即联系当地急救或前往正规医疗机构。"
      tone="warning"
      icon="lucide:triangle-alert"
    />

    <ClientStatePanel
      v-if="loading"
      class="mt-8"
      title="正在读取咨询记录"
      tone="muted"
      icon="lucide:loader-circle"
    />

    <ClientStatePanel
      v-else-if="errorMessage"
      class="mt-8"
      title="暂时无法读取咨询记录"
      :description="errorMessage"
      tone="danger"
      icon="lucide:circle-alert"
    >
      <el-button
        class="!h-11 !rounded-xl"
        @click="loadConsultations"
      >
        重试
      </el-button>
    </ClientStatePanel>

    <ClientStatePanel
      v-else-if="consultations.length === 0"
      class="mt-8"
      title="还没有咨询记录"
      description="如需工作人员协助，可以发起一条在线咨询。"
      tone="muted"
      icon="lucide:message-circle-more"
    >
      <el-button
        type="primary"
        class="!h-11 !rounded-xl"
        @click="router.push({ name: 'ClientConsultationNew' })"
      >
        发起咨询
      </el-button>
    </ClientStatePanel>

    <div
      v-else
      class="mt-8 border-t border-border"
    >
      <button
        v-for="item in consultations"
        :key="item.id"
        type="button"
        class="w-full border-b border-border py-5 text-left transition-colors hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary active:scale-[0.99]"
        @click="openConsultation(item.id)"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <p class="truncate text-base font-semibold">{{ item.subject }}</p>
            <p class="mt-2 text-sm text-muted-foreground">提交于 {{ formatTaskTime(item.openedAt) }}</p>
          </div>
          <ClientStatusBadge
            :label="statusLabel(item.status)"
            :tone="statusTone(item.status)"
          />
        </div>
        <div class="mt-4 flex items-center justify-between gap-3 text-xs text-muted-foreground">
          <span>{{ urgencyLabel(item.urgency) }}</span>
          <span class="inline-flex items-center gap-1" aria-hidden="true">
            查看详情
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
  import { getClientConsultations } from '@/api/sleep-care/client-access'
  import { formatTaskTime, unwrapClientResponse } from './state'
  import ClientStatePanel from '@/components/client-mobile/client-state-panel.vue'
  import ClientStatusBadge from '@/components/client-mobile/client-status-badge.vue'

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
      return 'success'
    }
    if (value === 'CLOSED') {
      return 'muted'
    }
    if (value === 'WAITING_CLIENT') {
      return 'warning'
    }
    return 'primary'
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
