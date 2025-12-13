export type Platform = 'github' | 'gitlab'

export type ReviewStatus = 'pending' | 'processing' | 'completed' | 'failed'

export type ReviewRecommendation = 'APPROVE' | 'REQUEST_CHANGES' | 'COMMENT'

export type Severity = 'error' | 'warning' | 'info'

export type Category = 'bugs' | 'security' | 'performance' | 'code_quality' | 'architecture'

export type Plan = 'free' | 'pro' | 'team' | 'enterprise'

export interface Organization {
  id: string
  name: string
  slug: string
  plan: Plan
  stripe_customer_id?: string
  stripe_subscription_id?: string
  plan_activated_at?: string
  created_at: string
  updated_at: string
}

export interface Repository {
  id: string
  org_id: string
  platform: Platform
  external_id: string
  name: string
  full_name: string
  is_private: boolean
  is_active: boolean
  config: string
  created_at: string
}

export interface Review {
  id: string
  org_id: string
  repo_id: string
  pr_number: number
  head_sha: string
  status: ReviewStatus
  result?: string
  comments_posted: number
  duration_ms?: number
  ai_provider: string
  created_at: string
  completed_at?: string
}

export interface ReviewComment {
  file: string
  line: number
  severity: Severity
  category: Category
  message: string
  fix?: string
}

export interface ReviewResult {
  recommendation: ReviewRecommendation
  summary: ReviewSummary
  comments: ReviewComment[]
}

export interface ReviewSummary {
  total_issues: number
  errors: number
  warnings: number
  suggestions: number
}

export interface RepoConfig {
  ai?: {
    provider?: string
    model?: string
  }
  review?: {
    enabled?: boolean
    on_open?: boolean
    on_push?: boolean
    incremental?: boolean
  }
  comments?: {
    post_summary?: boolean
    post_inline?: boolean
    max_comments?: number
  }
  focus?: {
    bugs?: boolean
    security?: boolean
    performance?: boolean
    code_quality?: boolean
    architecture?: boolean
  }
  rules?: string[]
  ignore?: {
    paths?: string[]
    files?: string[]
  }
}


