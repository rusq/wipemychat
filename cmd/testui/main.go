// Command testui emulates the work of the Text UI for making screenshots
package main

import (
	"context"
	"math/rand"
	"sort"
	"time"

	"github.com/gotd/td/tdp"
	"github.com/gotd/td/telegram/query/messages"
	"github.com/rusq/dlog"

	"github.com/rusq/wipemychat/internal/mtp"
	"github.com/rusq/wipemychat/internal/tui"
)

const (
	fakeSearchDelay      = 500 * time.Microsecond
	fakeDeleteMultiplier = 500 * time.Microsecond
	maxFakeMessages      = 5000
)

func init() {
	rand.Seed(time.Now().Unix())
}

func main() {
	chats := generateChats(fakechats)
	app := tui.New(FakeTelegram{chats: chats})

	if err := app.Run(context.Background(), chats); err != nil {
		dlog.Fatal(err)
	}

}

type FakeChat struct {
	id       int64
	title    string
	typeInfo string
}

func (f FakeChat) GetID() int64 {
	return f.id
}

func (f FakeChat) GetTitle() string {
	return f.title
}

func (f FakeChat) TypeInfo() tdp.Type {
	return tdp.Type{Name: f.typeInfo}
}

func (f FakeChat) Zero() bool {
	return f.title == "" && f.id == 0 && f.typeInfo == ""
}

var fakechats = []string{
	"Get to the Chopper",
	"Kelly Green",
	"Invest with us, quickly!",
	"NFT: pay $$$ get JPG",
	"Biohacking: your butt",
	"Crypto mining: y u no mine",
	"🔞 18+ LINQ expressions in C#",
	"Everything you need to know about everything you need to know about",
	"Dumbass: Breaking News",
	"Slackdump",
}

func generateChats(titles []string) []mtp.Entity {
	sort.Strings(titles)
	var ret = make([]mtp.Entity, len(titles))
	for i := range titles {
		ret[i] = FakeChat{
			title:    titles[i],
			id:       rand.Int63(),
			typeInfo: randType(),
		}
	}
	return ret
}

func randType() string {
	if rand.Int()%8 == 0 {
		return "chat"
	}
	return "channel"
}

type FakeTelegram struct {
	chats []mtp.Entity
}

func (ft FakeTelegram) GetChats(ctx context.Context) ([]mtp.Entity, error) {
	return ft.chats, nil
}
func (FakeTelegram) SearchAllMyMessages(ctx context.Context, dlg mtp.Entity, cb func(n int)) ([]messages.Elem, error) {
	var n = rand.Int() % maxFakeMessages
	var ret = make([]messages.Elem, n)
	for i := 0; i < len(ret); i++ {
		cb(1)
		time.Sleep(fakeSearchDelay)
	}
	return ret, nil
}
func (FakeTelegram) DeleteMessages(ctx context.Context, dlg mtp.Entity, messages []messages.Elem) (int, error) {
	time.Sleep(time.Duration(len(messages)) * fakeDeleteMultiplier)
	return len(messages), nil
}
