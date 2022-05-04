package mtp

import "github.com/gotd/contrib/storage"

type FilterFunc func(storage.Peer) (ent Entity, ok bool)

func FilterChat() FilterFunc {
	return func(peer storage.Peer) (Entity, bool) {
		if peer.Chat != nil {
			return peer.Chat, true
		} else if peer.Channel != nil && !peer.Channel.Broadcast {
			return peer.Channel, true
		}
		return nil, false
	}
}

func FilterChannel() FilterFunc {
	return func(peer storage.Peer) (Entity, bool) {
		if peer.Channel != nil && peer.Channel.Broadcast {
			return peer.Channel, true
		}
		return nil, false
	}
}
