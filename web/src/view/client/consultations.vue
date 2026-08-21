<template>
  <section class="px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-8">
    <ClientPageHero
      eyebrow="服务沟通记录"
      title="联系服务"
      description="查看留言和每次沟通的处理进度"
      :illustration="serviceRibbons"
    >
      <template #action>
        <el-button
          type="primary"
          class="!min-h-11 !rounded-xl !px-4 !font-semibold"
          @click="router.push({ name: 'ClientConsultationNew' })"
        >
          <svg-icon
            icon="lucide:message-circle-plus"
            class="mr-1"
          />
          发起咨询
        </el-button>
      </template>
    </ClientPageHero>

    <ClientStatePanel
      class="mt-5"
      title="此处不提供急救服务"
      description="系统可随时接收留言，人工回复时间以服务安排为准。如遇紧急情况，请立即联系当地急救或前往正规医疗机构。"
      tone="warning"
      icon="lucide:triangle-alert"
    />

    <ClientStatePanel
      v-if="loading"
      class="mt-6"
      title="正在读取咨询记录"
      tone="muted"
      icon="lucide:loader-circle"
    />

    <ClientStatePanel
      v-else-if="errorMessage"
      class="mt-6"
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
      class="mt-6"
      title="还没有咨询记录"
      description="如需工作人员协助，可以发起一条在线咨询。"
      tone="muted"
      icon="lucide:message-circle-more"
    />

    <div
      v-else
      class="mt-6 space-y-3"
    >
      <ClientRecordCard
        v-for="item in consultations"
        :key="item.id"
        icon="lucide:message-square-text"
        category="在线咨询"
        :title="item.subject"
        :description="`提交于 ${formatTaskTime(item.openedAt)}`"
        :status-label="statusLabel(item.status)"
        :status-tone="statusTone(item.status)"
        :status-icon="statusIcon(item.status)"
        :action-label="item.status === 'WAITING_CLIENT' ? '补充信息' : '查看详情'"
        :emphasized="item.status === 'WAITING_CLIENT'"
        @click="openConsultation(item.id)"
      >
        <template #meta>
          <span class="inline-flex items-center gap-1.5">
            <svg-icon
              icon="lucide:user-round-check"
              aria-hidden="true"
            />
            {{ urgencyLabel(item.urgency) }}
          </span>
        </template>
      </ClientRecordCard>
    </div>
  </section>
</template>

<script setup>
  import { onMounted, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { getClientConsultations } from '@/api/sleep-care/client-access'
  import { formatTaskTime, unwrapClientResponse } from './state'
  import serviceRibbons from '@/assets/client/service-ribbons.webp'
  import ClientPageHero from '@/components/client-mobile/client-page-hero.vue'
  import ClientStatePanel from '@/components/client-mobile/client-state-panel.vue'
  import ClientRecordCard from '@/components/client-mobile/client-record-card.vue'

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

  const statusIcon = (value) => ({
    NEW: 'lucide:message-circle-plus',
    WAITING_ASSIGNMENT: 'lucide:clock-3',
    ASSIGNED: 'lucide:user-round-check',
    HANDLING: 'lucide:messages-square',
    WAITING_CLIENT: 'lucide:message-circle-more',
    WAITING_COLLABORATION: 'lucide:users-round',
    RESOLVED: 'lucide:circle-check',
    CLOSED: 'lucide:circle-minus'
  }[value] || 'lucide:messages-square')

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
