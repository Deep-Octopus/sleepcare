<template>
  <TasksIndex
    :key="route.params.id"
    :initial-detail-id="route.params.id"
  />
</template>

<script setup>
  import { useRoute } from 'vue-router'
  import TasksIndex from './index.vue'

  defineOptions({ name: 'CareTaskDetail' })

  const route = useRoute()
</script>
