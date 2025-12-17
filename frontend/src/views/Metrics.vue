<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div class="mb-8">
        <h1 class="text-3xl font-bold text-gray-900">Analytics Dashboard</h1>
        <p class="mt-2 text-gray-600">Comprehensive insights into your code reviews</p>
      </div>

      <!-- Time Range Selector -->
      <div class="mb-6 flex items-center gap-4">
        <label class="text-sm font-medium text-gray-700">Time Range:</label>
        <select
          v-model="selectedDays"
          @change="loadAnalytics"
          class="px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        >
          <option :value="7">Last 7 days</option>
          <option :value="30">Last 30 days</option>
          <option :value="90">Last 90 days</option>
          <option :value="180">Last 6 months</option>
        </select>
        <button
          @click="loadAnalytics"
          :disabled="loading"
          class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
        >
          {{ loading ? 'Loading...' : 'Refresh' }}
        </button>
      </div>

      <!-- Rates Section -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <StatCard
          title="Cache Hit Rate"
          :value="formatPercentage(metrics?.rates?.cache_hit_rate || 0)"
          subtitle="Higher is better"
          :color="getCacheHitColor(metrics?.rates?.cache_hit_rate || 0)"
        />
        <StatCard
          title="Success Rate"
          :value="formatPercentage(metrics?.rates?.success_rate || 0)"
          subtitle="Reviews completed successfully"
          :color="getSuccessColor(metrics?.rates?.success_rate || 0)"
        />
        <StatCard
          v-if="metrics?.quality"
          title="Average Quality Score"
          :value="formatPercentage(metrics.quality.average_score)"
          subtitle="Review quality across all reviews"
          :color="getQualityColor(metrics.quality.average_score)"
        />
      </div>

      <!-- Quality Trends Chart -->
      <div v-if="qualityTrends.length > 0" class="mb-6">
        <MultiLineChart
          :title="'Quality Metrics Trend (' + selectedDays + ' days)'"
          :datasets="qualityTrendDatasets"
          :height="350"
        />
      </div>

      <!-- Issue Trends and Severity Trends -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <!-- Issue Trends -->
        <div v-if="issueTrends.length > 0">
          <LineChart
            title="Total Issues Over Time"
            :data="issueTrends"
            :height="300"
            line-color="#ef4444"
          />
        </div>

        <!-- Severity Trends -->
        <div v-if="severityTrends.errors && severityTrends.errors.length > 0">
          <MultiLineChart
            title="Issues by Severity"
            :datasets="severityDatasets"
            :height="300"
          />
        </div>
      </div>

      <!-- Category Breakdown and Repository Comparison -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <!-- Category Breakdown -->
        <div v-if="categoryBreakdown.length > 0">
          <PieChart
            title="Issues by Category"
            :data="categoryPieData"
            :size="350"
          />
        </div>

        <!-- Repository Comparison -->
        <div v-if="repositoryComparison.length > 0">
          <BarChart
            title="Repository Comparison"
            :data="repositoryBarData"
            :height="350"
          />
        </div>
      </div>

      <!-- Quality Metrics Section -->
      <div v-if="metrics?.quality" class="bg-white rounded-lg shadow p-6 mb-6">
        <h2 class="text-xl font-semibold mb-4">Review Quality Metrics</h2>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div class="text-center">
            <div class="text-sm text-gray-600 mb-2">Average Quality Score</div>
            <div class="text-4xl font-bold" :class="getQualityColorClass(metrics.quality.average_score)">
              {{ metrics.quality.average_score.toFixed(1) }}
            </div>
            <div class="text-xs text-gray-500 mt-1">out of 100</div>
          </div>
          <div class="text-center">
            <div class="text-sm text-gray-600 mb-2">Total Reviews Scored</div>
            <div class="text-4xl font-bold text-blue-600">{{ metrics.quality.total_scores }}</div>
            <div class="text-xs text-gray-500 mt-1">reviews with quality data</div>
          </div>
          <div class="text-center">
            <div class="text-sm text-gray-600 mb-2">Quality Coverage</div>
            <div class="text-4xl font-bold text-purple-600">
              {{ formatPercentage((metrics.quality.total_scores / (metrics.reviews.total || 1)) * 100) }}
            </div>
            <div class="text-xs text-gray-500 mt-1">% of reviews with quality metrics</div>
          </div>
        </div>
      </div>

      <!-- Review Stats -->
      <div class="bg-white rounded-lg shadow p-6 mb-6">
        <h2 class="text-xl font-semibold mb-4">Review Statistics</h2>
        <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div>
            <div class="text-sm text-gray-600">Total Reviews</div>
            <div class="text-2xl font-bold">{{ metrics?.reviews?.total || 0 }}</div>
          </div>
          <div>
            <div class="text-sm text-gray-600">Successful</div>
            <div class="text-2xl font-bold text-green-600">{{ metrics?.reviews?.successful || 0 }}</div>
          </div>
          <div>
            <div class="text-sm text-gray-600">Failed</div>
            <div class="text-2xl font-bold text-red-600">{{ metrics?.reviews?.failed || 0 }}</div>
          </div>
          <div>
            <div class="text-sm text-gray-600">Avg Duration</div>
            <div class="text-2xl font-bold">{{ formatDuration(metrics?.reviews?.average_duration_ms || 0) }}</div>
          </div>
        </div>
      </div>

      <!-- Cache Stats -->
      <div class="bg-white rounded-lg shadow p-6 mb-6">
        <h2 class="text-xl font-semibold mb-4">Cache Performance</h2>
        <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div>
            <div class="text-sm text-gray-600">Cache Hits</div>
            <div class="text-2xl font-bold text-green-600">{{ metrics?.reviews?.cache_hits || 0 }}</div>
          </div>
          <div>
            <div class="text-sm text-gray-600">Cache Misses</div>
            <div class="text-2xl font-bold text-orange-600">{{ metrics?.reviews?.cache_misses || 0 }}</div>
          </div>
          <div>
            <div class="text-sm text-gray-600">Incremental</div>
            <div class="text-2xl font-bold text-blue-600">{{ metrics?.reviews?.incremental || 0 }}</div>
          </div>
          <div>
            <div class="text-sm text-gray-600">Full Reviews</div>
            <div class="text-2xl font-bold">{{ metrics?.reviews?.full || 0 }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getMetrics, type MetricsResponse } from '../api/metrics'
