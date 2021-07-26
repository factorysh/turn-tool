package cmd

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/factorysh/turn-tool/parse"
	"github.com/factorysh/turn-tool/ping"
	"github.com/spf13/cobra"
)

//var TCP bool
var (
	realm  string
	id     string
	secret string
	peer   string
	npings int
)

func init() {
	rootCmd.AddCommand(pingCmd)
	//pingCmd.LocalFlags().BoolVar(&TCP, "tcp", false, "use TURN with tcp")
	pingCmd.MarkFlagRequired("host")
	pingCmd.PersistentFlags().StringVarP(&realm, "realm", "r", "", "Realm")
	pingCmd.PersistentFlags().StringVarP(&id, "id", "i", os.Getenv("USER"), "Coturn REST id")
	pingCmd.PersistentFlags().StringVarP(&secret, "secret", "s", "", "Coturn REST secret")
	pingCmd.PersistentFlags().StringVarP(&peer, "peer", "e", "0.0.0.0", "Peer iface or address")
	pingCmd.PersistentFlags().IntVarP(&npings, "number", "n", 5, "Number of ping")
}

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping a TURN server",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("host is mandatory")
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

		for _, urlRaw := range args {
			if !strings.Contains(urlRaw, ":") {
				urlRaw = "turn:" + urlRaw
			}
			u, err := parse.Parse(urlRaw)
			if err != nil {
				return err
			}
			host := u.Host
			if realm == "" {
				realm = host
			}
			var port int
			if u.Port == "" {
				port = 3478
			} else {
				port64, err := strconv.ParseInt(u.Port, 10, 32)
				if err != nil {
					return err
				}
				port = int(port64)
			}
			addr := fmt.Sprintf("%s:%d", host, port)
			fmt.Printf("\n#%s\n\n", u.Scheme)
			switch u.Scheme {
			case "turn":
				username, password, err := buildRestPasswor(id, []byte(secret), 1*time.Hour)
				if err != nil {
					return err
				}
				if u.Transport == "udp" {

					err = ping.PingUDP(peer, realm, addr, "", "", 1)
					if err == nil {
						return errors.New("server Authentifcation is mandatory")
					}
					if err.Error() != "turn allocate error : Allocate error response (error 401: Unauthorized)" {
						return err
					}
					err = ping.PingUDP(peer, realm, addr, username, password, npings)
					if err != nil {
						return err
					}
				} else {
					err = ping.PingTCP(peer, realm, addr, username, password, npings)
					if err != nil {
						return err
					}
				}
			case "stun":
				err = ping.Stun(addr)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("wrong scheme : %s", u.Scheme)
			}
		}
		return nil
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
