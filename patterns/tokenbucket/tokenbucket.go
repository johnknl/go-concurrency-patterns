// MIT License
//
// Copyright (C) 2026 John Kleijn
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//

package tokenbucket

import (
	"context"
	"sync"
	"time"
)

// bucket holds up to burst tokens and gains rate tokens per second. It has no goroutine: the tokens earned since
// the last call are added when the next call arrives. A negative token count means callers have reserved tokens
// that are still being earned.
type bucket struct {
	last   time.Time
	tokens float64
	rate   float64
	burst  float64
	mu     sync.Mutex
}

// newBucket returns a full bucket.
func newBucket(rate float64, burst int) *bucket {
	return &bucket{last: time.Now(), tokens: float64(burst), rate: rate, burst: float64(burst)}
}

// refill adds the tokens earned since the last call, up to burst. The caller holds mu.
func (b *bucket) refill() {
	now := time.Now()
	b.tokens = min(b.burst, b.tokens+now.Sub(b.last).Seconds()*b.rate)
	b.last = now
}

// allow takes a token if one is available now.
func (b *bucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill()

	if b.tokens < 1 {
		return false
	}

	b.tokens--

	return true
}

// wait takes a token, waiting until it has been earned, or returns the cause of ctx if ctx ends first. The token
// is reserved before waiting, so concurrent callers are served in the order they reserved. A caller whose ctx
// ends returns its token; callers that reserved after it keep their wait times.
func (b *bucket) wait(ctx context.Context) error {
	b.mu.Lock()
	b.refill()
	b.tokens--
	delay := time.Duration(-b.tokens / b.rate * float64(time.Second))
	b.mu.Unlock()

	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		b.mu.Lock()
		b.refill()
		b.tokens++
		b.mu.Unlock()

		return context.Cause(ctx)
	}
}

// newChanBucket returns a token bucket that holds up to burst tokens and starts full. A goroutine adds one token
// for every value received from tick, and drops it when the bucket is full.
func newChanBucket(ctx context.Context, burst int, tick <-chan time.Time) <-chan struct{} {
	tokens := make(chan struct{}, burst)

	for range burst {
		tokens <- struct{}{}
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick:
				select {
				case tokens <- struct{}{}:
				default:
				}
			}
		}
	}()

	return tokens
}

// take takes one token, or returns the cause of ctx if ctx ends first.
func take(ctx context.Context, tokens <-chan struct{}) error {
	select {
	case <-tokens:
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}
