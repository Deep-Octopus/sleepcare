<template>
  <main class="h-full overflow-y-auto bg-[#f3f6f4] text-[#18312c] dark:bg-slate-950 dark:text-slate-100">
    <div class="mx-auto min-h-full w-full max-w-[32rem] bg-white/92 dark:bg-slate-900">
      <header class="sticky top-0 z-20 border-b border-[#dce8e3] bg-white/95 px-5 py-3 backdrop-blur dark:border-slate-800 dark:bg-slate-900/95">
        <div class="flex items-center justify-between gap-4">
          <SleepCareBrand size="sm" />
          <span class="rounded-full border border-[#bfd4cc] bg-[#edf5f1] px-3 py-1 text-xs text-[#35675c] dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-200">
            安全访问
          </span>
        </div>
      </header>
      <router-view />
    </div>
  </main>
</template>

<script setup>
  import SleepCareBrand from '@/components/sleep-care-brand/index.vue'

  defineOptions({
    name: 'ClientLayout'
  })
</script>
