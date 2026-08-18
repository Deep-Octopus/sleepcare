<template>
  <section class="flex min-h-[calc(100vh-4.5rem)] flex-col justify-between px-5 pb-[max(1.5rem,env(safe-area-inset-bottom))] pt-14">
    <div>
      <div class="mb-8 flex h-16 w-16 items-center justify-center rounded-[1.35rem] bg-[#dcece6] text-2xl font-semibold text-[#1f6c5a] dark:bg-emerald-950 dark:text-emerald-200">
        ✓
      </div>
      <p class="text-sm font-medium text-[#47766b] dark:text-emerald-300">
        一次访问 · 仅限指定任务
      </p>
      <h1 class="mt-3 text-[2rem] font-semibold leading-tight tracking-[-0.035em]">
        {{ heading }}
      </h1>
      <p class="mt-4 max-w-sm text-base leading-7 text-[#5e746e] dark:text-slate-300">
        {{ description }}
      </p>
    </div>

    <div class="mt-12">
      <div
        v-if="state === 'checking'"
        class="flex items-center gap-3 rounded-2xl border border-[#dce8e3] bg-[#f7faf8] p-4 dark:border-slate-800 dark:bg-slate-900"
      >
        <span class="h-2.5 w-2.5 animate-pulse rounded-full bg-[#2c806c] motion-reduce:animate-none" />
        <span class="text-sm text-[#49645d] dark:text-slate-300">正在验证访问范围</span>
      </div>
      <el-button
        v-else-if="state === 'invalid'"
        class="!h-12 !w-full !rounded-xl"
        size="large"
        @click="reload"
      >
        重新检查
      </el-button>
      <p class="mt-4 text-center text-xs leading-5 text-[#7b8e89] dark:text-slate-500">
        页面不会显示姓名、联系方式或内部任务信息
      </p>
    </div>
  </section>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { redeemClientAccess } from '@/api/sleep-care/client-access'
  import { clearAllClientLocalState, unwrapClientResponse } from './state'

  defineOptions({
    name: 'ClientAccess'
  })

  const route = useRoute()
  const router = useRouter()
  const state = ref('checking')

  const heading = computed(() => state.value === 'invalid' ? '这个访问链接已失效' : '正在打开你的任务')
  const description = computed(() => state.value === 'invalid'
    ? '请返回原来的通知入口，使用最新的访问链接。'
    : '验证完成后，只会展示本次允许访问的任务。')

  const clearGrantFromAddress = async () => {
    await router.replace({ name: 'ClientAccess' })
    const cleanUrl = `${window.location.pathname}${window.location.search}#/client/access`
    window.history.replaceState(window.history.state, '', cleanUrl)
  }

  const redeem = async () => {
    const grant = typeof route.query.grant === 'string' ? route.query.grant : ''
    if (!grant) {
      state.value = 'invalid'
      return
    }
    await clearGrantFromAddress()
    clearAllClientLocalState()
    try {
      unwrapClientResponse(await redeemClientAccess(grant))
      await router.replace({ name: 'ClientTasks' })
    } catch {
      state.value = 'invalid'
    }
  }

  const reload = () => {
    window.location.reload()
  }

  onMounted(redeem)
</script>
