<template>
  <button
    type="button"
    class="client-record-card group relative w-full overflow-hidden rounded-2xl border p-4 text-left shadow-card transition-[transform,border-color,background-color] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
    :class="cardClass"
    :disabled="disabled"
    @click="emit('click')"
  >
    <div class="flex items-start gap-3.5">
      <span
        class="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-primary-100 bg-primary-50 text-xl text-primary-700 shadow-card"
        aria-hidden="true"
      >
        <svg-icon :icon="icon" />
      </span>

      <span class="min-w-0 flex-1">
        <span class="flex items-start justify-between gap-2.5">
          <span class="min-w-0">
            <span
              v-if="category"
              class="block text-xs font-medium text-muted-foreground"
            >
              {{ category }}
            </span>
            <span
              class="block break-words text-base font-semibold leading-6 text-base-text"
              :class="category ? 'mt-1' : ''"
            >
              {{ title }}
            </span>
          </span>

          <ClientStatusBadge
            v-if="statusLabel"
            :label="statusLabel"
            :tone="statusTone"
            :icon="statusIcon"
          />
        </span>

        <span
          v-if="description"
          class="mt-2 block text-sm leading-5 text-muted-foreground"
        >
          {{ description }}
        </span>
      </span>
    </div>

    <span class="mt-4 flex items-center justify-between gap-3 border-t border-border pt-3.5">
      <span class="min-w-0 text-xs leading-5 text-muted-foreground">
        <slot name="meta" />
      </span>
      <span
        class="inline-flex min-h-8 shrink-0 items-center gap-1 rounded-lg px-2.5 text-sm font-medium transition-colors"
        :class="actionClass"
      >
        {{ actionLabel }}
        <svg-icon
          icon="lucide:chevron-right"
          class="transition-transform group-hover:translate-x-0.5"
          aria-hidden="true"
        />
      </span>
    </span>
  </button>
</template>

<script setup>
  import { computed } from 'vue'
  import ClientStatusBadge from './client-status-badge.vue'

  defineOptions({
    name: 'ClientRecordCard'
  })

  const props = defineProps({
    icon: {
      type: String,
      required: true
    },
    category: {
      type: String,
      default: ''
    },
    title: {
      type: String,
      required: true
    },
    description: {
      type: String,
      default: ''
    },
    statusLabel: {
      type: String,
      default: ''
    },
    statusTone: {
      type: String,
      default: 'muted'
    },
    statusIcon: {
      type: String,
      default: ''
    },
    actionLabel: {
      type: String,
      default: '查看详情'
    },
    emphasized: {
      type: Boolean,
      default: false
    },
    disabled: {
      type: Boolean,
      default: false
    }
  })

  const emit = defineEmits(['click'])

  const cardClass = computed(() => {
    if (props.disabled) {
      return 'cursor-default border-border bg-muted text-muted-foreground'
    }
    if (props.emphasized) {
      return 'border-primary-100 bg-primary-50 hover:border-primary-300 active:scale-[0.99]'
    }
    return 'border-border bg-container hover:-translate-y-0.5 hover:border-primary-200 active:scale-[0.99]'
  })

  const actionClass = computed(() => {
    if (props.disabled) {
      return 'bg-muted text-muted-foreground'
    }
    if (props.emphasized) {
      return 'bg-primary text-white'
    }
    return 'bg-muted text-primary group-hover:bg-primary-50'
  })
</script>

<style scoped>
  .client-record-card::after {
    position: absolute;
    top: -3.5rem;
    right: -3.5rem;
    width: 7rem;
    height: 7rem;
    border: 1px solid rgb(var(--primary-100-color));
    border-radius: 999px;
    content: '';
    opacity: 0.55;
    pointer-events: none;
  }
</style>
