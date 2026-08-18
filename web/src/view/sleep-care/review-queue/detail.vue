<template>
  <ReviewQueueIndex
    :key="route.params.id"
    :initial-attention-case-id="route.params.id"
  />
</template>

<script setup>
  import { useRoute } from 'vue-router'
  import ReviewQueueIndex from './index.vue'

  defineOptions({ name: 'CareReviewDetail' })

  const route = useRoute()
</script>
