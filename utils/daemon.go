package utils

import (
	"runtime"
	"sync"
)

func StartAction(action func(), wg *sync.WaitGroup) {
	wg.Add(1)

	go func() {
		defer wg.Done()
		action()
	}()
}

func SafeStartGoroutine(do func()) {
	defer func() {
		if r := recover(); r != nil {
			println("Caught panic in goroutine: ", r)

			buf := make([]byte, 4*1024)
			n := runtime.Stack(buf, true)
			println("panic stack trace:\n", string(buf[:n]), "\n")
		}
	}()

	do()
}

func StartDeamon(do func()) {
	var wg sync.WaitGroup

	for {
		go func() {
			defer func() {
				wg.Done()
			}()
			SafeStartGoroutine(do)
		}()
		wg.Add(1)
		wg.Wait()
	}
}
