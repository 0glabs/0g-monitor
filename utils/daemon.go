package utils

import "sync"

func StartAction(action func(), wg *sync.WaitGroup) {
	wg.Add(1)

	go func() {
		defer wg.Done()
		action()
	}()
}
