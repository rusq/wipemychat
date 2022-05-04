package sys

import (
	"bytes"
	"testing"
)

var testMAC = []byte{0, 0, 0, 0, 0, 0}

func TestFindIface(t *testing.T) {
	t.Run("normal run", func(t *testing.T) {
		nif, err := FindIface()
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if nif == "" {
			t.Errorf("expected interface name, got: %q", nif)
		}
	})
}

func TestIfaceMAC(t *testing.T) {
	t.Run("mac for real interface", func(t *testing.T) {
		nif, err := FindIface()
		if err != nil {
			t.Errorf("FindIface() unexpected error: %s", err)
		}
		mac, err := IfaceMAC(nif)
		if err != nil {
			t.Fatalf("IfaceMAC() unexpected error: %s", err)
		}
		if len(mac) == 0 || bytes.Equal(testMAC, mac) {
			t.Errorf("unexpected mac value: %v", mac)
		}
	})
}
