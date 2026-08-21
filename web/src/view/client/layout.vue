<template>
  <main class="client-shell h-full overflow-y-auto bg-layout text-base-text">
    <div class="relative mx-auto min-h-full w-full max-w-[30rem] bg-container shadow-sider">
      <header
        v-if="showChrome"
        class="sticky top-0 z-20 border-b border-border bg-container px-5 pb-3.5 pt-[max(0.875rem,env(safe-area-inset-top))]"
      >
        <div class="flex items-center justify-between gap-4">
          <router-link
            :to="{ name: 'ClientHome' }"
            class="min-w-0 rounded-lg focus-visible:outline-2 focus-visible:outline-primary"
            aria-label="返回我的康养服务首页"
          >
            <SleepCareBrand
              size="sm"
              :show-tagline="false"
            />
          </router-link>
          <g-dropdown-menu
            :arrow="false"
            align="end"
            :side-offset="10"
            content-class="!w-[15rem] !rounded-2xl !p-2"
          >
            <button
              type="button"
              class="inline-flex h-11 w-11 items-center justify-center rounded-xl border border-border bg-muted text-sm font-semibold text-primary transition-[transform,border-color,background-color] hover:border-primary-200 hover:bg-primary-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary active:scale-[0.98]"
              :aria-label="profile.displayName ? `打开${profile.displayName}的账号菜单` : '打开账号菜单'"
            >
              {{ profileInitial }}
            </button>

            <template #content>
              <g-dropdown-menu-label class="!px-3 !py-3">
                <span class="block text-xs text-muted-foreground">当前登录</span>
                <span class="mt-1 block truncate text-sm font-semibold text-base-text">
                  {{ profile.displayName || '康养用户' }}
                </span>
              </g-dropdown-menu-label>
              <g-dropdown-menu-separator />
              <g-dropdown-menu-item @select="openHome">
                <svg-icon icon="lucide:house" />
                我的首页
              </g-dropdown-menu-item>
              <g-dropdown-menu-item
                danger
                :disabled="signingOut"
                @select="signOut"
              >
                <svg-icon icon="lucide:log-out" />
                {{ signingOut ? '正在退出' : '退出当前账号' }}
              </g-dropdown-menu-item>
            </template>
          </g-dropdown-menu>
        </div>
      </header>

      <div :class="showNavigation ? 'pb-[calc(8rem+env(safe-area-inset-bottom))]' : ''">
        <router-view />
      </div>

      <nav
        v-if="showNavigation"
        class="fixed bottom-[max(2.75rem,env(safe-area-inset-bottom))] left-1/2 z-30 w-[calc(100%-1rem)] max-w-[29rem] -translate-x-1/2 rounded-2xl border border-border bg-container px-2 py-2 shadow-sider"
        aria-label="康养用户端主导航"
      >
        <div class="grid grid-cols-4 gap-1">
          <router-link
            v-for="item in navigationItems"
            :key="item.key"
            :to="{ name: item.routeName }"
            class="relative flex min-h-13 flex-col items-center justify-center gap-1 rounded-xl px-2 text-xs transition-colors focus-visible:outline-2 focus-visible:outline-primary active:scale-[0.98]"
            :class="activeNavigation === item.key ? 'font-semibold text-primary' : 'text-muted-foreground hover:bg-muted hover:text-base-text'"
          >
            <span
              v-if="activeNavigation === item.key"
              class="absolute top-0 h-0.5 w-5 rounded-full bg-primary"
              aria-hidden="true"
            />
            <svg-icon
              :icon="item.icon"
              class="text-xl"
            />
            <span>{{ item.label }}</span>
          </router-link>
        </div>
      </nav>
    </div>
  </main>
</template>

<script setup>
  import { computed, ref, watch } from 'vue'
  import { ElMessage } from 'element-plus'
  import { useRoute, useRouter } from 'vue-router'
  import { getClientProfile, logoutClient } from '@/api/sleep-care/client-access'
  import SleepCareBrand from '@/components/sleep-care-brand/index.vue'
  import {
    clearClientAuthMode,
    clearClientDraftState
  } from '@/utils/client-session'
  import { unwrapClientResponse } from './state'

  defineOptions({
    name: 'ClientLayout'
  })

  const route = useRoute()
  const router = useRouter()
  const signingOut = ref(false)
  const profile = ref({ displayName: '', displayCode: '' })

  const navigationItems = [
    { key: 'home', label: '首页', routeName: 'ClientHome', icon: 'lucide:house' },
    { key: 'tasks', label: '随访', routeName: 'ClientTasks', icon: 'lucide:clipboard-check' },
    { key: 'consultations', label: '咨询', routeName: 'ClientConsultations', icon: 'lucide:messages-square' },
    { key: 'satisfaction', label: '评价', routeName: 'ClientSatisfaction', icon: 'lucide:star' }
  ]

  const showChrome = computed(() => route.meta.clientChrome !== false)
  const activeNavigation = computed(() => route.meta.clientNav || '')
  const showNavigation = computed(() => Boolean(activeNavigation.value))
  const profileInitial = computed(() => profile.value.displayName?.slice(0, 1) || '我')

  const loadProfile = async () => {
    if (!showChrome.value) {
      return
    }
    try {
      profile.value = unwrapClientResponse(await getClientProfile())
    } catch {
      profile.value = { displayName: '', displayCode: '' }
    }
  }

  const openHome = () => {
    router.push({ name: 'ClientHome' })
  }

  const signOut = async () => {
    signingOut.value = true
    try {
      unwrapClientResponse(await logoutClient())
      clearClientDraftState()
      clearClientAuthMode()
      profile.value = { displayName: '', displayCode: '' }
      await router.replace({ name: 'ClientLogin' })
    } catch (error) {
      ElMessage.error(error.message || '暂时无法退出，请稍后重试。')
    } finally {
      signingOut.value = false
    }
  }

  watch(showChrome, (visible) => {
    if (visible) {
      loadProfile()
      return
    }
    profile.value = { displayName: '', displayCode: '' }
  }, { immediate: true })
</script>

<style scoped>
  .client-shell {
    --primary-color: 35 113 95;
    --primary-50-color: 239 248 245;
    --primary-100-color: 217 238 231;
    --primary-200-color: 177 219 205;
    --primary-300-color: 126 192 172;
    --primary-400-color: 76 158 135;
    --primary-500-color: 35 113 95;
    --primary-600-color: 29 96 81;
    --primary-700-color: 26 77 66;
    --primary-800-color: 24 63 55;
    --primary-900-color: 21 53 47;
    --primary-950-color: 10 31 27;
    --el-color-primary: rgb(var(--primary-color));
    --el-color-primary-light-3: rgb(var(--primary-300-color));
    --el-color-primary-light-5: rgb(var(--primary-200-color));
    --el-color-primary-light-7: rgb(var(--primary-100-color));
    --el-color-primary-light-8: rgb(var(--primary-100-color));
    --el-color-primary-light-9: rgb(var(--primary-50-color));
    --el-color-primary-dark-2: rgb(var(--primary-700-color));
  }

  :global(.dark) .client-shell {
    --primary-color: 55 156 128;
    --primary-50-color: 16 48 43;
    --primary-100-color: 19 63 55;
    --primary-200-color: 24 82 70;
    --primary-300-color: 30 105 88;
    --primary-400-color: 39 130 106;
    --primary-500-color: 55 156 128;
    --primary-600-color: 85 181 152;
    --primary-700-color: 127 204 179;
    --primary-800-color: 178 226 209;
    --primary-900-color: 216 242 232;
    --primary-950-color: 238 250 245;
  }
</style>
