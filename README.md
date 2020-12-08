![](misc/yardstick.png)

Benchmark arbitrary things in Go.

Yardstick is a benchmarking library that allows you to benchmark anything.
Think of it as the [Vegeta](https://github.com/tsenart/vegeta) for arbitrary
code. (Because, quite honestly we're stealing everything we can from that
excellent HTTP benchmark tool.)

Yardstick allows you to write custom "request" functions and then call them
at a given rate and duration. E.g.

```
attacker := yardstick.NewAttacker()

attacker.AddRequestData(SomeData{...})
attacker.AddRequestData(SomeData{...})

attacker.RequestFunc = func(thing interface{}) (yardstick.ResponseData, error) {
  data, ok := thing.(SomeData)
  ret := yardstick.ResponseData{}

  ... do something ...

  ret.Code = 0
  return ret, nil
}

metrics := yardstick.NewMetricsWithDefaults()
for res := range attacker.Attack(
  yardstick.NewRate(100, time.Second), 10*time.Second, "my benchmark") {
  metrics.Add(res)
}
metrics.Close()

fmt.Printf("99th percentile: %s\n", metrics.Get().Latencies.P99)
m, _ := json.Marshal(metrics.Get())
fmt.Printf("Metrics:\n%s\n", m)
```

What `RequestFunc` can do is limited only by your imagination. You can
benchmark anything, HTTP, RPC, Redis, Postgres, filesystem, USB,
&lt;whatever device&gt;, etc.

For a more detailed example see [Hello world](examples/helowrld/main.go).
