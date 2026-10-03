package phase06

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"time"
)

// ErrGateClosed reports that a fake transport was released without producing data.
var ErrGateClosed = errors.New("fake transport gate closed")

// Gate is a deterministic, context-interruptible blocking point for fake I/O.
// It replaces wall-clock blocking so timeout and cancellation cases stay exact.
type Gate struct {
	mu       sync.Mutex
	ch       chan struct{}
	opened   bool
	waiters  int
	blocking bool
}

func NewGate() *Gate { return &Gate{ch: make(chan struct{})} }

// Blocking returns a gate that holds every read or write until released.
func NewBlockingGate() *Gate { g := NewGate(); g.blocking = true; return g }

// Open releases every waiter. Opening twice is safe and has no extra effect.
func (g *Gate) Open() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.opened {
		g.opened = true
		close(g.ch)
	}
}

func (g *Gate) isOpen() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.opened
}

// Wait blocks until the gate is open or the context ends. It returns the context
// error on cancellation, or err when a terminal stream failure is supplied.
func (g *Gate) Wait(ctx context.Context, err error) error {
	if !g.blocking {
		return nil
	}
	g.mu.Lock()
	g.waiters++
	g.mu.Unlock()
	select {
	case <-g.ch:
		return nil
	case <-ctx.Done():
		if err != nil {
			return err
		}
		return ctx.Err()
	}
}

// Waiters reports how many operations are currently held by this gate.
func (g *Gate) Waiters() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.waiters
}

// fakeTimer is one scheduled deterministic wait.
type fakeTimer struct {
	id       int64
	deadline time.Time
	ch       chan time.Time
}

// FakeClock is the deterministic clock. It never consults wall time, so a
// fixture run reports the same instants, durations, and wait counts every time.
// It satisfies the host runtime Clock seam, so every budget the runtime enforces
// is observable without wall-clock waiting.
type FakeClock struct {
	mu       sync.Mutex
	now      time.Time
	waits    int
	slept    time.Duration
	timers   map[int64]*fakeTimer
	nextID   int64
	waitHook func(time.Duration)
}

var FixtureEpoch = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func NewFakeClock() *FakeClock {
	return &FakeClock{now: FixtureEpoch, timers: map[int64]*fakeTimer{}}
}

// OnSleep registers an observer invoked at the start of each simulated sleep.
func (c *FakeClock) OnSleep(hook func(time.Duration)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.waitHook = hook
}

func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Since reports the deterministic elapsed time from an instant.
func (c *FakeClock) Since(t time.Time) time.Duration { return c.Now().Sub(t) }

// NewTimer implements the host runtime Clock seam. The returned channel fires
// when Advance reaches the deadline; the stop function discards a pending wait.
func (c *FakeClock) NewTimer(d time.Duration) (<-chan time.Time, func() bool) {
	ch := make(chan time.Time, 1)
	c.mu.Lock()
	if d <= 0 {
		now := c.now
		c.mu.Unlock()
		ch <- now
		return ch, func() bool { return false }
	}
	c.nextID++
	timer := &fakeTimer{id: c.nextID, deadline: c.now.Add(d), ch: ch}
	c.timers[timer.id] = timer
	c.mu.Unlock()
	return ch, func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		if _, live := c.timers[timer.id]; live {
			delete(c.timers, timer.id)
			return true
		}
		return false
	}
}

// After returns a channel that fires once the clock reaches now+d.
func (c *FakeClock) After(d time.Duration) <-chan time.Time {
	ch, _ := c.NewTimer(d)
	return ch
}

// Advance moves the deterministic clock forward and fires every due timer.
func (c *FakeClock) Advance(delta time.Duration) {
	if delta < 0 {
		return
	}
	c.mu.Lock()
	c.now = c.now.Add(delta)
	due := c.dueLocked()
	c.mu.Unlock()
	fire(c, due)
}

// Sleep advances the deterministic clock by d and returns the context error when
// the context ends first, so deadline tests need no real waiting.
func (c *FakeClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	c.waits++
	c.slept += d
	hook := c.waitHook
	c.now = c.now.Add(d)
	due := c.dueLocked()
	c.mu.Unlock()
	if hook != nil {
		hook(d)
	}
	fire(c, due)
	return ctx.Err()
}

func (c *FakeClock) dueLocked() []*fakeTimer {
	due := make([]*fakeTimer, 0, len(c.timers))
	for id, timer := range c.timers {
		if !timer.deadline.After(c.now) {
			due = append(due, timer)
			delete(c.timers, id)
		}
	}
	return due
}

