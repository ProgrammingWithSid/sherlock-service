import { apiClient } from './client';

export interface ReviewMetrics {
  total: number;
  successful: number;
  failed: number;
  average_duration_ms: number;
  cache_hits: number;
  cache_misses: number;
  incremental: number;
  full: number;
}

export interface MetricsRates {
  cache_hit_rate: number;
  success_rate: number;
}

export interface MetricsResponse {
  reviews: ReviewMetrics;
  rates: MetricsRates;
}

export async function getMetrics(): Promise<MetricsResponse> {
  return apiClient.get<MetricsResponse>('/api/v1/metrics');
}
