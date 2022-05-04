package sys

import (
	"errors"
	"net"
)

// Errors.
var (
	ErrNetNoIface = errors.New("unable find a suitable interface")
)

// FindIface returns name of the  first active interface with non-nil MAC
// address.
func FindIface() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, i := range interfaces {
		if i.Flags&net.FlagUp != 0 && len(i.HardwareAddr) > 0 {
			return i.Name, nil
		}
	}
	return "", ErrNetNoIface
}

// IfaceMAC returns the MAC address for the Interface.
func IfaceMAC(name string) (net.HardwareAddr, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}
	return iface.HardwareAddr, nil
}
