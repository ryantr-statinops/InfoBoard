package phase06

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

func drain(r io.Reader) ([]byte, error) {
	var out bytes.Buffer
	buf := make([]byte, 7)
	for {
		n, err := r.Read(buf)
		out.Write(buf[:n])
		if err != nil {
			if errors.Is(err, io.EOF) {
				return out.Bytes(), nil
			}
			return out.Bytes(), err
		}
	}
}

func TestFakeReaderIsDeterministicAndBounded(t *testing.T) {
	payload := EncodeFrame([]byte(`{"type":"hello"}`)) //nolint:gosec // bounded synthetic frame
	ended := func() *FakeReader {
		reader := NewFakeReader(context.Background(), payload, EncodeFrame([]byte(`{"type":"snapshot"}`)))
		reader.EndStream()
		return reader
	}
	first, err := drain(ended())
	if err != nil {
		t.Fatal(err)
	}
	second, err := drain(ended())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("fake reader is not deterministic across runs")
	}
	reader := NewFakeReader(context.Background(), payload)
	reader.EndStream()
	got, err := drain(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("stream mismatch: %d bytes", len(got))
	}
	if reader.Reads() < 2 {
		t.Fatalf("reads = %d, want the drain loop to have consumed more than one call", reader.Reads())
	}
	if reader.BytesRead() != len(payload) {
		t.Fatalf("bytes_read = %d, want %d", reader.BytesRead(), len(payload))
	}
	if reader.Pending() {
		t.Fatal("reader reports pending chunks after a complete drain")
	}
}

func TestFakeReaderSurfacesInjectedStreamFailure(t *testing.T) {
	boom := errors.New("broken pipe")
	reader := NewFakeReader(context.Background(), EncodeFrame([]byte(`{"type":"hello"}`)))
	reader.FailAfter(boom)
	if _, err := drain(reader); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the injected stream failure", err)
	}
}

func TestFakeReaderHoldsAnOpenPipeUntilEnded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader := NewFakeReader(ctx, EncodeFrame([]byte(`{"type":"hello"}`)))
	observed := make(chan error, 1)
	go func() {
		_, err := drain(reader)
		observed <- err
	}()
	waitForCondition(t, "the drained read to hold the open pipe", func() bool { return reader.Holding() > 0 })
	reader.EndStream()
	select {
	case err := <-observed:
		if err != nil {
			t.Fatalf("err = %v, want a clean drain once the stream ends", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ending the stream did not release the held read")
	}
}

func TestBlockedReadIsInterruptibleWithoutWallClockWaiting(t *testing.T) {
	before := runtime.NumGoroutine()
	for attempt := 0; attempt < 8; attempt++ {
		ctx, cancel := context.WithCancel(context.Background())
		reader := NewFakeReader(ctx)
		gate := reader.Block()
		done := make(chan error, 1)
		go func() {
			_, err := reader.Read(make([]byte, 8))
			done <- err
		}()
		waitForGate(t, gate)
		_ = gate
		cancel()
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("err = %v, want context.Canceled", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("blocked read did not observe cancellation")
		}
		gate.Open()
	}
	if leaked := runtime.NumGoroutine() - before; leaked > 2 {
		t.Fatalf("goroutines leaked by interrupted blocked reads: %d", leaked)
	}
}

func TestBlockedWriteIsInterruptibleAndSerialized(t *testing.T) {
	before := runtime.NumGoroutine()
	for attempt := 0; attempt < 8; attempt++ {
		ctx, cancel := context.WithCancel(context.Background())
		writer := NewFakeWriter(ctx)
		gate := writer.Block()
		var wg sync.WaitGroup
		results := make([]error, 2)
		for index := range results {
			wg.Add(1)
			go func(slot int) {
				defer wg.Done()
				_, err := writer.Write(EncodeFrame([]byte(`{"type":"query_result"}`)))
				results[slot] = err
			}(index)
		}
		waitForWaiters(t, gate, len(results))
		if writer.OverlappingWrites() == 0 {
			t.Fatal("two in-flight writes through one fake writer were not detected")
		}
		cancel()
		waitGroup(t, &wg)
		for index, err := range results {
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("write %d err = %v, want context.Canceled", index, err)
			}
		}
		gate.Open()
	}
	if leaked := runtime.NumGoroutine() - before; leaked > 2 {
		t.Fatalf("goroutines leaked by interrupted blocked writes: %d", leaked)
	}
}

