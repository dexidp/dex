package tokens

import (
	"context"
	"slices"

	"github.com/dexidp/dex/storage"
)

// CrossClientTrusted reports whether clientID may request an audience scope for
// peerID: true when clientID is peerID itself, or when peerID lists clientID as
// a trusted peer.
func CrossClientTrusted(ctx context.Context, s storage.Storage, clientID, peerID string) (bool, error) {
	if peerID == clientID {
		return true, nil
	}
	peer, err := s.GetClient(ctx, peerID)
	if err != nil {
		if err == storage.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return slices.Contains(peer.TrustedPeers, clientID), nil
}
