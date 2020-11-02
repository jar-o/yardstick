package main

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	yardstick "github.com/jar-o/yardstick/lib"
)

type TestThing struct {
	Ohai string
}

func MyCustomTargeter(tgts ...interface{}) yardstick.Targeter {
	i := int64(-1)
	return func(tgt *interface{}) error {
		if len(tgts) == 0 {
			return nil
		}
		if tgt == nil {
			return fmt.Errorf("ErrNilTarget TODO")
		}
		// Fetch every second item... for no decent reason.
		*tgt = tgts[atomic.AddInt64(&i, 2)%int64(len(tgts))]
		return nil
	}
}

func main() {
	// Start by creating a new Attacker.
	attacker := yardstick.NewAttacker()

	// Step 1, add any data specific to the RequestFunc you will define below.
	// Need to specify HTTP requests? Generate key/vals for a redis server,
	// etc? Do that first. Also note that this is optional.
	attacker.AddRequestData(TestThing{Ohai: "helo"})
	attacker.AddRequestData(TestThing{Ohai: "wrld"})
	attacker.AddRequestData(TestThing{Ohai: "HELO"})
	attacker.AddRequestData(TestThing{Ohai: "WRLD"})
	attacker.AddRequestData(TestThing{Ohai: "HELOWRLD"})
	attacker.AddRequestData(TestThing{Ohai: "emiterror"})

	// Step 2, create your custom request function. It can do whatever you want.
	attacker.RequestFunc = func(thing interface{}) (yardstick.ResponseData, error) {
		testthing, ok := thing.(TestThing)
		ret := yardstick.ResponseData{}
		if !ok {
			ret.Code = 1
			return ret, fmt.Errorf("Not ok %+v", testthing)
		}
		time.Sleep(250 * time.Millisecond)
		fmt.Printf("# %s\n", testthing.Ohai)
		ret.BytesIn += uint64(len(testthing.Ohai))
		ret.BytesOut += uint64(len(testthing.Ohai))
		if testthing.Ohai == "emiterror" {
			ret.Code = 2
			return ret, fmt.Errorf("Some error")
		}
		ret.Code = 0
		return ret, nil
	}

	// Optionally, add your own targeter. If you don't, a basic round-robin targeter is used.
	//attacker.Targeter = MyCustomTargeter(attacker.RequestData...)

	// Step 3, run your attack, collect metrics, profit.
	metrics := yardstick.NewMetricsWithDefaults()
	//                                         Rate per second            Duration        Test name
	for res := range attacker.Attack(yardstick.NewRate(100, time.Second), 10*time.Second, "helowrld") {
		metrics.Add(res)
	}
	metrics.Close()

	m := metrics.Get()
	fmt.Printf("# 99th percentile: %s\n", m.Latencies.P99)
	mj, _ := json.Marshal(m)
	fmt.Printf("# Metrics:\n%s\n", mj)
}
