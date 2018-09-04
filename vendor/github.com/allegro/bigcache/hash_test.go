package bigcache

type hashStub uint64

func (stub hashStub) Sum64(_ string) uint64 {
	return uint64(stub)
}
