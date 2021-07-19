package ping

import (
	"fmt"
	"net"
	"time"
)

type Ping struct {
	Size   int
	Source net.Addr
	RTT    time.Duration
}

type Pinger struct {
	ip      string
	conn    net.PacketConn
	looping bool
	ping    chan Ping
}

func NewPinger(ip string) *Pinger {
	return &Pinger{
		ip:      ip,
		looping: true,
		ping:    make(chan Ping),
	}
}

func (p *Pinger) Start() error {
	var err error
	p.conn, err = net.ListenPacket("udp4", fmt.Sprintf("%s:0", p.ip))
	if err != nil {
		return err
	}

	buf := make([]byte, 1600)
	for p.looping {
		n, from, pingerErr := p.conn.ReadFrom(buf)
		if pingerErr != nil {
			if !p.looping {
				return pingerErr
			}
			continue
		}

		msg := string(buf[:n])
		sentAt, pingerErr := time.Parse(time.RFC3339Nano, msg)
		if pingerErr != nil {
			return pingerErr
		} else {
			rtt := time.Since(sentAt)
			p.ping <- Ping{
				Size:   n,
				Source: from,
				RTT:    rtt,
			}
		}
	}
	return nil
}

func (p *Pinger) WriteTo(msg []byte, addr net.Addr) (int, error) {
	return p.conn.WriteTo(msg, addr)
}

func (p *Pinger) Close() {
	p.looping = false
	p.conn.Close()
}

func (p *Pinger) Ping() chan Ping {
	return p.ping
}
