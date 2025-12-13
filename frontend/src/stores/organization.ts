import { defineStore } from 'pinia'
import { authAPI } from '@/api/auth'
import type { Organization } from '@/types'

interface OrganizationState {
  current: Organization | null
  organizations: Organization[]
  loading: boolean
  error: string | null
}

export const useOrganizationStore = defineStore('organization', {
  state: (): OrganizationState => ({
    current: null,
    organizations: [],
    loading: false,
    error: null,
  }),

  actions: {
    setOrganization(org: Organization): void {
      this.current = org
      localStorage.setItem('org_id', org.id)
    },

    clearOrganization(): void {
      this.current = null
      localStorage.removeItem('org_id')
    },

    setLoading(loading: boolean): void {
      this.loading = loading
    },

    setError(error: string | null): void {
      this.error = error
    },

    async fetchOrganizations(): Promise<void> {
      this.loading = true
      this.error = null

      try {
        const orgs = await authAPI.listOrganizations()
        this.organizations = orgs
      } catch (err) {
        this.error = err instanceof Error ? err.message : 'Failed to fetch organizations'
        console.error('Failed to fetch organizations:', err)
      } finally {
        this.loading = false
      }
    },
  },

  getters: {
    isFreePlan: (state): boolean => {
      return state.current?.plan === 'free'
    },

    isProPlan: (state): boolean => {
      return state.current?.plan === 'pro'
    },

    isTeamPlan: (state): boolean => {
      return state.current?.plan === 'team'
    },

    isEnterprisePlan: (state): boolean => {
      return state.current?.plan === 'enterprise'
    },
  },
})
