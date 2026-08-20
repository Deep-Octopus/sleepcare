<template>
  <section class="flex min-h-[calc(100vh-4.5rem)] flex-col justify-between px-5 pb-[max(1.5rem,env(safe-area-inset-bottom))] pt-14">
    <div>
      <div class="mb-8 flex h-16 w-16 items-center justify-center rounded-[1.35rem] bg-muted text-2xl text-primary">
        <svg-icon icon="lucide:shield-check" />
      </div>
      <p class="text-sm font-medium text-primary">
        一次访问 · 仅限指定任务
      </p>
      <h1 class="mt-3 text-[2rem] font-semibold leading-tight tracking-[-0.035em]">
        {{ heading }}
      </h1>
      <p class="mt-4 max-w-sm text-base leading-7 text-muted-foreground">
        {{ description }}
      </p>
    </div>

    <div class="mt-12">
      <div
        v-if="state === 'checking'"
        class="flex items-center gap-3 rounded-2xl border border-border bg-muted p-4"
      >
        <span class="h-2.5 w-2.5 animate-pulse rounded-full bg-primary motion-reduce:animate-none" />
        <span class="text-sm text-muted-foreground">正在确认访问权限</span>
      </div>
      <div v-else-if="state === 'invalid'" class="space-y-3">
        <el-button
          class="!m-0 !h-12 !w-full !rounded-xl"
          size="large"
          @click="reload"
        >
          重新检查
        </el-button>
        <el-button
          type="primary"
          class="!m-0 !h-12 !w-full !rounded-xl"
          size="large"
          @click="router.replace({ name: 'ClientLogin' })"
        >
          使用账号登录
        </el-button>
      </div>
      <p class="mt-4 text-center text-xs leading-5 text-muted-foreground">
        为保护隐私，本页不会显示多余的个人信息
      </p>
    </div>
  </section>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { redeemClientAccess } from '@/api/sleep-care/client-access'
  import { clearAllClientLocalState, unwrapClientResponse } from './state'
  import {
    CLIENT_AUTH_MODE_GRANT,
    writeClientAuthMode
  } from '@/utils/client-session'

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
      writeClientAuthMode(CLIENT_AUTH_MODE_GRANT)
      await router.replace({ name: 'ClientHome' })
    } catch {
      state.value = 'invalid'
    }
  }

  const reload = () => {
    window.location.reload()
  }

  onMounted(redeem)
</script>
