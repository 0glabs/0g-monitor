package utils

import (
	"context"
	"math"
	"sync"
)

type SizedWaitGroup struct {
	Size int

	current chan struct{}
	wg      sync.WaitGroup
}

func NewSizedWaitGroup(limit int) SizedWaitGroup {
	size := math.MaxInt32
	if limit > 0 {
		size = limit
	}
	return SizedWaitGroup{
		Size: size,

		current: make(chan struct{}, size),
		wg:      sync.WaitGroup{},
	}
}

func (s *SizedWaitGroup) Add() {
	_ = s.AddWithContext(context.Background())
}

func (s *SizedWaitGroup) AddWithContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.current <- struct{}{}:
		break
	}
	s.wg.Add(1)
	return nil
}

func (s *SizedWaitGroup) Done() {
	<-s.current
	s.wg.Done()
}

func (s *SizedWaitGroup) Wait() {
	s.wg.Wait()
}
