<template>
  <ConsultationsIndex
    :key="route.params.id"
    :initial-detail-id="route.params.id"
  />
</template>

<script setup>
  import { useRoute } from 'vue-router'
  import ConsultationsIndex from './index.vue'

  defineOptions({
    name: 'CareConsultationDetail'
  })

  const route = useRoute()
</script>
