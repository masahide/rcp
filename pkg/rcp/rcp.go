package rcp

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
/*
	dialAddr   = ""
	output     = ""
	input      = ""
	discard    = false
	listenAddr = "0.0.0.0:1987"
*/
)

// Rcp configs
type Rcp struct {
	MaxBufNum  int
	BufSize    int
	ThreadCopy bool
	Discard    bool
	DialAddr   string
	Output     string
	Input      string
	ListenAddr string
}

func init() {
	/*
		flag.IntVar(&maxBufNum, "maxBufNum", maxBufNum, "Maximum number of buffers (with thread copy mode)")
		flag.IntVar(&bufSize, "bufsize", bufSize, "Buffer size(with thread copy mode)")
		flag.StringVar(&dialAddr, "d", dialAddr, "dial address (ex: 198.51.100.1:1987 )")
		flag.StringVar(&listenAddr, "l", listenAddr, "listen address")
		flag.StringVar(&output, "o", output, "output filename")
		flag.BoolVar(&discard, "discard", discard, "discard output")
		flag.BoolVar(&thread, "t", thread, "thread copy mode")
		flag.StringVar(&input, "i", input, "input filename")
		flag.Parse()
	*/

}

func main() {
	var (
		t    time.Duration
		size int64
		err  error
		rcp  *Rcp
	)
	switch {
	case len(rcp.DialAddr) > 0:
		t, size, err = rcp.send(rcp.Input, rcp.DialAddr)
	case len(rcp.Output) > 0:
		t, size, err = rcp.receiveFile(rcp.ListenAddr, rcp.Output)
	case rcp.Discard:
		t, size, err = rcp.receive(rcp.ListenAddr, ioutil.Discard)
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

func (rcp *Rcp) send(from, to string) (t time.Duration, size int64, err error) {
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
	return rcp.copy(conn, file)
}

func (rcp *Rcp) receiveFile(listen, filename string) (t time.Duration, size int64, err error) {
	var file *os.File
	file, err = os.Create(filename)
	if err != nil {
		return
	}
	defer file.Close()
	return rcp.receive(listen, file)
}

func (rcp *Rcp) receive(listen string, w io.Writer) (t time.Duration, size int64, err error) {
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
	return rcp.copy(w, conn)
}

func (rcp *Rcp) copy(w io.Writer, r io.Reader) (time.Duration, int64, error) {
	now := time.Now()
	c, err := map[bool]func(io.Writer, io.Reader) (int64, error){
		true:  rcp.bufCopy,
		false: io.Copy,
	}[rcp.ThreadCopy](w, r)
	t := time.Since(now)
	return t, c, err
}

type buffers struct {
	limit chan struct{}
	pool  sync.Pool
}

func newBuffers(size, n int) *buffers {
	bs := buffers{}
	bs.limit = make(chan struct{}, n)
	bs.pool = sync.Pool{New: func() interface{} {
		return make([]byte, size)
	}}
	return &bs
}

func (bs *buffers) Len() int {
	return len(bs.limit)
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

func (rcp *Rcp) bufCopy(w io.Writer, r io.Reader) (int64, error) {
	tc := &threadCopy{
		w:     w,
		r:     r,
		bs:    newBuffers(rcp.BufSize, rcp.MaxBufNum),
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
	size := int64(0)
	var err error
	defer close(tc.queue)
	defer func() { res <- result{int64(size), err} }()
	for {
		var c int
		buf := tc.bs.Get()
		c, err = tc.r.Read(buf)
		size += int64(c)
		if err != nil && err != io.EOF {
			return
		}
		select {
		case <-ctx.Done():
			return
		case tc.queue <- buf[:c]:
			if err == io.EOF {
				return
			}
		}
	}
}

func (tc *threadCopy) writeWorker(ctx context.Context, res chan result) {
	size := int64(0)
	var err error
	defer func() { res <- result{size, err} }()
	for {
		select {
		case <-ctx.Done():
			return
		case buf, ok := <-tc.queue:
			if !ok {
				return
			}
			var c int
			if c, err = tc.w.Write(buf); err != nil {
				return
			}
			tc.bs.Put(buf)
			size += int64(c)
		}
	}
}
