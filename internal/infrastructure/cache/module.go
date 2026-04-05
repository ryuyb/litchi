package cache

import (
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/pkg/fxutil"
	"go.uber.org/fx"
)

func init() {
	fxutil.RegisterModule(fxutil.ModuleInfo{
		Name:     "cache",
		Provides: []string{"repository.CacheRepository"},
		Invokes:  []string{},
		Depends:  []string{"*zap.Logger"},
	})
}

// CacheModule provides file-based cache implementations via Fx.
var CacheModule = fx.Module("cache",
	fx.Provide(
		fx.Annotate(
			NewFileCacheRepository,
			fx.As(new(repository.CacheRepository)),
		),
	),
)