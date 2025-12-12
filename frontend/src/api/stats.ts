import { apiClient } from './client'

export interface UsageStats {
  reviews_this_month: number
  commands_this_month: number
  total_reviews: number
  total_commands: number
  total_api_calls: number
  plan: string
  period: {
    start: string
    end: string
  }
}

export const statsAPI = {
  get: async (): Promise<UsageStats> => {
    return apiClient.get<UsageStats>('/api/v1/stats')
  },
}

