package yardstick

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	vegeta "github.com/tsenart/vegeta/lib"
)

// TODO stolen straight outta Vegeta. Probably need to clean up some vestigial
// fields ... e.g. redirects?
type Attacker struct {
	stopch      chan struct{}
	workers     uint64
	maxWorkers  uint64
	maxBody     int64
	redirects   int
	seqmu       sync.Mutex
	seq         uint64
	began       time.Time
	chunked     bool
	RequestFunc func(interface{}) (ResponseData, error)
	RequestData []interface{}
	Targeter    Targeter
}

type ResponseData struct {
	Code     uint16
	BytesIn  uint64
	BytesOut uint64
}

const (
	// DefaultRedirects is the default number of times an Attacker follows
	// redirects.
	DefaultRedirects = 10
	// DefaultTimeout is the default amount of time an Attacker waits for a request
	// before it times out.
	DefaultTimeout = 30 * time.Second
	// DefaultConnections is the default amount of max open idle connections per
	// target host.
	DefaultConnections = 10000
	// DefaultMaxConnections is the default amount of connections per target
	// host.
	DefaultMaxConnections = 0
	// DefaultWorkers is the default initial number of workers used to carry an attack.
	DefaultWorkers = 10
	// DefaultMaxWorkers is the default maximum number of workers used to carry an attack.
	DefaultMaxWorkers = math.MaxUint64
	// DefaultMaxBody is the default max number of bytes to be read from response bodies.
	// Defaults to no limit.
	DefaultMaxBody = int64(-1)
	// NoFollow is the value when redirects are not followed but marked successful
	NoFollow = -1
)

func NewAttacker() *Attacker {
	return &Attacker{
		stopch:      make(chan struct{}),
		workers:     DefaultWorkers,
		maxWorkers:  DefaultMaxWorkers,
		maxBody:     DefaultMaxBody,
		began:       time.Now(),
		RequestData: make([]interface{}, 0),
	}
}

func (a *Attacker) AddRequestData(thing interface{}) {
	a.RequestData = append(a.RequestData, thing)
}

// Attack reads its Targets from the passed Targeter and attacks them at
// the rate specified by the Pacer. When the duration is zero the attack
// runs until Stop is called. Results are sent to the returned channel as soon
// as they arrive and will have their Attack field set to the given name.
func (a *Attacker) Attack(p vegeta.Pacer, du time.Duration, name string) <-chan *vegeta.Result {
	var wg sync.WaitGroup

	workers := a.workers
	if workers > a.maxWorkers {
		workers = a.maxWorkers
	}

	if a.Targeter == nil {
		a.Targeter = NewStaticTargeter(a.RequestData...)
	}

	results := make(chan *vegeta.Result)
	ticks := make(chan struct{})
	for i := uint64(0); i < workers; i++ {
		wg.Add(1)
		go a.attack(name, &wg, ticks, results)
	}

	go func() {
		defer close(results)
		defer wg.Wait()
		defer close(ticks)

		began, count := time.Now(), uint64(0)
		for {
			elapsed := time.Since(began)
			if du > 0 && elapsed > du {
				return
			}

			wait, stop := p.Pace(elapsed, count)
			if stop {
				return
			}

			time.Sleep(wait)

			if workers < a.maxWorkers {
				select {
				case ticks <- struct{}{}:
					count++
					continue
				case <-a.stopch:
					return
				default:
					// all workers are blocked. start one more and try again
					workers++
					wg.Add(1)
					go a.attack(name, &wg, ticks, results)
				}
			}

			select {
			case ticks <- struct{}{}:
				count++
			case <-a.stopch:
				return
			}
		}
	}()

	return results
}

// Stop stops the current attack.
func (a *Attacker) Stop() {
	select {
	case <-a.stopch:
		return
	default:
		close(a.stopch)
	}
}

func (a *Attacker) attack(name string, workers *sync.WaitGroup, ticks <-chan struct{}, results chan<- *vegeta.Result) {
	defer workers.Done()
	for range ticks {
		results <- a.hit(name)
	}
}

func (a *Attacker) hit(name string) *vegeta.Result {
	var err error
	var resp ResponseData

	res := vegeta.Result{Attack: name}

	a.seqmu.Lock()
	res.Timestamp = a.began.Add(time.Since(a.began))
	res.Seq = a.seq
	a.seq++
	a.seqmu.Unlock()

	defer func() {
		res.Latency = time.Since(res.Timestamp)
		if err != nil {
			res.Error = err.Error()
		}
	}()

	var thing interface{}
	if err = a.Targeter(&thing); err != nil {
		a.Stop()
		return &res
	}

	if a.RequestFunc == nil {
		panic("Um, you need to define a RequestFunc.")
	}

	resp, err = a.RequestFunc(thing)
	res.BytesIn = resp.BytesIn
	res.BytesOut = resp.BytesOut
	res.Code = resp.Code
	return &res
}

func NewRate(frequency int, dur time.Duration) vegeta.Rate {
	return vegeta.Rate{
		Freq: frequency,
		Per:  dur,
	}
}

// Targeting, or how to get the RequestData into the RequestFunc for the
// attack.
type Targeter func(*interface{}) error

func NewStaticTargeter(tgts ...interface{}) Targeter {
	i := int64(-1)
	return func(tgt *interface{}) error {
		if len(tgts) == 0 {
			return nil
		}
		if tgt == nil {
			return fmt.Errorf("ErrNilTarget TODO")
		}
		*tgt = tgts[atomic.AddInt64(&i, 1)%int64(len(tgts))]
		return nil
	}
}
