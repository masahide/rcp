package rcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Rcp configs
type Rcp struct {
	MaxBufNum    int
	BufSize      int
	SingleThread bool
	DummyInput   int64
	DummyOutput  bool
	DialAddr     string
	Output       string
	Input        string
	ListenAddr   string
	*SpeedDashboard
}

// ErrInput  error type of source is not specified
var ErrInput = errors.New("The source is not specified")

// ErrOutput error type of destination is not specified
var ErrOutput = errors.New("The destination is not specified")

func (rcp *Rcp) openReader() (r io.ReadCloser, err error) {
	switch {
	case rcp.DummyInput > 0:
		r = openDummyRead(rcp.DummyInput)
		rcp.InputName = "dummy input"
		rcp.TotalSize = rcp.DummyInput
	case len(rcp.Input) > 0:
		var f *os.File
		if f, err = os.Open(rcp.Input); err != nil {
			return
		}
		rcp.InputName = rcp.Input
		var fi os.FileInfo
		if fi, err = f.Stat(); err != nil {
			return
		}
		r = f
		rcp.TotalSize = fi.Size()
	case len(rcp.ListenAddr) > 0:
		if r, err = reciveStreamOpen(rcp.ListenAddr); err != nil {
			return
		}
		rcp.InputName = rcp.ListenAddr
	default:
		return r, ErrInput
	}
	return
}

func (rcp *Rcp) openWriter() (w io.WriteCloser, err error) {
	switch {
	case rcp.DummyOutput:
		w = openDummyWrite()
		rcp.OutputName = "dummy output"
	case len(rcp.Output) > 0:
		if w, err = os.Create(rcp.Output); err != nil {
			return
		}
		rcp.OutputName = rcp.Output
	case len(rcp.DialAddr) > 0:
		if w, err = net.Dial("tcp", rcp.DialAddr); err != nil {
			return
		}
		rcp.OutputName = rcp.DialAddr
	default:
		return w, ErrOutput
	}
	return
}

// ReadWrite mode
func (rcp *Rcp) ReadWrite() (size int64, err error) {
	var w io.WriteCloser
	var r io.ReadCloser
	rcp.SpeedDashboard = NewSpeedDashboard()
	r, err = rcp.openReader()
	if err != nil {
		return
	}
	defer r.Close()
	w, err = rcp.openWriter()
	if err != nil {
		return
	}
	defer w.Close()
	return map[bool]func(io.Writer, io.Reader) (int64, error){
		true:  io.Copy,
		false: rcp.bufCopy,
	}[rcp.SingleThread](w, r)
}

type buffers struct {
	limit chan struct{}
	pool  sync.Pool
}

func newBuffers(size, n int) *buffers {
	bs := buffers{}
	bs.limit = make(chan struct{}, n)
	bs.pool = sync.Pool{New: func() interface{} {
		buf := make([]byte, size)
		return &buf
	}}
	return &bs
}

func (bs *buffers) Len() int {
	return len(bs.limit)
}

func (bs *buffers) Get() *[]byte {
	bs.limit <- struct{}{} // 空くまで待つ
	return bs.pool.Get().(*[]byte)
}

func (bs *buffers) Put(b *[]byte) {
	bs.pool.Put(b)
	<-bs.limit // 解放
}

type threadCopy struct {
	queue   chan *[]byte
	bufSize int
	bs      *buffers
	r       io.Reader
	w       io.Writer

	// atomic counter
	inputBytes  uint64
	outputBytes uint64
}

type result struct {
	size uint64
	err  error
}

