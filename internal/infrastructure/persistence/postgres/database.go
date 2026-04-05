// Package postgres provides database connection and migration functionality.
// It is integrated as an Fx module for dependency injection.
package postgres

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ryuyb/litchi/internal/infrastructure/config"
)

// DatabaseModule provides GORM database connection via Fx.
var DatabaseModule = fx.Module("database",
	fx.Provide(NewDB),
	fx.Invoke(RegisterLifecycle),
)

// DB wraps gorm.DB with additional functionality.
type DB struct {
	*gorm.DB
	logger *zap.Logger
	config *config.DatabaseConfig
}

// Params holds the dependencies for creating a DB instance.
type Params struct {
	fx.In

	DatabaseConfig *config.DatabaseConfig
	ServerConfig   *config.ServerConfig // for mode (debug/release/test)
	Logger         *zap.Logger
}

// NewDB creates a new GORM database instance with connection pool configuration.
func NewDB(p Params) (*DB, error) {
	dbConfig := p.DatabaseConfig

	// Build connection string with proper URL encoding for special characters in password
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		url.PathEscape(dbConfig.User),
		url.PathEscape(dbConfig.Password),
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.Name,
		dbConfig.SSLMode,
	)

	// Configure GORM logger based on application mode
	gormLogger := NewGormLogger(p.Logger, p.ServerConfig.Mode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB for connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)
	sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)

	// Parse connection max lifetime from config
	connMaxLifetime, err := time.ParseDuration(dbConfig.ConnMaxLifetime)
	if err != nil {
		p.Logger.Warn("invalid conn_max_lifetime, using default 1h", zap.Error(err))
		connMaxLifetime = time.Hour
	}
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	// Parse connection max idle time from config
	connMaxIdleTime, err := time.ParseDuration(dbConfig.ConnMaxIdleTime)
	if err != nil {
		p.Logger.Warn("invalid conn_max_idle_time, using default 10m", zap.Error(err))
		connMaxIdleTime = 10 * time.Minute
	}
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	p.Logger.Info("database connection established",
		zap.String("host", dbConfig.Host),
		zap.Int("port", dbConfig.Port),
		zap.String("database", dbConfig.Name),
		zap.Int("max_open_conns", dbConfig.MaxOpenConns),
		zap.Int("max_idle_conns", dbConfig.MaxIdleConns),
		zap.Duration("conn_max_lifetime", connMaxLifetime),
		zap.Duration("conn_max_idle_time", connMaxIdleTime),
	)

	return &DB{
		DB:     db,
		logger: p.Logger,
		config: dbConfig,
	}, nil
}

// RegisterLifecycle registers database lifecycle hooks with Fx.
func RegisterLifecycle(lc fx.Lifecycle, db *DB) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Verify connection is working
			sqlDB, err := db.DB.DB()
			if err != nil {
				return fmt.Errorf("failed to get sql.DB: %w", err)
			}
			if err := sqlDB.PingContext(ctx); err != nil {
				return fmt.Errorf("failed to ping database: %w", err)
			}
			db.logger.Info("database connection verified")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			db.logger.Info("closing database connection")
			sqlDB, err := db.DB.DB()
			if err != nil {
				db.logger.Error("failed to get sql.DB for closing", zap.Error(err))
				return err
			}
			// Use channel to handle context cancellation during close
			done := make(chan error, 1)
			go func() { done <- sqlDB.Close() }()
			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				db.logger.Warn("database close timeout, context cancelled")
				return ctx.Err()
			}
		},
	})
}

// Transaction executes a function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func (db *DB) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(fn)
}

// Ping checks if the database connection is still alive.
func (db *DB) Ping(ctx context.Context) error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

// Stats returns database connection pool statistics.
func (db *DB) Stats() (map[string]interface{}, error) {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}
	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration_ms":     stats.WaitDuration.Milliseconds(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}, nil
}

// ============================================
// GORM Logger Integration with Zap
// ============================================

// GormLogger wraps zap.Logger for GORM logging.
type GormLogger struct {
	zapLogger     *zap.Logger
	logLevel      logger.LogLevel
	slowThreshold time.Duration
}

// NewGormLogger creates a new GORM logger integrated with Zap.
func NewGormLogger(zapLogger *zap.Logger, mode string) *GormLogger {
	logLevel := logger.Info
	if mode == "release" {
		logLevel = logger.Error
	} else if mode == "test" {
		logLevel = logger.Silent
	}

	return &GormLogger{
		zapLogger:     zapLogger,
		logLevel:      logLevel,
		slowThreshold: 200 * time.Millisecond,
	}
}

// LogMode sets the logger level.
func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.logLevel = level
	return &newLogger
}

// Info logs info messages.
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Info {
		l.zapLogger.Sugar().Infof(msg, data...)
	}
}

// Warn logs warn messages.
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Warn {
		l.zapLogger.Sugar().Warnf(msg, data...)
	}
}

// Error logs error messages.
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Error {
		l.zapLogger.Sugar().Errorf(msg, data...)
	}
}

// Trace logs trace messages (SQL queries).
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.logLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	if err != nil {
		l.zapLogger.Error("sql error",
			zap.Error(err),
			zap.Duration("duration", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
		return
	}

	if elapsed > l.slowThreshold && l.slowThreshold != 0 {
		l.zapLogger.Warn("slow sql",
			zap.Duration("duration", elapsed),
			zap.Duration("threshold", l.slowThreshold),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
		return
	}

	l.zapLogger.Debug("sql",
		zap.Duration("duration", elapsed),
		zap.Int64("rows", rows),
		zap.String("sql", sql),
	)
}
