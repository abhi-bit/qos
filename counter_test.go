package qos

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCounter(t *testing.T) {
	windowInterval := int64(3)
	counter := NewCounter(windowInterval)

	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	var publishCounter int64

	for publishCounter < windowInterval {
		select {
		case <-tick.C:
			counter.AddValue(1)
			publishCounter++
		}
	}

	assert.EqualValues(t, counter.Sum(), windowInterval)
}

func TestPurgeStaleBuckets(t *testing.T) {
	windowInterval := int64(2)
	counter := NewCounter(windowInterval)

	tick := time.NewTicker(time.Second)
	var publishCounter int64

	for publishCounter < windowInterval+2 {
		select {
		case <-tick.C:
			counter.AddValue(1)
			publishCounter++
		}
	}

	assert.EqualValues(t, counter.Sum(), windowInterval)
}
