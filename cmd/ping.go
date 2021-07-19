package cmd

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/factorysh/turn-tool/ping"
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
	npings int
)

func init() {
	rootCmd.AddCommand(pingCmd)
	//pingCmd.LocalFlags().BoolVar(&TCP, "tcp", false, "use TURN with tcp")
	pingCmd.PersistentFlags().StringVarP(&host, "host", "H", "", "Host")
	pingCmd.MarkFlagRequired("host")
	pingCmd.PersistentFlags().IntVarP(&port, "port", "p", 3478, "Port")
	pingCmd.PersistentFlags().StringVarP(&realm, "realm", "r", "", "Realm")
	pingCmd.PersistentFlags().StringVarP(&id, "id", "i", os.Getenv("USER"), "Coturn REST id")
	pingCmd.PersistentFlags().StringVarP(&secret, "secret", "s", "", "Coturn REST secret")
	pingCmd.PersistentFlags().StringVarP(&peer, "peer", "e", "0.0.0.0", "Peer iface or address")
	pingCmd.PersistentFlags().IntVarP(&npings, "number", "n", 10, "Number of ping")
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

		if realm == "" {
			realm = host
		}

		ip := net.ParseIP(peer)
		if ip == nil {
			iface, err := net.InterfaceByName(peer)
			if err != nil {
				return err
			}
			addrs, err := iface.Addrs()
			if err != nil {
				return err
			}
			if len(addrs) > 1 {
				return fmt.Errorf("iface %s has more than one address %v", peer, addrs)
			}
			if len(addrs) == 0 {
				return fmt.Errorf("iface %s has noaddress", peer)
			}
			peer = strings.Split(addrs[0].String(), "/")[0]
		}

		turnServerAddr := fmt.Sprintf("%s:%d", host, port)
		username, password, err := buildRestPasswor(id, []byte(secret), 1*time.Hour)
		if err != nil {
			return err
		}
		return ping.PingUDP(peer, realm, turnServerAddr, username, password, npings)
	},
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
