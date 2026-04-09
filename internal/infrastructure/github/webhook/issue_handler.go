package webhook

import (
	"context"
	"strings"
	"time"

	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"go.uber.org/zap"
)

// BotCommandPrefix is the prefix for bot commands in issue comments.
const BotCommandPrefix = "@litchi "

// SupportedIssueActions defines which issue actions trigger session creation.
var SupportedIssueActions = map[string]bool{
	"opened":   true,
	"reopened": true,
}

// IssueProcessor defines the interface for processing issue events.
// This interface is satisfied by service.IssueService, decoupling webhook from the application layer.
type IssueProcessor interface {
	ProcessIssueEvent(
		ctx context.Context,
		repoName string,
		issueNumber int,
		issueTitle string,
		issueBody string,
		author string,
		labels []string,
		issueURL string,
		createdAt time.Time,
	) (session *aggregate.WorkSession, isNew bool, err error)

	ProcessIssueCommandEvent(
		ctx context.Context,
		repoName string,
		issueNumber int,
		actor string,
		command string,
	) (session *aggregate.WorkSession, err error)
}

// IssueHandler handles issues and issue_comment webhook events.
// It connects GitHub webhook events to the application layer IssueService.
type IssueHandler struct {
	issueService IssueProcessor
	logger       *zap.Logger
}

// NewIssueHandler creates a new IssueHandler.
func NewIssueHandler(issueService IssueProcessor, logger *zap.Logger) *IssueHandler {
	return &IssueHandler{
		issueService: issueService,
		logger:       logger.Named("webhook.issue"),
	}
}

// Handle processes issues and issue_comment webhook events.
func (h *IssueHandler) Handle(ctx context.Context, event WebhookEvent) error {
	switch e := event.(type) {
	case *IssueEvent:
		return h.handleIssuesEvent(ctx, e)
	case *IssueCommentEvent:
		return h.handleIssueCommentEvent(ctx, e)
	default:
		h.logger.Warn("unexpected event type for issue handler",
			zap.String("event_type", event.EventType()),
		)
		return nil
	}
}

// handleIssuesEvent processes issues events (opened/reopened).
func (h *IssueHandler) handleIssuesEvent(ctx context.Context, event *IssueEvent) error {
	if !SupportedIssueActions[event.Action()] {
		h.logger.Debug("ignoring issue action",
			zap.String("action", event.Action()),
			zap.String("repository", event.Repository()),
			zap.Int("issue_number", event.IssueNumber()),
		)
		return nil
	}

	h.logger.Info("processing issue event",
		zap.String("action", event.Action()),
		zap.String("repository", event.Repository()),
		zap.Int("issue_number", event.IssueNumber()),
		zap.String("author", event.IssueAuthor()),
	)

	session, isNew, err := h.issueService.ProcessIssueEvent(
		ctx,
		event.Repository(),
		event.IssueNumber(),
		event.IssueTitle(),
		event.IssueBody(),
		event.IssueAuthor(),
		event.IssueLabels(),
		event.IssueURL(),
		event.Issue.CreatedAt,
	)
	if err != nil {
		h.logger.Error("failed to process issue event",
			zap.String("repository", event.Repository()),
			zap.Int("issue_number", event.IssueNumber()),
			zap.Error(err),
		)
		return err
	}

	h.logger.Info("issue event processed",
		zap.String("session_id", session.ID.String()),
		zap.Bool("is_new", isNew),
		zap.String("repository", event.Repository()),
		zap.Int("issue_number", event.IssueNumber()),
	)

	return nil
}

// handleIssueCommentEvent processes issue_comment events.
// It parses bot commands from comment body and delegates to IssueService.
func (h *IssueHandler) handleIssueCommentEvent(ctx context.Context, event *IssueCommentEvent) error {
	if event.Action() != "created" {
		return nil
	}

	command := parseCommand(event.CommentBody())
	if command == "" {
		return nil
	}

	h.logger.Info("processing issue comment command",
		zap.String("repository", event.Repository()),
		zap.Int("issue_number", event.IssueNumber()),
		zap.String("actor", event.Actor()),
		zap.String("command", command),
	)

	session, err := h.issueService.ProcessIssueCommandEvent(
		ctx,
		event.Repository(),
		event.IssueNumber(),
		event.Actor(),
		command,
	)
	if err != nil {
		h.logger.Error("failed to process issue command",
			zap.String("repository", event.Repository()),
			zap.Int("issue_number", event.IssueNumber()),
			zap.String("command", command),
			zap.Error(err),
		)
		return err
	}

	h.logger.Info("issue command processed",
		zap.String("session_id", session.ID.String()),
		zap.String("repository", event.Repository()),
		zap.Int("issue_number", event.IssueNumber()),
		zap.String("command", command),
	)

	return nil
}

// parseCommand extracts a bot command from a comment body.
// Returns empty string if the comment is not a command.
// Command format: "@litchi <command>" (case-insensitive prefix).
func parseCommand(body string) string {
	if body == "" {
		return ""
	}

	// Check case-insensitive prefix match
	lowerBody := strings.ToLower(body)
	if !strings.HasPrefix(lowerBody, BotCommandPrefix) {
		return ""
	}

	// Extract command after prefix
	command := strings.TrimSpace(body[len(BotCommandPrefix):])
	if command == "" {
		return ""
	}

	return command
}
