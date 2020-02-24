package qos

import (
	"sync"
	"time"
)

type Counter struct {
	buckets          map[int64]uint64
	rw               *sync.RWMutex
	runningWindowInS int64
}

func NewCounter(runningWindowInterval int64) *Counter {
	return &Counter{
		buckets:          make(map[int64]uint64),
		rw:               &sync.RWMutex{},
		runningWindowInS: runningWindowInterval,
	}
}

func (c *Counter) AddValue(v uint64) {
	c.rw.Lock()

	now := time.Now().Unix()
	if _, ok := c.buckets[now]; !ok {
		c.buckets[now] = 0
	}
	c.buckets[now] += v
	c.rw.Unlock()

	c.purgeStaleBuckets()
}

func (c *Counter) Sum() uint64 {
	c.rw.RLock()
	defer c.rw.RUnlock()

	sum := uint64(0)
	lastValidTs := time.Now().Unix() - c.runningWindowInS

	for ts, bv := range c.buckets {
		if ts >= lastValidTs {
			sum += bv
		}
	}

	return sum
}

func (c *Counter) purgeStaleBuckets() {
	lastValidTs := time.Now().Unix() - c.runningWindowInS

	for ts := range c.buckets {
		if lastValidTs >= ts {
			c.rw.Lock()
			delete(c.buckets, ts)
			c.rw.Unlock()
		}
	}
}
