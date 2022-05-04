//go:build !windows

package authflow

import "io"

func clrscr(w io.Writer) {
	w.Write([]byte("\033c"))
}
