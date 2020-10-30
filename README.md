![](aux/yardstick.png)

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

attacker.RequestFunc = func(thing interface{}) (uint16, error) {
  data, ok := thing.(SomeData)
  ... do something ...
  return 0, nil
}

var metrics yardstick.Metrics
for res := range attacker.Attack(
  yardstick.NewRate(100, time.Second), 10*time.Second, "my benchmark") {
  metrics.Add(res)
}
metrics.Close()

fmt.Printf("99th percentile: %s\n", metrics.Latencies.P99)
m, _ := json.Marshal(metrics)
fmt.Printf("Metrics:\n%s\n", m)
```

For a more detailed example see [Hello world](examples/helowrld/main.go).
