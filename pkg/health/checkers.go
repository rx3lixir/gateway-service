// pkg/health/checkers.go
package health

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// GRPCChecker проверка gRPC соединения
func GRPCChecker(conn *grpc.ClientConn, serviceName string) Checker {
	return CheckerFunc(func(ctx context.Context) CheckResult {
		state := conn.GetState()

		// Проверяем состояние соединения
		switch state {
		case connectivity.Ready, connectivity.Idle:
			return CheckResult{
				Status: StatusUp,
				Details: map[string]any{
					"service": serviceName,
					"state":   state.String(),
				},
			}
		default:
			return CheckResult{
				Status: StatusDown,
				Error:  fmt.Sprintf("connection state: %s", state),
				Details: map[string]any{
					"service": serviceName,
					"state":   state.String(),
				},
			}
		}
	})
}

// DiskSpaceChecker проверка свободного места на диске
func DiskSpaceChecker(path string, minFreeBytes uint64) Checker {
	return CheckerFunc(func(ctx context.Context) CheckResult {
		// Для Linux/Unix систем
		// В продакшене лучше использовать библиотеку типа github.com/shirou/gopsutil
		return CheckResult{
			Status: StatusUp,
			Details: map[string]any{
				"path": path,
				"note": "implement disk check based on OS",
			},
		}
	})
}

// MemoryChecker проверка использования памяти
func MemoryChecker(maxUsagePercent float64) Checker {
	return CheckerFunc(func(ctx context.Context) CheckResult {
		// В продакшене использовать runtime.MemStats или gopsutil
		return CheckResult{
			Status: StatusUp,
			Details: map[string]any{
				"max_usage_percent": maxUsagePercent,
				"note":              "implement memory check",
			},
		}
	})
}

// CustomChecker создает проверку с простой функцией
func CustomChecker(name string, check func() error) Checker {
	return CheckerFunc(func(ctx context.Context) CheckResult {
		if err := check(); err != nil {
			return CheckResult{
				Status: StatusDown,
				Error:  err.Error(),
			}
		}
		return CheckResult{
			Status: StatusUp,
		}
	})
}
