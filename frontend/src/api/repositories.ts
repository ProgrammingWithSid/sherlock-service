import type { RepoConfig, Repository } from '@/types'
import { apiClient } from './client'

export const repositoriesAPI = {
  list: async (): Promise<Repository[]> => {
    return apiClient.get<Repository[]>('/api/v1/repositories')
  },

  get: async (id: string): Promise<Repository> => {
    return apiClient.get<Repository>(`/api/v1/repositories/${id}`)
  },

  getConfig: async (owner: string, repo: string): Promise<RepoConfig> => {
    return apiClient.get<RepoConfig>(`/api/v1/repos/${owner}/${repo}/config`)
  },

  updateConfig: async (owner: string, repo: string, config: RepoConfig): Promise<{ status: string }> => {
    return apiClient.put<{ status: string }>(`/api/v1/repos/${owner}/${repo}/config`, config)
  },

  setActive: async (id: string, isActive: boolean): Promise<void> => {
    return apiClient.put<void>(`/api/v1/repositories/${id}/active`, { is_active: isActive })
  },

  connect: async (platform: string, owner: string, repo: string): Promise<Repository> => {
    return apiClient.post<Repository>('/api/v1/repositories', {
      platform,
      owner,
      repo,
    })
  },
}
