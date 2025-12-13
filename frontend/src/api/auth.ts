import { apiClient } from './client'
import type { Organization } from '@/types'

export interface CurrentUser {
  org_id?: string
  name: string
  plan?: string
  role?: string
  token: string
}

export interface SignupRequest {
  name: string
  email: string
  password: string
  org_name: string
  org_slug?: string
}

export interface SignupResponse {
  user: {
    id: string
    email: string
    name: string
    role: string
    org_id: string
  }
  organization: {
    id: string
    name: string
    slug: string
    plan: string
  }
  token: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  user: {
    id: string
    email: string
    name: string
    role: string
    org_id?: string
  }
  organization?: {
    id: string
    name: string
    slug: string
    plan: string
  }
  token: string
}

export const authAPI = {
  signup: async (data: SignupRequest): Promise<SignupResponse> => {
    return apiClient.post<SignupResponse>('/api/v1/auth/signup', data)
  },

  login: async (data: LoginRequest): Promise<LoginResponse> => {
    return apiClient.post<LoginResponse>('/api/v1/auth/login', data)
  },

  getCurrentUser: async (): Promise<CurrentUser> => {
    return apiClient.get<CurrentUser>('/api/v1/auth/me')
  },

  listOrganizations: async (): Promise<Organization[]> => {
    return apiClient.get<Organization[]>('/api/v1/auth/organizations')
  },

  logout: async (): Promise<void> => {
    return apiClient.post<void>('/api/v1/auth/logout')
  },
}
