package mtp

import (
	"context"
	"fmt"
	"runtime/trace"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/query"
	"github.com/gotd/td/telegram/query/messages"
	"github.com/gotd/td/tg"
)

// SearchAllMyMessages returns the current authorized user messages from chat or
// channel `dlg`.  For each API call, the callback function will be invoked, if
// not nil.
func (c *Client) SearchAllMyMessages(ctx context.Context, dlg Entity, cb func(n int)) ([]messages.Elem, error) {
	return c.SearchAllMessages(ctx, dlg, &tg.InputPeerSelf{}, cb)
}

// SearchAllMessages search messages in the chat or channel `dlg`. It finds ALL
// messages from the person `who`. returns a slice of message.Elem. For each API
// call, the callback function will be invoked, if not nil.
func (c *Client) SearchAllMessages(ctx context.Context, dlg Entity, who tg.InputPeerClass, cb func(n int)) ([]messages.Elem, error) {
	if cached, err := c.cache.Get(cacheKey(dlg.GetID())); err == nil {
		msgs := cached.([]messages.Elem)
		if cb != nil {
			cb(len(msgs))
		}
		return msgs, nil
	}

	ip, err := asInputPeer(dlg)
	if err != nil {
		return nil, err
	}

	bld := query.Messages(c.cl.API()).
		Search(ip).
		BatchSize(defBatchSize).
		FromID(who).
		Filter(&tg.InputMessagesFilterEmpty{})
	elems, err := collectMessages(ctx, bld, cb)
	if err != nil {
		return nil, err
	}

	if err := c.cache.Set(cacheKey(dlg.GetID()), elems); err != nil {
		return nil, err
	}
	return elems, err
}

func (c *Client) DeleteMessages(ctx context.Context, dlg Entity, messages []messages.Elem) (int, error) {
	ctx, task := trace.NewTask(ctx, "DeleteMessages")
	defer task.End()

	ip, err := asInputPeer(dlg)
	if err != nil {
		trace.Log(ctx, "logic", err.Error())
		return 0, err
	}
	ids := splitBy(defBatchSize, messages, func(i int) int { return messages[i].Msg.GetID() })
	trace.Logf(ctx, "logic", "split chunks: %d", len(ids))

	// clearing cache.
	if c.cache.Remove(cacheKey(dlg.GetID())) {
		trace.Log(ctx, "logic", "cache cleared")
	}

	total := 0
	for _, chunk := range ids {
		resp, err := message.NewSender(c.cl.API()).To(ip).Revoke().Messages(ctx, chunk...)
		if err != nil {
			trace.Logf(ctx, "api", "revoke error: %s", err)
			return 0, fmt.Errorf("failed to delete: %w", err)
		}
		total += resp.GetPtsCount()
	}
	trace.Log(ctx, "logic", "ok")
	return total, nil
}

func asInputPeer(ent Entity) (tg.InputPeerClass, error) {
	switch peer := ent.(type) {
	case *tg.Chat:
		return peer.AsInputPeer(), nil
	case *tg.Channel:
		return peer.AsInputPeer(), nil
	default:
		return nil, fmt.Errorf("unsupported input peer type: %T", peer)
	}
	// unreachable
}

// splitBy splits the chunk input of M items to X chunks of `n` items.
// For each element of input, the fn is called, that should return
// the value.
func splitBy[T, S any](n int, input []S, fn func(i int) T) [][]T {
	var out [][]T = make([][]T, 0, len(input)/n)
	var chunk []T
	for i := range input {
		if i > 0 && i%n == 0 {
			out = append(out, chunk)
			chunk = make([]T, 0, n)
		}
		chunk = append(chunk, fn(i))
	}
	if len(chunk) > 0 {
		out = append(out, chunk)
	}
	return out
}

// collectMessages is the copy/pasta from the td/telegram/message package with added
// optional callback function. It creates iterator and collects all elements to
// slice, calling callback function for each iteration, if it's not nil.
func collectMessages(ctx context.Context, b *messages.SearchQueryBuilder, cb func(n int)) ([]messages.Elem, error) {
	iter := b.Iter()
	c, err := iter.Total(ctx)
	if err != nil {
		return nil, fmt.Errorf("get total: %w", err)
	}

	r := make([]messages.Elem, 0, c)
	for iter.Next(ctx) {
		r = append(r, iter.Value())
		if cb != nil {
			cb(1)
		}
	}

	return r, iter.Err()
}
