package webhook

import (
	"context"

	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"go.uber.org/zap"
)

// InstallationHandler handles GitHub App installation events.
// It automatically updates repository records with installation IDs when
// the app is installed, uninstalled, or when repositories are added/removed.
type InstallationHandler struct {
	repoRepo repository.RepositoryRepository
	logger   *zap.Logger
}

// NewInstallationHandler creates a new installation event handler.
func NewInstallationHandler(repoRepo repository.RepositoryRepository, logger *zap.Logger) *InstallationHandler {
	return &InstallationHandler{
		repoRepo: repoRepo,
		logger:   logger.Named("webhook.installation"),
	}
}

// Handle processes installation events.
func (h *InstallationHandler) Handle(ctx context.Context, event WebhookEvent) error {
	switch e := event.(type) {
	case *InstallationEvent:
		return h.handleInstallationEvent(ctx, e)
	case *InstallationRepositoriesEvent:
		return h.handleInstallationRepositoriesEvent(ctx, e)
	default:
		h.logger.Warn("unexpected event type for installation handler",
			zap.String("event_type", event.EventType()),
		)
		return nil
	}
}

// handleInstallationEvent handles the main installation event.
func (h *InstallationHandler) handleInstallationEvent(ctx context.Context, event *InstallationEvent) error {
	h.logger.Info("processing installation event",
		zap.String("action", event.Action()),
		zap.Int64("installation_id", event.InstallationID()),
		zap.String("account", event.AccountLogin()),
	)

	switch event.Action() {
	case "created":
		return h.handleInstallationCreated(ctx, event)
	case "deleted":
		return h.handleInstallationDeleted(ctx, event)
	case "suspend":
		return h.handleInstallationSuspend(ctx, event)
	case "unsuspend":
		return h.handleInstallationUnsuspend(ctx, event)
	case "new_permissions_accepted":
		h.logger.Info("new permissions accepted for installation",
			zap.Int64("installation_id", event.InstallationID()),
		)
		return nil
	default:
		h.logger.Debug("unhandled installation action",
			zap.String("action", event.Action()),
		)
		return nil
	}
}

// handleInstallationCreated handles when the app is installed.
func (h *InstallationHandler) handleInstallationCreated(ctx context.Context, event *InstallationEvent) error {
	installationID := event.InstallationID()
	repoNames := event.GetRepositoryNames()

	if len(repoNames) == 0 {
		h.logger.Info("installation created with no repositories (all repositories access)",
			zap.Int64("installation_id", installationID),
			zap.String("account", event.AccountLogin()),
		)
		return nil
	}

	// Update or create repository records for each installed repository
	for _, repoName := range repoNames {
		if err := h.updateRepositoryInstallation(ctx, repoName, installationID, true); err != nil {
			h.logger.Error("failed to update repository after installation",
				zap.String("repo", repoName),
				zap.Error(err),
			)
			// Continue with other repositories
		}
	}

	h.logger.Info("installation created",
		zap.Int64("installation_id", installationID),
		zap.Strings("repositories", repoNames),
	)

	return nil
}

// handleInstallationDeleted handles when the app is uninstalled.
func (h *InstallationHandler) handleInstallationDeleted(ctx context.Context, event *InstallationEvent) error {
	installationID := event.InstallationID()
	repoNames := event.GetRepositoryNames()

	// Clear installation ID from affected repositories
	for _, repoName := range repoNames {
		if err := h.updateRepositoryInstallation(ctx, repoName, 0, false); err != nil {
			h.logger.Error("failed to clear repository installation",
				zap.String("repo", repoName),
				zap.Error(err),
			)
		}
	}

	h.logger.Info("installation deleted",
		zap.Int64("installation_id", installationID),
		zap.Strings("repositories", repoNames),
	)

	return nil
}

// handleInstallationSuspend handles when the app is suspended.
func (h *InstallationHandler) handleInstallationSuspend(ctx context.Context, event *InstallationEvent) error {
	installationID := event.InstallationID()
	repoNames := event.GetRepositoryNames()

	// Disable repositories when app is suspended
	for _, repoName := range repoNames {
		if err := h.updateRepositoryInstallation(ctx, repoName, installationID, false); err != nil {
			h.logger.Error("failed to disable repository on suspend",
				zap.String("repo", repoName),
				zap.Error(err),
			)
		}
	}

	h.logger.Info("installation suspended",
		zap.Int64("installation_id", installationID),
		zap.Strings("repositories", repoNames),
	)

	return nil
}

// handleInstallationUnsuspend handles when the app is unsuspended.
func (h *InstallationHandler) handleInstallationUnsuspend(ctx context.Context, event *InstallationEvent) error {
	installationID := event.InstallationID()
	repoNames := event.GetRepositoryNames()

	// Re-enable repositories when app is unsuspended
	for _, repoName := range repoNames {
		if err := h.updateRepositoryInstallation(ctx, repoName, installationID, true); err != nil {
			h.logger.Error("failed to enable repository on unsuspend",
				zap.String("repo", repoName),
				zap.Error(err),
			)
		}
	}

	h.logger.Info("installation unsuspended",
		zap.Int64("installation_id", installationID),
		zap.Strings("repositories", repoNames),
	)

	return nil
}

// handleInstallationRepositoriesEvent handles when repositories are added/removed from an installation.
func (h *InstallationHandler) handleInstallationRepositoriesEvent(ctx context.Context, event *InstallationRepositoriesEvent) error {
	installationID := event.InstallationID()

	h.logger.Info("processing installation_repositories event",
		zap.String("action", event.Action()),
		zap.Int64("installation_id", installationID),
		zap.Int("added", len(event.RepositoriesAdded)),
		zap.Int("removed", len(event.RepositoriesRemoved)),
	)

	// Handle added repositories
	for _, repoName := range event.GetAddedRepositoryNames() {
		if err := h.updateRepositoryInstallation(ctx, repoName, installationID, true); err != nil {
			h.logger.Error("failed to update repository on add",
				zap.String("repo", repoName),
				zap.Error(err),
			)
		}
	}

	// Handle removed repositories
	for _, repoName := range event.GetRemovedRepositoryNames() {
		if err := h.updateRepositoryInstallation(ctx, repoName, 0, false); err != nil {
			h.logger.Error("failed to update repository on remove",
				zap.String("repo", repoName),
				zap.Error(err),
			)
		}
	}

	return nil
}

// updateRepositoryInstallation updates a repository's installation ID and enabled status.
func (h *InstallationHandler) updateRepositoryInstallation(ctx context.Context, repoName string, installationID int64, enabled bool) error {
	repo, err := h.repoRepo.FindByName(ctx, repoName)
	if err != nil {
		return err
	}

	if repo == nil {
		// Create new repository record
		repo = entity.NewRepository(repoName)
	}

	repo.SetInstallationID(installationID)
	if enabled {
		repo.Enable()
	} else {
		repo.Disable()
	}

	return h.repoRepo.Save(ctx, repo)
}