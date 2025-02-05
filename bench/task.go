package bench

import (
	"os"
	"os/signal"
	"sync"
	"time"
)

type SubTasker interface {
	Init()
	Process(chan struct{})
	Cancel()
	WaitAll()
}

type Task struct {
	Duration   time.Duration //压测时间
	Number     int           //压测次数
	Concurrent int           //并发数
	Rate       int           //压测频率

	work chan struct{}

	ok bool

	wg sync.WaitGroup
}

func (t *Task) init() {
	t.work = make(chan struct{})
	if t.Concurrent == 0 {
		t.Concurrent = 1
	}
	t.ok = true
}

func (t *Task) producer() {
	if t.ok == false {
		panic("task must be init")
	}

	work := t.work
	// 控制压测时间
	if t.Duration > 0 {
		tk := time.NewTicker(t.Duration)
		go func() {
			defer close(work)
			for {
				select {
				case <-tk.C:
					return
				case work <- struct{}{}:
				}
			}
		}()

		return
	}

	go func() {
		defer close(work)

		switch {
		case t.Number == 0:
			return
		case t.Number > 0:
			for i, n := 0, t.Number; i < n; i++ {
				work <- struct{}{}
			}
		default: // t.Number < 0
			for {
				work <- struct{}{}
			}
		}

	}()

}

func (t *Task) run(sub SubTasker) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	interval := 0
	work := t.work
	wg := &t.wg

	allDone := make(chan struct{})
	if t.Rate > 0 {
		interval = int(time.Second) / t.Rate
	}

	begin := time.Now()
	if interval > 0 {
		oldwork := work
		count := 0
		work = make(chan struct{}, 1)

		wg.Add(1)
		go func() {
			defer func() {
				close(work)
				wg.Done()
			}()

			for {
				next := begin.Add(time.Duration(count * interval))
				time.Sleep(next.Sub(time.Now()))
				select {
				case _, ok := <-oldwork:
					if !ok {
						return
					}
				default:
				}

				work <- struct{}{}
				count++
			}
		}()

	}

	wg.Add(t.Concurrent)
	for i, c := 0, t.Concurrent; i < c; i++ {
		go func() {
			defer wg.Done()
			sub.Process(work)
		}()
	}

	go func() {
		wg.Wait()
		close(allDone)
	}()

	select {
	case <-sig:
		sub.Cancel()
		sub.WaitAll()
	case <-allDone:
		sub.Cancel()
		sub.WaitAll()
	}
}

func (t *Task) Run(sub SubTasker) {
	t.init()

	sub.Init()

	t.producer()

	t.run(sub)
}
