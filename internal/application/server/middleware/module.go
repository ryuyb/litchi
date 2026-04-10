// Package middleware provides HTTP middleware for the server.
package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/utils/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ryuyb/litchi/internal/application/server/middleware/auth"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
)

// Module provides middleware for Fx.
var Module = fx.Module("middleware",
	fx.Provide(NewErrorHandler),
	fx.Invoke(RegisterMiddlewares),
	// Include auth module
	fx.Options(auth.Module),
)

// MiddlewareParams contains dependencies for middleware registration.
type MiddlewareParams struct {
	fx.In

	App    *fiber.App
	Logger *zap.Logger
	Config *config.Config
}

// RegisterMiddlewares registers all middlewares on the Fiber app.
// Middlewares are registered in order: recover -> requestid -> logger -> cors -> helmet -> compress -> limiter -> csrf
func RegisterMiddlewares(p MiddlewareParams) {
	p.Logger.Info("registering core middlewares")

	cfg := p.Config.Middleware

	// 1. Recover middleware (must be first to catch panics)
	if cfg.Recover.Enabled {
		p.App.Use(createRecoverMiddleware(p.Logger, cfg.Recover))
		p.Logger.Debug("middleware registered: recover")
	}

	// 2. Request ID middleware (must be before logger for request ID in logs)
	if cfg.RequestID.Enabled {
		p.App.Use(createRequestIDMiddleware(cfg.RequestID))
		p.Logger.Debug("middleware registered: requestid")
	}

	// 3. Logger middleware (custom zap-based logging)
	p.App.Use(createLoggerMiddleware(p.Logger))
	p.Logger.Debug("middleware registered: logger")

	// 4. CORS middleware
	if cfg.CORS.Enabled {
		p.App.Use(createCORSMiddleware(cfg.CORS))
		p.Logger.Debug("middleware registered: cors")
	}

	// 5. Helmet middleware (security headers, always enabled)
	p.App.Use(createHelmetMiddleware())
	p.Logger.Debug("middleware registered: helmet")

	// 6. Compress middleware
	if cfg.Compress.Enabled {
		p.App.Use(createCompressMiddleware(cfg.Compress))
		p.Logger.Debug("middleware registered: compress")
	}

	// 7. Rate limiter middleware
	if cfg.Limiter.Enabled {
		p.App.Use(createLimiterMiddleware(p.Logger, cfg.Limiter))
		p.Logger.Debug("middleware registered: limiter")
	}

	// 8. CSRF middleware (protects against Cross-Site Request Forgery)
	if cfg.CSRF.Enabled {
		p.App.Use(createCSRFMiddleware(p.Config, cfg.CSRF, p.Logger))
		p.Logger.Debug("middleware registered: csrf")
	}

	p.Logger.Info("core middlewares registered successfully")
}

// createRecoverMiddleware creates recover middleware.
// Note: Fiber v3 recover middleware catches panics and forwards to ErrorHandler.
// We use StackTraceHandler to log panic details with stack trace.
func createRecoverMiddleware(logger *zap.Logger, cfg config.RecoverMiddlewareConfig) fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: cfg.EnableStackTrace,
		StackTraceHandler: func(c fiber.Ctx, err any) {
			logger.Error("panic recovered",
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
				zap.String("ip", c.IP()),
				zap.Any("error", err),
			)
		},
	})
}

// createRequestIDMiddleware creates request ID middleware.
func createRequestIDMiddleware(cfg config.RequestIDMiddlewareConfig) fiber.Handler {
	header := cfg.Header
	if header == "" {
		header = "X-Request-ID"
	}
	return requestid.New(requestid.Config{
		Header:    header,
		Generator: utils.SecureToken,
	})
}

// createLoggerMiddleware creates a custom logger middleware using zap.
func createLoggerMiddleware(logger *zap.Logger) fiber.Handler {
	httpLogger := logger.Named("http")

	return func(c fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		latency := time.Since(start)
		reqID := requestid.FromContext(c)
		status := c.Response().StatusCode()
		method := c.Method()
		path := c.Path()
		ip := c.IP()

		// If handler returned an error, the global ErrorHandler hasn't run yet,
		// so c.Response().StatusCode() is still 200 (default). Derive the actual
		// status code from the error itself.
		if err != nil {
			apiErr := litchierrors.ToAPIError(err)
			if apiErr.Code > 0 {
				status = apiErr.Code
			}
		}

		// Determine log level based on status code
		var logLevel zapcore.Level
		switch {
		case status >= 500:
			logLevel = zapcore.ErrorLevel
		case status >= 400:
			logLevel = zapcore.WarnLevel
		default:
			logLevel = zapcore.InfoLevel
		}

		fields := []zap.Field{
			zap.String("request_id", reqID),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("ip", ip),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
		}

		httpLogger.Log(logLevel, "request", fields...)

		return err
	}
}

