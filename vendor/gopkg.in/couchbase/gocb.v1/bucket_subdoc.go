package gocb

import (
	"encoding/json"
	"log"

	"github.com/opentracing/opentracing-go"
	"gopkg.in/couchbase/gocbcore.v7"
)

type subDocResult struct {
	path string
	data []byte
	err  error
}

// DocumentFragment represents multiple chunks of a full Document.
type DocumentFragment struct {
	cas      Cas
	mt       MutationToken
	contents []subDocResult
	pathMap  map[string]int
}

// Cas returns the Cas of the Document
func (frag *DocumentFragment) Cas() Cas {
	return frag.cas
}

// MutationToken returns the MutationToken for the change represented by this DocumentFragment.
func (frag *DocumentFragment) MutationToken() MutationToken {
	return frag.mt
}

// ContentByIndex retrieves the value of the operation by its index. The index is the position of
// the operation as it was added to the builder.
func (frag *DocumentFragment) ContentByIndex(idx int, valuePtr interface{}) error {
	res := frag.contents[idx]
	if res.err != nil {
		return res.err
	}
	if valuePtr == nil {
		return nil
	}

	if valuePtr, ok := valuePtr.(*[]byte); ok {
		*valuePtr = res.data
		return nil
	}

	return json.Unmarshal(res.data, valuePtr)
}

// Content retrieves the value of the operation by its path. The path is the path provided
// to the operation
func (frag *DocumentFragment) Content(path string, valuePtr interface{}) error {
	if frag.pathMap == nil {
		frag.pathMap = make(map[string]int)
		for i, v := range frag.contents {
			frag.pathMap[v.path] = i
		}
	}
	return frag.ContentByIndex(frag.pathMap[path], valuePtr)
}

// Exists checks whether the indicated path exists in this DocumentFragment and no
// errors were returned from the server.
func (frag *DocumentFragment) Exists(path string) bool {
	err := frag.Content(path, nil)
	return err == nil
}

// LookupInBuilder is a builder used to create a set of sub-document lookup operations.
type LookupInBuilder struct {
	bucket *Bucket
	opName string
	name   string
	flags  gocbcore.SubdocDocFlag
	ops    []gocbcore.SubDocOp
}

func (set *LookupInBuilder) execute(tracectx opentracing.SpanContext) (*DocumentFragment, error) {
	return set.bucket.lookupIn(tracectx, set)
}

// Execute executes this set of lookup operations on the bucket.
func (set *LookupInBuilder) Execute() (*DocumentFragment, error) {
	return set.execute(nil)
}

// GetEx allows you to perform a sub-document Get operation with flags
func (set *LookupInBuilder) GetEx(path string, flags SubdocFlag) *LookupInBuilder {
	if path == "" {
		op := gocbcore.SubDocOp{
			Op:    gocbcore.SubDocOpGetDoc,
			Flags: gocbcore.SubdocFlag(flags),
		}
		set.ops = append(set.ops, op)
		return set
	}

	op := gocbcore.SubDocOp{
		Op:    gocbcore.SubDocOpGet,
		Path:  path,
		Flags: gocbcore.SubdocFlag(flags),
	}
	set.ops = append(set.ops, op)
	return set
}

// Get indicates a path to be retrieved from the document.  The value of the path
// can later be retrieved (after .Execute()) using the Content or ContentByIndex
// method. The path syntax follows N1QL's path syntax (e.g. `foo.bar.baz`).
func (set *LookupInBuilder) Get(path string) *LookupInBuilder {
	return set.GetEx(path, SubdocFlagNone)
}

// ExistsEx allows you to perform a sub-document Exists operation with flags
func (set *LookupInBuilder) ExistsEx(path string, flags SubdocFlag) *LookupInBuilder {
	op := gocbcore.SubDocOp{
		Op:    gocbcore.SubDocOpExists,
		Path:  path,
		Flags: gocbcore.SubdocFlag(flags),
	}
	set.ops = append(set.ops, op)
	return set
}

