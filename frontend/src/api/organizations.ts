import { apiClient } from './client'

export interface GlobalRulesResponse {
  rules: string[]
}

export interface GlobalRulesRequest {
  rules: string[]
}

export const organizationsAPI = {
  /**
   * Get global rules for an organization
   */
  async getGlobalRules(orgId: string): Promise<string[]> {
    const response = await apiClient.get<GlobalRulesResponse>(`/api/v1/organizations/${orgId}/rules`)
    return response.rules
  },

  /**
   * Update global rules for an organization
   */
  async updateGlobalRules(orgId: string, rules: string[]): Promise<string[]> {
    const response = await apiClient.put<GlobalRulesResponse>(
      `/api/v1/organizations/${orgId}/rules`,
      { rules }
    )
    return response.rules
  },
}
