package waipu

import (
	"context"

	"github.com/gotd/td/telegram/query/messages"
	mtp "github.com/rusq/mtpwrap"
)

type Telegramer interface {
	GetChats(ctx context.Context) ([]mtp.Entity, error)
	SearchAllMyMessages(ctx context.Context, dlg mtp.Entity, cb func(n int)) ([]messages.Elem, error)
	DeleteMessages(ctx context.Context, dlg mtp.Entity, messages []messages.Elem) (int, error)
}
