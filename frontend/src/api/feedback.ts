import { apiClient } from './client';

export interface FeedbackRequest {
  review_id: string;
  comment_id: string;
  file_path: string;
  line_number: number;
  feedback: 'accepted' | 'dismissed' | 'fixed';
}

export interface FeedbackPatterns {
  feedback_distribution: Record<string, number>;
  total_feedback: number;
  acceptance_rate?: number;
}

export interface TeamPreferences {
  patterns: FeedbackPatterns;
  learned_rules: string[];
}

export const feedbackAPI = {
  record: async (feedback: FeedbackRequest): Promise<{ status: string }> => {
    return apiClient.post('/api/v1/feedback', feedback);
  },

  getPatterns: async (): Promise<FeedbackPatterns> => {
    return apiClient.get('/api/v1/feedback/patterns');
  },

  getPreferences: async (): Promise<TeamPreferences> => {
    return apiClient.get('/api/v1/feedback/preferences');
  },
};

