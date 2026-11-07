package oauth2

import (
	"net/url"
	"path"
)

// IssuerURL is the dex issuer URL together with the helpers for building paths
// and absolute URLs under its path. Handlers hold this instead of a bare
// url.URL so the "join onto the issuer path" logic lives in one place rather
// than being reimplemented per handler. The embedded url.URL keeps String,
// JoinPath, Query, and the fields available directly.
type IssuerURL struct {
	url.URL
}

// AbsPath joins the given items onto the issuer path, e.g. issuer path "/dex"
// plus "auth" gives "/dex/auth".
func (i IssuerURL) AbsPath(items ...string) string {
	return path.Join(append([]string{i.Path}, items...)...)
}

// AbsURL returns the absolute issuer URL with the given items joined onto its
// path.
func (i IssuerURL) AbsURL(items ...string) string {
	u := i.URL
	u.Path = i.AbsPath(items...)
	return u.String()
}
