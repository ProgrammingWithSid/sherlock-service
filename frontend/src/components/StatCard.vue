<template>
  <div class="bg-white rounded-lg shadow p-6">
    <div class="flex items-center justify-between">
      <div>
        <p class="text-sm font-medium text-gray-600">{{ title }}</p>
        <p :class="['text-2xl font-bold mt-2', color || 'text-gray-900']">{{ value }}</p>
        <p v-if="subtitle" class="text-xs text-gray-500 mt-1">{{ subtitle }}</p>
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
  subtitle?: string
  color?: string
  change?: number
}>()
</script>
