// Package dto provides Data Transfer Objects for API request/response structures.
package dto

import (
	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
)

// RepositoryResponse represents a repository configuration.
type RepositoryResponse struct {
	ID      uuid.UUID   `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name    string      `json:"name" example:"owner/repo"`
	Enabled bool        `json:"enabled" example:"true"`
	Config  RepoConfigDTO `json:"config"`
} // @name Repository

// RepoConfigDTO represents repository configuration overrides.
type RepoConfigDTO struct {
	MaxConcurrency      *int    `json:"maxConcurrency,omitempty" example:"5"`
	ComplexityThreshold *int    `json:"complexityThreshold,omitempty" example:"70"`
	ForceDesignConfirm  *bool   `json:"forceDesignConfirm,omitempty" example:"false"`
	DefaultModel        *string `json:"defaultModel,omitempty" example:"claude-3"`
	TaskRetryLimit      *int    `json:"taskRetryLimit,omitempty" example:"3"`
} // @name RepoConfig

// CreateRepositoryRequest represents create repository request.
type CreateRepositoryRequest struct {
	Name   string        `json:"name" example:"owner/repo" validate:"required"`
	Config *RepoConfigDTO `json:"config,omitempty"`
} // @name CreateRepository

// UpdateRepositoryRequest represents update repository request.
type UpdateRepositoryRequest struct {
	Config RepoConfigDTO `json:"config"`
} // @name UpdateRepository

// EffectiveConfigResponse represents effective configuration for a repository.
type EffectiveConfigResponse struct {
	RepositoryName string        `json:"repositoryName" example:"owner/repo"`
	RepositoryID   uuid.UUID     `json:"repositoryId,omitempty"`
	Enabled        bool          `json:"enabled"`
	GlobalConfig   RepoConfigDTO `json:"globalConfig"`
	RepoConfig     RepoConfigDTO `json:"repoConfig"`
	Effective      RepoConfigDTO `json:"effective"`
	HasRepoConfig  bool          `json:"hasRepoConfig"`
} // @name EffectiveConfig

// RepositoryListRequest represents query parameters for listing repositories.
type RepositoryListRequest struct {
	Page     int    `query:"page" default:"1"`
	PageSize int    `query:"pageSize" default:"20"`
	Enabled  string `query:"enabled" example:"true"` // true, false, all (default: all)
}

// ToRepositoryResponse converts entity.Repository to DTO.
func ToRepositoryResponse(repo *entity.Repository) RepositoryResponse {
	return RepositoryResponse{
		ID:      repo.ID,
		Name:    repo.Name,
		Enabled: repo.Enabled,
		Config:  ToRepoConfigDTO(repo.Config),
	}
}

// ToRepoConfigDTO converts entity.RepoConfig to DTO.
func ToRepoConfigDTO(config entity.RepoConfig) RepoConfigDTO {
	return RepoConfigDTO{
		MaxConcurrency:      config.MaxConcurrency,
		ComplexityThreshold: config.ComplexityThreshold,
		ForceDesignConfirm:  config.ForceDesignConfirm,
		DefaultModel:        config.DefaultModel,
		TaskRetryLimit:      config.TaskRetryLimit,
	}
}

// ToRepositoryList converts repositories to paginated response.
func ToRepositoryList(repos []*entity.Repository, page, pageSize int, total int64) PaginatedResponse[RepositoryResponse] {
	data := make([]RepositoryResponse, len(repos))
	for i, repo := range repos {
		data[i] = ToRepositoryResponse(repo)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return PaginatedResponse[RepositoryResponse]{
		Data: data,
		Pagination: PaginationDTO{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}
}