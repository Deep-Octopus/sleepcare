<template>
  <section class="flex min-h-[calc(100vh-4.5rem)] flex-col justify-between px-5 pb-[max(2rem,env(safe-area-inset-bottom))] pt-14">
    <div>
      <div class="flex h-20 w-20 items-center justify-center rounded-[1.75rem] bg-[#dcece6] text-3xl text-[#1f6c5a] dark:bg-emerald-950 dark:text-emerald-200">
        ✓
      </div>
      <p class="mt-8 text-sm font-medium text-[#47766b] dark:text-emerald-300">任务状态已更新</p>
      <h1 class="mt-2 text-[2rem] font-semibold leading-tight tracking-[-0.035em]">已提交，等待处理</h1>
      <p class="mt-4 max-w-sm text-base leading-7 text-[#60766f] dark:text-slate-300">
        本页面只表示内容已经收到，不代表后续处理已经完成。
      </p>

      <div v-if="checking" class="mt-8 rounded-2xl bg-[#f3f7f5] p-4 text-sm text-[#60766f] dark:bg-slate-800 dark:text-slate-300">
        正在确认提交状态…
      </div>
      <p v-else-if="errorMessage" class="mt-8 rounded-2xl bg-red-50 p-4 text-sm text-red-700 dark:bg-red-950/40 dark:text-red-200">
        {{ errorMessage }}
      </p>
    </div>

    <el-button
      class="!h-12 !w-full !rounded-xl"
      :disabled="checking"
      @click="router.push({ name: 'ClientTasks' })"
    >
      返回任务列表
    </el-button>
  </section>
</template>

<script setup>
  import { onMounted, ref } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { getClientTask } from '@/api/sleep-care/client-access'
  import { unwrapClientResponse } from './state'

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
