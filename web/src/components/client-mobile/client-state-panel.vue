<template>
  <section
    class="rounded-2xl border px-4 py-4 shadow-card"
    :class="toneClass"
    :role="tone === 'danger' ? 'alert' : undefined"
  >
    <div class="flex items-start gap-3">
      <span
        class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-border bg-container text-lg shadow-card"
        aria-hidden="true"
      >
        <svg-icon
          :icon="icon"
          :class="iconClass"
        />
      </span>
      <div class="min-w-0 flex-1">
        <p class="text-sm font-semibold leading-6">{{ title }}</p>
        <p
          v-if="description"
          class="mt-0.5 text-sm leading-6 opacity-80"
        >
          {{ description }}
        </p>
        <div v-if="$slots.default" class="mt-3">
          <slot />
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
  import { computed } from 'vue'

  defineOptions({
    name: 'ClientStatePanel'
  })

  const props = defineProps({
    title: {
      type: String,
      required: true
    },
    description: {
      type: String,
      default: ''
    },
    tone: {
      type: String,
      default: 'muted'
    },
    icon: {
      type: String,
      default: 'lucide:info'
    }
  })

  const toneClass = computed(() => ({
    muted: 'border-border bg-muted text-muted-foreground',
    primary: 'border-primary-100 bg-primary-50 text-primary-700',
    warning: 'border-warning-100 bg-warning-50 text-warning-800',
    danger: 'border-error-100 bg-error-50 text-error-700',
    success: 'border-success-100 bg-success-50 text-success-700'
  })[props.tone] || 'border-border bg-muted text-muted-foreground')

  const iconClass = computed(() => props.icon === 'lucide:loader-circle'
    ? 'animate-spin text-primary'
    : '')
</script>
