package waipu

import (
	"context"
	"fmt"
	"io"
	"sort"
)

func List(ctx context.Context, w io.Writer, cl Telegramer) error {
	chats, err := cl.GetChats(ctx)
	if err != nil {
		return err
	}
	sort.Slice(chats, func(i, j int) bool {
		return chats[i].GetTitle() < chats[j].GetTitle()
	})
	for _, chat := range chats {
		if _, err := fmt.Fprintf(w, "%15d - %s\n", chat.GetID(), chat.GetTitle()); err != nil {
			return err
		}
	}
	return nil
}