// Exists is similar to Get(), but does not actually retrieve the value from the server.
// This may save bandwidth if you only need to check for the existence of a
// path (without caring for its content). You can check the status of this
// operation by using .Content (and ignoring the value) or .Exists()
func (set *LookupInBuilder) Exists(path string) *LookupInBuilder {
	return set.ExistsEx(path, SubdocFlagNone)
}

// GetCountEx allows you to perform a sub-document GetCount operation with flags
func (set *LookupInBuilder) GetCountEx(path string, flags SubdocFlag) *LookupInBuilder {
	op := gocbcore.SubDocOp{
		Op:    gocbcore.SubDocOpGetCount,
		Path:  path,
		Flags: gocbcore.SubdocFlag(flags),
	}
	set.ops = append(set.ops, op)
	return set
}

// GetCount allows you to retrieve the number of items in an array or keys within an
// dictionary within an element of a document.
func (set *LookupInBuilder) GetCount(path string) *LookupInBuilder {
	return set.GetCountEx(path, SubdocFlagNone)
}

func (b *Bucket) lookupIn(tracectx opentracing.SpanContext, set *LookupInBuilder) (resOut *DocumentFragment, errOut error) {
	if tracectx == nil {
		lispan := b.startKvOpTrace(set.opName)
		defer lispan.Finish()
		tracectx = lispan.Context()
	}

	signal := make(chan bool, 1)
	op, err := b.client.LookupInEx(gocbcore.LookupInOptions{
		Key:          []byte(set.name),
		Flags:        set.flags,
		Ops:          set.ops,
		TraceContext: tracectx,
	}, func(res *gocbcore.LookupInResult, err error) {
		errOut = err

		if res != nil {
			resSet := &DocumentFragment{}
			resSet.contents = make([]subDocResult, len(res.Ops))
			resSet.cas = Cas(res.Cas)

			for i, opRes := range res.Ops {
				resSet.contents[i].path = set.ops[i].Path
				resSet.contents[i].err = opRes.Err
				if opRes.Value != nil {
					resSet.contents[i].data = append([]byte(nil), opRes.Value...)
				}
			}

			resOut = resSet
		}

		signal <- true
	})
	if err != nil {
		return nil, err
	}

	timeoutTmr := gocbcore.AcquireTimer(b.opTimeout)
	select {
	case <-signal:
		gocbcore.ReleaseTimer(timeoutTmr, false)
		return
	case <-timeoutTmr.C:
		gocbcore.ReleaseTimer(timeoutTmr, true)
		if !op.Cancel() {
			<-signal
			return
		}
		return nil, ErrTimeout
	}
}

func (b *Bucket) startLookupIn(opName string, key string, flags SubdocDocFlag) *LookupInBuilder {
	return &LookupInBuilder{
		bucket: b,
		name:   key,
		flags:  gocbcore.SubdocDocFlag(flags),
		opName: opName,
	}
}

// LookupInEx creates a sub-document lookup operation builder.
func (b *Bucket) LookupInEx(key string, flags SubdocDocFlag) *LookupInBuilder {
	return b.startLookupIn("LookupInEx", key, flags)
}

// LookupIn creates a sub-document lookup operation builder.
func (b *Bucket) LookupIn(key string) *LookupInBuilder {
	return b.startLookupIn("LookupIn", key, 0)
}

// MutateInBuilder is a builder used to create a set of sub-document mutation operations.
type MutateInBuilder struct {
	bucket    *Bucket
	opName    string
	name      string
	flags     gocbcore.SubdocDocFlag
	cas       gocbcore.Cas
	expiry    uint32
	ops       []gocbcore.SubDocOp
	errs      MultiError
	replicaTo uint
	persistTo uint
}

func (set *MutateInBuilder) execute(tracectx opentracing.SpanContext) (*DocumentFragment, error) {
	return set.bucket.mutateIn(tracectx, set)
}

