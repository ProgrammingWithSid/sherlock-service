<template>
  <div class="quality-chart-container">
    <div v-if="metrics" class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <!-- Overall Score Gauge -->
      <div class="bg-white rounded-lg shadow p-6">
        <h3 class="text-lg font-semibold mb-4">Overall Quality Score</h3>
        <div class="relative w-48 h-48 mx-auto">
          <svg class="transform -rotate-90" width="192" height="192">
            <!-- Background circle -->
            <circle
              cx="96"
              cy="96"
              r="80"
              fill="none"
              stroke="#e5e7eb"
              stroke-width="16"
            />
            <!-- Progress circle -->
            <circle
              cx="96"
              cy="96"
              r="80"
              fill="none"
              :stroke="getScoreColor(props.metrics.overallScore)"
              stroke-width="16"
              stroke-linecap="round"
              :stroke-dasharray="circumference"
              :stroke-dashoffset="getDashOffset(props.metrics.overallScore)"
              class="transition-all duration-500"
            />
          </svg>
          <div class="absolute inset-0 flex items-center justify-center">
              <div class="text-center">
              <div class="text-4xl font-bold" :class="getScoreColorClass(props.metrics.overallScore)">
                {{ props.metrics.overallScore.toFixed(1) }}
              </div>
              <div class="text-sm text-gray-500">out of 100</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Metrics Breakdown -->
      <div class="bg-white rounded-lg shadow p-6">
        <h3 class="text-lg font-semibold mb-4">Quality Breakdown</h3>
        <div class="space-y-4">
          <MetricBar
            label="Accuracy"
            :value="props.metrics.accuracy"
            color="blue"
          />
          <MetricBar
            label="Actionability"
            :value="props.metrics.actionability"
            color="green"
          />
          <MetricBar
            label="Coverage"
            :value="props.metrics.coverage"
            color="purple"
          />
          <MetricBar
            label="Precision"
            :value="props.metrics.precision"
            color="orange"
          />
          <MetricBar
            label="Recall"
            :value="props.metrics.recall"
            color="pink"
          />
          <MetricBar
            label="Confidence"
            :value="props.metrics.confidence"
            color="indigo"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ReviewQualityMetrics } from '@/types'
import MetricBar from './MetricBar.vue'

interface Props {
  metrics?: ReviewQualityMetrics
}

defineProps<Props>()

const circumference = 2 * Math.PI * 80 // radius = 80

const getDashOffset = (score: number): number => {
  const percentage = score / 100
  return circumference * (1 - percentage)
}

const getScoreColor = (score: number): string => {
  if (score >= 80) return '#10b981' // green
  if (score >= 60) return '#f59e0b' // yellow
  return '#ef4444' // red
}

const getScoreColorClass = (score: number): string => {
  if (score >= 80) return 'text-green-600'
  if (score >= 60) return 'text-yellow-600'
  return 'text-red-600'
}
</script>

<style scoped>
.quality-chart-container {
  width: 100%;
}
</style>
