<template>
  <section class="flex min-h-[calc(100dvh-4.5rem)] flex-col justify-between px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-12">
    <div>
      <div class="flex items-center gap-3 text-primary">
        <svg-icon
          icon="lucide:circle-check"
          class="text-3xl"
        />
        <span class="text-sm font-medium">内容已经收到</span>
      </div>
      <h1 class="mt-6 max-w-[21rem] text-[2.25rem] font-semibold leading-[1.12] tracking-[-0.045em]">
        已提交，等待处理
      </h1>
      <p class="mt-5 max-w-sm text-base leading-7 text-muted-foreground">
        本页面只表示内容已经收到，不代表后续处理已经完成。
      </p>

      <ClientStatePanel
        v-if="checking"
        class="mt-9"
        title="正在确认提交状态"
        tone="muted"
        icon="lucide:loader-circle"
      />
      <ClientStatePanel
        v-else-if="errorMessage"
        class="mt-9"
        title="暂时无法确认提交状态"
        :description="errorMessage"
        tone="danger"
        icon="lucide:circle-alert"
      />
    </div>

    <div class="mt-12 border-t border-border pt-6">
      <el-button
        class="!h-13 !w-full !rounded-xl"
        :disabled="checking"
        @click="router.push({ name: 'ClientTasks' })"
      >
        返回随访列表
      </el-button>
    </div>
  </section>
</template>

<script setup>
  import { onMounted, ref } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { getClientTask } from '@/api/sleep-care/client-access'
  import { unwrapClientResponse } from './state'
  import ClientStatePanel from '@/components/client-mobile/client-state-panel.vue'

  defineOptions({
    name: 'ClientTaskSuccess'
  })

  const route = useRoute()
  const router = useRouter()
  const taskId = Number(route.params.taskId)
  const checking = ref(true)
  const errorMessage = ref('')

  const verify = async () => {
    try {
      const task = unwrapClientResponse(await getClientTask(taskId))
      if (task.executionStatus !== 'SUBMITTED') {
        await router.replace({ name: 'ClientTask', params: { taskId } })
      }
    } catch (error) {
      errorMessage.value = error.message || '暂时无法确认状态，请返回任务列表查看。'
    } finally {
      checking.value = false
    }
  }

  onMounted(verify)
</script>
