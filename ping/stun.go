package ping

import (
	"fmt"

	"github.com/pion/stun"
)

func Stun(addr string) error {
	c, err := stun.Dial("udp4", addr)
	if err != nil {
		return err
	}
	var myError error
	if err = c.Do(stun.MustBuild(stun.TransactionID, stun.BindingRequest), func(res stun.Event) {
		if res.Error != nil {
			myError = res.Error
			return
		}
		var xorAddr stun.XORMappedAddress
		if getErr := xorAddr.GetFrom(res.Message); getErr != nil {
			myError = getErr
			return
		}
		fmt.Println(xorAddr)
	}); err != nil {
		return err
	}
	if myError != nil {
		return myError
	}
	if err := c.Close(); err != nil {
		return err
	}
	return nil
}
