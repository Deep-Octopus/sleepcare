<template>
  <section
    class="border-l-2 px-4 py-3.5"
    :class="toneClass"
    :role="tone === 'danger' ? 'alert' : undefined"
  >
    <div class="flex items-start gap-3">
      <svg-icon
        :icon="icon"
        class="mt-0.5 shrink-0 text-lg"
      />
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
    primary: 'border-primary bg-primary-50 text-primary-700',
    warning: 'border-warning bg-warning-50 text-warning-800',
    danger: 'border-error bg-error-50 text-error-700',
    success: 'border-success bg-success-50 text-success-700'
  })[props.tone] || 'border-border bg-muted text-muted-foreground')
</script>
