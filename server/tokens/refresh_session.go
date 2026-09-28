package tokens

import "github.com/dexidp/dex/storage"

// RefreshReferenceSessionID returns the browser session ID recorded on an
// offline session's refresh reference for clientID. The empty string means
// there is no reference, or the reference carries no sid (token minted outside
// a browser flow). The refresh grant and token introspection both read the sid
// through this helper so they cannot disagree about which session a token names.
func RefreshReferenceSessionID(offline storage.OfflineSessions, clientID string) string {
	ref, ok := offline.Refresh[clientID]
	if !ok || ref == nil {
		return ""
	}
	return ref.SessionID
}
