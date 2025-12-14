<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <h1 class="text-2xl font-bold mb-6">Feedback Patterns</h1>

      <div v-if="loading" class="text-center py-12">
        <p class="text-gray-500">Loading feedback patterns...</p>
      </div>

      <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4">
        <p class="text-red-800">{{ error }}</p>
      </div>

      <div v-else>
        <!-- Feedback Distribution -->
        <div class="bg-white rounded-lg shadow p-6 mb-6">
          <h2 class="text-xl font-semibold mb-4">Feedback Distribution</h2>
          <div v-if="patterns?.total_feedback === 0" class="text-center py-8 text-gray-500">
            No feedback collected yet. Start reviewing PRs and provide feedback on comments to see patterns here.
          </div>
          <div v-else class="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div
              v-for="(count, feedback) in patterns?.feedback_distribution"
              :key="feedback"
              class="border rounded-lg p-4"
            >
              <div class="text-sm text-gray-600 capitalize">{{ feedback }}</div>
              <div class="text-2xl font-bold mt-2">{{ count }}</div>
              <div class="text-xs text-gray-500 mt-1">
                {{ ((count / (patterns?.total_feedback || 1)) * 100).toFixed(1) }}%
              </div>
            </div>
          </div>
        </div>

        <!-- Acceptance Rate -->
        <div v-if="patterns?.acceptance_rate !== undefined && patterns?.total_feedback > 0" class="bg-white rounded-lg shadow p-6 mb-6">
          <h2 class="text-xl font-semibold mb-4">Acceptance Rate</h2>
          <div class="flex items-center">
            <div class="flex-1">
              <div class="text-sm text-gray-600 mb-2">Overall Acceptance Rate</div>
              <div class="text-3xl font-bold" :class="getAcceptanceColor(patterns.acceptance_rate)">
                {{ patterns.acceptance_rate.toFixed(1) }}%
              </div>
            </div>
            <div class="w-32 h-32">
              <svg class="transform -rotate-90" viewBox="0 0 100 100">
                <circle
                  cx="50"
                  cy="50"
                  r="45"
                  stroke="#e5e7eb"
                  stroke-width="8"
                  fill="none"
                />
                <circle
                  cx="50"
                  cy="50"
                  r="45"
                  :stroke="getAcceptanceColor(patterns.acceptance_rate)"
                  stroke-width="8"
                  fill="none"
                  stroke-dasharray="283"
                  :stroke-dashoffset="283 - (patterns.acceptance_rate / 100) * 283"
                  stroke-linecap="round"
                />
              </svg>
            </div>
          </div>
        </div>

        <!-- Team Preferences -->
        <div v-if="preferences" class="bg-white rounded-lg shadow p-6">
          <h2 class="text-xl font-semibold mb-4">Learned Preferences</h2>
          <div v-if="preferences.learned_rules && preferences.learned_rules.length > 0">
            <ul class="list-disc list-inside space-y-2">
              <li v-for="(rule, index) in preferences.learned_rules" :key="index" class="text-gray-700">
                {{ rule }}
              </li>
            </ul>
          </div>
          <p v-else class="text-gray-500">
            No learned preferences yet. Feedback will be analyzed to build team preferences.
          </p>
        </div>
      </div>

      <div class="mt-6 flex justify-end">
        <button
          @click="loadData"
          :disabled="loading"
          class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
        >
          {{ loading ? 'Loading...' : 'Refresh' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import NavBar from '@/components/NavBar.vue';
import { feedbackAPI, type FeedbackPatterns, type TeamPreferences } from '@/api/feedback';

const patterns = ref<FeedbackPatterns | null>(null);
const preferences = ref<TeamPreferences | null>(null);
const loading = ref(false);
const error = ref<string | null>(null);

const loadData = async () => {
  loading.value = true;
  error.value = null;
  try {
    [patterns.value, preferences.value] = await Promise.all([
      feedbackAPI.getPatterns(),
      feedbackAPI.getPreferences(),
    ]);
    
    // Ensure patterns has default structure if empty
    if (patterns.value && !patterns.value.feedback_distribution) {
      patterns.value.feedback_distribution = {};
    }
    if (patterns.value && patterns.value.total_feedback === undefined) {
      patterns.value.total_feedback = 0;
    }
  } catch (err: any) {
    console.error('Failed to load feedback data:', err);
    const errorMessage = err.response?.data?.error || err.message || 'Failed to load feedback patterns';
    error.value = errorMessage;
    
    // Set empty defaults on error
    patterns.value = {
      feedback_distribution: {},
      total_feedback: 0,
    };
    preferences.value = {
      patterns: {
        feedback_distribution: {},
        total_feedback: 0,
      },
      learned_rules: [],
    };
  } finally {
    loading.value = false;
  }
};

const getAcceptanceColor = (rate: number): string => {
  if (rate >= 70) return 'text-green-600';
  if (rate >= 50) return 'text-yellow-600';
  return 'text-red-600';
};

onMounted(() => {
  loadData();
});
</script>
