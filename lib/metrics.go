package yardstick

// A very skinny shim over vegeta.Metrics. The only assumption vegeta.Metrics
// makes that doesn't work for Yardstick is that it defaults to HTTP status
// codes for success/failure. This shim allows you to override those with
// custom codes.

import (
	vegeta "github.com/tsenart/vegeta/lib"
)

type YMetrics struct {
	m               *vegeta.Metrics
	successCodes    map[uint16]bool
	successOverride uint64
}

// Default is 0 = success, everything else is failure.
func NewMetricsWithDefaults() *YMetrics {
	return NewMetrics([]uint16{0})
}

// Define your own set of success codes
func NewMetrics(successCodes []uint16) *YMetrics {
	sc := make(map[uint16]bool)
	for _, i := range successCodes {
		sc[i] = true
	}
	return &YMetrics{
		m:            &vegeta.Metrics{},
		successCodes: sc,
	}
}

func (o *YMetrics) Add(r *vegeta.Result) {
	o.m.Add(r)
	if _, ok := o.successCodes[r.Code]; ok {
		o.successOverride++
	}
}

func (o *YMetrics) Close() {
	o.m.Close()
	o.m.Success = float64(o.successOverride) / float64(o.m.Requests)
}

func (o *YMetrics) Get() *vegeta.Metrics {
	return o.m
}
