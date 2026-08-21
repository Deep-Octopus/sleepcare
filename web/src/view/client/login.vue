<template>
  <section class="flex min-h-[100dvh] flex-col px-6 pb-[max(4.5rem,env(safe-area-inset-bottom))] pt-[max(2rem,env(safe-area-inset-top))]">
    <div class="flex-1">
      <SleepCareBrand
        size="md"
        :show-tagline="false"
      />

      <section class="client-login-scene relative isolate mt-8 min-h-48 overflow-hidden rounded-2xl border border-border shadow-card">
        <img
          :src="wellnessHorizon"
          alt=""
          width="960"
          height="640"
          class="pointer-events-none absolute inset-0 h-full w-full select-none object-cover object-center"
          aria-hidden="true"
          draggable="false"
          fetchpriority="high"
        >
        <span
          class="client-login-scene__scrim absolute inset-0"
          aria-hidden="true"
        />
        <div class="relative max-w-[15rem] p-5">
          <div class="flex items-center gap-2 text-sm font-medium text-primary-700">
            <svg-icon icon="lucide:shield-check" />
            <span>私密服务入口</span>
          </div>
          <h1 class="mt-3 text-[2rem] font-semibold leading-[1.12] tracking-[-0.045em]">
            欢迎回来
          </h1>
          <p class="mt-3 text-sm leading-6 text-muted-foreground">
            登录后查看随访安排、服务消息和评价邀请。
          </p>
        </div>
      </section>

      <ClientStatePanel
        v-if="route.query.state === 'session'"
        class="mt-7"
        title="登录状态已失效"
        description="请重新输入账号和密码。"
        tone="warning"
        icon="lucide:clock-3"
      />

      <form
        class="mt-7 space-y-6"
        @submit.prevent="submitLogin"
      >
        <label class="block">
          <span class="mb-2.5 block text-sm font-medium">账号</span>
          <el-input
            v-model="form.username"
            class="client-login-field"
            size="large"
            autocomplete="username"
            placeholder="请输入账号"
            :disabled="submitting"
            @input="errorMessage = ''"
          />
        </label>

        <label class="block">
          <span class="mb-2.5 block text-sm font-medium">密码</span>
          <el-input
            v-model="form.password"
            class="client-login-field"
            size="large"
            type="password"
            autocomplete="current-password"
            placeholder="请输入密码"
            show-password
            :disabled="submitting"
            @input="errorMessage = ''"
          />
        </label>

        <ClientStatePanel
          v-if="errorMessage"
          title="暂时无法登录"
          :description="errorMessage"
          tone="danger"
          icon="lucide:circle-alert"
        />

        <el-button
          native-type="submit"
          type="primary"
          size="large"
          class="client-login-submit !h-14 !w-full !rounded-xl !px-3 !text-base !font-semibold"
          :loading="submitting"
          :disabled="!canSubmit"
        >
          <span class="flex w-full items-center justify-between pl-2">
            登录
            <span class="inline-flex h-9 w-9 items-center justify-center rounded-lg bg-container text-primary shadow-card">
              <svg-icon icon="lucide:arrow-right" />
            </span>
          </span>
        </el-button>
      </form>
    </div>

    <div class="mt-12 border-t border-border pt-5 text-center text-xs leading-5 text-muted-foreground">
      <p>账号由服务团队提供</p>
      <p class="mt-1">无法登录时，请联系工作人员协助处理</p>
    </div>
  </section>
</template>

<script setup>
  import { computed, reactive, ref } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { loginClient } from '@/api/sleep-care/client-access'
  import SleepCareBrand from '@/components/sleep-care-brand/index.vue'
  import ClientStatePanel from '@/components/client-mobile/client-state-panel.vue'
  import wellnessHorizon from '@/assets/client/wellness-horizon.webp'
  import {
    CLIENT_AUTH_MODE_ACCOUNT,
    clearClientDraftState,
    writeClientAuthMode
  } from '@/utils/client-session'
  import { unwrapClientResponse } from './state'

  defineOptions({
    name: 'ClientLogin'
  })

  const route = useRoute()
  const router = useRouter()
  const submitting = ref(false)
  const errorMessage = ref('')
  const form = reactive({
    username: '',
    password: ''
  })

  const canSubmit = computed(() => (
    form.username.trim().length >= 3 &&
    form.password.length > 0 &&
    !submitting.value
  ))

  const submitLogin = async () => {
    if (!canSubmit.value) {
      return
    }
    submitting.value = true
    errorMessage.value = ''
    try {
      unwrapClientResponse(await loginClient({
        username: form.username.trim(),
        password: form.password
      }))
      clearClientDraftState()
      writeClientAuthMode(CLIENT_AUTH_MODE_ACCOUNT)
      await router.replace({ name: 'ClientHome' })
    } catch (error) {
      errorMessage.value = error.response?.data?.msg || error.message || '暂时无法登录，请稍后重试。'
    } finally {
      form.password = ''
      submitting.value = false
    }
  }
</script>

<style scoped>
  .client-login-scene {
    background-color: rgb(var(--primary-50-color));
    box-shadow:
      inset 0 1px 0 rgb(var(--container-bg-color) / 0.72),
      var(--card-box-shadow);
  }

  .client-login-scene__scrim {
    background: linear-gradient(
      90deg,
      rgb(var(--container-bg-color)) 0%,
      rgb(var(--container-bg-color) / 0.94) 48%,
      rgb(var(--container-bg-color) / 0.28) 76%,
      transparent 100%
    );
  }

  :global(.dark) .client-login-scene > img {
    filter: brightness(0.72) saturate(0.72);
  }

  :deep(.client-login-submit) {
    box-shadow:
      inset 0 1px 0 rgb(var(--primary-300-color)),
      0 10px 24px rgb(var(--primary-950-color) / 0.16);
  }

  :deep(.client-login-field .el-input__wrapper) {
    min-height: 3.5rem;
    border-radius: 0.75rem;
    padding-inline: 1rem;
  }

  :deep(.client-login-field .el-input__inner) {
    font-size: 1rem;
  }
</style>
