package webhook

import (
	"encoding/json"
	"time"

	"github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
)

// EventType constants for GitHub webhook events.
const (
	EventTypeIssues            = "issues"
	EventTypeIssueComment      = "issue_comment"
	EventTypePullRequest       = "pull_request"
	EventTypePullRequestReview = "pull_request_review"
	EventTypePush              = "push"
)

// WebhookEvent represents a parsed webhook event.
type WebhookEvent interface {
	EventType() string
	Repository() string
	Actor() string
	Action() string
}

// EventParser parses GitHub webhook events.
type EventParser struct {
	logger *zap.Logger
}

// NewEventParser creates a new event parser.
func NewEventParser(logger *zap.Logger) *EventParser {
	return &EventParser{
		logger: logger.Named("webhook.parser"),
	}
}

// Parse parses the raw payload into a WebhookEvent.
func (p *EventParser) Parse(eventType string, payload []byte) (WebhookEvent, error) {
	switch eventType {
	case EventTypeIssues:
		return p.parseIssuesEvent(payload)
	case EventTypeIssueComment:
		return p.parseIssueCommentEvent(payload)
	case EventTypePullRequest:
		return p.parsePullRequestEvent(payload)
	case EventTypePullRequestReview:
		return p.parsePullRequestReviewEvent(payload)
	case EventTypePush:
		return p.parsePushEvent(payload)
	default:
		p.logger.Debug("unsupported event type",
			zap.String("event_type", eventType),
		)
		return &IgnoredEvent{eventType: eventType}, nil
	}
}

// IssueEvent represents an issues webhook event.
type IssueEvent struct {
	EventAction string `json:"action"`
	Issue       struct {
		Number    int       `json:"number"`
		Title     string    `json:"title"`
		Body      string    `json:"body"`
		HTMLURL   string    `json:"html_url"`
		State     string    `json:"state"`
		User      struct {
			Login string `json:"login"`
		} `json:"user"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	} `json:"issue"`
	Repo struct {
		FullName string `json:"full_name"`
		ID       int64  `json:"id"`
	} `json:"repository"`
	EventSender struct {
		Login string `json:"login"`
		ID    int64  `json:"id"`
	} `json:"sender"`
}

func (e *IssueEvent) EventType() string   { return EventTypeIssues }
func (e *IssueEvent) Repository() string  { return e.Repo.FullName }
func (e *IssueEvent) Actor() string       { return e.EventSender.Login }
func (e *IssueEvent) Action() string      { return e.EventAction }
func (e *IssueEvent) IssueNumber() int    { return e.Issue.Number }
func (e *IssueEvent) IssueTitle() string  { return e.Issue.Title }
func (e *IssueEvent) IssueBody() string   { return e.Issue.Body }
func (e *IssueEvent) IssueAuthor() string { return e.Issue.User.Login }
func (e *IssueEvent) IssueState() string  { return e.Issue.State }
func (e *IssueEvent) IssueURL() string    { return e.Issue.HTMLURL }
func (e *IssueEvent) IssueLabels() []string {
	labels := make([]string, len(e.Issue.Labels))
	for i, l := range e.Issue.Labels {
		labels[i] = l.Name
	}
	return labels
}

func (p *EventParser) parseIssuesEvent(payload []byte) (*IssueEvent, error) {
	var event IssueEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, errors.Wrap(errors.ErrBadRequest, err).
			WithDetail("failed to parse issues event payload")
	}
	return &event, nil
}

// IssueCommentEvent represents an issue_comment webhook event.
type IssueCommentEvent struct {
	EventAction string `json:"action"`
	Issue       struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		State  string `json:"state"`
		User   struct {
			Login string `json:"login"`
		} `json:"user"`
		HTMLURL string `json:"html_url"`
	} `json:"issue"`
	Comment struct {
		ID        int64     `json:"id"`
		Body      string    `json:"body"`
		User      struct {
			Login string `json:"login"`
		} `json:"user"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		HTMLURL   string    `json:"html_url"`
	} `json:"comment"`
	Repo struct {
		FullName string `json:"full_name"`
		ID       int64  `json:"id"`
	} `json:"repository"`
	EventSender struct {
		Login string `json:"login"`
		ID    int64  `json:"id"`
	} `json:"sender"`
}

