<template>
  <ClientsIndex
    :key="route.params.id"
    :initial-detail-id="route.params.id"
  />
</template>

<script setup>
  import { useRoute } from 'vue-router'
  import ClientsIndex from './index.vue'

  defineOptions({ name: 'CareClientDetail' })

  const route = useRoute()
</script>
