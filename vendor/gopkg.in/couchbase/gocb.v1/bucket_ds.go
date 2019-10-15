package gocb

import "fmt"

// MapGet retrieves a single item from a map document by its key.
func (b *Bucket) MapGet(key, path string, valuePtr interface{}) (Cas, error) {
	tracespan := b.startKvOpTrace("MapGet")
	defer tracespan.Finish()

	frag, err := b.startLookupIn("", key, 0).Get(path).execute(tracespan.Context())
	if err != nil {
		return 0, err
	}
	err = frag.ContentByIndex(0, valuePtr)
	if err != nil {
		return 0, err
	}
	return frag.Cas(), nil
}

// MapRemove removes a specified key from the specified map document.
func (b *Bucket) MapRemove(key, path string) (Cas, error) {
	tracespan := b.startKvOpTrace("MapRemove")
	defer tracespan.Finish()

	frag, err := b.startMutateIn("", key, 0, 0, 0, 0, 0).Remove(path).execute(tracespan.Context())
	if err != nil {
		return 0, err
	}
	return frag.Cas(), nil
}

// MapSize returns the current number of items in a map document.
// PERFORMANCE NOTICE: This currently performs a full document fetch...
func (b *Bucket) MapSize(key string) (uint, Cas, error) {
	var mapContents map[string]interface{}
	cas, err := b.Get(key, &mapContents)
	if err != nil {
		return 0, 0, err
	}

	return uint(len(mapContents)), cas, nil
}

// MapAdd inserts an item to a map document.
func (b *Bucket) MapAdd(key, path string, value interface{}, createMap bool) (Cas, error) {
	for {
		frag, err := b.startMutateIn("MapAdd", key, 0, 0, 0, 0, 0).
			Insert(path, value, false).Execute()
		if err != nil {
			if IsKeyNotFoundError(err) && createMap {
				data := make(map[string]interface{})
				data[path] = value
				cas, err := b.Insert(key, data, 0)
				if err != nil {
					if IsKeyExistsError(err) {
						continue
					}

					return 0, err
				}
				return cas, nil
			}
			return 0, err
		}
		return frag.Cas(), nil
	}
}

// ListGet retrieves an item from a list document by index.
func (b *Bucket) ListGet(key string, index uint, valuePtr interface{}) (Cas, error) {
	frag, err := b.LookupIn(key).Get(fmt.Sprintf("[%d]", index)).Execute()
	if err != nil {
		return 0, err
	}
	err = frag.ContentByIndex(0, valuePtr)
	if err != nil {
		return 0, err
	}
	return frag.Cas(), nil
}

// ListAppend inserts an item to the end of a list document.
func (b *Bucket) ListAppend(key string, value interface{}, createList bool) (Cas, error) {
	for {
		frag, err := b.MutateIn(key, 0, 0).ArrayAppend("", value, false).Execute()
		if err != nil {
			if IsKeyNotFoundError(err) && createList {
				var data []interface{}
				data = append(data, value)
				cas, err := b.Insert(key, data, 0)
				if err != nil {
					if IsKeyExistsError(err) {
						continue
					}

					return 0, err
				}
				return cas, nil
			}
			return 0, err
		}
		return frag.Cas(), nil
	}
}

// ListPrepend inserts an item to the beginning of a list document.
func (b *Bucket) ListPrepend(key string, value interface{}, createList bool) (Cas, error) {
	for {
		frag, err := b.MutateIn(key, 0, 0).ArrayPrepend("", value, false).Execute()
		if err != nil {
			if IsKeyNotFoundError(err) && createList {
				var data []interface{}
				data = append(data, value)
				cas, err := b.Insert(key, data, 0)
				if err != nil {
					if IsKeyExistsError(err) {
						continue
					}

					return 0, err
				}
				return cas, nil
			}
			return 0, err
		}
		return frag.Cas(), nil
	}
}

// ListRemove removes an item from a list document by its index.
func (b *Bucket) ListRemove(key string, index uint) (Cas, error) {
	frag, err := b.MutateIn(key, 0, 0).Remove(fmt.Sprintf("[%d]", index)).Execute()
	if err != nil {
		return 0, err
	}
	return frag.Cas(), nil
}

