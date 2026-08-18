<template>
  <AttentionCasesIndex
    :key="route.params.id"
    :initial-detail-id="route.params.id"
  />
</template>

<script setup>
  import { useRoute } from 'vue-router'
  import AttentionCasesIndex from './index.vue'

  defineOptions({ name: 'CareAttentionCaseDetail' })

  const route = useRoute()
</script>
