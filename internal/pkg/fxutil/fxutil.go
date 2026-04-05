package fxutil

import (
	"fmt"
	"sort"
	"sync"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// FxEventLogger adapts Zap logger for Fx event logging.
type FxEventLogger struct {
	logger *zap.Logger
}

// NewFxEventLogger creates a new Fx event logger.
func NewFxEventLogger(logger *zap.Logger) *FxEventLogger {
	return &FxEventLogger{logger: logger.Named("fx")}
}

func (l *FxEventLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		l.logger.Debug("OnStart hook executing")
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			l.logger.Error("OnStart hook failed", zap.Error(e.Err))
		} else {
			l.logger.Debug("OnStart hook executed", zap.Duration("runtime", e.Runtime))
		}
	case *fxevent.OnStopExecuting:
		l.logger.Debug("OnStop hook executing")
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			l.logger.Error("OnStop hook failed", zap.Error(e.Err))
		} else {
			l.logger.Debug("OnStop hook executed", zap.Duration("runtime", e.Runtime))
		}
	case *fxevent.Supplied:
		l.logger.Debug("Supplied", zap.String("type", e.TypeName))
	case *fxevent.Provided:
		l.logger.Debug("Provided",
			zap.String("constructor", e.ConstructorName),
			zap.Strings("output_types", e.OutputTypeNames),
		)
	case *fxevent.Invoked:
		l.logger.Debug("Invoked", zap.String("function", e.FunctionName))
		if e.Err != nil {
			l.logger.Error("Invoke failed",
				zap.String("function", e.FunctionName),
				zap.Error(e.Err),
			)
		}
	case *fxevent.Stopping:
		l.logger.Info("Application stopping")
	case *fxevent.Stopped:
		if e.Err != nil {
			l.logger.Error("Application stopped with error", zap.Error(e.Err))
		} else {
			l.logger.Info("Application stopped")
		}
	case *fxevent.RollingBack:
		l.logger.Warn("Rolling back")
	case *fxevent.RolledBack:
		if e.Err != nil {
			l.logger.Error("Rollback failed", zap.Error(e.Err))
		} else {
			l.logger.Info("Rollback completed")
		}
	case *fxevent.Started:
		l.logger.Info("Application started")
	case *fxevent.LoggerInitialized:
		if e.Err != nil {
			l.logger.Error("Logger initialization failed", zap.Error(e.Err))
		} else {
			l.logger.Debug("Logger initialized")
		}
	}
}

// VisualizeDependencyGraph returns the dependency graph visualization.
func VisualizeDependencyGraph(app *fx.App) string {
	// Note: Fx doesn't provide direct graph visualization
	// This is a placeholder for future implementation
	return "Dependency graph visualization requires fxtool or custom implementation"
}

// ModuleInfo holds metadata about an Fx module.
type ModuleInfo struct {
	Name     string
	Provides []string
	Invokes  []string
	Depends  []string
}

// Registry holds information about all registered modules.
// Using sync.Map for concurrent-safe access during init().
var Registry sync.Map

// RegisterModule registers module metadata for documentation.
func RegisterModule(info ModuleInfo) {
	Registry.Store(info.Name, info)
}

// GetModuleInfo returns information about a registered module.
func GetModuleInfo(name string) (ModuleInfo, bool) {
	if v, ok := Registry.Load(name); ok {
		return v.(ModuleInfo), true
	}
	return ModuleInfo{}, false
}

// ListModules returns all registered modules sorted by name.
func ListModules() []ModuleInfo {
	modules := make([]ModuleInfo, 0)
	Registry.Range(func(key, value interface{}) bool {
		modules = append(modules, value.(ModuleInfo))
		return true
	})
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Name < modules[j].Name
	})
	return modules
}

// PrintModules prints all registered modules in sorted order.
func PrintModules() {
	fmt.Println("Registered Fx Modules:")
	fmt.Println("========================")
	modules := ListModules()
	for _, info := range modules {
		fmt.Printf("\nModule: %s\n", info.Name)
		if len(info.Provides) > 0 {
			fmt.Println("  Provides:")
			for _, p := range info.Provides {
				fmt.Printf("    - %s\n", p)
			}
		}
		if len(info.Invokes) > 0 {
			fmt.Println("  Invokes:")
			for _, i := range info.Invokes {
				fmt.Printf("    - %s\n", i)
			}
		}
		if len(info.Depends) > 0 {
			fmt.Println("  Depends on:")
			for _, d := range info.Depends {
				fmt.Printf("    - %s\n", d)
			}
		}
	}
}
