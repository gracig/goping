package goping

import (
	"fmt"
	"runtime"
	"sync"
)

var memStatPool = sync.Pool{
	New: func() interface{} {
		return new(runtime.MemStats)
	},
}

func statsNow() *runtime.MemStats {
	//Getting the before memstat
	mb := memStatPool.Get().(*runtime.MemStats)
	runtime.GC()
	runtime.ReadMemStats(mb)
	return mb
}

func traceMem(fname string, mb *runtime.MemStats) {
	//Getting the after memstats
	ma := memStatPool.Get().(*runtime.MemStats)
	runtime.GC()
	runtime.ReadMemStats(ma)

	//Printing
	fmt.Printf("%v:     Alloc: %v %v %v\n", fname, mb.Alloc, ma.Alloc, (ma.Alloc - mb.Alloc))
	fmt.Printf("%v: HeapInUse: %v %v %v\n", fname, mb.HeapInuse, ma.HeapInuse, (ma.HeapInuse - mb.HeapInuse))
	fmt.Printf("%v:   Mallocs: %v %v %v\n", fname, mb.Mallocs, ma.Mallocs-3, (ma.Mallocs - 3 - mb.Mallocs))
	fmt.Printf("%v:     Frees: %v %v %v\n", fname, mb.Frees, ma.Frees-2, (ma.Frees - 2 - mb.Frees))

	//Returning objects to pool
	memStatPool.Put(mb)
	memStatPool.Put(ma)
}