// ListSet replaces the item at a particular index of a list document.
func (b *Bucket) ListSet(key string, index uint, value interface{}) (Cas, error) {
	frag, err := b.MutateIn(key, 0, 0).Replace(fmt.Sprintf("[%d]", index), value).Execute()
	if err != nil {
		return 0, err
	}
	return frag.Cas(), nil
}

// ListSize returns the current number of items in a list.
// PERFORMANCE NOTICE: This currently performs a full document fetch...
func (b *Bucket) ListSize(key string) (uint, Cas, error) {
	var listContents []interface{}
	cas, err := b.Get(key, &listContents)
	if err != nil {
		return 0, 0, err
	}

	return uint(len(listContents)), cas, nil
}

// SetAdd adds a new value to a set document.
func (b *Bucket) SetAdd(key string, value interface{}, createSet bool) (Cas, error) {
	for {
		frag, err := b.MutateIn(key, 0, 0).ArrayAddUnique("", value, false).Execute()
		if err != nil {
			if IsKeyNotFoundError(err) && createSet {
				var data []interface{}
				data = append(data, value)
				cas, err := b.Insert(key, data, 0)
				if err != nil {
					if IsKeyExistsError(err) {
						continue
					}

					return 0, err
				}
				return cas, nil
			}
			return 0, err
		}
		return frag.Cas(), nil
	}
}

// SetExists checks if a particular value exists within the specified set document.
// PERFORMANCE WARNING: This performs a full set fetch and compare.
func (b *Bucket) SetExists(key string, value interface{}) (bool, Cas, error) {
	var setContents []interface{}
	cas, err := b.Get(key, &setContents)
	if err != nil {
		return false, 0, err
	}

	for _, item := range setContents {
		if item == value {
			return true, cas, nil
		}
	}

	return false, 0, nil
}

// SetSize returns the current number of values in a set.
// PERFORMANCE NOTICE: This currently performs a full document fetch...
func (b *Bucket) SetSize(key string) (uint, Cas, error) {
	var setContents []interface{}
	cas, err := b.Get(key, &setContents)
	if err != nil {
		return 0, 0, err
	}

	return uint(len(setContents)), cas, nil
}

// SetRemove removes a single specified value from the specified set document.
// WARNING: This relies on Go's interface{} comparison behaviour!
// PERFORMANCE WARNING: This performs full set fetch, modify, store cycles.
func (b *Bucket) SetRemove(key string, value interface{}) (Cas, error) {
	for {
		var setContents []interface{}
		cas, err := b.Get(key, &setContents)
		if err != nil {
			return 0, err
		}

		foundItem := false
		newSetContents := make([]interface{}, 0)
		for _, item := range setContents {
			if item == value {
				foundItem = true
			} else {
				newSetContents = append(newSetContents, item)
			}
		}

		if !foundItem {
			return 0, ErrRangeError
		}

		cas, err = b.Replace(key, newSetContents, cas, 0)
		if err != nil {
			if IsKeyExistsError(err) {
				// If this is just a CAS error, try again!
				continue
			}

			return 0, err
		}

		return cas, nil
	}
}

// QueuePush adds a new item to the end of a queue.
func (b *Bucket) QueuePush(key string, value interface{}, createQueue bool) (Cas, error) {
	return b.ListPrepend(key, value, createQueue)
}

// QueuePop pops the oldest item from a queue and returns it.
func (b *Bucket) QueuePop(key string, valuePtr interface{}) (Cas, error) {
	for {
		getFrag, err := b.LookupIn(key).Get("[-1]").Execute()
		if err != nil {
			return 0, err
		}

		rmFrag, err := b.MutateIn(key, getFrag.Cas(), 0).Remove("[-1]").Execute()
		if err != nil {
			if IsKeyExistsError(err) {
				// If this is just a CAS error, try again!
				continue
			}

			return 0, err
		}

		err = getFrag.ContentByIndex(0, valuePtr)
		if err != nil {
			return 0, err
		}

		return rmFrag.Cas(), nil
	}
}

// QueueSize returns the current size of a queue.
func (b *Bucket) QueueSize(key string) (uint, Cas, error) {
	var queueContents []interface{}
	cas, err := b.Get(key, &queueContents)
	if err != nil {
		return 0, 0, err
	}

	return uint(len(queueContents)), cas, nil
}
