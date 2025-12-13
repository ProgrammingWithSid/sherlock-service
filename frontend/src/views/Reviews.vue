<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div class="mb-8">
        <h1 class="text-3xl font-bold text-gray-900">Reviews</h1>
        <p class="mt-2 text-gray-600">All code reviews</p>
      </div>

      <div v-if="reviewsStore.loading" class="text-center py-12">
        <p class="text-gray-500">Loading reviews...</p>
      </div>

      <div v-else-if="reviewsStore.error" class="bg-red-50 border border-red-200 rounded-lg p-4">
        <p class="text-red-800">{{ reviewsStore.error }}</p>
      </div>

      <div v-else-if="reviewsStore.items.length === 0" class="text-center py-12">
        <p class="text-gray-500">No reviews found</p>
      </div>

      <div v-else class="bg-white shadow rounded-lg overflow-hidden">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                PR
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Status
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Comments
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Created
              </th>
              <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <tr v-for="review in reviewsStore.items" :key="review.id">
              <td class="px-6 py-4 whitespace-nowrap">
                <router-link
                  :to="`/reviews/${review.id}`"
                  class="text-sm font-medium text-blue-600 hover:text-blue-800"
                >
                  PR #{{ review.pr_number }}
                </router-link>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <span
                  :class="[
                    'px-2 py-1 text-xs font-medium rounded',
                    getStatusClass(review.status)
                  ]"
                >
                  {{ review.status }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                {{ review.comments_posted }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                {{ formatDate(review.created_at) }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <button
                  v-if="review.status === 'failed'"
                  @click="handleRetry(review.id)"
                  class="text-blue-600 hover:text-blue-900"
                >
                  Retry
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import NavBar from '@/components/NavBar.vue'
import { useReviewsStore } from '@/stores/reviews'

const reviewsStore = useReviewsStore()

const formatDate = (dateString: string): string => {
  const date = new Date(dateString)
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString()
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

const handleRetry = async (id: string): Promise<void> => {
  await reviewsStore.retryReview(id)
}

onMounted(() => {
  reviewsStore.fetchReviews(50, 0)
})
</script>


