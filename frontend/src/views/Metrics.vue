<template>
  <div class="metrics-container">
    <h1 class="text-2xl font-bold mb-6">Metrics Dashboard</h1>

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

    <!-- Refresh Button -->
    <div class="flex justify-end">
      <button
        @click="loadMetrics"
        :disabled="loading"
        class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
      >
        {{ loading ? 'Loading...' : 'Refresh' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { getMetrics, type MetricsResponse } from '../api/metrics';
import StatCard from '../components/StatCard.vue';

const metrics = ref<MetricsResponse | null>(null);
const loading = ref(false);

const loadMetrics = async () => {
  loading.value = true;
  try {
    metrics.value = await getMetrics();
  } catch (error) {
    console.error('Failed to load metrics:', error);
  } finally {
    loading.value = false;
  }
};

const formatPercentage = (value: number): string => {
  return `${value.toFixed(1)}%`;
};

const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
};

const getCacheHitColor = (rate: number): string => {
  if (rate >= 70) return 'text-green-600';
  if (rate >= 50) return 'text-yellow-600';
  return 'text-red-600';
};

const getSuccessColor = (rate: number): string => {
  if (rate >= 95) return 'text-green-600';
  if (rate >= 90) return 'text-yellow-600';
  return 'text-red-600';
};

const getQualityColor = (score: number): string => {
  if (score >= 80) return 'text-green-600';
  if (score >= 60) return 'text-yellow-600';
  return 'text-red-600';
};

const getQualityColorClass = (score: number): string => {
  if (score >= 80) return 'text-green-600';
  if (score >= 60) return 'text-yellow-600';
  return 'text-red-600';
};

onMounted(() => {
  loadMetrics();
  // Auto-refresh every 30 seconds
  setInterval(loadMetrics, 30000);
});
</script>

<style scoped>
.metrics-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}
</style>
