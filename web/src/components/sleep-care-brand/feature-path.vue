<template>
  <ol
    class="grid grid-cols-5"
    aria-label="持续服务路径：计划、任务、关注、协作、复盘"
  >
    <li
      v-for="(stage, index) in stages"
      :key="stage"
      class="min-w-0 text-center"
    >
      <div class="flex h-4 items-center">
        <span
          class="h-px flex-1"
          :class="index === 0 ? 'bg-transparent' : 'bg-border'"
        />
        <span
          class="h-2.5 w-2.5 shrink-0 rounded-full ring-4"
          :class="index === attentionIndex
            ? 'bg-warning ring-warning-50'
            : 'bg-primary ring-primary-50'"
        />
        <span
          class="h-px flex-1"
          :class="index === stages.length - 1 ? 'bg-transparent' : 'bg-border'"
        />
      </div>
      <span class="mt-1 block truncate text-[11px] font-medium text-muted-foreground">
        {{ stage }}
      </span>
    </li>
  </ol>
</template>

<script setup>
  defineOptions({ name: 'SleepCareFeaturePath' })

  const stages = ['计划', '任务', '关注', '协作', '复盘']
  const attentionIndex = 2
</script>
