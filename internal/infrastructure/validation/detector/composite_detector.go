package detector

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// CompositeDetector combines multiple language detectors.
type CompositeDetector struct {
	mu        sync.RWMutex
	detectors []service.ProjectDetector
	logger    *zap.Logger
}

// CompositeDetectorParams contains dependencies for CompositeDetector.
type CompositeDetectorParams struct {
	fx.In

	Detectors []service.ProjectDetector `group:"detectors"`
	Logger    *zap.Logger
}

// NewCompositeDetector creates a new composite detector.
func NewCompositeDetector(p CompositeDetectorParams) service.CompositeProjectDetector {
	// Sort detectors by priority (higher priority first)
	detectors := p.Detectors
	sort.Slice(detectors, func(i, j int) bool {
		return detectors[i].Priority() > detectors[j].Priority()
	})

	logger := p.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &CompositeDetector{
		detectors: detectors,
		logger:    logger.Named("composite-detector"),
	}
}

// DetectWithAll executes all detectors in priority order.
func (d *CompositeDetector) DetectWithAll(ctx context.Context, worktreePath string) (*valueobject.DetectedProject, error) {
	d.logger.Debug("starting project detection", zap.String("path", worktreePath))

	d.mu.RLock()
	detectors := d.detectors
	d.mu.RUnlock()

	for _, detector := range detectors {
		project, err := detector.Detect(ctx, worktreePath)
		if err != nil {
			d.logger.Warn("detector failed",
				zap.String("detector", detectorName(detector)),
				zap.Error(err),
			)
			continue
		}
		if project != nil {
			d.logger.Info("project detected",
				zap.String("type", string(project.Type)),
				zap.String("language", project.PrimaryLanguage),
				zap.Int("confidence", project.Confidence),
				zap.Int("tools", len(project.DetectedTools)),
			)
			return project, nil
		}
	}

	// No project detected
	d.logger.Warn("no project detected", zap.String("path", worktreePath))
	return nil, fmt.Errorf("no project type detected at path: %s", worktreePath)
}

// DetectByLanguage executes only the detector for the specified language.
func (d *CompositeDetector) DetectByLanguage(ctx context.Context, worktreePath string, language string) (*valueobject.DetectedProject, error) {
	language = strings.ToLower(language)

	d.mu.RLock()
	detectors := d.detectors
	d.mu.RUnlock()

	for _, detector := range detectors {
		if detector.SupportsLanguage(language) {
			return detector.Detect(ctx, worktreePath)
		}
	}

	d.logger.Warn("no detector for language",
		zap.String("language", language),
		zap.String("path", worktreePath),
	)
	return nil, nil
}

// RegisterDetector registers a language detector.
func (d *CompositeDetector) RegisterDetector(detector service.ProjectDetector) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.detectors = append(d.detectors, detector)
	// Re-sort by priority
	sort.Slice(d.detectors, func(i, j int) bool {
		return d.detectors[i].Priority() > d.detectors[j].Priority()
	})
}

// GetDetectors returns all registered detectors sorted by priority.
func (d *CompositeDetector) GetDetectors() []service.ProjectDetector {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.detectors
}

// GetSupportedLanguages returns all supported language names.
func (d *CompositeDetector) GetSupportedLanguages() []string {
	d.mu.RLock()
	detectors := d.detectors
	d.mu.RUnlock()

	languages := []string{}
	for _, detector := range detectors {
		// We need to know what languages each detector supports
		// For now, use detector name as language hint
		name := detectorName(detector)
		if name != "" {
			languages = append(languages, name)
		}
	}
	return languages
}

// detectorName extracts a name from detector for logging.
func detectorName(detector service.ProjectDetector) string {
	// Try to get a meaningful name based on SupportsLanguage
	// Common languages to check
	commonLanguages := []string{"go", "nodejs", "node", "python", "rust", "java"}
	for _, lang := range commonLanguages {
		if detector.SupportsLanguage(lang) {
			return lang
		}
	}
	return "unknown"
}