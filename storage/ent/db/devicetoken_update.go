// Code generated by entc, DO NOT EDIT.

package db

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/dexidp/dex/storage/ent/db/devicetoken"
	"github.com/dexidp/dex/storage/ent/db/predicate"
)

// DeviceTokenUpdate is the builder for updating DeviceToken entities.
type DeviceTokenUpdate struct {
	config
	hooks    []Hook
	mutation *DeviceTokenMutation
}

// Where appends a list predicates to the DeviceTokenUpdate builder.
func (dtu *DeviceTokenUpdate) Where(ps ...predicate.DeviceToken) *DeviceTokenUpdate {
	dtu.mutation.Where(ps...)
	return dtu
}

// SetDeviceCode sets the "device_code" field.
func (dtu *DeviceTokenUpdate) SetDeviceCode(s string) *DeviceTokenUpdate {
	dtu.mutation.SetDeviceCode(s)
	return dtu
}

// SetStatus sets the "status" field.
func (dtu *DeviceTokenUpdate) SetStatus(s string) *DeviceTokenUpdate {
	dtu.mutation.SetStatus(s)
	return dtu
}

// SetToken sets the "token" field.
func (dtu *DeviceTokenUpdate) SetToken(b []byte) *DeviceTokenUpdate {
	dtu.mutation.SetToken(b)
	return dtu
}

// ClearToken clears the value of the "token" field.
func (dtu *DeviceTokenUpdate) ClearToken() *DeviceTokenUpdate {
	dtu.mutation.ClearToken()
	return dtu
}

// SetExpiry sets the "expiry" field.
func (dtu *DeviceTokenUpdate) SetExpiry(t time.Time) *DeviceTokenUpdate {
	dtu.mutation.SetExpiry(t)
	return dtu
}

// SetLastRequest sets the "last_request" field.
func (dtu *DeviceTokenUpdate) SetLastRequest(t time.Time) *DeviceTokenUpdate {
	dtu.mutation.SetLastRequest(t)
	return dtu
}

// SetPollInterval sets the "poll_interval" field.
func (dtu *DeviceTokenUpdate) SetPollInterval(i int) *DeviceTokenUpdate {
	dtu.mutation.ResetPollInterval()
	dtu.mutation.SetPollInterval(i)
	return dtu
}

// AddPollInterval adds i to the "poll_interval" field.
func (dtu *DeviceTokenUpdate) AddPollInterval(i int) *DeviceTokenUpdate {
	dtu.mutation.AddPollInterval(i)
	return dtu
}

// Mutation returns the DeviceTokenMutation object of the builder.
func (dtu *DeviceTokenUpdate) Mutation() *DeviceTokenMutation {
	return dtu.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (dtu *DeviceTokenUpdate) Save(ctx context.Context) (int, error) {
	var (
		err      error
		affected int
	)
	if len(dtu.hooks) == 0 {
		if err = dtu.check(); err != nil {
			return 0, err
		}
		affected, err = dtu.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*DeviceTokenMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = dtu.check(); err != nil {
				return 0, err
			}
			dtu.mutation = mutation
			affected, err = dtu.sqlSave(ctx)
			mutation.done = true
			return affected, err
		})
		for i := len(dtu.hooks) - 1; i >= 0; i-- {
			if dtu.hooks[i] == nil {
				return 0, fmt.Errorf("db: uninitialized hook (forgotten import db/runtime?)")
			}
			mut = dtu.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, dtu.mutation); err != nil {
			return 0, err
		}
	}
	return affected, err
}

