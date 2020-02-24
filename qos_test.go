package qos

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQOS_UpdateConnCap(t *testing.T) {
	qs := WithDefaultConfig()
	assert.EqualValues(t, qs.config.connBandwidthCap, connCap)

	newCap := uint64(1024)
	qs.UpdateConnCap(newCap)
	assert.EqualValues(t, qs.config.connBandwidthCap, newCap)
}

func TestQOS_UpdateGlobalCap(t *testing.T) {
	qs := WithConfig(&Config{})
	assert.EqualValues(t, qs.config.globalBandwidthCap, globalCap)

	newCap := uint64(1024)
	qs.UpdateGlobalCap(newCap)
	assert.EqualValues(t, qs.config.globalBandwidthCap, newCap)
}

func TestQOSCustomConf_UpdateGlobalCap(t *testing.T) {
	qs := WithConfig(&Config{globalBandwidthCap: 1024 * 10})
	assert.EqualValues(t, qs.config.globalBandwidthCap, 1024*10)

	newCap := uint64(1024)
	qs.UpdateGlobalCap(newCap)
	assert.EqualValues(t, qs.config.globalBandwidthCap, newCap)
}

func TestQOS_TrackConn(t *testing.T) {
	qs := WithDefaultConfig()

	mcw := &MockConnWriter{}
	bytesWritten := uint64(1024)
	qs.TrackConn(mcw, bytesWritten)
	assert.EqualValues(t, 1, len(qs.connBandwidthTrackingMap))
	assert.EqualValues(t, bytesWritten, qs.connBandwidthTrackingMap[mcw].Sum())

	qs.RemoveConn(mcw)
	assert.EqualValues(t, 0, len(qs.connBandwidthTrackingMap))
}

func TestQOS_GlobalBandwidthRateLimited(t *testing.T) {
	qs := WithConfig(&Config{globalBandwidthCap: 1})

	mcw := &MockConnWriter{}
	bytesWritten := uint64(1024)
	qs.TrackConn(mcw, bytesWritten)

	assert.True(t, qs.RateLimited(mcw))
}

func TestQOS_ConnBandwidthRateLimited(t *testing.T) {
	qs := WithConfig(&Config{connBandwidthCap: 1})

	mcw := &MockConnWriter{}
	bytesWritten := uint64(1024)
	qs.TrackConn(mcw, bytesWritten)

	assert.True(t, qs.RateLimited(mcw))

	mcwNew := &MockConnWriter{}
	assert.False(t, qs.RateLimited(mcwNew))
}

func TestQOS_NotRateLimited(t *testing.T) {
	qs := WithDefaultConfig()

	mcw := &MockConnWriter{}
	bytesWritten := uint64(1024)
	qs.TrackConn(mcw, bytesWritten)

	assert.False(t, qs.RateLimited(mcw))
	assert.True(t, qs.Allowed(mcw))
}

func TestQOS_Allowed(t *testing.T) {
	qs := WithConfig(&Config{
		connBandwidthCap:      1024,
		runningWindowInterval: 1,
	})

	mcw := &MockConnWriter{}
	bytesWritten := uint64(2048)
	qs.TrackConn(mcw, bytesWritten)

	assert.True(t, qs.RateLimited(mcw))
	go func(t *testing.T, qs *QOS) {
		assert.True(t, qs.Allowed(mcw))
	}(t, qs)

	time.Sleep(time.Duration(2*qs.config.runningWindowInterval) * time.Second)
	assert.False(t, qs.RateLimited(mcw))

	assert.True(t, qs.Allowed(mcw))
}

func TestQOS_MultipleConnections(t *testing.T) {
	qs := WithDefaultConfig()

	mcw1 := &MockConnWriter{}
	mcw2 := &MockConnWriter{}

	qs.TrackConn(mcw1, 1024*1024)
	qs.TrackConn(mcw2, 1024*1024)

	assert.True(t, qs.RateLimited(mcw1))
	assert.True(t, qs.RateLimited(mcw2))
}

type MockConnWriter struct {
	writer *io.PipeWriter
}

func (mcw MockConnWriter) Close() error                         { return mcw.writer.Close() }
func (mcw MockConnWriter) Read([]byte) (int, error)             { return 0, nil }
func (mcw MockConnWriter) Write(data []byte) (n int, err error) { return mcw.writer.Write(data) }

func (mcw MockConnWriter) LocalAddr() net.Addr {
	return addr{
		NetworkString: "tcp",
		AddrString:    "127.0.0.1",
	}
}

func (mcw MockConnWriter) RemoteAddr() net.Addr {
	return addr{
		NetworkString: "tcp",
		AddrString:    "127.0.0.1",
	}
}

func (mcw MockConnWriter) SetDeadline(time.Time) error      { return nil }
func (mcw MockConnWriter) SetReadDeadline(time.Time) error  { return nil }
func (mcw MockConnWriter) SetWriteDeadline(time.Time) error { return nil }

type addr struct {
	NetworkString string
	AddrString    string
}

func (a addr) Network() string { return a.NetworkString }
func (a addr) String() string  { return a.AddrString }
