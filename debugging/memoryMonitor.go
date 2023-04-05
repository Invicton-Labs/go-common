package debugging

import (
	"context"
	"time"

	"github.com/Invicton-Labs/go-concurrency"
	"github.com/Invicton-Labs/go-stackerr"
)

var memoryMonitorCtx context.Context = nil
var maxReserved uint64 = 0
var maxInUse uint64 = 0

func StartMemoryMonitor(ctx context.Context) context.Context {
	if memoryMonitorCtx != nil {
		return memoryMonitorCtx
	}
	executor := concurrency.ContinuousFinal(
		ctx, concurrency.ContinuousFinalInput{
			Name: "memory-monitor",
			Func: func(ctx context.Context, metadata *concurrency.RoutineFunctionMetadata) (err stackerr.Error) {
				mem := GetMemUsage()
				if mem.Sys > maxReserved {
					maxReserved = mem.Sys
				}
				if mem.HeapInuse+mem.StackInuse > maxInUse {
					maxInUse = mem.HeapInuse + mem.StackInuse
				}
				return nil
			},
		}, 10*time.Microsecond)
	memoryMonitorCtx = executor.Ctx()
	return memoryMonitorCtx
}

func GetMaxMemoryUsageMb() (maxReserved uint64, maxInUse uint64) {
	return bToMb(maxReserved), bToMb(maxInUse)
}
