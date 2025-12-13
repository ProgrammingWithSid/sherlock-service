import { defineStore } from 'pinia'
import { authAPI } from '@/api/auth'

interface AuthState {
  isAuthenticated: boolean
  token: string | null
  orgId: string | null
  user: {
    org_id: string
    name: string
    plan: string
  } | null
  loading: boolean
  error: string | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    isAuthenticated: false,
    token: null,
    orgId: null,
    user: null,
    loading: false,
    error: null,
  }),

  actions: {
    async setAuth(token: string, orgId: string): Promise<void> {
      this.token = token
      this.orgId = orgId
      this.isAuthenticated = true
      localStorage.setItem('session_token', token)
      localStorage.setItem('org_id', orgId)
    },

    async fetchCurrentUser(): Promise<void> {
      this.loading = true
      this.error = null

      try {
        const user = await authAPI.getCurrentUser()
        this.user = user
        this.isAuthenticated = true
      } catch (err) {
        this.error = err instanceof Error ? err.message : 'Failed to fetch user'
        this.logout()
      } finally {
        this.loading = false
      }
    },

    async init(): Promise<void> {
      const token = localStorage.getItem('session_token')
      const orgId = localStorage.getItem('org_id')

      if (token && orgId) {
        await this.setAuth(token, orgId)
        await this.fetchCurrentUser()
      }
    },

    logout(): void {
      this.isAuthenticated = false
      this.token = null
      this.orgId = null
      this.user = null
      localStorage.removeItem('session_token')
      localStorage.removeItem('org_id')
    },
  },
})


