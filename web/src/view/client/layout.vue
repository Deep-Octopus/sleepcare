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
          <router-link
            :to="{ name: 'ClientHome' }"
            class="inline-flex h-10 w-10 items-center justify-center rounded-xl text-muted-foreground transition-colors hover:bg-muted hover:text-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary active:scale-[0.98]"
            aria-label="打开首页"
          >
            <svg-icon
              icon="lucide:house"
              class="text-lg"
            />
          </router-link>
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
  import { computed } from 'vue'
  import { useRoute } from 'vue-router'
  import SleepCareBrand from '@/components/sleep-care-brand/index.vue'

  defineOptions({
    name: 'ClientLayout'
  })

  const route = useRoute()

  const navigationItems = [
    { key: 'home', label: '首页', routeName: 'ClientHome', icon: 'lucide:house' },
    { key: 'tasks', label: '随访', routeName: 'ClientTasks', icon: 'lucide:clipboard-check' },
    { key: 'consultations', label: '咨询', routeName: 'ClientConsultations', icon: 'lucide:messages-square' },
    { key: 'satisfaction', label: '评价', routeName: 'ClientSatisfaction', icon: 'lucide:star' }
  ]

  const showChrome = computed(() => route.meta.clientChrome !== false)
  const activeNavigation = computed(() => route.meta.clientNav || '')
  const showNavigation = computed(() => Boolean(activeNavigation.value))
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