func (e *IssueCommentEvent) EventType() string     { return EventTypeIssueComment }
func (e *IssueCommentEvent) Repository() string    { return e.Repo.FullName }
func (e *IssueCommentEvent) Actor() string         { return e.EventSender.Login }
func (e *IssueCommentEvent) Action() string        { return e.EventAction }
func (e *IssueCommentEvent) IssueNumber() int      { return e.Issue.Number }
func (e *IssueCommentEvent) IssueTitle() string    { return e.Issue.Title }
func (e *IssueCommentEvent) IssueState() string    { return e.Issue.State }
func (e *IssueCommentEvent) IssueAuthor() string   { return e.Issue.User.Login }
func (e *IssueCommentEvent) CommentID() int64      { return e.Comment.ID }
func (e *IssueCommentEvent) CommentBody() string   { return e.Comment.Body }
func (e *IssueCommentEvent) CommentAuthor() string { return e.Comment.User.Login }
func (e *IssueCommentEvent) CommentURL() string    { return e.Comment.HTMLURL }

func (p *EventParser) parseIssueCommentEvent(payload []byte) (*IssueCommentEvent, error) {
	var event IssueCommentEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, errors.Wrap(errors.ErrBadRequest, err).
			WithDetail("failed to parse issue_comment event payload")
	}
	return &event, nil
}

// PullRequestEvent represents a pull_request webhook event.
type PullRequestEvent struct {
	EventAction string `json:"action"`
	Number      int    `json:"number"`
	PR          struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		State     string `json:"state"`
		HTMLURL   string `json:"html_url"`
		Draft     bool   `json:"draft"`
		Merged    bool   `json:"merged"`
		Mergeable *bool  `json:"mergeable"`
		User      struct {
			Login string `json:"login"`
		} `json:"user"`
		Head struct {
			Ref  string `json:"ref"`
			SHA  string `json:"sha"`
			Repo struct {
				FullName string `json:"full_name"`
			} `json:"repo"`
		} `json:"head"`
		Base struct {
			Ref  string `json:"ref"`
			SHA  string `json:"sha"`
			Repo struct {
				FullName string `json:"full_name"`
			} `json:"repo"`
		} `json:"base"`
		CreatedAt time.Time  `json:"created_at"`
		UpdatedAt time.Time  `json:"updated_at"`
		MergedAt  *time.Time `json:"merged_at"`
	} `json:"pull_request"`
	Repo struct {
		FullName string `json:"full_name"`
		ID       int64  `json:"id"`
	} `json:"repository"`
	EventSender struct {
		Login string `json:"login"`
		ID    int64  `json:"id"`
	} `json:"sender"`
}

func (e *PullRequestEvent) EventType() string      { return EventTypePullRequest }
func (e *PullRequestEvent) Repository() string     { return e.Repo.FullName }
func (e *PullRequestEvent) Actor() string          { return e.EventSender.Login }
func (e *PullRequestEvent) Action() string         { return e.EventAction }
func (e *PullRequestEvent) PRNumber() int          { return e.PR.Number }
func (e *PullRequestEvent) PRTitle() string        { return e.PR.Title }
func (e *PullRequestEvent) PRBody() string         { return e.PR.Body }
func (e *PullRequestEvent) PRState() string        { return e.PR.State }
func (e *PullRequestEvent) PRURL() string          { return e.PR.HTMLURL }
func (e *PullRequestEvent) PRAuthor() string       { return e.PR.User.Login }
func (e *PullRequestEvent) PRHeadBranch() string   { return e.PR.Head.Ref }
func (e *PullRequestEvent) PRBaseBranch() string   { return e.PR.Base.Ref }
func (e *PullRequestEvent) PRHeadSHA() string      { return e.PR.Head.SHA }
func (e *PullRequestEvent) PRIsDraft() bool        { return e.PR.Draft }
func (e *PullRequestEvent) PRIsMerged() bool       { return e.PR.Merged }
func (e *PullRequestEvent) PRIsMergeable() *bool   { return e.PR.Mergeable }

func (p *EventParser) parsePullRequestEvent(payload []byte) (*PullRequestEvent, error) {
	var event PullRequestEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, errors.Wrap(errors.ErrBadRequest, err).
			WithDetail("failed to parse pull_request event payload")
	}
	return &event, nil
}

