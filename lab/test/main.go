package main

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

func fnChTimeout(sec int, wg *sync.WaitGroup) (chtimeout chan int, chdone chan struct{}) {
	chtimeout = make(chan int)
	chdone = make(chan struct{})

	var cchtimeout chan int
	var cchdone chan struct{}
	if sec > 0 {
		cchtimeout, cchdone = fnChTimeout(sec-1, wg) //Recursive call with sec -1
	}

	go func(chtimeout, cchtimeout chan int, cchdone chan struct{}) {
		if cchtimeout != nil {
			defer close(cchtimeout)
			defer close(cchdone)
		}

		l := list.New()

		tk := time.NewTicker(time.Second)
		for {
			select {
			case p := <-chtimeout:
				fmt.Printf("REceiving %d on %d\n", p, sec)
				l.PushBack(p)
			case <-tk.C:
				for e := l.Front(); e != nil; e = l.Front() {
					if cchtimeout != nil {
						cchtimeout <- e.Value.(int)
					} else {
						fmt.Printf("Timing out %d on %d\n", e.Value.(int), sec)
						wg.Done()
					}
					l.Remove(e)
				}
			case <-chdone:
				for e := l.Front(); e != nil; e = l.Front() {
					if cchtimeout != nil {
						cchtimeout <- e.Value.(int)
					} else {
						fmt.Printf("Timing out %d on %d\n", e.Value.(int), sec)
						wg.Done()
					}
					l.Remove(e)
				}
				if cchdone != nil {
					<-tk.C
					cchdone <- struct{}{}
				}
				return
			}
		}
	}(chtimeout, cchtimeout, cchdone)
	return
}

func main() {
	numbers := []int{3, 9, 20, 344, 21}

	var wg sync.WaitGroup
	wg.Add(len(numbers))
	chtimeout, chdone := fnChTimeout(10, &wg)

	for _, n := range numbers {
		fmt.Printf("Adding Number %d\n", n)
		chtimeout <- n
	}
	chdone <- struct{}{}
	wg.Wait()

}
