package github

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
)

// MergeMethod represents the method to merge a PR.
type MergeMethod string

const (
	MergeMethodSquash MergeMethod = "squash"
	MergeMethodMerge  MergeMethod = "merge"
	MergeMethodRebase MergeMethod = "rebase"
)

// PullRequestService provides PR API operations.
type PullRequestService struct {
	clientManager *ClientManager
	logger        *zap.Logger
}

// NewPullRequestService creates a new PullRequestService.
func NewPullRequestService(clientManager *ClientManager, logger *zap.Logger) *PullRequestService {
	return &PullRequestService{
		clientManager: clientManager,
		logger:        logger.Named("github.pr"),
	}
}

// PRCreateRequest represents a request to create a PR.
type PRCreateRequest struct {
	Title      string
	Body       string
	HeadBranch string
	BaseBranch string
	Draft      bool
}

// PRUpdateRequest represents a request to update a PR.
type PRUpdateRequest struct {
	Title *string
	Body  *string
	State *string // "open", "closed"
	Draft *bool
}

// PRInfo represents basic PR information.
type PRInfo struct {
	Number       int
	Title        string
	Body         string
	State        string
	HeadBranch   string
	BaseBranch   string
	Mergeable    *bool
	Merged       bool
	Draft        bool
	HTMLURL      string
	CreatedAt    time.Time
	User         string
	ReviewStatus string
}

// PullRequest represents detailed PR information.
type PullRequest struct {
	PRInfo
	Commits        int
	Additions      int
	Deletions      int
	Changed        int
	Comments       int
	ReviewComments int
}

// getClient gets a GitHub client for the specified repository.
func (s *PullRequestService) getClient(ctx context.Context, owner, repo string) (*Client, error) {
	return s.clientManager.GetClient(ctx, owner+"/"+repo)
}

