package cache

import (
	"github.com/ryuyb/litchi/internal/domain/repository"
	"go.uber.org/fx"
)

// CacheModule provides file-based cache implementations via Fx.
var CacheModule = fx.Module("cache",
	fx.Provide(
		fx.Annotate(
			NewFileCacheRepository,
			fx.As(new(repository.CacheRepository)),
		),
	),
)