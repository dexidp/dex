package pkiutil

import "context"

type key struct{}

var dnKey = key{}

func ContextWithDistinguishedName(parent context.Context, dn string) context.Context {
	return context.WithValue(parent, dnKey, dn)
}

func DistinguishedNameFromContext(ctx context.Context) (dn string, ok bool) {
	u, ok := ctx.Value(dnKey).(string)
	return u, ok
}
