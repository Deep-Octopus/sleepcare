<template>
  <section class="flex min-h-[100dvh] flex-col justify-between px-6 pb-[max(4.5rem,env(safe-area-inset-bottom))] pt-[max(2rem,env(safe-area-inset-top))]">
    <div>
      <SleepCareBrand
        size="md"
        :show-tagline="false"
      />

      <div class="mt-12 border-t border-border pt-10">
        <div class="flex items-center gap-2 text-sm font-medium text-primary">
          <svg-icon icon="lucide:shield-check" />
          <span>一次访问，仅限指定任务</span>
        </div>
        <h1 class="mt-4 max-w-[20rem] text-[2.25rem] font-semibold leading-[1.12] tracking-[-0.045em]">
          {{ heading }}
        </h1>
        <p class="mt-4 max-w-sm text-base leading-7 text-muted-foreground">
          {{ description }}
        </p>
      </div>
    </div>

    <div class="mt-12">
      <ClientStatePanel
        v-if="state === 'checking'"
        title="正在确认访问权限"
        description="验证完成后会自动进入可访问的任务。"
        tone="primary"
        icon="lucide:loader-circle"
      />
      <div v-else-if="state === 'invalid'" class="grid grid-cols-1 gap-3">
        <el-button
          class="!m-0 !h-13 !w-full !rounded-xl"
          size="large"
          @click="reload"
        >
          重新检查
        </el-button>
        <el-button
          type="primary"
          class="!m-0 !h-13 !w-full !rounded-xl"
          size="large"
          @click="router.replace({ name: 'ClientLogin' })"
        >
          使用账号登录
        </el-button>
      </div>
      <p class="mt-6 border-t border-border pt-5 text-center text-xs leading-5 text-muted-foreground">
        为保护隐私，本页不会显示多余的个人信息
      </p>
    </div>
  </section>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { redeemClientAccess } from '@/api/sleep-care/client-access'
  import SleepCareBrand from '@/components/sleep-care-brand/index.vue'
  import ClientStatePanel from '@/components/client-mobile/client-state-panel.vue'
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
