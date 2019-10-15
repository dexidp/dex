package gocb

// RemoveMt performs a Remove operation and includes MutationToken in the results.
func (b *Bucket) RemoveMt(key string, cas Cas) (Cas, MutationToken, error) {
	if !b.mtEnabled {
		panic("You must use OpenBucketMt with Mt operation variants.")
	}

	span := b.startKvOpTrace("RemoveMt")
	defer span.Finish()

	return b.remove(span.Context(), key, cas)
}

// UpsertMt performs a Upsert operation and includes MutationToken in the results.
func (b *Bucket) UpsertMt(key string, value interface{}, expiry uint32) (Cas, MutationToken, error) {
	if !b.mtEnabled {
		panic("You must use OpenBucketMt with Mt operation variants.")
	}

	span := b.startKvOpTrace("UpsertMt")
	defer span.Finish()

	return b.upsert(span.Context(), key, value, expiry)
}

// InsertMt performs a Insert operation and includes MutationToken in the results.
func (b *Bucket) InsertMt(key string, value interface{}, expiry uint32) (Cas, MutationToken, error) {
	if !b.mtEnabled {
		panic("You must use OpenBucketMt with Mt operation variants.")
	}

	span := b.startKvOpTrace("InsertMt")
	defer span.Finish()

	return b.insert(span.Context(), key, value, expiry)
}

// ReplaceMt performs a Replace operation and includes MutationToken in the results.
func (b *Bucket) ReplaceMt(key string, value interface{}, cas Cas, expiry uint32) (Cas, MutationToken, error) {
	if !b.mtEnabled {
		panic("You must use OpenBucketMt with Mt operation variants.")
	}

	span := b.startKvOpTrace("ReplaceMt")
	defer span.Finish()

	return b.replace(span.Context(), key, value, cas, expiry)
}

// AppendMt performs a Append operation and includes MutationToken in the results.
func (b *Bucket) AppendMt(key, value string) (Cas, MutationToken, error) {
	if !b.mtEnabled {
		panic("You must use OpenBucketMt with Mt operation variants.")
	}

	span := b.startKvOpTrace("AppendMt")
	defer span.Finish()

	return b.append(span.Context(), key, value)
}

// PrependMt performs a Prepend operation and includes MutationToken in the results.
func (b *Bucket) PrependMt(key, value string) (Cas, MutationToken, error) {
	if !b.mtEnabled {
		panic("You must use OpenBucketMt with Mt operation variants.")
	}

	span := b.startKvOpTrace("PrependMt")
	defer span.Finish()

	return b.prepend(span.Context(), key, value)
}

// CounterMt performs a Counter operation and includes MutationToken in the results.
func (b *Bucket) CounterMt(key string, delta, initial int64, expiry uint32) (uint64, Cas, MutationToken, error) {
	if !b.mtEnabled {
		panic("You must use OpenBucketMt with Mt operation variants.")
	}

	span := b.startKvOpTrace("CounterMt")
	defer span.Finish()

	return b.counter(span.Context(), key, delta, initial, expiry)
}
