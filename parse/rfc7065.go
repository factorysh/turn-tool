package parse

import (
	"fmt"
	"regexp"
)

type URI struct {
	Scheme    string
	Host      string
	Port      string
	Transport string
}

var parseRegexp *regexp.Regexp

func init() {
	parseRegexp = regexp.MustCompile(`(?P<scheme>turn|turns|stun):(?P<host>[a-zA-Z0-9\-.]+)(:(?P<port>\d+))?(\?transport=(?P<transport>udp|tcp))?`)
}

func Parse(raw string) (*URI, error) {
	s := parseRegexp.FindStringSubmatch(raw)
	if len(s) == 0 {
		return nil, fmt.Errorf("wrong uri : %s", raw)
	}
	return &URI{
		Scheme:    s[parseRegexp.SubexpIndex("scheme")],
		Host:      s[parseRegexp.SubexpIndex("host")],
		Port:      s[parseRegexp.SubexpIndex("port")],
		Transport: s[parseRegexp.SubexpIndex("transport")],
	}, nil

}
