import { defineStore } from 'pinia'
import type { Organization } from '@/types'

interface OrganizationState {
  current: Organization | null
  loading: boolean
  error: string | null
}

export const useOrganizationStore = defineStore('organization', {
  state: (): OrganizationState => ({
    current: null,
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


