package waipu

import (
	"context"
	"errors"
	"fmt"

	"github.com/rusq/dlog"
	mtp "github.com/rusq/mtpwrap"
	"github.com/schollz/progressbar/v3"
)

func Batch(ctx context.Context, cl Telegramer, ids []int64) error {
	chats, err := cl.GetChats(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if n, err := wipe(ctx, cl, chats, id); err != nil {
			dlog.Printf("SKIPPED: chat %d: error deleting messages %s", id, err)
		} else {
			dlog.Printf("OK: chat: %d: messages deleted: %d", id, n)
		}
	}
	return nil
}

func wipe(ctx context.Context, cl Telegramer, chats []mtp.Entity, id int64) (int, error) {
	idx, err := findIdxOf(chats, id)
	if err != nil {
		return 0, err
	}

	pb := progressbar.New(-1)
	pb.Describe(fmt.Sprintf("scanning %d (%s)", id, chats[idx].GetTitle()))
	pb.RenderBlank()
	messages, err := cl.SearchAllMyMessages(ctx, chats[idx], func(n int) {
		pb.Add(1)
	})
	pb.Finish()
	fmt.Print("\r")
	if err != nil {
		return 0, err
	}

	return cl.DeleteMessages(ctx, chats[idx], messages)
}

func findIdxOf(chats []mtp.Entity, id int64) (int, error) {
	idx := -1
	for i := range chats {
		if chats[i].GetID() == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return 0, errors.New("chat not found")
	}
	return idx, nil
}