// SaveX is like Save, but panics if an error occurs.
func (dtu *DeviceTokenUpdate) SaveX(ctx context.Context) int {
	affected, err := dtu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (dtu *DeviceTokenUpdate) Exec(ctx context.Context) error {
	_, err := dtu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (dtu *DeviceTokenUpdate) ExecX(ctx context.Context) {
	if err := dtu.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (dtu *DeviceTokenUpdate) check() error {
	if v, ok := dtu.mutation.DeviceCode(); ok {
		if err := devicetoken.DeviceCodeValidator(v); err != nil {
			return &ValidationError{Name: "device_code", err: fmt.Errorf("db: validator failed for field \"device_code\": %w", err)}
		}
	}
	if v, ok := dtu.mutation.Status(); ok {
		if err := devicetoken.StatusValidator(v); err != nil {
			return &ValidationError{Name: "status", err: fmt.Errorf("db: validator failed for field \"status\": %w", err)}
		}
	}
	return nil
}

func (dtu *DeviceTokenUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   devicetoken.Table,
			Columns: devicetoken.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeInt,
				Column: devicetoken.FieldID,
			},
		},
	}
	if ps := dtu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := dtu.mutation.DeviceCode(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: devicetoken.FieldDeviceCode,
		})
	}
	if value, ok := dtu.mutation.Status(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: devicetoken.FieldStatus,
		})
	}
	if value, ok := dtu.mutation.Token(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBytes,
			Value:  value,
			Column: devicetoken.FieldToken,
		})
	}
	if dtu.mutation.TokenCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeBytes,
			Column: devicetoken.FieldToken,
		})
	}
	if value, ok := dtu.mutation.Expiry(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: devicetoken.FieldExpiry,
		})
	}
	if value, ok := dtu.mutation.LastRequest(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: devicetoken.FieldLastRequest,
		})
	}
	if value, ok := dtu.mutation.PollInterval(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: devicetoken.FieldPollInterval,
		})
	}
	if value, ok := dtu.mutation.AddedPollInterval(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: devicetoken.FieldPollInterval,
		})
	}
	if n, err = sqlgraph.UpdateNodes(ctx, dtu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{devicetoken.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{err.Error(), err}
		}
		return 0, err
	}
	return n, nil
}

// DeviceTokenUpdateOne is the builder for updating a single DeviceToken entity.
type DeviceTokenUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *DeviceTokenMutation
}

// SetDeviceCode sets the "device_code" field.
func (dtuo *DeviceTokenUpdateOne) SetDeviceCode(s string) *DeviceTokenUpdateOne {
	dtuo.mutation.SetDeviceCode(s)
	return dtuo
}

// SetStatus sets the "status" field.
func (dtuo *DeviceTokenUpdateOne) SetStatus(s string) *DeviceTokenUpdateOne {
	dtuo.mutation.SetStatus(s)
	return dtuo
}

// SetToken sets the "token" field.
func (dtuo *DeviceTokenUpdateOne) SetToken(b []byte) *DeviceTokenUpdateOne {
	dtuo.mutation.SetToken(b)
	return dtuo
}

// ClearToken clears the value of the "token" field.
func (dtuo *DeviceTokenUpdateOne) ClearToken() *DeviceTokenUpdateOne {
	dtuo.mutation.ClearToken()
	return dtuo
}

// SetExpiry sets the "expiry" field.
func (dtuo *DeviceTokenUpdateOne) SetExpiry(t time.Time) *DeviceTokenUpdateOne {
	dtuo.mutation.SetExpiry(t)
	return dtuo
}

// SetLastRequest sets the "last_request" field.
func (dtuo *DeviceTokenUpdateOne) SetLastRequest(t time.Time) *DeviceTokenUpdateOne {
	dtuo.mutation.SetLastRequest(t)
	return dtuo
}

// SetPollInterval sets the "poll_interval" field.
func (dtuo *DeviceTokenUpdateOne) SetPollInterval(i int) *DeviceTokenUpdateOne {
	dtuo.mutation.ResetPollInterval()
	dtuo.mutation.SetPollInterval(i)
	return dtuo
}

// AddPollInterval adds i to the "poll_interval" field.
func (dtuo *DeviceTokenUpdateOne) AddPollInterval(i int) *DeviceTokenUpdateOne {
	dtuo.mutation.AddPollInterval(i)
	return dtuo
}

