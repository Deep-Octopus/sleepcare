<template>
  <section class="flex min-h-[100dvh] flex-col px-6 pb-[max(4rem,env(safe-area-inset-bottom))] pt-[max(2rem,env(safe-area-inset-top))]">
    <div class="flex-1">
      <SleepCareBrand
        size="md"
        :show-tagline="false"
      />

      <div class="mt-14">
        <p class="text-sm font-medium text-primary">我的康养服务</p>
        <h1 class="mt-2 text-3xl font-semibold tracking-tight">欢迎回来</h1>
        <p class="mt-3 max-w-sm text-base leading-7 text-muted-foreground">
          登录后查看随访安排、服务消息和评价邀请。
        </p>
      </div>

      <div
        v-if="route.query.state === 'session'"
        class="mt-6 flex items-start gap-3 rounded-2xl border border-warning-200 bg-warning-50 p-4 text-sm leading-6 text-warning-800"
      >
        <svg-icon
          icon="lucide:clock-3"
          class="mt-0.5 shrink-0 text-lg"
        />
        <span>登录状态已失效，请重新登录。</span>
      </div>

      <form
        class="mt-8 space-y-5"
        @submit.prevent="submitLogin"
      >
        <label class="block">
          <span class="mb-2 block text-sm font-medium">账号</span>
          <el-input
            v-model="form.username"
            class="!h-12"
            size="large"
            autocomplete="username"
            placeholder="请输入账号"
            :disabled="submitting"
            @input="errorMessage = ''"
          />
        </label>

        <label class="block">
          <span class="mb-2 block text-sm font-medium">密码</span>
          <el-input
            v-model="form.password"
            class="!h-12"
            size="large"
            type="password"
            autocomplete="current-password"
            placeholder="请输入密码"
            show-password
            :disabled="submitting"
            @input="errorMessage = ''"
          />
        </label>

        <p
          v-if="errorMessage"
          class="rounded-xl bg-error-50 px-4 py-3 text-sm leading-6 text-error-700"
          role="alert"
        >
          {{ errorMessage }}
        </p>

        <el-button
          native-type="submit"
          type="primary"
          size="large"
          class="!h-13 !w-full !rounded-xl !text-base"
          :loading="submitting"
          :disabled="!canSubmit"
        >
          登录
        </el-button>
      </form>
    </div>

    <p class="mt-10 text-center text-xs leading-5 text-muted-foreground">
      账号由服务团队提供。无法登录时，请联系工作人员协助处理。
    </p>
  </section>
</template>

<script setup>
  import { computed, reactive, ref } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { loginClient } from '@/api/sleep-care/client-access'
  import SleepCareBrand from '@/components/sleep-care-brand/index.vue'
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
