package github

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"go.uber.org/zap"
)

// IssueService provides Issue API operations.
type IssueService struct {
	client *Client
	logger *zap.Logger
}

// NewIssueService creates a new IssueService.
func NewIssueService(client *Client, logger *zap.Logger) *IssueService {
	return &IssueService{
		client: client,
		logger: logger.Named("github.issue"),
	}
}

// GetIssue fetches an issue by number.
func (s *IssueService) GetIssue(ctx context.Context, owner, repo string, number int) (*entity.Issue, error) {
	issue, err := executeWithRetry(s.client, ctx, func() (*github.Issue, *github.Response, error) {
		return s.client.GitHub().Issues.Get(ctx, owner, repo, number)
	})

	if err != nil {
		return nil, err
	}

	return s.toEntity(issue, owner+"/"+repo), nil
}

// CreateComment adds a comment to an issue.
func (s *IssueService) CreateComment(ctx context.Context, owner, repo string, number int, body string) (int64, error) {
	comment, err := executeWithRetry(s.client, ctx, func() (*github.IssueComment, *github.Response, error) {
		return s.client.GitHub().Issues.CreateComment(ctx, owner, repo, number, &github.IssueComment{
			Body: &body,
		})
	})

	if err != nil {
		return 0, err
	}

	s.logger.Info("comment created",
		zap.String("repository", owner+"/"+repo),
		zap.Int("issue_number", number),
		zap.Int64("comment_id", comment.GetID()),
	)

	return comment.GetID(), nil
}

// UpdateComment updates an existing comment.
func (s *IssueService) UpdateComment(ctx context.Context, owner, repo string, commentID int64, body string) error {
	_, err := executeWithRetry(s.client, ctx, func() (*github.IssueComment, *github.Response, error) {
		return s.client.GitHub().Issues.EditComment(ctx, owner, repo, commentID, &github.IssueComment{
			Body: &body,
		})
	})

	if err != nil {
		return err
	}

	s.logger.Info("comment updated",
		zap.String("repository", owner+"/"+repo),
		zap.Int64("comment_id", commentID),
	)

	return nil
}

// CloseIssue closes an issue.
func (s *IssueService) CloseIssue(ctx context.Context, owner, repo string, number int) error {
	state := "closed"

	_, err := executeWithRetry(s.client, ctx, func() (*github.Issue, *github.Response, error) {
		return s.client.GitHub().Issues.Edit(ctx, owner, repo, number, &github.IssueRequest{
			State: &state,
		})
	})

	if err != nil {
		return err
	}

	s.logger.Info("issue closed",
		zap.String("repository", owner+"/"+repo),
		zap.Int("issue_number", number),
	)

	return nil
}

// ReopenIssue reopens a closed issue.
func (s *IssueService) ReopenIssue(ctx context.Context, owner, repo string, number int) error {
	state := "open"

	_, err := executeWithRetry(s.client, ctx, func() (*github.Issue, *github.Response, error) {
		return s.client.GitHub().Issues.Edit(ctx, owner, repo, number, &github.IssueRequest{
			State: &state,
		})
	})

	if err != nil {
		return err
	}

	s.logger.Info("issue reopened",
		zap.String("repository", owner+"/"+repo),
		zap.Int("issue_number", number),
	)

	return nil
}

// AddLabels adds labels to an issue.
func (s *IssueService) AddLabels(ctx context.Context, owner, repo string, number int, labels []string) error {
	_, err := executeWithRetry(s.client, ctx, func() ([]*github.Label, *github.Response, error) {
		return s.client.GitHub().Issues.AddLabelsToIssue(ctx, owner, repo, number, labels)
	})

	if err != nil {
		return err
	}

	s.logger.Info("labels added",
		zap.String("repository", owner+"/"+repo),
		zap.Int("issue_number", number),
		zap.Strings("labels", labels),
	)

	return nil
}

// RemoveLabel removes a label from an issue.
func (s *IssueService) RemoveLabel(ctx context.Context, owner, repo string, number int, label string) error {
	err := executeWithRetryResponse(s.client, ctx, func() (*github.Response, error) {
		return s.client.GitHub().Issues.RemoveLabelForIssue(ctx, owner, repo, number, label)
	})

	if err != nil {
		return err
	}

	s.logger.Info("label removed",
		zap.String("repository", owner+"/"+repo),
		zap.Int("issue_number", number),
		zap.String("label", label),
	)

	return nil
}

// GetLabels gets all labels for an issue.
func (s *IssueService) GetLabels(ctx context.Context, owner, repo string, number int) ([]string, error) {
	labels, err := executeWithRetry(s.client, ctx, func() ([]*github.Label, *github.Response, error) {
		return s.client.GitHub().Issues.ListLabelsByIssue(ctx, owner, repo, number, nil)
	})

	if err != nil {
		return nil, err
	}

	result := make([]string, len(labels))
	for i, l := range labels {
		result[i] = l.GetName()
	}

	return result, nil
}

