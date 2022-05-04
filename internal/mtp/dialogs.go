package mtp

import (
	"context"
	"errors"
	"runtime/trace"

	"github.com/gotd/contrib/storage"

	"github.com/gotd/td/telegram/query/dialogs"
	"github.com/gotd/td/tg"
)

// GetChats retrieves the account chats.
func (c *Client) GetChats(ctx context.Context) ([]Entity, error) {
	return c.GetEntities(ctx, FilterChat())
}

// GetChannels retrieves the account channels.
func (c *Client) GetChannels(ctx context.Context) ([]Entity, error) {
	return c.GetEntities(ctx, FilterChannel())
}

// GetEntities ensures that storage is populated, then iterates through storage
// peers calling filterFn for each peer. The filterFn should return Entity and
// true, if the peer satisfies the criteria, or nil and false, otherwise.
func (c *Client) GetEntities(ctx context.Context, filterFn FilterFunc) ([]Entity, error) {
	ctx, task := trace.NewTask(ctx, "GetEntities")
	defer task.End()

	if err := c.ensureStoragePopulated(ctx); err != nil {
		return nil, err
	}

	peerIter, err := c.storage.Iterate(ctx)
	if err != nil {
		return nil, err
	}
	defer peerIter.Close()

	var ee []Entity

	for peerIter.Next(ctx) {
		ent, ok := filterFn(peerIter.Value())
		if !ok {
			continue
		}
		ee = append(ee, ent)
	}
	if err := peerIter.Err(); err != nil {
		return nil, err
	}
	return ee, nil
}

// ensureStoragePopulated ensures that the peer storage has been populated within
// defCacheEvict duration.
func (c *Client) ensureStoragePopulated(ctx context.Context) error {
	if cached, err := c.cache.Get(cacheDlgStorage); err == nil && cached.(bool) {
		trace.Log(ctx, "cache", "hit")
		return nil
	}
	// populating the storage
	trace.Log(ctx, "cache", "miss")

	dlgIter := dialogs.NewQueryBuilder(c.cl.API()).
		GetDialogs().
		BatchSize(defBatchSize).
		Iter()
	if err := storage.CollectPeers(c.storage).Dialogs(ctx, dlgIter); err != nil {
		return err
	}
	if err := c.cache.SetWithExpire(cacheDlgStorage, true, defCacheEvict); err != nil {
		return err
	}

	return nil
}

// CreateChat creates a Chat (not a Mega- or Gigagroup).
//
// Example
//
//  if err := cl.CreateChat(ctx, "mtproto-test",123455678, 312849128); err != nil {
//		return err
//	}
func (c *Client) CreateChat(ctx context.Context, title string, userIDs ...int64) error {
	if len(userIDs) == 0 {
		return errors.New("at least one user is required")
	}

	var others = make([]tg.InputUserClass, len(userIDs))
	for i := range userIDs {
		others[i] = &tg.InputUser{UserID: userIDs[i]}
	}

	var users = append([]tg.InputUserClass{&tg.InputUserSelf{}}, others...)

	var resp tg.Updates
	if err := c.cl.Invoke(ctx,
		&tg.MessagesCreateChatRequest{
			Users: users,
			Title: title,
		},
		&resp,
	); err != nil {
		return err
	}
	return nil
}