// Execute executes this set of mutation operations on the bucket.
func (set *MutateInBuilder) Execute() (*DocumentFragment, error) {
	return set.execute(nil)
}

func (set *MutateInBuilder) marshalValue(value interface{}) []byte {
	if value, ok := value.([]byte); ok {
		return value
	}

	if value, ok := value.(*[]byte); ok {
		return *value
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		set.errs.add(err)
		return nil
	}
	return bytes
}

// InsertEx allows you to perform a sub-document Insert operation with flags
func (set *MutateInBuilder) InsertEx(path string, value interface{}, flags SubdocFlag) *MutateInBuilder {
	if path == "" {
		op := gocbcore.SubDocOp{
			Op:    gocbcore.SubDocOpAddDoc,
			Flags: gocbcore.SubdocFlag(flags),
			Value: set.marshalValue(value),
		}
		set.ops = append(set.ops, op)
		return set
	}

	op := gocbcore.SubDocOp{
		Op:    gocbcore.SubDocOpDictAdd,
		Path:  path,
		Flags: gocbcore.SubdocFlag(flags),
		Value: set.marshalValue(value),
	}
	set.ops = append(set.ops, op)
	return set
}

// Insert adds an insert operation to this mutation operation set.
func (set *MutateInBuilder) Insert(path string, value interface{}, createParents bool) *MutateInBuilder {
	var flags SubdocFlag
	if createParents {
		flags |= SubdocFlagCreatePath
	}

	return set.InsertEx(path, value, flags)
}

// UpsertEx allows you to perform a sub-document Upsert operation with flags
func (set *MutateInBuilder) UpsertEx(path string, value interface{}, flags SubdocFlag) *MutateInBuilder {
	if path == "" {
		op := gocbcore.SubDocOp{
			Op:    gocbcore.SubDocOpSetDoc,
			Flags: gocbcore.SubdocFlag(flags),
			Value: set.marshalValue(value),
		}
		set.ops = append(set.ops, op)
		return set
	}

	op := gocbcore.SubDocOp{
		Op:    gocbcore.SubDocOpDictSet,
		Path:  path,
		Flags: gocbcore.SubdocFlag(flags),
		Value: set.marshalValue(value),
	}
	set.ops = append(set.ops, op)
	return set
}

// Upsert adds an upsert operation to this mutation operation set.
func (set *MutateInBuilder) Upsert(path string, value interface{}, createParents bool) *MutateInBuilder {
	var flags SubdocFlag
	if createParents {
		flags |= SubdocFlagCreatePath
	}

	return set.UpsertEx(path, value, flags)
}

// ReplaceEx allows you to perform a sub-document Replace operation with flags
func (set *MutateInBuilder) ReplaceEx(path string, value interface{}, flags SubdocFlag) *MutateInBuilder {
	op := gocbcore.SubDocOp{
		Op:    gocbcore.SubDocOpReplace,
		Path:  path,
		Flags: gocbcore.SubdocFlag(flags),
		Value: set.marshalValue(value),
	}
	set.ops = append(set.ops, op)
	return set
}

// Replace adds an replace operation to this mutation operation set.
func (set *MutateInBuilder) Replace(path string, value interface{}) *MutateInBuilder {
	return set.ReplaceEx(path, value, SubdocFlagNone)
}

func (set *MutateInBuilder) marshalArrayMulti(in interface{}) (out []byte) {
	out, err := json.Marshal(in)
	if err != nil {
		log.Panic(err)
	}

	// Assert first character is a '['
	if len(out) < 2 || out[0] != '[' {
		log.Panic("Not a JSON array")
	}

	out = out[1 : len(out)-1]
	return
}

