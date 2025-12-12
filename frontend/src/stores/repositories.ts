import { repositoriesAPI } from '@/api/repositories'
import type { Repository } from '@/types'
import { defineStore } from 'pinia'

interface RepositoriesState {
  items: Repository[]
  loading: boolean
  error: string | null
}

export const useRepositoriesStore = defineStore('repositories', {
  state: (): RepositoriesState => ({
    items: [],
    loading: false,
    error: null,
  }),

  actions: {
    async fetchRepositories(): Promise<void> {
      this.loading = true
      this.error = null

      try {
        const repos = await repositoriesAPI.list()
        this.items = repos
      } catch (error) {
        this.error = error instanceof Error ? error.message : 'Failed to fetch repositories'
      } finally {
        this.loading = false
      }
    },

    async fetchRepository(id: string): Promise<Repository | null> {
      this.loading = true
      this.error = null

      try {
        const repo = await repositoriesAPI.get(id)
        const index = this.items.findIndex((r) => r.id === repo.id)
        if (index >= 0) {
          this.items[index] = repo
        } else {
          this.items.push(repo)
        }
        return repo
      } catch (error) {
        this.error = error instanceof Error ? error.message : 'Failed to fetch repository'
        return null
      } finally {
        this.loading = false
      }
    },

    async setActive(id: string, isActive: boolean): Promise<void> {
      this.loading = true
      this.error = null

      try {
        await repositoriesAPI.setActive(id, isActive)
        const index = this.items.findIndex((r) => r.id === id)
        if (index >= 0) {
          this.items[index].is_active = isActive
        }
      } catch (error) {
        this.error = error instanceof Error ? error.message : 'Failed to update repository'
      } finally {
        this.loading = false
      }
    },

    setRepositories(repos: Repository[]): void {
      this.items = repos
    },

    addRepository(repo: Repository): void {
      this.items.push(repo)
    },

    updateRepository(repo: Repository): void {
      const index = this.items.findIndex((r) => r.id === repo.id)
      if (index >= 0) {
        this.items[index] = repo
      }
    },

    removeRepository(id: string): void {
      this.items = this.items.filter((r) => r.id !== id)
    },

    setLoading(loading: boolean): void {
      this.loading = loading
    },

    setError(error: string | null): void {
      this.error = error
    },
  },

  getters: {
    activeRepositories: (state): Repository[] => {
      return state.items.filter((r) => r.is_active)
    },

    byPlatform: (state) => {
      return (platform: string): Repository[] => {
        return state.items.filter((r) => r.platform === platform)
      }
    },
  },
})
