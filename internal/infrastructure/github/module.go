// Package github provides GitHub API integration with Fx module support.
package github

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/infrastructure/github/webhook"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/postgres/repositories"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides GitHub integration via Fx.
var Module = fx.Module("github",
	// Providers
	fx.Provide(
		// GitHub client and services
		NewClient,
		NewIssueService,
		NewPullRequestService,

		// Webhook components
		NewSignatureVerifier,
		webhook.NewEventParser,
		webhook.NewEventDispatcher,
		NewWebhookHandler,

		// Cleanup service
		NewWebhookCleanupService,
	),

	// Invokes
	fx.Invoke(
		RegisterWebhookRoutes,
		StartWebhookCleanup,
	),
)

// SignatureVerifierParams contains dependencies for creating a SignatureVerifier.
type SignatureVerifierParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

// NewSignatureVerifier creates a new signature verifier.
func NewSignatureVerifier(p SignatureVerifierParams) *webhook.SignatureVerifier {
	return webhook.NewSignatureVerifier(p.Config.GitHub.WebhookSecret, p.Logger)
}

// WebhookHandlerParams contains dependencies for creating a WebhookHandler.
type WebhookHandlerParams struct {
	fx.In

	Verifier   *webhook.SignatureVerifier
	Parser     *webhook.EventParser
	Dispatcher *webhook.EventDispatcher
	DedupRepo  repository.WebhookDeliveryRepository
	Logger     *zap.Logger
	Config     *config.Config
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(p WebhookHandlerParams) *webhook.Handler {
	return webhook.NewHandler(webhook.HandlerParams{
		Verifier:   p.Verifier,
		Parser:     p.Parser,
		Dispatcher: p.Dispatcher,
		DedupRepo:  p.DedupRepo,
		Logger:     p.Logger,
		Config:     &p.Config.Webhook,
	})
}

// WebhookCleanupParams contains dependencies for creating a cleanup service.
type WebhookCleanupParams struct {
	fx.In

	Repo   repository.WebhookDeliveryRepository
	Logger *zap.Logger
	Config *config.Config
}

// NewWebhookCleanupService creates a new webhook cleanup service.
func NewWebhookCleanupService(p WebhookCleanupParams) *repositories.CleanupService {
	interval, err := time.ParseDuration(p.Config.Webhook.Idempotency.CleanupInterval)
	if err != nil || interval == 0 {
		interval = time.Hour
	}

	return repositories.NewCleanupService(p.Repo, p.Logger, interval)
}

// RegisterWebhookRoutes registers webhook routes on the Fiber app.
func RegisterWebhookRoutes(app *fiber.App, handler *webhook.Handler) {
	webhook.RegisterRoutes(app, handler)
}

// StartWebhookCleanup starts the cleanup service lifecycle.
func StartWebhookCleanup(lc fx.Lifecycle, cleanupSvc *repositories.CleanupService, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return cleanupSvc.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return cleanupSvc.Stop(ctx)
		},
	})
}