// CreatePullRequest creates a new PR.
func (s *PullRequestService) CreatePullRequest(ctx context.Context, owner, repo string, req *PRCreateRequest) (*PRInfo, error) {
	client, err := s.getClient(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	newPR := &github.NewPullRequest{
		Title: &req.Title,
		Body:  &req.Body,
		Head:  &req.HeadBranch,
		Base:  &req.BaseBranch,
		Draft: &req.Draft,
	}

	pr, err := executeWithRetry(client, ctx, func() (*github.PullRequest, *github.Response, error) {
		return client.GitHub().PullRequests.Create(ctx, owner, repo, newPR)
	})

	if err != nil {
		return nil, err
	}

	s.logger.Info("PR created",
		zap.String("repository", owner+"/"+repo),
		zap.Int("pr_number", pr.GetNumber()),
		zap.String("head", req.HeadBranch),
		zap.String("base", req.BaseBranch),
	)

	return s.toPRInfo(pr), nil
}

// GetPullRequest fetches a PR by number.
func (s *PullRequestService) GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	client, err := s.getClient(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	pr, err := executeWithRetry(client, ctx, func() (*github.PullRequest, *github.Response, error) {
		return client.GitHub().PullRequests.Get(ctx, owner, repo, number)
	})

	if err != nil {
		return nil, err
	}

	return s.toPullRequest(pr), nil
}

// UpdatePullRequest updates an existing PR.
func (s *PullRequestService) UpdatePullRequest(ctx context.Context, owner, repo string, number int, req *PRUpdateRequest) error {
	client, err := s.getClient(ctx, owner, repo)
	if err != nil {
		return err
	}

	update := &github.PullRequest{}

	if req.Title != nil {
		update.Title = req.Title
	}
	if req.Body != nil {
		update.Body = req.Body
	}
	if req.State != nil {
		update.State = req.State
	}
	if req.Draft != nil {
		update.Draft = req.Draft
	}

	_, err = executeWithRetry(client, ctx, func() (*github.PullRequest, *github.Response, error) {
		return client.GitHub().PullRequests.Edit(ctx, owner, repo, number, update)
	})

	if err != nil {
		return err
	}

	s.logger.Info("PR updated",
		zap.String("repository", owner+"/"+repo),
		zap.Int("pr_number", number),
	)

	return nil
}

// ClosePullRequest closes a PR without merging.
func (s *PullRequestService) ClosePullRequest(ctx context.Context, owner, repo string, number int) error {
	client, err := s.getClient(ctx, owner, repo)
	if err != nil {
		return err
	}

	state := "closed"

	_, err = executeWithRetry(client, ctx, func() (*github.PullRequest, *github.Response, error) {
		return client.GitHub().PullRequests.Edit(ctx, owner, repo, number, &github.PullRequest{
			State: &state,
		})
	})

	if err != nil {
		return err
	}

	s.logger.Info("PR closed",
		zap.String("repository", owner+"/"+repo),
		zap.Int("pr_number", number),
	)

	return nil
}

// ReopenPullRequest reopens a closed PR.
func (s *PullRequestService) ReopenPullRequest(ctx context.Context, owner, repo string, number int) error {
	client, err := s.getClient(ctx, owner, repo)
	if err != nil {
		return err
	}

	state := "open"

	_, err = executeWithRetry(client, ctx, func() (*github.PullRequest, *github.Response, error) {
		return client.GitHub().PullRequests.Edit(ctx, owner, repo, number, &github.PullRequest{
			State: &state,
		})
	})

	if err != nil {
		return err
	}

	s.logger.Info("PR reopened",
		zap.String("repository", owner+"/"+repo),
		zap.Int("pr_number", number),
	)

	return nil
}

// MergePullRequest merges a PR.
func (s *PullRequestService) MergePullRequest(ctx context.Context, owner, repo string, number int, method MergeMethod, commitTitle, commitMessage string) error {
	client, err := s.getClient(ctx, owner, repo)
	if err != nil {
		return err
	}

	mergeMethod := "squash" // default
	switch method {
	case MergeMethodMerge:
		mergeMethod = "merge"
	case MergeMethodRebase:
		mergeMethod = "rebase"
	}

	opts := &github.PullRequestOptions{
		MergeMethod: mergeMethod,
	}
	if commitTitle != "" {
		opts.CommitTitle = commitTitle
	}

	result, err := executeWithRetry(client, ctx, func() (*github.PullRequestMergeResult, *github.Response, error) {
		return client.GitHub().PullRequests.Merge(ctx, owner, repo, number, commitMessage, opts)
	})

	if err != nil {
		return err
	}

	if !result.GetMerged() {
		return errors.New(errors.ErrGitHubAPIError).
			WithDetail(fmt.Sprintf("PR merge failed: %s", result.GetMessage()))
	}

	s.logger.Info("PR merged",
		zap.String("repository", owner+"/"+repo),
		zap.Int("pr_number", number),
		zap.String("method", mergeMethod),
		zap.String("sha", result.GetSHA()),
	)

	return nil
}

// RequestReview requests reviewers for a PR.
func (s *PullRequestService) RequestReview(ctx context.Context, owner, repo string, number int, reviewers []string) error {
	client, err := s.getClient(ctx, owner, repo)
	if err != nil {
		return err
	}

	reviewReq := github.ReviewersRequest{
		Reviewers: reviewers,
	}

	_, err = executeWithRetry(client, ctx, func() (*github.PullRequest, *github.Response, error) {
		return client.GitHub().PullRequests.RequestReviewers(ctx, owner, repo, number, reviewReq)
	})

	if err != nil {
		return err
	}

	s.logger.Info("review requested",
		zap.String("repository", owner+"/"+repo),
		zap.Int("pr_number", number),
		zap.Strings("reviewers", reviewers),
	)

	return nil
}

// RemoveReviewRequest removes requested reviewers from a PR.
func (s *PullRequestService) RemoveReviewRequest(ctx context.Context, owner, repo string, number int, reviewers []string) error {
	client, err := s.getClient(ctx, owner, repo)
	if err != nil {
		return err
	}

	reviewReq := github.ReviewersRequest{
		Reviewers: reviewers,
	}

	err = executeWithRetryResponse(client, ctx, func() (*github.Response, error) {
		return client.GitHub().PullRequests.RemoveReviewers(ctx, owner, repo, number, reviewReq)
	})

	if err != nil {
		return err
	}

	s.logger.Info("review request removed",
		zap.String("repository", owner+"/"+repo),
		zap.Int("pr_number", number),
		zap.Strings("reviewers", reviewers),
	)

	return nil
}

// ListReviews lists all reviews for a PR.
func (s *PullRequestService) ListReviews(ctx context.Context, owner, repo string, number int) ([]*PullRequestReview, error) {
	client, err := s.getClient(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	reviews, err := executeWithRetry(client, ctx, func() ([]*github.PullRequestReview, *github.Response, error) {
		return client.GitHub().PullRequests.ListReviews(ctx, owner, repo, number, nil)
	})

	if err != nil {
		return nil, err
	}

	result := make([]*PullRequestReview, len(reviews))
	for i, r := range reviews {
		result[i] = &PullRequestReview{
			ID:          r.GetID(),
			User:        r.User.GetLogin(),
			State:       r.GetState(),
			Body:        r.GetBody(),
			SubmittedAt: r.GetSubmittedAt().Time,
		}
	}

	return result, nil
}

// PullRequestReview represents a PR review.
type PullRequestReview struct {
	ID          int64
	User        string
	State       string // APPROVED, CHANGES_REQUESTED, COMMENTED, PENDING
	Body        string
	SubmittedAt time.Time
}

// CreateReviewComment adds a comment to a specific line in PR.
func (s *PullRequestService) CreateReviewComment(ctx context.Context, owner, repo string, number int, req *CreateReviewCommentRequest) (int64, error) {
	client, err := s.getClient(ctx, owner, repo)
	if err != nil {
		return 0, err
	}

	comment, err := executeWithRetry(client, ctx, func() (*github.PullRequestComment, *github.Response, error) {
		return client.GitHub().PullRequests.CreateComment(ctx, owner, repo, number, &github.PullRequestComment{
			Body:     &req.Body,
			Path:     &req.Path,
			Position: &req.Position,
		})
	})

	if err != nil {
		return 0, err
	}

	s.logger.Info("review comment created",
		zap.String("repository", owner+"/"+repo),
		zap.Int("pr_number", number),
		zap.Int64("comment_id", comment.GetID()),
		zap.String("path", req.Path),
	)

	return comment.GetID(), nil
}

// CreateReviewCommentRequest represents a request to create a review comment.
type CreateReviewCommentRequest struct {
	Body     string
	Path     string // File path
	Position int    // Line position in the diff
}

// ListComments lists all review comments on a PR.
func (s *PullRequestService) ListComments(ctx context.Context, owner, repo string, number int) ([]*PullRequestReviewComment, error) {
	client, err := s.getClient(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	comments, err := executeWithRetry(client, ctx, func() ([]*github.PullRequestComment, *github.Response, error) {
		return client.GitHub().PullRequests.ListComments(ctx, owner, repo, number, nil)
	})

	if err != nil {
		return nil, err
	}

	result := make([]*PullRequestReviewComment, len(comments))
	for i, c := range comments {
		result[i] = &PullRequestReviewComment{
			ID:        c.GetID(),
			Body:      c.GetBody(),
			Path:      c.GetPath(),
			User:      c.User.GetLogin(),
			CreatedAt: c.GetCreatedAt().Time,
		}
	}

	return result, nil
}

// PullRequestReviewComment represents a review comment on a PR.
type PullRequestReviewComment struct {
	ID        int64
	Body      string
	Path      string
	User      string
	CreatedAt time.Time
}

// ListFiles lists the files changed in a PR.
func (s *PullRequestService) ListFiles(ctx context.Context, owner, repo string, number int) ([]*PullRequestFile, error) {
	client, err := s.getClient(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	files, err := executeWithRetry(client, ctx, func() ([]*github.CommitFile, *github.Response, error) {
		return client.GitHub().PullRequests.ListFiles(ctx, owner, repo, number, nil)
	})

	if err != nil {
		return nil, err
	}

	result := make([]*PullRequestFile, len(files))
	for i, f := range files {
		result[i] = &PullRequestFile{
			Filename:  f.GetFilename(),
			Status:    f.GetStatus(),
			Additions: f.GetAdditions(),
			Deletions: f.GetDeletions(),
			Changes:   f.GetChanges(),
			Patch:     f.GetPatch(),
		}
	}

	return result, nil
}

// PullRequestFile represents a file changed in a PR.
type PullRequestFile struct {
	Filename  string
	Status    string // added, modified, removed, renamed
	Additions int
	Deletions int
	Changes   int
	Patch     string
}

// toPRInfo converts github.PullRequest to PRInfo.
func (s *PullRequestService) toPRInfo(pr *github.PullRequest) *PRInfo {
	if pr == nil {
		return nil
	}

	info := &PRInfo{
		Number:     pr.GetNumber(),
		Title:      pr.GetTitle(),
		Body:       pr.GetBody(),
		State:      pr.GetState(),
		HeadBranch: pr.Head.GetRef(),
		BaseBranch: pr.Base.GetRef(),
		Merged:     pr.GetMerged(),
		Draft:      pr.GetDraft(),
		HTMLURL:    pr.GetHTMLURL(),
		CreatedAt:  pr.GetCreatedAt().Time,
		User:       pr.User.GetLogin(),
	}

	if pr.Mergeable != nil {
		info.Mergeable = pr.Mergeable
	}

	return info
}

// toPullRequest converts github.PullRequest to PullRequest.
func (s *PullRequestService) toPullRequest(pr *github.PullRequest) *PullRequest {
	if pr == nil {
		return nil
	}

	info := s.toPRInfo(pr)
	return &PullRequest{
		PRInfo:         *info,
		Commits:        pr.GetCommits(),
		Additions:      pr.GetAdditions(),
		Deletions:      pr.GetDeletions(),
		Changed:        pr.GetChangedFiles(),
		Comments:       pr.GetComments(),
		ReviewComments: pr.GetReviewComments(),
	}
}