// RemoveEx allows you to perform a sub-document Remove operation with flags
func (set *MutateInBuilder) RemoveEx(path string, flags SubdocFlag) *MutateInBuilder {
	if path == "" {
		op := gocbcore.SubDocOp{
			Op:    gocbcore.SubDocOpDeleteDoc,
			Flags: gocbcore.SubdocFlag(flags),
		}
		set.ops = append(set.ops, op)
		return set
	}

	op := gocbcore.SubDocOp{
		Op:    gocbcore.SubDocOpDelete,
		Path:  path,
		Flags: gocbcore.SubdocFlag(flags),
	}
	set.ops = append(set.ops, op)
	return set
}

// Remove adds an remove operation to this mutation operation set.
func (set *MutateInBuilder) Remove(path string) *MutateInBuilder {
	return set.RemoveEx(path, SubdocFlagNone)
}

// ArrayPrependEx allows you to perform a sub-document ArrayPrepend operation with flags
func (set *MutateInBuilder) ArrayPrependEx(path string, value interface{}, flags SubdocFlag) *MutateInBuilder {
	return set.arrayPrependValue(path, set.marshalValue(value), flags)
}

// ArrayPrepend adds an element to the beginning (i.e. left) of an array
func (set *MutateInBuilder) ArrayPrepend(path string, value interface{}, createParents bool) *MutateInBuilder {
	var flags SubdocFlag
	if createParents {
		flags |= SubdocFlagCreatePath
	}

	return set.ArrayPrependEx(path, value, flags)
}

func (set *MutateInBuilder) arrayPrependValue(path string, bytes []byte, flags SubdocFlag) *MutateInBuilder {
	op := gocbcore.SubDocOp{
		Op:    gocbcore.SubDocOpArrayPushFirst,
		Path:  path,
		Flags: gocbcore.SubdocFlag(flags),
		Value: bytes,
	}
	set.ops = append(set.ops, op)
	return set
}

// ArrayAppendEx allows you to perform a sub-document ArrayAppend operation with flags
func (set *MutateInBuilder) ArrayAppendEx(path string, value interface{}, flags SubdocFlag) *MutateInBuilder {
	return set.arrayAppendValue(path, set.marshalValue(value), flags)
}

// ArrayAppend adds an element to the end (i.e. right) of an array
func (set *MutateInBuilder) ArrayAppend(path string, value interface{}, createParents bool) *MutateInBuilder {
	var flags SubdocFlag
	if createParents {
		flags |= SubdocFlagCreatePath
	}

	return set.ArrayAppendEx(path, value, flags)
}

func (set *MutateInBuilder) arrayAppendValue(path string, bytes []byte, flags SubdocFlag) *MutateInBuilder {
	op := gocbcore.SubDocOp{
		Op:    gocbcore.SubDocOpArrayPushLast,
		Path:  path,
		Flags: gocbcore.SubdocFlag(flags),
		Value: bytes,
	}
	set.ops = append(set.ops, op)
	return set
}

// ArrayInsertEx allows you to perform a sub-document ArrayInsert operation with flags
func (set *MutateInBuilder) ArrayInsertEx(path string, value interface{}, flags SubdocFlag) *MutateInBuilder {
	return set.arrayInsertValue(path, set.marshalValue(value), flags)
}

// ArrayInsert inserts an element at a given position within an array. The position should be
// specified as part of the path, e.g. path.to.array[3]
func (set *MutateInBuilder) ArrayInsert(path string, value interface{}) *MutateInBuilder {
	return set.ArrayInsertEx(path, value, SubdocFlagNone)
}

func (set *MutateInBuilder) arrayInsertValue(path string, bytes []byte, flags SubdocFlag) *MutateInBuilder {
	op := gocbcore.SubDocOp{
		Op:    gocbcore.SubDocOpArrayInsert,
		Path:  path,
		Flags: gocbcore.SubdocFlag(flags),
		Value: bytes,
	}
	set.ops = append(set.ops, op)
	return set
}

