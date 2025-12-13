import { defineStore } from 'pinia'
import { reviewsAPI } from '@/api/reviews'
import type { Review } from '@/types'

interface ReviewsState {
  items: Review[]
  loading: boolean
  error: string | null
  total: number
}

export const useReviewsStore = defineStore('reviews', {
  state: (): ReviewsState => ({
    items: [],
    loading: false,
    error: null,
    total: 0,
  }),

  actions: {
    async fetchReviews(limit = 50, offset = 0): Promise<void> {
      this.loading = true
      this.error = null

      try {
        const reviews = await reviewsAPI.list({ limit, offset })
        this.items = reviews
        this.total = reviews.length
      } catch (error) {
        this.error = error instanceof Error ? error.message : 'Failed to fetch reviews'
      } finally {
        this.loading = false
      }
    },

    async fetchReview(id: string): Promise<Review | null> {
      this.loading = true
      this.error = null

      try {
        const review = await reviewsAPI.get(id)
        const index = this.items.findIndex((r) => r.id === review.id)
        if (index >= 0) {
          this.items[index] = review
        } else {
          this.items.push(review)
        }
        return review
      } catch (error) {
        this.error = error instanceof Error ? error.message : 'Failed to fetch review'
        return null
      } finally {
        this.loading = false
      }
    },

    async retryReview(id: string): Promise<void> {
      this.loading = true
      this.error = null

      try {
        await reviewsAPI.retry(id)
        await this.fetchReview(id)
      } catch (error) {
        this.error = error instanceof Error ? error.message : 'Failed to retry review'
      } finally {
        this.loading = false
      }
    },
  },
})


