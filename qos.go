package qos

import (
	"log"
	"net"
	"sync"
	"time"
)

const (
	globalCap             = 1024 * 1024
	connCap               = 1024 * 20
	runningWindowInterval = 60
	backOffIntervalInMS   = 100
)

type Config struct {
	globalBandwidthCap    uint64
	connBandwidthCap      uint64
	runningWindowInterval int64
	backOffInterval       int64
}

type QOS struct {
	globalBandwidthCounter   *Counter
	config                   *Config
	connBandwidthTrackingMap map[net.Conn]*Counter
	rw                       *sync.RWMutex
}

func WithDefaultConfig() *QOS {
	return WithConfig(&Config{
		globalBandwidthCap:    globalCap,
		connBandwidthCap:      connCap,
		runningWindowInterval: runningWindowInterval,
		backOffInterval:       backOffIntervalInMS,
	})
}

func WithConfig(c *Config) *QOS {
	if c.globalBandwidthCap <= 0 {
		c.globalBandwidthCap = globalCap
	}

	if c.connBandwidthCap <= 0 {
		c.connBandwidthCap = connCap
	}

	if c.runningWindowInterval <= 0 {
		c.runningWindowInterval = runningWindowInterval
	}

	if c.backOffInterval <= 0 {
		c.backOffInterval = backOffIntervalInMS
	}

	return &QOS{
		globalBandwidthCounter:   NewCounter(c.runningWindowInterval),
		config:                   c,
		connBandwidthTrackingMap: make(map[net.Conn]*Counter),
		rw:                       &sync.RWMutex{},
	}
}

func (q *QOS) UpdateGlobalCap(cap uint64) {
	q.config.globalBandwidthCap = cap
}

func (q *QOS) UpdateConnCap(cap uint64) {
	q.config.connBandwidthCap = cap
}

func (q *QOS) TrackConn(conn net.Conn, bytesTx uint64) {
	q.rw.Lock()
	defer q.rw.Unlock()

	if _, ok := q.connBandwidthTrackingMap[conn]; !ok {
		q.connBandwidthTrackingMap[conn] = NewCounter(q.config.runningWindowInterval)
	}
	q.connBandwidthTrackingMap[conn].AddValue(bytesTx)
	q.globalBandwidthCounter.AddValue(bytesTx)
}

func (q *QOS) RemoveConn(conn net.Conn) {
	q.rw.Lock()
	defer q.rw.Unlock()
	delete(q.connBandwidthTrackingMap, conn)
}

func (q *QOS) RateLimited(conn net.Conn) bool {
	if q.globalBandwidthCounter.Sum() > q.config.globalBandwidthCap {
		log.Printf("Throttling conn: %s Current global counter: %d cap: %d\n",
			conn.RemoteAddr().String(), q.globalBandwidthCounter.Sum(), q.config.globalBandwidthCap)
		return true
	}

	q.rw.RLock()
	defer q.rw.RUnlock()

	if connBandwidth, ok := q.connBandwidthTrackingMap[conn]; !ok {
		return false
	} else {
		if connBandwidth.Sum() > q.config.connBandwidthCap {
			log.Printf("Throttling conn: %s current conn counter: %d cap: %d\n",
				conn.RemoteAddr().String(), connBandwidth.Sum(), q.config.connBandwidthCap)
			return true
		}
	}

	return false
}

func (q *QOS) Allowed(conn net.Conn) bool {
	if !q.RateLimited(conn) {
		return true
	}

	tick := time.NewTicker(time.Duration(q.config.backOffInterval) * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			if !q.RateLimited(conn) {
				return true
			}
		}
	}
}
