package plan

import (
	"fmt"

	"github.com/sherlock/service/internal/types"
)

type PlanLimits struct {
	MaxRepos           int
	MaxReviewsPerMonth int
	MaxRules           int
	MaxTeamMembers     int
	AllowPrivateRepos  bool
	AllowPriorityQueue bool
	AllowCustomAI      bool
	AllowAPI           bool
	AIProvider         string
}

var planLimits = map[types.Plan]PlanLimits{
	types.PlanFree: {
		MaxRepos:           3,
		MaxReviewsPerMonth: 50,
		MaxRules:           5,
		MaxTeamMembers:     1,
		AllowPrivateRepos:  false,
		AllowPriorityQueue: false,
		AllowCustomAI:      false,
		AllowAPI:           false,
		AIProvider:         "gpt-3.5",
	},
	types.PlanPro: {
		MaxRepos:           10,
		MaxReviewsPerMonth: 500,
		MaxRules:           20,
		MaxTeamMembers:     3,
		AllowPrivateRepos:  true,
		AllowPriorityQueue: false,
		AllowCustomAI:      false,
		AllowAPI:           true,
		AIProvider:         "gpt-4",
	},
	types.PlanTeam: {
		MaxRepos:           -1, // unlimited
		MaxReviewsPerMonth: -1, // unlimited
		MaxRules:           -1, // unlimited
		MaxTeamMembers:     -1, // unlimited
		AllowPrivateRepos:  true,
		AllowPriorityQueue: true,
		AllowCustomAI:      false,
		AllowAPI:           true,
		AIProvider:         "claude",
	},
	types.PlanEnterprise: {
		MaxRepos:           -1, // unlimited
		MaxReviewsPerMonth: -1, // unlimited
		MaxRules:           -1, // unlimited
		MaxTeamMembers:     -1, // unlimited
		AllowPrivateRepos:  true,
		AllowPriorityQueue: true,
		AllowCustomAI:      true,
		AllowAPI:           true,
		AIProvider:         "custom",
	},
}

type Service struct {
	getMonthlyReviewCount func(string) (int, error)
	getRepoCount          func(string) (int, error)
}

func NewService(
	getMonthlyReviewCount func(string) (int, error),
	getRepoCount func(string) (int, error),
) *Service {
	return &Service{
		getMonthlyReviewCount: getMonthlyReviewCount,
		getRepoCount:          getRepoCount,
	}
}

func (s *Service) GetLimits(plan types.Plan) PlanLimits {
	return planLimits[plan]
}

func (s *Service) CheckCanReview(orgID string, plan types.Plan) (bool, string) {
	limits := planLimits[plan]

	if limits.MaxReviewsPerMonth == -1 {
		return true, ""
	}

	count, err := s.getMonthlyReviewCount(orgID)
	if err != nil {
		return false, "Failed to check review count"
	}

	if count >= limits.MaxReviewsPerMonth {
		return false, fmt.Sprintf("Monthly review limit reached (%d). Upgrade to continue.", limits.MaxReviewsPerMonth)
	}

	return true, ""
}

func (s *Service) CheckCanAddRepo(orgID string, plan types.Plan, isPrivate bool) (bool, string) {
	limits := planLimits[plan]

	if limits.MaxRepos != -1 {
		count, err := s.getRepoCount(orgID)
		if err != nil {
			return false, "Failed to check repo count"
		}

		if count >= limits.MaxRepos {
			return false, fmt.Sprintf("Repository limit reached (%d). Upgrade for more repos.", limits.MaxRepos)
		}
	}

	if isPrivate && !limits.AllowPrivateRepos {
		return false, "Private repositories require Pro plan or higher."
	}

	return true, ""
}

func (s *Service) GetAIProvider(plan types.Plan, customProvider string) string {
	limits := planLimits[plan]
	if limits.AllowCustomAI && customProvider != "" {
		return customProvider
	}
	return limits.AIProvider
}

func (s *Service) GetQueuePriority(plan types.Plan) int {
	limits := planLimits[plan]
	if limits.AllowPriorityQueue {
		if plan == types.PlanEnterprise {
			return 100
		}
		return 50
	}
	return 1
}
