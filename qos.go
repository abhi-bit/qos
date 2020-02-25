package qos

import (
	"context"
	"math"
	"net"
	"sync"

	"golang.org/x/time/rate"
)

const (
	globalCap       = 1024 * 1024
	connCap         = 1024 * 20
	burstMultiplier = 5
)

type Config struct {
	GlobalBandwidthCap int
	ConnBandwidthCap   int
}

type QOS struct {
	globalBandwidthCounter   *rate.Limiter
	config                   *Config
	connBandwidthTrackingMap map[net.Conn]*rate.Limiter
	rw                       *sync.RWMutex
}

func WithDefaultConfig() *QOS {
	return WithConfig(&Config{
		GlobalBandwidthCap: globalCap,
		ConnBandwidthCap:   connCap,
	})
}

func WithConfig(c *Config) *QOS {
	if c.GlobalBandwidthCap <= 0 {
		c.GlobalBandwidthCap = globalCap
	}

	if c.ConnBandwidthCap <= 0 {
		c.ConnBandwidthCap = connCap
	}

	return &QOS{
		globalBandwidthCounter: rate.NewLimiter(
			rate.Limit(c.GlobalBandwidthCap),
			c.GlobalBandwidthCap),
		config:                   c,
		connBandwidthTrackingMap: make(map[net.Conn]*rate.Limiter),
		rw:                       &sync.RWMutex{},
	}
}

func (q *QOS) UpdateGlobalCap(cap int) {
	if cap == 0 {
		q.config.GlobalBandwidthCap = math.MaxInt32
	} else {
		q.config.GlobalBandwidthCap = cap
	}

	q.globalBandwidthCounter = rate.NewLimiter(
		rate.Limit(q.config.GlobalBandwidthCap),
		q.config.GlobalBandwidthCap)
}

func (q *QOS) UpdateConnCap(cap int) {
	if cap == 0 {
		q.config.ConnBandwidthCap = math.MaxInt32
	} else {
		q.config.ConnBandwidthCap = cap
	}

	q.rw.Lock()
	defer q.rw.Unlock()
	for conn := range q.connBandwidthTrackingMap {
		q.connBandwidthTrackingMap[conn] = rate.NewLimiter(
			rate.Limit(q.config.ConnBandwidthCap),
			q.config.ConnBandwidthCap*burstMultiplier)
	}
}

type LimitedListener struct {
	net.Listener
	qs *QOS
}

type llConn struct {
	net.Conn
	qs *QOS
}

func (ll *LimitedListener) Accept() (net.Conn, error) {
	c, err := ll.Listener.Accept()
	ll.qs.TrackConn(c, 0)
	return &llConn{
		Conn: c,
		qs:   ll.qs,
	}, err
}

func (llc *llConn) Read(b []byte) (int, error) {
	n, err := llc.Conn.Read(b)
	llc.qs.TrackConn(llc, uint64(n))
	return n, err
}

func (llc *llConn) Write(b []byte) (int, error) {
	n, err := llc.Conn.Write(b)
	llc.qs.TrackConn(llc, uint64(n))
	return n, err
}

func (q *QOS) NewListener(l net.Listener) net.Listener {
	return &LimitedListener{
		Listener: l,
		qs:       q,
	}
}

func (q *QOS) TrackConn(conn net.Conn, bytesTx uint64) {
	q.rw.Lock()
	defer q.rw.Unlock()

	if _, ok := q.connBandwidthTrackingMap[conn]; !ok {
		q.connBandwidthTrackingMap[conn] = rate.NewLimiter(
			rate.Limit(q.config.ConnBandwidthCap),
			q.config.ConnBandwidthCap*burstMultiplier)
	}
	q.globalBandwidthCounter.WaitN(context.Background(), int(bytesTx))
	q.connBandwidthTrackingMap[conn].WaitN(context.Background(), int(bytesTx))
}

func (q *QOS) RemoveConn(conn net.Conn) {
	q.rw.Lock()
	defer q.rw.Unlock()
	delete(q.connBandwidthTrackingMap, conn)
}
