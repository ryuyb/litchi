package repositories

import (
	"context"
	"time"

	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/models"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WebhookDeliveryRepoParams contains dependencies for creating WebhookDeliveryRepository.
type WebhookDeliveryRepoParams struct {
	fx.In

	DB     *gorm.DB `name:"gorm_db"`
	Logger *zap.Logger
	Config *config.Config
}

// webhookDeliveryRepository implements repository.WebhookDeliveryRepository.
type webhookDeliveryRepository struct {
	db     *gorm.DB
	logger *zap.Logger
	ttl    time.Duration // TTL for delivery records
}

// NewWebhookDeliveryRepository creates a new WebhookDeliveryRepository.
func NewWebhookDeliveryRepository(p WebhookDeliveryRepoParams) repository.WebhookDeliveryRepository {
	// Parse TTL from config, default to 24 hours
	ttl, err := time.ParseDuration(p.Config.Webhook.Idempotency.TTL)
	if err != nil || ttl == 0 {
		ttl = 24 * time.Hour
	}

	return &webhookDeliveryRepository{
		db:     p.DB,
		logger: p.Logger.Named("webhook_delivery_repo"),
		ttl:    ttl,
	}
}

// TryAcquire atomically acquires processing rights for a delivery.
// Returns true if this handler should process the webhook (new delivery_id).
// Returns false if already processed or being processed by another handler.
func (r *webhookDeliveryRepository) TryAcquire(ctx context.Context, deliveryID, eventType, repository, payloadHash string) (bool, error) {
	// Set expiration time based on configured TTL
	expiresAt := time.Now().Add(r.ttl)

	// Insert a "processing" record. If delivery_id exists, ON CONFLICT DO NOTHING.
	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "delivery_id"}},
			DoNothing: true,
		}).
		Create(&models.WebhookDelivery{
			DeliveryID:    deliveryID,
			EventType:     eventType,
			Repository:    repository,
			PayloadHash:   payloadHash,
			Processed:     false,
			ProcessResult: entity.ProcessResultProcessing,
			ExpiresAt:     &expiresAt,
		})

	if result.Error != nil {
		r.logger.Error("failed to acquire webhook delivery",
			zap.Error(result.Error),
			zap.String("delivery_id", deliveryID),
		)
		return false, result.Error
	}

	// RowsAffected == 1 means we successfully acquired the right to process
	acquired := result.RowsAffected == 1
	if !acquired {
		r.logger.Debug("webhook delivery already exists (idempotent)",
			zap.String("delivery_id", deliveryID),
		)
	}

	return acquired, nil
}

// Create creates a new webhook delivery record.
// Returns error if delivery_id already exists (unique constraint).
func (r *webhookDeliveryRepository) Create(ctx context.Context, delivery *entity.WebhookDelivery) error {
	model := &models.WebhookDelivery{
		ID:             delivery.ID,
		DeliveryID:     delivery.DeliveryID,
		EventType:      delivery.EventType,
		Repository:     delivery.Repository,
		PayloadHash:    delivery.PayloadHash,
		Processed:      delivery.Processed,
		ProcessResult:  delivery.ProcessResult,
		ProcessMessage: delivery.ProcessMessage,
		ExpiresAt:      &delivery.ExpiresAt,
	}

	// Use OnConflict DoNothing to silently ignore duplicate delivery_id
	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "delivery_id"}},
			DoNothing: true,
		}).
		Create(model)

	if result.Error != nil {
		r.logger.Error("failed to create webhook delivery",
			zap.Error(result.Error),
			zap.String("delivery_id", delivery.DeliveryID),
		)
		return result.Error
	}

	// Check if the record was actually inserted
	if result.RowsAffected == 0 {
		r.logger.Debug("webhook delivery already exists (idempotent)",
			zap.String("delivery_id", delivery.DeliveryID),
		)
	}

	return nil
}

// FindByDeliveryID finds a delivery by its GitHub delivery ID.
// Returns nil if not found.
func (r *webhookDeliveryRepository) FindByDeliveryID(ctx context.Context, deliveryID string) (*entity.WebhookDelivery, error) {
	var model models.WebhookDelivery
	err := r.db.WithContext(ctx).
		Where("delivery_id = ?", deliveryID).
		First(&model).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		r.logger.Error("failed to find webhook delivery",
			zap.Error(err),
			zap.String("delivery_id", deliveryID),
		)
		return nil, err
	}

	return r.toEntity(&model), nil
}

