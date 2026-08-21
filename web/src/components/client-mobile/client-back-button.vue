<template>
  <button
    type="button"
    class="inline-flex min-h-11 items-center gap-2 rounded-xl px-1.5 text-sm font-medium text-muted-foreground transition-colors hover:text-base-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary active:scale-[0.98]"
    @click="$emit('click')"
  >
    <svg-icon
      icon="lucide:arrow-left"
      class="text-base"
    />
    <span>{{ label }}</span>
  </button>
</template>

<script setup>
  defineOptions({
    name: 'ClientBackButton'
  })

  defineProps({
    label: {
      type: String,
      required: true
    }
  })

  defineEmits(['click'])
</script>
