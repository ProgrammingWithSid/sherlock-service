import { apiClient } from './client'
import type { Review, ReviewResult } from '@/types'

export interface ListReviewsParams {
  limit?: number
  offset?: number
}

export const reviewsAPI = {
  list: async (params?: ListReviewsParams): Promise<Review[]> => {
    const queryParams = new URLSearchParams()
    if (params?.limit) {
      queryParams.append('limit', params.limit.toString())
    }
    if (params?.offset) {
      queryParams.append('offset', params.offset.toString())
    }
    const url = `/api/v1/reviews${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
    return apiClient.get<Review[]>(url)
  },

  get: async (id: string): Promise<Review> => {
    return apiClient.get<Review>(`/api/v1/reviews/${id}`)
  },

  retry: async (id: string): Promise<{ status: string }> => {
    return apiClient.post<{ status: string }>(`/api/v1/reviews/${id}/retry`)
  },

  cancel: async (id: string): Promise<void> => {
    return apiClient.delete<void>(`/api/v1/reviews/${id}`)
  },

  listByRepo: async (owner: string, repo: string, params?: ListReviewsParams): Promise<Review[]> => {
    const queryParams = new URLSearchParams()
    if (params?.limit) {
      queryParams.append('limit', params.limit.toString())
    }
    if (params?.offset) {
      queryParams.append('offset', params.offset.toString())
    }
    const url = `/api/v1/repos/${owner}/${repo}/reviews${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
    return apiClient.get<Review[]>(url)
  },
}


