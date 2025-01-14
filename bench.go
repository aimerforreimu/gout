package gout

import (
	"context"
	"github.com/guonaihong/gout/bench"
	"time"
)

type Bench struct {
	bench.Task

	df *DataFlow
}

func (b *Bench) Concurrent(c int) *Bench {
	b.Task.Concurrent = c
	return b
}

func (b *Bench) Number(n int) *Bench {
	b.Task.Number = n
	return b
}

func (b *Bench) Rate(rate int) *Bench {
	b.Task.Rate = rate
	return b
}

func (b *Bench) Durations(d time.Duration) *Bench {
	b.Task.Duration = d
	return b
}

func (b *Bench) Do() error {
	// 报表插件
	req, err := b.df.Req.request()
	if err != nil {
		return err
	}

	client := b.df.out.Client
	if client == &DefaultClient {
		client = &DefaultBenchClient
	}

	r := bench.NewReport(context.Background(),
		b.Task.Concurrent,
		b.Task.Number,
		b.Task.Duration,
		req,
		client)

	// task是并发控制模块
	b.Run(r)
	return nil
}
