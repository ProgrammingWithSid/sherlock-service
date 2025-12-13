<template>
  <div class="bg-white rounded-lg shadow p-6">
    <div class="flex items-center justify-between">
      <div>
        <p class="text-sm font-medium text-gray-600">{{ title }}</p>
        <p class="text-2xl font-bold text-gray-900 mt-2">{{ value }}</p>
      </div>
      <div v-if="change !== undefined" class="flex items-center">
        <span
          :class="[
            'text-sm font-medium',
            change >= 0 ? 'text-green-600' : 'text-red-600'
          ]"
        >
          {{ change >= 0 ? '▲' : '▼' }} {{ Math.abs(change) }}%
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  value: string
  change?: number
}>()
</script>


