import { apiClient } from './client'

export interface CurrentUser {
  org_id: string
  name: string
  plan: string
  token: string
}

export const authAPI = {
  getCurrentUser: async (): Promise<CurrentUser> => {
    return apiClient.get<CurrentUser>('/api/v1/auth/me')
  },

  logout: async (): Promise<void> => {
    return apiClient.post<void>('/api/v1/auth/logout')
  },
}