// Mutation returns the DeviceTokenMutation object of the builder.
func (dtuo *DeviceTokenUpdateOne) Mutation() *DeviceTokenMutation {
	return dtuo.mutation
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (dtuo *DeviceTokenUpdateOne) Select(field string, fields ...string) *DeviceTokenUpdateOne {
	dtuo.fields = append([]string{field}, fields...)
	return dtuo
}

// Save executes the query and returns the updated DeviceToken entity.
func (dtuo *DeviceTokenUpdateOne) Save(ctx context.Context) (*DeviceToken, error) {
	var (
		err  error
		node *DeviceToken
	)
	if len(dtuo.hooks) == 0 {
		if err = dtuo.check(); err != nil {
			return nil, err
		}
		node, err = dtuo.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*DeviceTokenMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = dtuo.check(); err != nil {
				return nil, err
			}
			dtuo.mutation = mutation
			node, err = dtuo.sqlSave(ctx)
			mutation.done = true
			return node, err
		})
		for i := len(dtuo.hooks) - 1; i >= 0; i-- {
			if dtuo.hooks[i] == nil {
				return nil, fmt.Errorf("db: uninitialized hook (forgotten import db/runtime?)")
			}
			mut = dtuo.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, dtuo.mutation); err != nil {
			return nil, err
		}
	}
	return node, err
}

// SaveX is like Save, but panics if an error occurs.
func (dtuo *DeviceTokenUpdateOne) SaveX(ctx context.Context) *DeviceToken {
	node, err := dtuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (dtuo *DeviceTokenUpdateOne) Exec(ctx context.Context) error {
	_, err := dtuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (dtuo *DeviceTokenUpdateOne) ExecX(ctx context.Context) {
	if err := dtuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (dtuo *DeviceTokenUpdateOne) check() error {
	if v, ok := dtuo.mutation.DeviceCode(); ok {
		if err := devicetoken.DeviceCodeValidator(v); err != nil {
			return &ValidationError{Name: "device_code", err: fmt.Errorf("db: validator failed for field \"device_code\": %w", err)}
		}
	}
	if v, ok := dtuo.mutation.Status(); ok {
		if err := devicetoken.StatusValidator(v); err != nil {
			return &ValidationError{Name: "status", err: fmt.Errorf("db: validator failed for field \"status\": %w", err)}
		}
	}
	return nil
}

func (dtuo *DeviceTokenUpdateOne) sqlSave(ctx context.Context) (_node *DeviceToken, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   devicetoken.Table,
			Columns: devicetoken.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeInt,
				Column: devicetoken.FieldID,
			},
		},
	}
	id, ok := dtuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "ID", err: fmt.Errorf("missing DeviceToken.ID for update")}
	}
	_spec.Node.ID.Value = id
	if fields := dtuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, devicetoken.FieldID)
		for _, f := range fields {
			if !devicetoken.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != devicetoken.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := dtuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := dtuo.mutation.DeviceCode(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: devicetoken.FieldDeviceCode,
		})
	}
	if value, ok := dtuo.mutation.Status(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: devicetoken.FieldStatus,
		})
	}
	if value, ok := dtuo.mutation.Token(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBytes,
			Value:  value,
			Column: devicetoken.FieldToken,
		})
	}
	if dtuo.mutation.TokenCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeBytes,
			Column: devicetoken.FieldToken,
		})
	}
	if value, ok := dtuo.mutation.Expiry(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: devicetoken.FieldExpiry,
		})
	}
	if value, ok := dtuo.mutation.LastRequest(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: devicetoken.FieldLastRequest,
		})
	}
	if value, ok := dtuo.mutation.PollInterval(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: devicetoken.FieldPollInterval,
		})
	}
	if value, ok := dtuo.mutation.AddedPollInterval(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: devicetoken.FieldPollInterval,
		})
	}
	_node = &DeviceToken{config: dtuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, dtuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{devicetoken.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{err.Error(), err}
		}
		return nil, err
	}
	return _node, nil
}
