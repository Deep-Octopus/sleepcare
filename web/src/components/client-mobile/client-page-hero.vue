<template>
  <section class="client-page-hero relative isolate overflow-hidden rounded-2xl border border-primary-100 bg-primary-50 px-5 py-5 shadow-card">
    <span
      class="client-page-hero__orbit absolute -bottom-22 -right-14 h-44 w-44 rounded-full border border-primary-200"
      aria-hidden="true"
    />

    <img
      v-if="illustration"
      :src="illustration"
      alt=""
      width="320"
      height="320"
      class="pointer-events-none absolute -bottom-8 -right-8 h-40 w-40 select-none object-contain"
      aria-hidden="true"
      draggable="false"
    >
    <span
      v-else-if="icon"
      class="pointer-events-none absolute -bottom-5 -right-3 text-[8rem] leading-none text-primary-200"
      aria-hidden="true"
    >
      <svg-icon :icon="icon" />
    </span>

    <div class="relative max-w-[13.75rem]">
      <p class="text-sm font-medium text-primary-700">
        {{ eyebrow }}
      </p>
      <h1 class="mt-1.5 text-[2rem] font-semibold leading-tight tracking-[-0.04em] text-base-text">
        {{ title }}
      </h1>
      <p
        v-if="description"
        class="mt-2 text-sm leading-6 text-muted-foreground"
      >
        {{ description }}
      </p>
      <div
        v-if="$slots.action"
        class="mt-4"
      >
        <slot name="action" />
      </div>
    </div>
  </section>
</template>

<script setup>
  defineOptions({
    name: 'ClientPageHero'
  })

  defineProps({
    eyebrow: {
      type: String,
      required: true
    },
    title: {
      type: String,
      required: true
    },
    description: {
      type: String,
      default: ''
    },
    illustration: {
      type: String,
      default: ''
    },
    icon: {
      type: String,
      default: ''
    }
  })
</script>

<style scoped>
  .client-page-hero {
    box-shadow:
      inset 0 1px 0 rgb(var(--container-bg-color) / 0.78),
      var(--card-box-shadow);
  }

  .client-page-hero__orbit {
    opacity: 0.7;
  }

  :global(.dark) .client-page-hero img {
    filter: brightness(0.82) saturate(0.82);
  }
</style>