// ListComments lists all comments on an issue.
func (s *IssueService) ListComments(ctx context.Context, owner, repo string, number int) ([]*IssueComment, error) {
	comments, err := executeWithRetry(s.client, ctx, func() ([]*github.IssueComment, *github.Response, error) {
		return s.client.GitHub().Issues.ListComments(ctx, owner, repo, number, nil)
	})

	if err != nil {
		return nil, err
	}

	result := make([]*IssueComment, len(comments))
	for i, c := range comments {
		result[i] = &IssueComment{
			ID:        c.GetID(),
			Body:      c.GetBody(),
			Author:    c.User.GetLogin(),
			CreatedAt: c.GetCreatedAt().Time,
			UpdatedAt: c.GetUpdatedAt().Time,
		}
	}

	return result, nil
}

// IssueComment represents a comment on an issue.
type IssueComment struct {
	ID        int64
	Body      string
	Author    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// toEntity converts github.Issue to entity.Issue.
func (s *IssueService) toEntity(ghIssue *github.Issue, repository string) *entity.Issue {
	if ghIssue == nil {
		return nil
	}

	labels := []string{}
	for _, l := range ghIssue.Labels {
		labels = append(labels, l.GetName())
	}

	var author string
	if ghIssue.User != nil {
		author = ghIssue.User.GetLogin()
	}

	return entity.NewIssueFromGitHub(
		ghIssue.GetNumber(),
		ghIssue.GetTitle(),
		ghIssue.GetBody(),
		repository,
		author,
		labels,
		ghIssue.GetHTMLURL(),
		ghIssue.GetCreatedAt().Time,
	)
}

// GetPermissionLevel gets the permission level for a user on a repository.
func (s *IssueService) GetPermissionLevel(ctx context.Context, owner, repo, username string) (string, error) {
	perm, err := executeWithRetry(s.client, ctx, func() (*github.RepositoryPermissionLevel, *github.Response, error) {
		return s.client.GitHub().Repositories.GetPermissionLevel(ctx, owner, repo, username)
	})

	if err != nil {
		return "", err
	}

	return perm.GetPermission(), nil
}

// IsRepoAdmin checks if a user has admin or maintain permission on a repository.
func (s *IssueService) IsRepoAdmin(ctx context.Context, owner, repo, username string) (bool, error) {
	perm, err := s.GetPermissionLevel(ctx, owner, repo, username)
	if err != nil {
		return false, err
	}

	return perm == "admin" || perm == "maintain", nil
}

// CreateIssue creates a new issue.
func (s *IssueService) CreateIssue(ctx context.Context, owner, repo, title, body string, labels []string) (*entity.Issue, error) {
	req := &github.IssueRequest{
		Title: &title,
		Body:  &body,
	}
	if len(labels) > 0 {
		req.Labels = &labels
	}

	issue, err := executeWithRetry(s.client, ctx, func() (*github.Issue, *github.Response, error) {
		return s.client.GitHub().Issues.Create(ctx, owner, repo, req)
	})

	if err != nil {
		return nil, err
	}

	s.logger.Info("issue created",
		zap.String("repository", owner+"/"+repo),
		zap.Int("issue_number", issue.GetNumber()),
	)

	return s.toEntity(issue, owner+"/"+repo), nil
}

// UpdateIssue updates an existing issue.
func (s *IssueService) UpdateIssue(ctx context.Context, owner, repo string, number int, title, body *string) error {
	req := &github.IssueRequest{}
	if title != nil {
		req.Title = title
	}
	if body != nil {
		req.Body = body
	}

	_, err := executeWithRetry(s.client, ctx, func() (*github.Issue, *github.Response, error) {
		return s.client.GitHub().Issues.Edit(ctx, owner, repo, number, req)
	})

	if err != nil {
		return err
	}

	s.logger.Info("issue updated",
		zap.String("repository", owner+"/"+repo),
		zap.Int("issue_number", number),
	)

	return nil
}

// AssignIssue assigns users to an issue.
func (s *IssueService) AssignIssue(ctx context.Context, owner, repo string, number int, assignees []string) error {
	_, err := executeWithRetry(s.client, ctx, func() (*github.Issue, *github.Response, error) {
		return s.client.GitHub().Issues.AddAssignees(ctx, owner, repo, number, assignees)
	})

	if err != nil {
		return err
	}

	s.logger.Info("issue assigned",
		zap.String("repository", owner+"/"+repo),
		zap.Int("issue_number", number),
		zap.Strings("assignees", assignees),
	)

	return nil
}

// IsNotFoundError checks if the error is a not found error.
func IsNotFoundError(err error) bool {
	var ghErr *github.ErrorResponse
	if stderrors.As(err, &ghErr) {
		return ghErr.Response.StatusCode == 404
	}
	return false
}