func fire(c *FakeClock, due []*fakeTimer) {
	for _, timer := range due {
		select {
		case timer.ch <- c.Now():
		default:
		}
	}
}

// Waits reports how many simulated sleeps occurred.
func (c *FakeClock) Waits() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.waits
}

// Slept reports the total simulated elapsed time.
func (c *FakeClock) Slept() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.slept
}

// PendingTimers reports how many scheduled waits have not yet fired.
func (c *FakeClock) PendingTimers() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.timers)
}

// FakeReader is a deterministic io.Reader. It serves queued chunks, can hold a
// read open indefinitely, can fail a read, and can end the stream at any point.
type FakeReader struct {
	ctx      context.Context
	mu       sync.Mutex
	chunks   [][]byte
	index    int
	gate     *Gate
	hold     *Gate
	readErr  error
	eof      bool
	reads    int
	bytes    int
	maxAsked int
}

func NewFakeReader(ctx context.Context, chunks ...[]byte) *FakeReader {
	return &FakeReader{ctx: ctx, chunks: chunks, hold: NewBlockingGate()}
}

// Block holds every subsequent read until the returned gate is opened.
func (r *FakeReader) Block() *Gate {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gate = NewBlockingGate()
	return r.gate
}

// FailAfter serves queued chunks and then returns err, releasing a read that is
// holding the stream open so the terminal failure is observed at once.
func (r *FakeReader) FailAfter(err error) {
	r.mu.Lock()
	r.readErr = err
	stale := r.rotateHoldLocked()
	r.mu.Unlock()
	stale.Open()
}

// EndStream reports io.EOF once the queued chunks are drained.
func (r *FakeReader) EndStream() {
	r.mu.Lock()
	r.eof = true
	stale := r.rotateHoldLocked()
	r.mu.Unlock()
	stale.Open()
}

// rotateHoldLocked installs a fresh closed hold gate and returns the stale one so
// the caller can release parked reads outside the lock.
func (r *FakeReader) rotateHoldLocked() *Gate {
	stale := r.hold
	r.hold = NewBlockingGate()
	return stale
}

// Reads reports how many Read calls were served.
func (r *FakeReader) Reads() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.reads
}

// BytesRead reports how many payload bytes were served.
func (r *FakeReader) BytesRead() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.bytes
}

// MaxRequestedBuffer reports the largest read buffer the host asked for, which
// exposes whether a frame limit was enforced before allocation.
func (r *FakeReader) MaxRequestedBuffer() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.maxAsked
}

// Holding reports how many reads currently wait on an open browser pipe.
func (r *FakeReader) Holding() int { return r.hold.Waiters() }

// Pending reports whether undelivered chunks remain.
func (r *FakeReader) Pending() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.index < len(r.chunks)
}

func (r *FakeReader) Read(p []byte) (int, error) {
	ctx := r.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		r.mu.Lock()
		r.reads++
		if len(p) > r.maxAsked {
			r.maxAsked = len(p)
		}
		gate := r.gate
		if gate != nil {
			r.mu.Unlock()
			if err := gate.Wait(ctx, ctx.Err()); err != nil {
				return 0, err
			}
			r.mu.Lock()
		}
		if r.index >= len(r.chunks) {
			readErr, eof, hold := r.readErr, r.eof, r.hold
			r.mu.Unlock()
			if readErr != nil {
				return 0, readErr
			}
			if eof {
				return 0, io.EOF
			}
			// A drained but unended fake stdin behaves like a live browser pipe:
			// the read holds open until the stream ends, fails, or the caller
			// cancels, so no fixture observes a premature EOF.
			if err := hold.Wait(ctx, ctx.Err()); err != nil {
				return 0, err
			}
			continue
		}
		n := copy(p, r.chunks[r.index])
		r.bytes += n
		r.chunks[r.index] = r.chunks[r.index][n:]
		if len(r.chunks[r.index]) == 0 {
			r.index++
		}
		r.mu.Unlock()
		return n, nil
	}
}

// FakeWriter records every stdout byte, can hold writes open, and detects
// overlapping write calls so serialized-writer ownership is observable.
type FakeWriter struct {
	ctx      context.Context
	mu       sync.Mutex
	buf      []byte
	writes   int
	inFlight int
	overlaps int
	gate     *Gate
	writeErr error
	closed   bool
	closeCnt int
}

