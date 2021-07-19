package ping

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/pion/logging"
	"github.com/pion/turn/v2"
)

func PingUDP(peer string, realm string, turnServerAddr string, username string, password string, npings int) error {

	// TURN client won't create a local listening socket by itself.
	conn, err := net.ListenPacket("udp4", fmt.Sprintf("%s:0", peer))
	if err != nil {
		return err
	}
	defer conn.Close()
	fmt.Println("conn", conn.LocalAddr().String())

	cfg := &turn.ClientConfig{
		STUNServerAddr: turnServerAddr,
		TURNServerAddr: turnServerAddr,
		Conn:           conn,
		Username:       username,
		Password:       password,
		Realm:          realm,
		Software:       "TurnTool",
		LoggerFactory:  logging.NewDefaultLoggerFactory(),
	}

	fmt.Println("Realm", realm)

	fmt.Printf(`
{
	"username" : "%s",
	"password" : "%s",
	"uris" : [
	  "turn:%s?transport=udp"
	]
}
`, username, password, turnServerAddr)

	client, err := turn.NewClient(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	// Start listening on the conn provided.
	err = client.Listen()
	if err != nil {
		return err
	}

	mappedAddr, err := client.SendBindingRequest()
	if err != nil {
		return err
	}
	fmt.Println("mappedAddr", mappedAddr)

	// Allocate a relay socket on the TURN server. On success, it
	// will return a net.PacketConn which represents the remote
	// socket.
	relayConn, err := client.Allocate()
	if err != nil {
		return fmt.Errorf("turn allocate error : %s", err.Error())
	}
	defer relayConn.Close()
	fmt.Println("relayConn", relayConn.LocalAddr().String())

	// Set up pinger socket (pingerConn)
	pingerConn, err := net.ListenPacket("udp4", fmt.Sprintf("%s:0", peer))
	if err != nil {
		return err
	}
	defer pingerConn.Close()
	fmt.Println("pingerConn", pingerConn.LocalAddr().String())

	// Punch a UDP hole for the relayConn by sending a data to the mappedAddr.
	// This will trigger a TURN client to generate a permission request to the
	// TURN server. After this, packets from the IP address will be accepted by
	// the TURN server.
	_, err = relayConn.WriteTo([]byte("Hello"), mappedAddr)
	if err != nil {
		return err
	}

	looping := true
	// Start read-loop on pingerConn
	go func() {
		buf := make([]byte, 1600)
		for looping {
			n, from, pingerErr := pingerConn.ReadFrom(buf)
			if pingerErr != nil {
				fmt.Println("pingError", pingerErr)
				if !looping {
					return
				}
				continue
			}

			msg := string(buf[:n])
			sentAt, pingerErr := time.Parse(time.RFC3339Nano, msg)
			if pingerErr != nil {
				fmt.Println("parse error", pingerErr)
			} else {
				rtt := time.Since(sentAt)
				log.Printf("%d bytes from from %s time=%d ms\n", n, from.String(), int(rtt.Seconds()*1000))
			}
		}
	}()

	// Start read-loop on relayConn
	go func() {
		buf := make([]byte, 1600)
		for looping {
			n, from, readerErr := relayConn.ReadFrom(buf)
			if readerErr != nil {
				if !looping {
					return
				}
				fmt.Println("readerErr", readerErr)
				continue
			}

			// Echo back
			if _, readerErr = relayConn.WriteTo(buf[:n], from); readerErr != nil {
				fmt.Println("readerErr echo back", readerErr)
				break
			}
		}
	}()

	// Send 10 packets from relayConn to the echo server
	for i := 0; i < npings; i++ {
		msg := time.Now().Format(time.RFC3339Nano)
		_, err = pingerConn.WriteTo([]byte(msg), relayConn.LocalAddr())
		if err != nil {
			return err
		}

		// For simplicity, this example does not wait for the pong (reply).
		// Instead, sleep 1 second.
		time.Sleep(time.Second)
	}
	looping = false

	return nil
}
