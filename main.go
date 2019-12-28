package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

var (
	maxBufNum  = 10 * 1024
	bufSize    = 10 * 1024 * 1024 // 10MByte
	dialAddr   = ""
	output     = ""
	input      = ""
	thread     = false
	discard    = false
	listenAddr = "0.0.0.0:1987"
	copyFunc   = map[bool]func(io.Writer, io.Reader) (int64, error){
		true:  bufCopy,
		false: io.Copy,
	}
)

func init() {
	flag.IntVar(&maxBufNum, "maxBufNum", maxBufNum, "Maximum number of buffers (with thread copy mode)")
	flag.IntVar(&bufSize, "bufsize", bufSize, "Buffer size(with thread copy mode)")
	flag.StringVar(&dialAddr, "d", dialAddr, "dial address (ex: 198.51.100.1:1987 )")
	flag.StringVar(&listenAddr, "l", listenAddr, "listen address")
	flag.StringVar(&output, "o", output, "output filename")
	flag.BoolVar(&discard, "discard", discard, "discard output")
	flag.BoolVar(&thread, "t", thread, "thread copy mode")
	flag.StringVar(&input, "i", input, "input filename")
	flag.Parse()

}

func main() {
	var (
		t    time.Duration
		size int64
		err  error
	)
	switch {
	case len(dialAddr) > 0:
		t, size, err = send(input, dialAddr)
	case len(output) > 0:
		t, size, err = receiveFile(listenAddr, output)
	case discard:
		t, size, err = receive(listenAddr, ioutil.Discard)
	default:
		flag.PrintDefaults()
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, "time:  %v\n", t)
	fmt.Fprintf(os.Stderr, "size:  %v\n", size)
	fmt.Fprintf(os.Stderr, "%.0f Byte/sec  (%.0f bit/sec)\n", float64(size)/t.Seconds(), float64(size)*8/t.Seconds())
	fmt.Fprintf(os.Stderr, "%.4f MByte/sec (%.4f Mbit/sec)\n", float64(size)/t.Seconds()/1024/1024, float64(size)*8/t.Seconds()/1024/1024)
	fmt.Fprintf(os.Stderr, "%.4f GByte/sec (%.4f Gbit/sec)\n", float64(size)/t.Seconds()/1024/1024/1024, float64(size)*8/t.Seconds()/1024/1024/1024)
}

func send(from, to string) (t time.Duration, size int64, err error) {
	var conn net.Conn
	conn, err = net.Dial("tcp", to)
	if err != nil {
		return
	}
	defer conn.Close()
	var file *os.File
	file, err = os.Open(from)
	if err != nil {
		return
	}
	defer file.Close()
	return copy(conn, file)
}

func receiveFile(listen, filename string) (t time.Duration, size int64, err error) {
	var file *os.File
	file, err = os.Create(filename)
	if err != nil {
		return
	}
	defer file.Close()
	return receive(listen, file)
}

func receive(listen string, w io.Writer) (t time.Duration, size int64, err error) {
	var ln net.Listener
	ln, err = net.Listen("tcp", listen)
	if err != nil {
		return
	}
	defer ln.Close()
	fmt.Printf("Listen: %s\n", listen)
	var conn net.Conn
	conn, err = ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	return copy(w, conn)
}

func copy(w io.Writer, r io.Reader) (time.Duration, int64, error) {
	now := time.Now()
	c, err := copyFunc[thread](w, r)
	t := time.Since(now)
	return t, c, err
}

type buffers struct {
	limit chan struct{}
	pool  sync.Pool
}

func newBuffers(n int) *buffers {
	bs := buffers{}
	bs.limit = make(chan struct{}, n)
	bs.pool = sync.Pool{New: func() interface{} {
		return make([]byte, bufSize)
	}}
	return &bs
}

func (bs *buffers) Get() []byte {
	bs.limit <- struct{}{} // 空くまで待つ
	return bs.pool.Get().([]byte)
}

func (bs *buffers) Put(b []byte) {
	bs.pool.Put(b)
	<-bs.limit // 解放
}

type threadCopy struct {
	queue chan []byte
	bs    *buffers
	r     io.Reader
	w     io.Writer
}

type result struct {
	size int64
	err  error
}

func bufCopy(w io.Writer, r io.Reader) (int64, error) {
	tc := &threadCopy{
		w:     w,
		r:     r,
		bs:    newBuffers(maxBufNum),
		queue: make(chan []byte),
	}
	ctx := context.Background()
	rResChan := make(chan result)
	wResChan := make(chan result)
	go tc.readWorker(ctx, rResChan)
	go tc.writeWorker(ctx, wResChan)
	rRes := <-rResChan
	if rRes.err != nil {
		log.Fatal(rRes.err)
	}
	wRes := <-wResChan
	if wRes.err != nil {
		log.Fatal(wRes.err)
	}
	close(rResChan)
	close(wResChan)
	return wRes.size, nil
}

func (tc *threadCopy) readWorker(ctx context.Context, res chan result) {
	defer close(tc.queue)
	eof := false
	size := int64(0)
	for {
		buf := tc.bs.Get()
		c, err := tc.r.Read(buf)
		size += int64(c)
		if err != nil {
			if err != io.EOF {
				res <- result{int64(size), err}
				return
			}
			eof = true
		}
		select {
		case <-ctx.Done():
			res <- result{int64(size), ctx.Err()}
			return
		case tc.queue <- buf[:c]:
			if eof {
				res <- result{int64(size), nil}
				return
			}
		}
	}
}

func (tc *threadCopy) writeWorker(ctx context.Context, res chan result) {
	size := int64(0)
	for {
		select {
		case <-ctx.Done():
			res <- result{size, ctx.Err()}
			return
		case buf, ok := <-tc.queue:
			if !ok {
				res <- result{size, nil}
				return
			}
			c, err := tc.w.Write(buf)
			if err != nil {
				res <- result{size, err}
				return
			}
			tc.bs.Put(buf)
			size += int64(c)
		}
	}
}