func NewFakeWriter(ctx context.Context) *FakeWriter { return &FakeWriter{ctx: ctx} }

// Block holds every subsequent write until the returned gate is opened.
func (w *FakeWriter) Block() *Gate {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.gate = NewBlockingGate()
	return w.gate
}

// FailWrites makes every subsequent write return err.
func (w *FakeWriter) FailWrites(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writeErr = err
}

func (w *FakeWriter) Write(p []byte) (int, error) {
	ctx := w.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	w.mu.Lock()
	w.writes++
	w.inFlight++
	if w.inFlight > 1 {
		w.overlaps++
	}
	gate, writeErr, closed := w.gate, w.writeErr, w.closed
	w.mu.Unlock()
	defer func() {
		w.mu.Lock()
		w.inFlight--
		w.mu.Unlock()
	}()
	if gate != nil {
		if err := gate.Wait(ctx, ctx.Err()); err != nil {
			return 0, err
		}
	}
	if writeErr != nil {
		return 0, writeErr
	}
	if closed {
		return 0, errors.New("write after close")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf, p...)
	return len(p), nil
}

// Close marks the stream closed so later writes fail like a real closed pipe.
func (w *FakeWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closeCnt++
	w.closed = true
	return nil
}

// CloseCalls reports how many times the host closed the stream.
func (w *FakeWriter) CloseCalls() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closeCnt
}

// OverlappingWrites reports writes that started while another write was in flight.
func (w *FakeWriter) OverlappingWrites() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.overlaps
}

// Gate exposes the blocking gate so a test can release a stuck stdout write.
func (w *FakeWriter) Gate() *Gate {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.gate == nil {
		w.gate = NewBlockingGate()
	}
	return w.gate
}

// Bytes returns a copy of every byte the host wrote.
func (w *FakeWriter) Bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]byte, len(w.buf))
	copy(out, w.buf)
	return out
}

// WriteCalls reports how many Write calls the host issued.
func (w *FakeWriter) WriteCalls() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writes
}

// EncodeFrame builds one browser-provided Native Messaging frame as fixture
// input bytes. This is test scaffolding for the fake stdin stream only: IP-06
// owns no codec, and IP-07 owns every envelope field and decode rule.
func EncodeFrame(payload []byte) []byte {
	frame := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint32(frame[:4], uint32(len(payload)))
	copy(frame[4:], payload)
	return frame
}

// DecodedFrame is one captured stdout frame plus the exact byte span it occupied.
type DecodedFrame struct {
	Payload []byte
	Offset  int
	Size    int
}

// ScanFrames splits a captured stdout stream on the browser-provided length
// prefix so a test can measure frame boundaries and stray bytes. It applies no
// size or validity rule of its own: every fail-closed transport outcome must come
// from the host runtime, never from this scaffolding.
func ScanFrames(raw []byte) ([]DecodedFrame, int) {
	frames := make([]DecodedFrame, 0, 4)
	offset := 0
	for offset+4 <= len(raw) {
		size := int(binary.LittleEndian.Uint32(raw[offset : offset+4]))
		if size < 0 || offset+4+size > len(raw) {
			break
		}
		payload := make([]byte, size)
		copy(payload, raw[offset+4:offset+4+size])
		frames = append(frames, DecodedFrame{Payload: payload, Offset: offset, Size: 4 + size})
		offset += 4 + size
	}
	return frames, len(raw) - offset
}

// Queue appends fixture input bytes to an open fake stdin stream and wakes a
// parked read so the host observes them. The hold gate is rotated to a fresh
// closed one, so the woken read that finds no further data parks again instead of
// reporting a premature end of stream.
func (r *FakeReader) Queue(chunk []byte) {
	r.mu.Lock()
	r.chunks = append(r.chunks, chunk)
	r.eof = false
	r.readErr = nil
	stale := r.rotateHoldLocked()
	r.mu.Unlock()
	stale.Open()
}

// Unblock releases every read parked on the open stream without ending it and
// rotates to a fresh closed hold gate. A host shutdown test uses it to prove a
// spurious wake is never mistaken for a clean end of stream, and to let the
// single reader goroutine observe the session cancellation instead of leaking.
func (r *FakeReader) Unblock() int {
	r.mu.Lock()
	released := r.hold.Waiters()
	stale := r.rotateHoldLocked()
	r.mu.Unlock()
	stale.Open()
	return released
}
