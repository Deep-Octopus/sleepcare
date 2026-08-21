<template>
  <span
    class="inline-flex min-h-7 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-xs font-medium"
    :class="toneClass"
  >
    <svg-icon
      :icon="resolvedIcon"
      class="text-sm"
      aria-hidden="true"
    />
    {{ label }}
  </span>
</template>

<script setup>
  import { computed } from 'vue'

  defineOptions({
    name: 'ClientStatusBadge'
  })

  const props = defineProps({
    label: {
      type: String,
      required: true
    },
    tone: {
      type: String,
      default: 'primary'
    },
    icon: {
      type: String,
      default: ''
    }
  })

  const resolvedIcon = computed(() => props.icon || ({
    primary: 'lucide:circle-play',
    success: 'lucide:circle-check',
    warning: 'lucide:clock-3',
    danger: 'lucide:circle-alert',
    muted: 'lucide:circle-minus'
  })[props.tone] || 'lucide:circle-minus')

  const toneClass = computed(() => ({
    primary: 'bg-primary-50 text-primary-700',
    success: 'bg-success-50 text-success-700',
    warning: 'bg-warning-50 text-warning-700',
    danger: 'bg-error-50 text-error-700',
    muted: 'bg-muted text-muted-foreground'
  })[props.tone] || 'bg-muted text-muted-foreground')
</script>
