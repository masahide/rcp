package rcp

import (
	"fmt"
	"io"
	"net"
	"os"
)

type reciveStream struct {
	ln   net.Listener
	conn net.Conn
}

func reciveStreamOpen(listen string) (*reciveStream, error) {
	rs := &reciveStream{}
	var err error
	if rs.ln, err = net.Listen("tcp", listen); err != nil {
		return nil, err
	}
	fmt.Printf("Listen: %s\n", listen)
	if rs.conn, err = rs.ln.Accept(); err != nil {
		return nil, err
	}
	return rs, nil
}
func (rs *reciveStream) Read(b []byte) (n int, err error) { return rs.conn.Read(b) }
func (rs *reciveStream) Close() error {
	if err := rs.conn.Close(); err != nil {
		return err
	}
	return rs.ln.Close()
}

func fileReadOpen(name string) (io.ReadCloser, error)   { return os.Open(name) }
func fileWriteOpen(name string) (io.WriteCloser, error) { return os.Create(name) }
func dialWriteOpen(name string) (io.WriteCloser, error) { return net.Dial("tcp", name) }

type dummyStream struct {
	size int64
	c    int64
}

func openDummyRead(size int64) *dummyStream { return &dummyStream{size: size} }

func (d *dummyStream) Read(b []byte) (n int, err error) {
	n = len(b)
	if int(d.size-d.c) < n {
		n = int(d.size - d.c)
		err = io.EOF
	}
	copy(b, make([]byte, n))
	d.c += int64(n)
	return
}
func (d *dummyStream) Close() error { return nil }

func openDummyWrite() *dummyStream { return &dummyStream{} }

func (d *dummyStream) Write(p []byte) (n int, err error) { return len(p), nil }
