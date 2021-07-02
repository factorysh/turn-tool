package cmd

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/pion/logging"
	"github.com/pion/turn/v2"
	"github.com/spf13/cobra"
)

//var TCP bool
var Host string
var Port int
var Realm string
var Id string
var Secret string

func init() {
	//pingCmd.LocalFlags().BoolVar(&TCP, "tcp", false, "use TURN with tcp")
	pingCmd.LocalFlags().StringVar(&Host, "host", "", "Host")
	pingCmd.MarkFlagRequired("host")
	pingCmd.LocalFlags().IntVar(&Port, "port", 3478, "Port")
	pingCmd.LocalFlags().StringVar(&Realm, "realm", "pion.ly", "Realm")
	pingCmd.LocalFlags().StringVar(&Id, "id", "bob", "Coturn REST id")
	pingCmd.LocalFlags().StringVar(&Secret, "secret", "", "Coturn REST secret")
	rootCmd.AddCommand(pingCmd)
}

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping a TURN server",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(Host) == 0 {
			fmt.Println(args)
			if len(args) >= 1 {
				Host = args[0]
			} else {
				return errors.New("host is mandatory")
			}
		}

		// TURN client won't create a local listening socket by itself.
		conn, err := net.ListenPacket("udp4", "0.0.0.0:0")
		if err != nil {
			return err
		}
		defer conn.Close()

		turnServerAddr := fmt.Sprintf("%s:%d", Host, Port)
		username, password, err := BuildRestPasswor(Id, []byte(Secret), 10*time.Minute)
		if err != nil {
			return err
		}

		cfg := &turn.ClientConfig{
			STUNServerAddr: turnServerAddr,
			TURNServerAddr: turnServerAddr,
			Conn:           conn,
			Username:       username,
			Password:       password,
			Realm:          Realm,
			LoggerFactory:  logging.NewDefaultLoggerFactory(),
		}

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
		// Allocate a relay socket on the TURN server. On success, it
		// will return a net.PacketConn which represents the remote
		// socket.
		relayConn, err := client.Allocate()
		if err != nil {
			return err
		}
		defer relayConn.Close()

		// Set up pinger socket (pingerConn)
		pingerConn, err := net.ListenPacket("udp4", "0.0.0.0:0")
		defer pingerConn.Close()

		// Start read-loop on pingerConn
		go func() {
			buf := make([]byte, 1600)
			for {
				n, from, pingerErr := pingerConn.ReadFrom(buf)
				if pingerErr != nil {
					break
				}

				msg := string(buf[:n])
				if sentAt, pingerErr := time.Parse(time.RFC3339Nano, msg); pingerErr == nil {
					rtt := time.Since(sentAt)
					log.Printf("%d bytes from from %s time=%d ms\n", n, from.String(), int(rtt.Seconds()*1000))
				}
			}
		}()

		// Start read-loop on relayConn
		go func() {
			buf := make([]byte, 1600)
			for {
				n, from, readerErr := relayConn.ReadFrom(buf)
				if readerErr != nil {
					break
				}

				// Echo back
				if _, readerErr = relayConn.WriteTo(buf[:n], from); readerErr != nil {
					break
				}
			}
		}()

		time.Sleep(500 * time.Millisecond)

		// Send 10 packets from relayConn to the echo server
		for i := 0; i < 10; i++ {
			msg := time.Now().Format(time.RFC3339Nano)
			_, err = pingerConn.WriteTo([]byte(msg), relayConn.LocalAddr())
			if err != nil {
				return err
			}

			// For simplicity, this example does not wait for the pong (reply).
			// Instead, sleep 1 second.
			time.Sleep(time.Second)
		}

		return nil
	},
}

func BuildRestPasswor(id string, secret []byte, ttl time.Duration) (string, string, error) {
	//https://stackoverflow.com/questions/35766382/coturn-how-to-use-turn-rest-api#54725092
	if ttl <= 0 {
		return "", "", errors.New("use a TTL > 0")
	}
	username := fmt.Sprintf("%d:%s", time.Now().Add(ttl).Unix(), id)
	mac := hmac.New(sha1.New, secret)
	_, err := mac.Write([]byte(username))
	if err != nil {
		return "", "", err
	}
	credential := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return username, credential, nil

}