func TestFakeWriterRecordsBytesAndRejectsUseAfterClose(t *testing.T) {
	writer := NewFakeWriter(context.Background())
	first := EncodeFrame([]byte(`{"type":"hello_ack"}`)) //nolint:gosec // bounded synthetic frame
	second := EncodeFrame([]byte(`{"type":"sync_ack"}`)) //nolint:gosec // bounded synthetic frame
	if _, err := writer.Write(first); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(second); err != nil {
		t.Fatal(err)
	}
	if writer.WriteCalls() != 2 {
		t.Fatalf("write_calls = %d, want 2", writer.WriteCalls())
	}
	if writer.OverlappingWrites() != 0 {
		t.Fatalf("sequential writes were reported as overlapping: %d", writer.OverlappingWrites())
	}
	if !bytes.Equal(writer.Bytes(), append(append([]byte{}, first...), second...)) {
		t.Fatal("captured stdout bytes do not match the written frames")
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if writer.CloseCalls() != 2 {
		t.Fatalf("close_calls = %d, want 2", writer.CloseCalls())
	}
	if _, err := writer.Write(first); err == nil {
		t.Fatal("write after close succeeded, so a closed stdout would silently accept frames")
	}
}

func TestFakeClockIsDeterministicAndDrivesDueTimers(t *testing.T) {
	run := func() []time.Time {
		clock := NewFakeClock()
		if !clock.Now().Equal(FixtureEpoch) {
			t.Fatalf("clock start = %s, want the fixed fixture epoch %s", clock.Now(), FixtureEpoch)
		}
		pending := clock.After(50 * time.Millisecond)
		early := clock.After(200 * time.Millisecond)
		if err := clock.Sleep(context.Background(), 25*time.Millisecond); err != nil {
			t.Fatal(err)
		}
		select {
		case <-pending:
			t.Fatal("timer fired before its deterministic deadline")
		default:
		}
		if err := clock.Sleep(context.Background(), 25*time.Millisecond); err != nil {
			t.Fatal(err)
		}
		select {
		case <-pending:
		default:
			t.Fatal("timer did not fire at its deterministic deadline")
		}
		select {
		case <-early:
			t.Fatal("later timer fired early")
		default:
		}
		clock.Advance(200 * time.Millisecond)
		select {
		case <-early:
		default:
			t.Fatal("Advance did not fire the due timer")
		}
		return []time.Time{clock.Now(), FixtureEpoch.Add(250 * time.Millisecond)}
	}
	first, second := run(), run()
	if !reflect.DeepEqual(first, second) {
		t.Fatal("deterministic clock produced different instants across identical runs")
	}
	if clock := NewFakeClock(); clock.Slept() != 0 || clock.Waits() != 0 || clock.PendingTimers() != 0 {
		t.Fatal("unused clock is not inert")
	}
}

func TestFakeClockSleepHonoursContextEnd(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	clock := NewFakeClock()
	if err := clock.Sleep(ctx, time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if clock.Waits() != 0 {
		t.Fatalf("waits = %d, want an ended context to be observed before any simulated sleep", clock.Waits())
	}
	if !clock.Now().Equal(FixtureEpoch) {
		t.Fatal("an ended context must not advance the deterministic clock")
	}
}

func TestFrameScaffoldingMeasuresCapturedStdoutBoundaries(t *testing.T) {
	frames := [][]byte{
		EncodeFrame([]byte(`{"type":"hello_ack","protocol":1}`)),          //nolint:gosec // bounded synthetic payload
		EncodeFrame([]byte(`{"type":"sync_ack","revision":1}`)),           //nolint:gosec // bounded synthetic payload
		EncodeFrame([]byte(`{"type":"health_result","state":"healthy"}`)), //nolint:gosec // bounded synthetic payload
	}
	stream := bytes.Join(frames, nil)
	decoded, stray := ScanFrames(stream)
	if len(decoded) != 3 || stray != 0 {
		t.Fatalf("frames = %d stray = %d, want 3 frames and 0 stray bytes", len(decoded), stray)
	}
	for index, frame := range decoded {
		if frame.Offset != len(bytes.Join(frames[:index], nil)) {
			t.Fatalf("frame %d offset = %d, want the exact byte span", index, frame.Offset)
		}
		if frame.Size != 4+len(frame.Payload) {
			t.Fatalf("frame %d size = %d, want length prefix plus payload", index, frame.Size)
		}
		if !bytes.HasSuffix(frame.Payload, []byte("}")) {
			t.Fatalf("frame %d payload is not a complete JSON object", index)
		}
	}
	if _, stray := ScanFrames(append(append([]byte{}, stream...), []byte("panic: stdout contamination")...)); stray == 0 {
		t.Fatal("trailing diagnostic text long enough to look like a frame was not reported as stray")
	}
	short, stray := ScanFrames(append(append([]byte{}, stream...), "xx"...))
	if stray == 0 || len(short) != 3 {
		t.Fatalf("short trailing contamination: stray=%d frames=%d", stray, len(short))
	}
}

func waitForGate(t *testing.T, gate *Gate) { waitForWaiters(t, gate, 1) }

func waitForWaiters(t *testing.T, gate *Gate, want int) {
	t.Helper()
	for spin := 0; spin < 100000; spin++ {
		if gate.Waiters() >= want {
			return
		}
		runtime.Gosched()
	}
	t.Fatalf("only %d operations reached the blocking gate, want %d", gate.Waiters(), want)
}

func waitGroup(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent writes did not finish")
	}
}
