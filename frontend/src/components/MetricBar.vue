<template>
  <div class="metric-bar">
    <div class="flex justify-between items-center mb-1">
      <span class="text-sm font-medium text-gray-700">{{ label }}</span>
      <span class="text-sm font-semibold" :class="getValueColorClass()">
        {{ value.toFixed(1) }}%
      </span>
    </div>
    <div class="w-full bg-gray-200 rounded-full h-2.5">
      <div
        class="h-2.5 rounded-full transition-all duration-500"
        :class="getBarColorClass()"
        :style="{ width: `${Math.min(value, 100)}%` }"
      ></div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  label: string
  value: number
  color?: 'blue' | 'green' | 'purple' | 'orange' | 'pink' | 'indigo'
}

const props = withDefaults(defineProps<Props>(), {
  color: 'blue',
})

const getBarColorClass = (): string => {
  const colors = {
    blue: 'bg-blue-500',
    green: 'bg-green-500',
    purple: 'bg-purple-500',
    orange: 'bg-orange-500',
    pink: 'bg-pink-500',
    indigo: 'bg-indigo-500',
  }
  return colors[props.color] || colors.blue
}

const getValueColorClass = (): string => {
  if (props.value >= 80) return 'text-green-600'
  if (props.value >= 60) return 'text-yellow-600'
  return 'text-red-600'
}
</script>

<style scoped>
.metric-bar {
  width: 100%;
}
</style>