func (rcp *Rcp) bufCopy(w io.Writer, r io.Reader) (int64, error) {
	tc := &threadCopy{
		w:       w,
		r:       r,
		bufSize: rcp.BufSize,
		bs:      newBuffers(rcp.BufSize, rcp.MaxBufNum),
		queue:   make(chan *[]byte, rcp.MaxBufNum),
	}
	ctx, cancel := context.WithCancel(context.Background())
	rResChan := make(chan result)
	wResChan := make(chan result)
	var wg sync.WaitGroup
	defer func() { cancel(); wg.Wait() }()
	wg.Add(2)
	go func() { tc.readWorker(ctx, rResChan); wg.Done() }()
	go func() { tc.writeWorker(ctx, wResChan); wg.Done() }()

	mctx, mCancel := context.WithCancel(ctx)
	wg.Add(2)
	go func() { tc.monitorWorker(mctx, rcp.Ch); wg.Done() }()
	go func() {
		if err := rcp.SpeedDashboard.Run(mctx); err != nil {
			fmt.Fprintf(os.Stderr, "SpeedDashboard.Run err: %s", err)
		}
		wg.Done()
		cancel()
	}()
	rRes := <-rResChan
	wRes := <-wResChan
	mCancel()
	if rRes.err != nil && rRes.err != io.EOF {
		return int64(rRes.size), rRes.err
	}
	if wRes.err != nil && wRes.err != io.EOF {
		return int64(wRes.size), wRes.err
	}
	return int64(wRes.size), nil
}

func (tc *threadCopy) readWorker(ctx context.Context, res chan result) {
	size := uint64(0)
	var err error
	defer close(tc.queue)
	defer func() { res <- result{size, err}; close(res) }()
	for {
		var c int
		buf := tc.bs.Get()
		c, err = tc.r.Read(*buf)
		size += uint64(c)
		if err != nil && err != io.EOF {
			return
		}
		*buf = (*buf)[:c]
		select {
		case <-ctx.Done():
			return
		case tc.queue <- buf:
			atomic.AddUint64(&tc.inputBytes, uint64(c))
			if err == io.EOF {
				return
			}
		}
	}
}

func (tc *threadCopy) writeWorker(ctx context.Context, res chan result) {
	size := uint64(0)
	var err error
	defer func() { res <- result{size, err}; close(res) }()
	for {
		select {
		case <-ctx.Done():
			return
		case buf, ok := <-tc.queue:
			if !ok {
				return
			}
			var c int
			if c, err = tc.w.Write(*buf); err != nil {
				return
			}
			atomic.AddUint64(&tc.outputBytes, uint64(c))
			tc.bs.Put(buf)
			size += uint64(c)
		}
	}
}

func (tc *threadCopy) monitorWorker(ctx context.Context, ch chan<- Metrics) {
	start := time.Now()
	prevTime := start
	oldInputBytes := uint64(0)
	oldOutputBytes := uint64(0)
	m := Metrics{}
	speedCalcFunc := func(t time.Time) {
		dur := t.Sub(start)
		inputBytes := atomic.LoadUint64(&tc.inputBytes)
		outputBytes := atomic.LoadUint64(&tc.outputBytes)
		m.Size = uint64(outputBytes)
		m.AvgByteSec = uint64(float64(outputBytes) / dur.Seconds())
		m.InputByteSec = uint64(float64(inputBytes-oldInputBytes) / t.Sub(prevTime).Seconds())
		if m.InputMaxByteSec < m.InputByteSec {
			m.InputMaxByteSec = m.InputByteSec
		}
		oldInputBytes = inputBytes
		m.OutputByteSec = uint64(float64(outputBytes-oldOutputBytes) / t.Sub(prevTime).Seconds())
		if m.OutputMaxByteSec < m.OutputByteSec {
			m.OutputMaxByteSec = m.OutputByteSec
		}
		oldOutputBytes = outputBytes
		m.BufferUsed = uint64(len(tc.queue) * tc.bufSize)
		if m.BufferMaxUsed < m.BufferUsed {
			m.BufferMaxUsed = m.BufferUsed
		}
		prevTime = t
	}
	postFunc := func() {
		select {
		case ch <- m:
		case <-ctx.Done():
		}
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			speedCalcFunc(t)
			postFunc()
		}
	}
}