// createCORSMiddleware creates CORS middleware.
func createCORSMiddleware(cfg config.CORSMiddlewareConfig) fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     cfg.AllowOrigins,
		AllowMethods:     cfg.AllowMethods,
		AllowHeaders:     cfg.AllowHeaders,
		AllowCredentials: cfg.AllowCredentials,
		ExposeHeaders:    cfg.ExposeHeaders,
		MaxAge:           cfg.MaxAge,
	})
}

// createHelmetMiddleware creates helmet middleware with default security headers.
func createHelmetMiddleware() fiber.Handler {
	return helmet.New(helmet.Config{
		// XSSProtection: "0" disables the deprecated XSS Auditor.
		// Modern browsers (Chrome 83+) have removed XSS Auditor support.
		// Setting "0" prevents issues with legacy browser implementations.
		XSSProtection:             "0",
		ContentTypeNosniff:        "nosniff",
		XFrameOptions:             "SAMEORIGIN",
		ReferrerPolicy:            "no-referrer",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		OriginAgentCluster:        "?1",
		XDNSPrefetchControl:       "off",
		XDownloadOptions:          "noopen",
		XPermittedCrossDomain:     "none",
	})
}

// createCompressMiddleware creates compress middleware.
func createCompressMiddleware(cfg config.CompressMiddlewareConfig) fiber.Handler {
	level := compress.Level(cfg.Level)
	// Validate level and use default if invalid
	if level < compress.LevelDisabled || level > compress.LevelBestCompression {
		level = compress.LevelDefault
	}

	return compress.New(compress.Config{
		Level: level,
	})
}

// createLimiterMiddleware creates rate limiter middleware.
func createLimiterMiddleware(logger *zap.Logger, cfg config.LimiterMiddlewareConfig) fiber.Handler {
	// Parse expiration duration
	expiration := 1 * time.Minute
	if cfg.Expiration != "" {
		if dur, err := time.ParseDuration(cfg.Expiration); err == nil {
			expiration = dur
		} else {
			logger.Warn("invalid limiter expiration, using default",
				zap.String("configured", cfg.Expiration),
				zap.Error(err),
				zap.Duration("default", expiration),
			)
		}
	}

	maxRequests := cfg.Max
	if maxRequests <= 0 {
		maxRequests = 100
	}

	return limiter.New(limiter.Config{
		Max:        maxRequests,
		Expiration: expiration,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP()
		},
		Next: func(c fiber.Ctx) bool {
			// Skip rate limiting for health check and public paths
			path := c.Path()
			return path == "/health" ||
				path == "/healthz" ||
				strings.HasPrefix(path, "/swagger")
		},
		LimitReached: func(c fiber.Ctx) error {
			logger.Warn("rate limit exceeded",
				zap.String("ip", c.IP()),
				zap.String("path", c.Path()),
			)
			return litchierrors.New(litchierrors.ErrRateLimitExceeded)
		},
	})
}

// createCSRFMiddleware creates CSRF protection middleware.
// It protects against Cross-Site Request Forgery attacks.
func createCSRFMiddleware(appCfg *config.Config, cfg config.CSRFMiddlewareConfig, logger *zap.Logger) fiber.Handler {
	// Default excluded paths (login, webhooks, etc.)
	defaultExcluded := []string{
		"/api/v1/auth/login",
		"/api/v1/webhooks",
	}

	// Merge with configured excluded paths
	excludedPaths := append(defaultExcluded, cfg.ExcludedPaths...)

	// Parse idle timeout
	idleTimeout := 30 * time.Minute
	if cfg.IdleTimeout != "" {
		if dur, err := time.ParseDuration(cfg.IdleTimeout); err == nil {
			idleTimeout = dur
		}
	}

	// Build CSRF config
	csrfConfig := csrf.Config{
		CookieName:     cfg.CookieName,
		IdleTimeout:    idleTimeout,
		TrustedOrigins: cfg.TrustedOrigins,
		CookieSecure:   appCfg.Server.Mode == "release",
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
		Next: func(c fiber.Ctx) bool {
			// Skip CSRF for safe methods
			method := c.Method()
			if method == "GET" || method == "HEAD" || method == "OPTIONS" || method == "TRACE" {
				return true
			}

			// Skip CSRF for excluded paths
			path := c.Path()
			for _, excluded := range excludedPaths {
				if strings.HasPrefix(path, excluded) {
					return true
				}
			}

			return false
		},
		ErrorHandler: func(c fiber.Ctx, err error) error {
			logger.Warn("csrf validation failed",
				zap.String("ip", c.IP()),
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
				zap.Error(err),
			)
			return litchierrors.New(litchierrors.ErrBadRequest).
				WithDetail("CSRF token invalid or missing")
		},
	}

	// Set defaults
	if csrfConfig.CookieName == "" {
		csrfConfig.CookieName = "csrf_"
	}

	return csrf.New(csrfConfig)
}