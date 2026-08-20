<template>
  <main class="h-full overflow-y-auto bg-layout text-base-text">
    <div class="relative mx-auto min-h-full w-full max-w-[30rem] bg-container shadow-card">
      <header
        v-if="showChrome"
        class="sticky top-0 z-20 border-b border-border bg-container px-5 pb-3 pt-[max(0.75rem,env(safe-area-inset-top))]"
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
            class="inline-flex h-10 w-10 items-center justify-center rounded-full bg-muted text-primary transition-colors hover:bg-primary-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
            aria-label="打开首页"
          >
            <svg-icon
              icon="lucide:user-round"
              class="text-xl"
            />
          </router-link>
        </div>
      </header>

      <div :class="showNavigation ? 'pb-[calc(6.75rem+env(safe-area-inset-bottom))]' : ''">
        <router-view />
      </div>

      <nav
        v-if="showNavigation"
        class="fixed bottom-[max(1.5rem,env(safe-area-inset-bottom))] left-1/2 z-30 w-full max-w-[30rem] -translate-x-1/2 border-y border-border bg-container px-3 pb-2 pt-2 shadow-header"
        aria-label="康养用户端主导航"
      >
        <div class="grid grid-cols-4 gap-1">
          <router-link
            v-for="item in navigationItems"
            :key="item.key"
            :to="{ name: item.routeName }"
            class="flex min-h-12 flex-col items-center justify-center gap-1 rounded-xl px-2 text-xs transition-colors focus-visible:outline-2 focus-visible:outline-primary"
            :class="activeNavigation === item.key ? 'bg-muted font-semibold text-primary' : 'text-muted-foreground hover:bg-muted hover:text-base-text'"
          >
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
