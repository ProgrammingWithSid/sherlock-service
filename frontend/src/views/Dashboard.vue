<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div class="mb-8">
        <h1 class="text-3xl font-bold text-gray-900">Dashboard</h1>
        <p class="mt-2 text-gray-600">Overview of your code reviews</p>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <StatCard
          title="Reviews This Month"
          :value="stats.reviews.toString()"
          :change="stats.reviewsChange"
        />
        <StatCard
          title="Issues Found"
          :value="stats.issues.toString()"
          :change="stats.issuesChange"
        />
        <StatCard
          title="Fixed"
          :value="stats.fixed.toString()"
          :change="stats.fixedChange"
        />
        <StatCard
          title="Quality Score"
          :value="`${stats.score}%`"
          :change="stats.scoreChange"
        />
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <RecentReviews />
        <ConnectedRepos />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reviewsAPI } from '@/api/reviews'
import { statsAPI } from '@/api/stats'
import ConnectedRepos from '@/components/ConnectedRepos.vue'
import NavBar from '@/components/NavBar.vue'
import RecentReviews from '@/components/RecentReviews.vue'
import StatCard from '@/components/StatCard.vue'
import { onMounted, ref } from 'vue'

interface Stats {
  reviews: number
  reviewsChange: number
  issues: number
  issuesChange: number
  fixed: number
  fixedChange: number
  score: number
  scoreChange: number
}

const stats = ref<Stats>({
  reviews: 0,
  reviewsChange: 0,
  issues: 0,
  issuesChange: 0,
  fixed: 0,
  fixedChange: 0,
  score: 0,
  scoreChange: 0,
})

const loading = ref(true)
const error = ref<string | null>(null)

onMounted(async () => {
  try {
    const usageStats = await statsAPI.get()
    const reviews = await reviewsAPI.list({ limit: 100 })

    // Calculate stats from reviews
    let totalIssues = 0
    let totalErrors = 0
    let totalFixed = 0

    reviews.forEach((review) => {
      if (review.result) {
        try {
          const result = JSON.parse(review.result)
          totalIssues += result.summary?.total_issues || 0
          totalErrors += result.summary?.errors || 0
          // Fixed would be calculated from resolved reviews
          if (review.status === 'completed' && result.summary?.errors === 0) {
            totalFixed++
          }
        } catch (e) {
          // Ignore parse errors
        }
      }
    })

    stats.value = {
      reviews: usageStats.reviews_this_month,
      reviewsChange: 0, // Would calculate from previous month
      issues: totalIssues,
      issuesChange: 0,
      fixed: totalFixed,
      fixedChange: 0,
      score: totalIssues > 0 ? Math.round((1 - totalErrors / totalIssues) * 100) : 100,
      scoreChange: 0,
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load stats'
  } finally {
    loading.value = false
  }
})
</script>
