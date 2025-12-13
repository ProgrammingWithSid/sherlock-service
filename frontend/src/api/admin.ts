import { apiClient } from './client'

export interface SystemStats {
  total_organizations: number
  total_users: number
  total_repositories: number
  total_reviews: number
  reviews_this_month: number
}

export interface Organization {
  id: string
  name: string
  slug: string
  plan: string
}

export interface User {
  id: string
  email: string
  name: string
  role: string
  org_id?: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export const adminAPI = {
  getStats: async (): Promise<SystemStats> => {
    return apiClient.get<SystemStats>('/api/v1/admin/stats')
  },

  listOrganizations: async (): Promise<Organization[]> => {
    return apiClient.get<Organization[]>('/api/v1/admin/organizations')
  },

  getOrganization: async (id: string): Promise<any> => {
    return apiClient.get<any>(`/api/v1/admin/organizations/${id}`)
  },

  listUsers: async (): Promise<User[]> => {
    return apiClient.get<User[]>('/api/v1/admin/users')
  },
}