// ArrayAppendMultiEx allows you to perform a sub-document ArrayAppendMulti operation with flags
func (set *MutateInBuilder) ArrayAppendMultiEx(path string, values interface{}, flags SubdocFlag) *MutateInBuilder {
	return set.arrayAppendValue(path, set.marshalArrayMulti(values), flags)
}

// ArrayAppendMulti adds multiple values as elements to an array.
// `values` must be an array type
// ArrayAppendMulti("path", []int{1,2,3,4}, true) =>
//   "path" [..., 1,2,3,4]
//
// This is a more efficient version (at both the network and server levels)
// of doing
// ArrayAppend("path", 1, true).ArrayAppend("path", 2, true).ArrayAppend("path", 3, true)
//
// See ArrayAppend() for more information
func (set *MutateInBuilder) ArrayAppendMulti(path string, values interface{}, createParents bool) *MutateInBuilder {
	var flags SubdocFlag
	if createParents {
		flags |= SubdocFlagCreatePath
	}

	return set.ArrayAppendMultiEx(path, values, flags)
}

// ArrayPrependMultiEx allows you to perform a sub-document ArrayPrependMulti operation with flags
func (set *MutateInBuilder) ArrayPrependMultiEx(path string, values interface{}, flags SubdocFlag) *MutateInBuilder {
	return set.arrayPrependValue(path, set.marshalArrayMulti(values), flags)
}

// ArrayPrependMulti adds multiple values at the beginning of an array.
// See ArrayAppendMulti for more information about multiple element operations
// and ArrayPrepend for the semantics of this operation
func (set *MutateInBuilder) ArrayPrependMulti(path string, values interface{}, createParents bool) *MutateInBuilder {
	var flags SubdocFlag
	if createParents {
		flags |= SubdocFlagCreatePath
	}

	return set.ArrayPrependMultiEx(path, values, flags)
}

// ArrayInsertMultiEx allows you to perform a sub-document ArrayInsertMulti operation with flags
func (set *MutateInBuilder) ArrayInsertMultiEx(path string, values interface{}, flags SubdocFlag) *MutateInBuilder {
	return set.arrayInsertValue(path, set.marshalArrayMulti(values), flags)
}

// ArrayInsertMulti inserts multiple elements at a specified position within the
// array. See ArrayAppendMulti for more information about multiple element
// operations, and ArrayInsert for more information about array insertion operations
func (set *MutateInBuilder) ArrayInsertMulti(path string, values interface{}) *MutateInBuilder {
	return set.ArrayInsertMultiEx(path, values, SubdocFlagNone)
}

// ArrayAddUniqueEx allows you to perform a sub-document ArrayAddUnique operation with flags
func (set *MutateInBuilder) ArrayAddUniqueEx(path string, value interface{}, flags SubdocFlag) *MutateInBuilder {
	op := gocbcore.SubDocOp{
		Op:    gocbcore.SubDocOpArrayAddUnique,
		Path:  path,
		Flags: gocbcore.SubdocFlag(flags),
		Value: set.marshalValue(value),
	}
	set.ops = append(set.ops, op)
	return set
}

// ArrayAddUnique adds an dictionary add unique operation to this mutation operation set.
func (set *MutateInBuilder) ArrayAddUnique(path string, value interface{}, createParents bool) *MutateInBuilder {
	var flags SubdocFlag
	if createParents {
		flags |= SubdocFlagCreatePath
	}

	return set.ArrayAddUniqueEx(path, value, flags)
}

// CounterEx allows you to perform a sub-document Counter operation with flags
func (set *MutateInBuilder) CounterEx(path string, delta int64, flags SubdocFlag) *MutateInBuilder {
	op := gocbcore.SubDocOp{
		Op:    gocbcore.SubDocOpCounter,
		Path:  path,
		Flags: gocbcore.SubdocFlag(flags),
		Value: set.marshalValue(delta),
	}
	set.ops = append(set.ops, op)
	return set
}

