<template>
  <div class="bg-white rounded-lg shadow">
    <div class="px-6 py-4 border-b border-gray-200">
      <h2 class="text-lg font-semibold text-gray-900">Recent Reviews</h2>
    </div>
    <div class="px-6 py-4">
      <div v-if="loading" class="text-center py-8 text-gray-500">
        Loading...
      </div>
      <div v-else-if="reviews.length === 0" class="text-center py-8 text-gray-500">
        No reviews yet
      </div>
      <ul v-else class="space-y-4">
        <li
          v-for="review in reviews"
          :key="review.id"
          class="flex items-center justify-between py-2 border-b border-gray-100 last:border-0"
        >
          <div>
            <router-link
              :to="`/reviews/${review.id}`"
              class="text-sm font-medium text-blue-600 hover:text-blue-800"
            >
              PR #{{ review.pr_number }}
            </router-link>
            <p class="text-xs text-gray-500 mt-1">
              {{ formatDate(review.created_at) }}
            </p>
          </div>
          <span
            :class="[
              'px-2 py-1 text-xs font-medium rounded',
              getStatusClass(review.status)
            ]"
          >
            {{ review.status }}
          </span>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useReviewsStore } from '@/stores/reviews'
import type { Review } from '@/types'
import { onMounted, ref } from 'vue'

const reviewsStore = useReviewsStore()
const reviews = ref<Review[]>([])
const loading = ref(true)

const formatDate = (dateString: string): string => {
  const date = new Date(dateString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  return `${diffDays}d ago`
}

const getStatusClass = (status: string): string => {
  switch (status) {
    case 'completed':
      return 'bg-green-100 text-green-800'
    case 'processing':
      return 'bg-blue-100 text-blue-800'
    case 'failed':
      return 'bg-red-100 text-red-800'
    default:
      return 'bg-gray-100 text-gray-800'
  }
}

onMounted(async () => {
  try {
    await reviewsStore.fetchReviews(5, 0)
    reviews.value = reviewsStore.items.slice(0, 5)
  } catch (err) {
    console.error('Failed to fetch reviews:', err)
  } finally {
    loading.value = false
  }
})
</script>