// UpdateResult updates the processing result of a delivery.
func (r *webhookDeliveryRepository) UpdateResult(ctx context.Context, deliveryID string, result, message string) error {
	update := map[string]any{
		"process_result":  result,
		"process_message": message,
		"processed":       true,
	}

	dbResult := r.db.WithContext(ctx).
		Model(&models.WebhookDelivery{}).
		Where("delivery_id = ?", deliveryID).
		Updates(update)

	if dbResult.Error != nil {
		r.logger.Error("failed to update webhook delivery result",
			zap.Error(dbResult.Error),
			zap.String("delivery_id", deliveryID),
		)
		return dbResult.Error
	}

	if dbResult.RowsAffected == 0 {
		r.logger.Warn("no record found to update",
			zap.String("delivery_id", deliveryID),
		)
	}

	return nil
}

// DeleteExpired deletes all expired delivery records.
// Returns the number of deleted records.
func (r *webhookDeliveryRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
		Delete(&models.WebhookDelivery{})

	if result.Error != nil {
		r.logger.Error("failed to delete expired webhook deliveries",
			zap.Error(result.Error),
		)
		return 0, result.Error
	}

	if result.RowsAffected > 0 {
		r.logger.Info("expired webhook deliveries deleted",
			zap.Int64("count", result.RowsAffected),
		)
	}

	return result.RowsAffected, nil
}

// Count counts total delivery records (for monitoring).
func (r *webhookDeliveryRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.WebhookDelivery{}).
		Count(&count).Error

	if err != nil {
		r.logger.Error("failed to count webhook deliveries",
			zap.Error(err),
		)
		return 0, err
	}

	return count, nil
}

// toEntity converts models.WebhookDelivery to entity.WebhookDelivery.
func (r *webhookDeliveryRepository) toEntity(m *models.WebhookDelivery) *entity.WebhookDelivery {
	if m == nil {
		return nil
	}

	e := &entity.WebhookDelivery{
		ID:             m.ID,
		DeliveryID:     m.DeliveryID,
		EventType:      m.EventType,
		Repository:     m.Repository,
		PayloadHash:    m.PayloadHash,
		Processed:      m.Processed,
		ProcessResult:  m.ProcessResult,
		ProcessMessage: m.ProcessMessage,
		CreatedAt:      m.CreatedAt,
	}

	if m.ExpiresAt != nil {
		e.ExpiresAt = *m.ExpiresAt
	}

	return e
}

// CleanupService handles periodic cleanup of expired webhook deliveries.
type CleanupService struct {
	repo     repository.WebhookDeliveryRepository
	logger   *zap.Logger
	interval time.Duration
	stopChan chan struct{}
}

// NewCleanupService creates a new cleanup service.
func NewCleanupService(
	repo repository.WebhookDeliveryRepository,
	logger *zap.Logger,
	interval time.Duration,
) *CleanupService {
	if interval == 0 {
		interval = time.Hour
	}
	return &CleanupService{
		repo:     repo,
		logger:   logger.Named("webhook_cleanup"),
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start starts the cleanup loop.
func (s *CleanupService) Start(ctx context.Context) error {
	go s.run(ctx)
	s.logger.Info("webhook cleanup service started",
		zap.Duration("interval", s.interval),
	)
	return nil
}

// Stop stops the cleanup loop.
func (s *CleanupService) Stop(ctx context.Context) error {
	close(s.stopChan)
	s.logger.Info("webhook cleanup service stopped")
	return nil
}

func (s *CleanupService) run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.cleanup(ctx)
		}
	}
}

func (s *CleanupService) cleanup(ctx context.Context) {
	count, err := s.repo.DeleteExpired(ctx)
	if err != nil {
		s.logger.Error("failed to delete expired webhook deliveries",
			zap.Error(err),
		)
		return
	}
	if count > 0 {
		s.logger.Info("cleanup completed",
			zap.Int64("deleted_count", count),
		)
	}
}