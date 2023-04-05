package debugging

import (
	"fmt"
	"runtime"
)

func GetMemUsage() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

func GetFormattedMemUsage() string {
	m := GetMemUsage()
	return fmt.Sprintf(`Memory Allocation
	Total Reserved: %d MiB
	Heap Reserved: %d MiB
	Heap In-Use: %d MiB
	Heap Allocated: %d MiB
	Stack Reserved: %d MiB
	Stack In-Use: %d MiB`, bToMb(m.Sys), bToMb(m.HeapSys), bToMb(m.HeapInuse), bToMb(m.HeapAlloc), bToMb(m.StackSys), bToMb(m.StackInuse))
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