// Counter adds an counter operation to this mutation operation set.
func (set *MutateInBuilder) Counter(path string, delta int64, createParents bool) *MutateInBuilder {
	var flags SubdocFlag
	if createParents {
		flags |= SubdocFlagCreatePath
	}

	return set.CounterEx(path, delta, flags)
}

func (b *Bucket) mutateIn(tracectx opentracing.SpanContext, set *MutateInBuilder) (resOut *DocumentFragment, errOut error) {
	// Perform the base operation
	res, err := b.mutateInBase(tracectx, set)
	if err != nil {
		return res, err
	}

	// Skip durability if there was none set
	if set.replicaTo == 0 && set.persistTo == 0 {
		return res, err
	}

	// Attempt to satisfy the durability requirements
	return res, b.durability(tracectx, set.name, res.cas, res.mt, set.replicaTo, set.persistTo, false)
}

func (b *Bucket) mutateInBase(tracectx opentracing.SpanContext, set *MutateInBuilder) (resOut *DocumentFragment, errOut error) {
	if tracectx == nil {
		mispan := b.startKvOpTrace(set.opName)
		defer mispan.Finish()
		tracectx = mispan.Context()
	}

	errOut = set.errs.get()
	if errOut != nil {
		return
	}

	signal := make(chan bool, 1)
	op, err := b.client.MutateInEx(gocbcore.MutateInOptions{
		Key:          []byte(set.name),
		Flags:        set.flags,
		Cas:          set.cas,
		Expiry:       set.expiry,
		Ops:          set.ops,
		TraceContext: tracectx,
	}, func(res *gocbcore.MutateInResult, err error) {
		errOut = err

		if res != nil {
			resSet := &DocumentFragment{
				cas: Cas(res.Cas),
				mt:  MutationToken{res.MutationToken, b},
			}
			resSet.contents = make([]subDocResult, len(res.Ops))

			for i, opRes := range res.Ops {
				resSet.contents[i].path = set.ops[i].Path
				resSet.contents[i].err = opRes.Err
				if opRes.Value != nil {
					resSet.contents[i].data = append([]byte(nil), opRes.Value...)
				}
			}

			resOut = resSet
		}

		signal <- true
	})
	if err != nil {
		return nil, err
	}

	timeoutTmr := gocbcore.AcquireTimer(b.opTimeout)
	select {
	case <-signal:
		gocbcore.ReleaseTimer(timeoutTmr, false)
		return
	case <-timeoutTmr.C:
		gocbcore.ReleaseTimer(timeoutTmr, true)
		if !op.Cancel() {
			<-signal
			return
		}
		return nil, ErrTimeout
	}
}

func (b *Bucket) startMutateIn(opName string, key string, flags SubdocDocFlag, cas Cas, expiry uint32, replicaTo, persistTo uint) *MutateInBuilder {
	return &MutateInBuilder{
		bucket:    b,
		opName:    opName,
		name:      key,
		flags:     gocbcore.SubdocDocFlag(flags),
		cas:       gocbcore.Cas(cas),
		expiry:    expiry,
		replicaTo: replicaTo,
		persistTo: persistTo,
	}
}

// MutateInEx creates a sub-document mutation operation builder.
func (b *Bucket) MutateInEx(key string, flags SubdocDocFlag, cas Cas, expiry uint32) *MutateInBuilder {
	return b.startMutateIn("MutateInEx", key, flags, cas, expiry, 0, 0)
}

// MutateInExDura creates a sub-document mutation operation builder with durability.
func (b *Bucket) MutateInExDura(key string, flags SubdocDocFlag, cas Cas, expiry uint32, replicaTo, persistTo uint) *MutateInBuilder {
	return b.startMutateIn("MutateInExDura", key, flags, cas, expiry, replicaTo, persistTo)
}

// MutateIn creates a sub-document mutation operation builder.
func (b *Bucket) MutateIn(key string, cas Cas, expiry uint32) *MutateInBuilder {
	return b.startMutateIn("MutateIn", key, 0, cas, expiry, 0, 0)
}
