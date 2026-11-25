//go:build linux

package mcu

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// Client owns the MCU tty: one reader goroutine and a serialized writer that
// waits for the ACK (300 ms) and retries up to 10 times, as the vendor does.
type Client struct {
	f     *os.File
	log   *slog.Logger
	wmu   sync.Mutex
	acks  chan struct{}
	mu    sync.Mutex
	temps map[byte]float64
}

func Open(dev string, log *slog.Logger) (*Client, error) {
	f, err := os.OpenFile(dev, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, err
	}
	if err := makeRaw(f.Fd()); err != nil {
		f.Close()
		return nil, fmt.Errorf("mcu: configure %s: %w", dev, err)
	}
	c := &Client{f: f, log: log, acks: make(chan struct{}, 4), temps: map[byte]float64{}}
	go c.readLoop()
	return c, nil
}

// Linux termios bits not exported by package syscall (same on arm and amd64).
const (
	crtscts = 0x80000000
	cbaud   = 0x100f
)

// makeRaw sets 19200 8N1, raw, no flow control, VMIN=1 VTIME=0.
func makeRaw(fd uintptr) error {
	var t syscall.Termios
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TCGETS, uintptr(unsafe.Pointer(&t))); e != 0 {
		return e
	}
	t.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON | syscall.IXOFF
	t.Oflag &^= syscall.OPOST
	t.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	t.Cflag &^= syscall.CSIZE | syscall.PARENB | syscall.CSTOPB | crtscts | cbaud
	t.Cflag |= syscall.CS8 | syscall.CREAD | syscall.CLOCAL | syscall.B19200
	t.Ispeed, t.Ospeed = syscall.B19200, syscall.B19200
	t.Cc[syscall.VMIN], t.Cc[syscall.VTIME] = 1, 0
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TCSETS, uintptr(unsafe.Pointer(&t))); e != 0 {
		return e
	}
	return nil
}

func (c *Client) readLoop() {
	var p Parser
	buf := make([]byte, 64)
	for {
		n, err := c.f.Read(buf)
		if err != nil {
			c.log.Error("mcu read", "err", err)
			time.Sleep(time.Second)
			continue
		}
		for _, fr := range p.Feed(buf[:n]) {
			switch {
			case IsAck(fr):
				select {
				case c.acks <- struct{}{}:
				default:
				}
			default:
				if sel, t, ok := ThermistorReply(fr); ok {
					c.mu.Lock()
					c.temps[sel] = t
					c.mu.Unlock()
				} else {
					c.log.Debug("mcu frame", "bytes", fmt.Sprintf("% X", fr))
				}
			}
		}
	}
}

func (c *Client) Send(frame []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	for len(c.acks) > 0 {
		<-c.acks
	}
	for try := 0; try < 10; try++ {
		if _, err := c.f.Write(frame); err != nil {
			return err
		}
		select {
		case <-c.acks:
			return nil
		case <-time.After(300 * time.Millisecond):
		}
	}
	return ErrNoAck
}

// Thermistor sends a query and waits briefly for the asynchronous reply.
func (c *Client) Thermistor(sel byte) (float64, bool) {
	if c.Send(GetThermistor(sel)) != nil {
		return 0, false
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		t, ok := c.temps[sel]
		delete(c.temps, sel)
		c.mu.Unlock()
		if ok {
			return t, true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return 0, false
}
