<template>
  <div
    class="flex min-w-0 items-center"
    :class="[sizeClasses.gap, layoutClass]"
    role="img"
    aria-label="睡眠康养随访，让持续关怀有路径"
  >
    <img
      :src="sleepCareMark"
      alt=""
      class="shrink-0 object-contain"
      :class="sizeClasses.mark"
    >
    <div
      class="min-w-0"
      :class="layout === 'stacked' ? 'text-center' : ''"
    >
      <p
        class="truncate font-semibold tracking-tight text-base-text"
        :class="sizeClasses.title"
      >
        睡眠康养随访
      </p>
      <p
        v-if="showTagline"
        class="mt-0.5 truncate text-muted-foreground"
        :class="sizeClasses.tagline"
      >
        让持续关怀有路径
      </p>
    </div>
  </div>
</template>

<script setup>
  import { computed } from 'vue'
  import sleepCareMark from '@/assets/brand/sleep-care-mark.svg'

  defineOptions({ name: 'SleepCareBrand' })

  const props = defineProps({
    size: {
      type: String,
      default: 'md',
      validator: (value) => ['sm', 'md', 'lg'].includes(value)
    },
    layout: {
      type: String,
      default: 'horizontal',
      validator: (value) => ['horizontal', 'stacked'].includes(value)
    },
    showTagline: {
      type: Boolean,
      default: true
    }
  })

  const sizeClasses = computed(() => ({
    sm: {
      mark: 'h-9 w-9',
      title: 'text-sm',
      tagline: 'text-[11px]',
      gap: 'gap-2.5'
    },
    md: {
      mark: 'h-12 w-12',
      title: 'text-lg',
      tagline: 'text-xs',
      gap: 'gap-3'
    },
    lg: {
      mark: 'h-16 w-16',
      title: 'text-xl',
      tagline: 'text-sm',
      gap: 'gap-4'
    }
  })[props.size])

  const layoutClass = computed(() => props.layout === 'stacked'
    ? 'flex-col justify-center'
    : '')
</script>