// PullRequestReviewEvent represents a pull_request_review webhook event.
type PullRequestReviewEvent struct {
	EventAction string `json:"action"`
	Review      struct {
		ID          int64      `json:"id"`
		State       string     `json:"state"`
		Body        string     `json:"body"`
		SubmittedAt *time.Time `json:"submitted_at"`
		User        struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"review"`
	PR struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		State   string `json:"state"`
		HTMLURL string `json:"html_url"`
		User    struct {
			Login string `json:"login"`
		} `json:"user"`
		Head struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"base"`
	} `json:"pull_request"`
	Repo struct {
		FullName string `json:"full_name"`
		ID       int64  `json:"id"`
	} `json:"repository"`
	EventSender struct {
		Login string `json:"login"`
		ID    int64  `json:"id"`
	} `json:"sender"`
}

func (e *PullRequestReviewEvent) EventType() string        { return EventTypePullRequestReview }
func (e *PullRequestReviewEvent) Repository() string       { return e.Repo.FullName }
func (e *PullRequestReviewEvent) Actor() string            { return e.EventSender.Login }
func (e *PullRequestReviewEvent) Action() string           { return e.EventAction }
func (e *PullRequestReviewEvent) PRNumber() int            { return e.PR.Number }
func (e *PullRequestReviewEvent) PRTitle() string          { return e.PR.Title }
func (e *PullRequestReviewEvent) ReviewID() int64          { return e.Review.ID }
func (e *PullRequestReviewEvent) ReviewState() string      { return e.Review.State }
func (e *PullRequestReviewEvent) ReviewBody() string       { return e.Review.Body }
func (e *PullRequestReviewEvent) Reviewer() string         { return e.Review.User.Login }
func (e *PullRequestReviewEvent) ReviewApproved() bool     { return e.Review.State == "approved" }
func (e *PullRequestReviewEvent) ChangesRequested() bool   { return e.Review.State == "changes_requested" }

func (p *EventParser) parsePullRequestReviewEvent(payload []byte) (*PullRequestReviewEvent, error) {
	var event PullRequestReviewEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, errors.Wrap(errors.ErrBadRequest, err).
			WithDetail("failed to parse pull_request_review event payload")
	}
	return &event, nil
}

// PushEvent represents a push webhook event.
type PushEvent struct {
	EventRef    string `json:"ref"`
	EventBefore string `json:"before"`
	EventAfter  string `json:"after"`
	EventCreated bool   `json:"created"`
	EventDeleted bool   `json:"deleted"`
	EventForced  bool   `json:"forced"`
	EventSender  struct {
		Login string `json:"login"`
		ID    int64  `json:"id"`
	} `json:"sender"`
	Repo struct {
		FullName string `json:"full_name"`
		ID       int64  `json:"id"`
	} `json:"repository"`
	Pusher struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"pusher"`
	Commits []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"commits"`
}

func (e *PushEvent) EventType() string    { return EventTypePush }
func (e *PushEvent) Repository() string   { return e.Repo.FullName }
func (e *PushEvent) Actor() string        { return e.EventSender.Login }
func (e *PushEvent) Action() string       { return "push" }
func (e *PushEvent) Ref() string          { return e.EventRef }
func (e *PushEvent) IsCreated() bool      { return e.EventCreated }
func (e *PushEvent) IsDeleted() bool      { return e.EventDeleted }
func (e *PushEvent) IsForced() bool       { return e.EventForced }
func (e *PushEvent) BeforeSHA() string    { return e.EventBefore }
func (e *PushEvent) AfterSHA() string     { return e.EventAfter }
func (e *PushEvent) CommitCount() int     { return len(e.Commits) }

func (p *EventParser) parsePushEvent(payload []byte) (*PushEvent, error) {
	var event PushEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, errors.Wrap(errors.ErrBadRequest, err).
			WithDetail("failed to parse push event payload")
	}
	return &event, nil
}

// IgnoredEvent represents an event we don't process.
type IgnoredEvent struct {
	eventType string
}

func (e *IgnoredEvent) EventType() string  { return e.eventType }
func (e *IgnoredEvent) Repository() string { return "" }
func (e *IgnoredEvent) Actor() string      { return "" }
func (e *IgnoredEvent) Action() string     { return "ignored" }

// IsIgnoredEvent checks if an event is ignored.
func IsIgnoredEvent(event WebhookEvent) bool {
	_, ok := event.(*IgnoredEvent)
	return ok
}