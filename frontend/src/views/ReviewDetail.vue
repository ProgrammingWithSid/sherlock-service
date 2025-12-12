<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div v-if="loading" class="text-center py-12">
        <p class="text-gray-500">Loading review...</p>
      </div>

      <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4">
        <p class="text-red-800">{{ error }}</p>
      </div>

      <div v-else-if="review">
        <div class="mb-8">
          <h1 class="text-3xl font-bold text-gray-900">Review #{{ review.pr_number }}</h1>
          <p class="mt-2 text-gray-600">Review details and comments</p>
        </div>

        <div class="bg-white shadow rounded-lg p-6 mb-6">
          <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div>
              <p class="text-sm text-gray-600">Status</p>
              <p
                :class="[
                  'mt-1 text-lg font-semibold',
                  getStatusClass(review.status)
                ]"
              >
                {{ review.status }}
              </p>
            </div>
            <div>
              <p class="text-sm text-gray-600">Comments</p>
              <p class="mt-1 text-lg font-semibold text-gray-900">
                {{ review.comments_posted }}
              </p>
            </div>
            <div>
              <p class="text-sm text-gray-600">Duration</p>
              <p class="mt-1 text-lg font-semibold text-gray-900">
                {{ review.duration_ms ? `${review.duration_ms}ms` : 'N/A' }}
              </p>
            </div>
            <div>
              <p class="text-sm text-gray-600">AI Provider</p>
              <p class="mt-1 text-lg font-semibold text-gray-900">
                {{ review.ai_provider }}
              </p>
            </div>
          </div>
        </div>

        <div v-if="reviewResult" class="bg-white shadow rounded-lg p-6">
          <h2 class="text-xl font-semibold text-gray-900 mb-4">Review Summary</h2>
          <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
            <div>
              <p class="text-sm text-gray-600">Total Issues</p>
              <p class="mt-1 text-2xl font-bold text-gray-900">
                {{ reviewResult.summary.total_issues }}
              </p>
            </div>
            <div>
              <p class="text-sm text-gray-600">Errors</p>
              <p class="mt-1 text-2xl font-bold text-red-600">
                {{ reviewResult.summary.errors }}
              </p>
            </div>
            <div>
              <p class="text-sm text-gray-600">Warnings</p>
              <p class="mt-1 text-2xl font-bold text-yellow-600">
                {{ reviewResult.summary.warnings }}
              </p>
            </div>
            <div>
              <p class="text-sm text-gray-600">Suggestions</p>
              <p class="mt-1 text-2xl font-bold text-blue-600">
                {{ reviewResult.summary.suggestions }}
              </p>
            </div>
          </div>

          <div v-if="reviewResult.comments.length > 0" class="mt-6">
            <h3 class="text-lg font-semibold text-gray-900 mb-4">Comments</h3>
            <div class="space-y-4">
              <div
                v-for="(comment, index) in reviewResult.comments"
                :key="index"
                class="border-l-4 border-gray-300 pl-4 py-2"
              >
                <div class="flex items-center mb-2">
                  <span
                    :class="[
                      'px-2 py-1 text-xs font-medium rounded mr-2',
                      getSeverityClass(comment.severity)
                    ]"
                  >
                    {{ comment.severity }}
                  </span>
                  <span class="text-sm font-medium text-gray-900">
                    {{ comment.file }}:{{ comment.line }}
                  </span>
                </div>
                <p class="text-sm text-gray-700">{{ comment.message }}</p>
                <p v-if="comment.fix" class="text-sm text-green-700 mt-2">
                  Fix: {{ comment.fix }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import NavBar from '@/components/NavBar.vue'
import { useReviewsStore } from '@/stores/reviews'
import type { Review, ReviewResult } from '@/types'

const route = useRoute()
const reviewsStore = useReviewsStore()

const review = ref<Review | null>(null)
const reviewResult = ref<ReviewResult | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

const getStatusClass = (status: string): string => {
  switch (status) {
    case 'completed':
      return 'text-green-600'
    case 'processing':
      return 'text-blue-600'
    case 'failed':
      return 'text-red-600'
    default:
      return 'text-gray-600'
  }
}

const getSeverityClass = (severity: string): string => {
  switch (severity) {
    case 'error':
      return 'bg-red-100 text-red-800'
    case 'warning':
      return 'bg-yellow-100 text-yellow-800'
    default:
      return 'bg-blue-100 text-blue-800'
  }
}

onMounted(async () => {
  const id = route.params.id as string
  const fetchedReview = await reviewsStore.fetchReview(id)

  if (!fetchedReview) {
    error.value = 'Review not found'
    loading.value = false
    return
  }

  review.value = fetchedReview

  if (fetchedReview.result) {
    try {
      reviewResult.value = JSON.parse(fetchedReview.result) as ReviewResult
    } catch (e) {
      console.error('Failed to parse review result', e)
    }
  }

  loading.value = false
})
</script>

