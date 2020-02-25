package qos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQOS_UpdateConnCap(t *testing.T) {
	qs := WithDefaultConfig()
	assert.EqualValues(t, qs.config.ConnBandwidthCap, connCap)

	newCap := uint64(1024)
	qs.UpdateConnCap(newCap)
	assert.EqualValues(t, qs.config.ConnBandwidthCap, newCap)
}

func TestQOS_UpdateGlobalCap(t *testing.T) {
	qs := WithConfig(&Config{})
	assert.EqualValues(t, qs.config.GlobalBandwidthCap, globalCap)

	newCap := uint64(1024)
	qs.UpdateGlobalCap(newCap)
	assert.EqualValues(t, qs.config.GlobalBandwidthCap, newCap)
}

func TestQOSCustomConf_UpdateGlobalCap(t *testing.T) {
	qs := WithConfig(&Config{GlobalBandwidthCap: 1024 * 10})
	assert.EqualValues(t, qs.config.GlobalBandwidthCap, 1024*10)

	newCap := uint64(1024)
	qs.UpdateGlobalCap(newCap)
	assert.EqualValues(t, qs.config.GlobalBandwidthCap, newCap)
}
