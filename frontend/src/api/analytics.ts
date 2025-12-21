import { apiClient } from './client'

export interface TimeSeriesPoint {
  date: string
  value: number
}

export interface QualityTrend {
  date: string
  overall_score: number
  accuracy: number
  actionability: number
  coverage: number
}

export interface IssueCategoryBreakdown {
  category: string
  count: number
  errors: number
  warnings: number
  info: number
}

export interface RepositoryMetrics {
  repository_id: string
  repository_name: string
  total_reviews: number
  total_issues: number
  average_score: number
  success_rate: number
}

export interface SeverityTrends {
  errors: TimeSeriesPoint[]
  warnings: TimeSeriesPoint[]
  suggestions: TimeSeriesPoint[]
}

export const analyticsAPI = {
  getQualityTrends: async (days: number = 30): Promise<QualityTrend[]> => {
    return apiClient.get<QualityTrend[]>(`/api/v1/analytics/quality-trends?days=${days}`)
  },

  getIssueTrends: async (days: number = 30): Promise<TimeSeriesPoint[]> => {
    return apiClient.get<TimeSeriesPoint[]>(`/api/v1/analytics/issue-trends?days=${days}`)
  },

  getSeverityTrends: async (days: number = 30): Promise<SeverityTrends> => {
    return apiClient.get<SeverityTrends>(`/api/v1/analytics/severity-trends?days=${days}`)
  },

  getCategoryBreakdown: async (days: number = 30): Promise<IssueCategoryBreakdown[]> => {
    return apiClient.get<IssueCategoryBreakdown[]>(`/api/v1/analytics/category-breakdown?days=${days}`)
  },

  getRepositoryComparison: async (days: number = 30): Promise<RepositoryMetrics[]> => {
    return apiClient.get<RepositoryMetrics[]>(`/api/v1/analytics/repository-comparison?days=${days}`)
  },
}
