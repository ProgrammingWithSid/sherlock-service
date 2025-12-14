import { defineStore } from 'pinia'
import { authAPI } from '@/api/auth'

interface AuthState {
  isAuthenticated: boolean
  token: string | null
  orgId: string | null
  user: {
    org_id?: string
    name: string
    plan?: string
    role?: string
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
        this.user = {
          org_id: user.org_id,
          name: user.name,
          plan: user.plan,
          role: user.role,
        }
        if (user.org_id) {
          this.orgId = user.org_id
          localStorage.setItem('org_id', user.org_id)
        } else {
          // Clear orgId if user doesn't have one (e.g., super admin)
          this.orgId = null
          localStorage.removeItem('org_id')
        }
        this.isAuthenticated = true
        // Update token in state if returned
        if (user.token) {
          this.token = user.token
          localStorage.setItem('session_token', user.token)
        }
      } catch (err) {
        this.error = err instanceof Error ? err.message : 'Failed to fetch user'
        // Don't logout here - let the caller decide
        throw err
      } finally {
        this.loading = false
      }
    },

    async init(): Promise<void> {
      const token = localStorage.getItem('session_token')
      const orgId = localStorage.getItem('org_id')

      if (token) {
        // Set auth state first (orgId might be null for super admins)
        this.token = token
        this.isAuthenticated = true
        if (orgId) {
          this.orgId = orgId
        }

        // Try to fetch current user to validate session
        try {
          await this.fetchCurrentUser()
        } catch (err) {
          // If fetch fails, session might be expired - clear auth but don't throw
          console.warn('Session validation failed:', err)
          this.logout()
        }
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

  getters: {
    isSuperAdmin: (state): boolean => {
      return state.user?.role === 'super_admin'
    },
  },
})