import { analyticsAPI, type QualityTrend, type TimeSeriesPoint, type IssueCategoryBreakdown, type RepositoryMetrics, type SeverityTrends } from '../api/analytics'
import StatCard from '../components/StatCard.vue'
import LineChart from '../components/LineChart.vue'
import MultiLineChart from '../components/MultiLineChart.vue'
import BarChart from '../components/BarChart.vue'
import PieChart from '../components/PieChart.vue'
import NavBar from '../components/NavBar.vue'

const metrics = ref<MetricsResponse | null>(null)
const qualityTrends = ref<QualityTrend[]>([])
const issueTrends = ref<TimeSeriesPoint[]>([])
const severityTrends = ref<SeverityTrends>({ errors: [], warnings: [], suggestions: [] })
const categoryBreakdown = ref<IssueCategoryBreakdown[]>([])
const repositoryComparison = ref<RepositoryMetrics[]>([])
const loading = ref(false)
const selectedDays = ref(30)

const qualityTrendDatasets = computed(() => {
  return [
    {
      label: 'Overall Score',
      data: qualityTrends.value.map(t => ({ date: t.date, value: t.overall_score })),
      color: '#3b82f6',
    },
    {
      label: 'Accuracy',
      data: qualityTrends.value.map(t => ({ date: t.date, value: t.accuracy })),
      color: '#10b981',
    },
    {
      label: 'Actionability',
      data: qualityTrends.value.map(t => ({ date: t.date, value: t.actionability })),
      color: '#f59e0b',
    },
    {
      label: 'Coverage',
      data: qualityTrends.value.map(t => ({ date: t.date, value: t.coverage })),
      color: '#8b5cf6',
    },
  ]
})

const severityDatasets = computed(() => {
  return [
    {
      label: 'Errors',
      data: severityTrends.value.errors,
      color: '#ef4444',
    },
    {
      label: 'Warnings',
      data: severityTrends.value.warnings,
      color: '#f59e0b',
    },
    {
      label: 'Suggestions',
      data: severityTrends.value.suggestions,
      color: '#3b82f6',
    },
  ]
})

const categoryPieData = computed(() => {
  return categoryBreakdown.value.map(cat => ({
    label: cat.category.charAt(0).toUpperCase() + cat.category.slice(1).replace('_', ' '),
    value: cat.count,
  }))
})

const repositoryBarData = computed(() => {
  return repositoryComparison.value.map(repo => ({
    label: repo.repository_name || 'Unknown',
    value: repo.total_issues,
    color: repo.average_score >= 80 ? '#10b981' : repo.average_score >= 60 ? '#f59e0b' : '#ef4444',
  }))
})

const loadMetrics = async () => {
  loading.value = true
  try {
    metrics.value = await getMetrics()
  } catch (error) {
    console.error('Failed to load metrics:', error)
  } finally {
    loading.value = false
  }
}

const loadAnalytics = async () => {
  loading.value = true
  try {
    const days = selectedDays.value

    // Load all analytics data in parallel
    const [
      qualityTrendsData,
      issueTrendsData,
      severityTrendsData,
      categoryBreakdownData,
      repositoryComparisonData,
    ] = await Promise.all([
      analyticsAPI.getQualityTrends(days).catch(() => []),
      analyticsAPI.getIssueTrends(days).catch(() => []),
      analyticsAPI.getSeverityTrends(days).catch(() => ({ errors: [], warnings: [], suggestions: [] })),
      analyticsAPI.getCategoryBreakdown(days).catch(() => []),
      analyticsAPI.getRepositoryComparison(days).catch(() => []),
    ])

    qualityTrends.value = qualityTrendsData
    issueTrends.value = issueTrendsData
    severityTrends.value = severityTrendsData
    categoryBreakdown.value = categoryBreakdownData
    repositoryComparison.value = repositoryComparisonData
  } catch (error) {
    console.error('Failed to load analytics:', error)
  } finally {
    loading.value = false
  }
}

const formatPercentage = (value: number): string => {
  return `${value.toFixed(1)}%`
}

const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

const getCacheHitColor = (rate: number): string => {
  if (rate >= 70) return 'text-green-600'
  if (rate >= 50) return 'text-yellow-600'
  return 'text-red-600'
}

const getSuccessColor = (rate: number): string => {
  if (rate >= 95) return 'text-green-600'
  if (rate >= 90) return 'text-yellow-600'
  return 'text-red-600'
}

const getQualityColor = (score: number): string => {
  if (score >= 80) return 'text-green-600'
  if (score >= 60) return 'text-yellow-600'
  return 'text-red-600'
}

const getQualityColorClass = (score: number): string => {
  if (score >= 80) return 'text-green-600'
  if (score >= 60) return 'text-yellow-600'
  return 'text-red-600'
}

onMounted(() => {
  loadMetrics()
  loadAnalytics()
  // Auto-refresh every 60 seconds
  setInterval(() => {
    loadMetrics()
    loadAnalytics()
  }, 60000)
})
</script>

<style scoped>
.metrics-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}
</style>
