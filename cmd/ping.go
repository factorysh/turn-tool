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
var (
	host   string
	port   int
	realm  string
	id     string
	secret string
	peer   string
)

func init() {
	rootCmd.AddCommand(pingCmd)
	//pingCmd.LocalFlags().BoolVar(&TCP, "tcp", false, "use TURN with tcp")
	pingCmd.PersistentFlags().StringVarP(&host, "host", "H", "", "Host")
	pingCmd.MarkFlagRequired("host")
	pingCmd.PersistentFlags().IntVarP(&port, "port", "p", 3478, "Port")
	pingCmd.PersistentFlags().StringVarP(&realm, "realm", "r", "pion.ly", "Realm")
	pingCmd.PersistentFlags().StringVarP(&id, "id", "i", "bob", "Coturn REST id")
	pingCmd.PersistentFlags().StringVarP(&secret, "secret", "s", "", "Coturn REST secret")
	pingCmd.PersistentFlags().StringVarP(&peer, "peer", "e", "0.0.0.0", "Peer address")
}

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping a TURN server",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(host) == 0 {
			if len(args) >= 1 {
				host = args[0]
			} else {
				return errors.New("host is mandatory")
			}
		}

		turnServerAddr := fmt.Sprintf("%s:%d", host, port)
		username, password, err := buildRestPasswor(id, []byte(secret), 1*time.Hour)
		if err != nil {
			return err
		}
		return pingUDP(turnServerAddr, username, password)
	},
}

func pingUDP(turnServerAddr string, username string, password string) error {

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
	// Allocate a relay socket on the TURN server. On success, it
	// will return a net.PacketConn which represents the remote
	// socket.
	relayConn, err := client.Allocate()
	if err != nil {
		return err
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

	// Start read-loop on pingerConn
	go func() {
		buf := make([]byte, 1600)
		for {
			n, from, pingerErr := pingerConn.ReadFrom(buf)
			if pingerErr != nil {
				fmt.Println("pingError", pingerErr)
				break
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
		for {
			n, from, readerErr := relayConn.ReadFrom(buf)
			if readerErr != nil {
				fmt.Println("readerErr", readerErr)
				break
			}

			// Echo back
			if _, readerErr = relayConn.WriteTo(buf[:n], from); readerErr != nil {
				fmt.Println("readerErr echo back", readerErr)
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
}

func buildRestPasswor(id string, secret []byte, ttl time.Duration) (string, string, error) {
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
