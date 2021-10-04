// Code generated by entc, DO NOT EDIT.

package db

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/dexidp/dex/storage/ent/db/predicate"
	"github.com/dexidp/dex/storage/ent/db/refreshtoken"
)

// RefreshTokenUpdate is the builder for updating RefreshToken entities.
type RefreshTokenUpdate struct {
	config
	hooks    []Hook
	mutation *RefreshTokenMutation
}

// Where appends a list predicates to the RefreshTokenUpdate builder.
func (rtu *RefreshTokenUpdate) Where(ps ...predicate.RefreshToken) *RefreshTokenUpdate {
	rtu.mutation.Where(ps...)
	return rtu
}

// SetClientID sets the "client_id" field.
func (rtu *RefreshTokenUpdate) SetClientID(s string) *RefreshTokenUpdate {
	rtu.mutation.SetClientID(s)
	return rtu
}

// SetScopes sets the "scopes" field.
func (rtu *RefreshTokenUpdate) SetScopes(s []string) *RefreshTokenUpdate {
	rtu.mutation.SetScopes(s)
	return rtu
}

// ClearScopes clears the value of the "scopes" field.
func (rtu *RefreshTokenUpdate) ClearScopes() *RefreshTokenUpdate {
	rtu.mutation.ClearScopes()
	return rtu
}

// SetNonce sets the "nonce" field.
func (rtu *RefreshTokenUpdate) SetNonce(s string) *RefreshTokenUpdate {
	rtu.mutation.SetNonce(s)
	return rtu
}

// SetClaimsUserID sets the "claims_user_id" field.
func (rtu *RefreshTokenUpdate) SetClaimsUserID(s string) *RefreshTokenUpdate {
	rtu.mutation.SetClaimsUserID(s)
	return rtu
}

// SetClaimsUsername sets the "claims_username" field.
func (rtu *RefreshTokenUpdate) SetClaimsUsername(s string) *RefreshTokenUpdate {
	rtu.mutation.SetClaimsUsername(s)
	return rtu
}

// SetClaimsEmail sets the "claims_email" field.
func (rtu *RefreshTokenUpdate) SetClaimsEmail(s string) *RefreshTokenUpdate {
	rtu.mutation.SetClaimsEmail(s)
	return rtu
}

// SetClaimsEmailVerified sets the "claims_email_verified" field.
func (rtu *RefreshTokenUpdate) SetClaimsEmailVerified(b bool) *RefreshTokenUpdate {
	rtu.mutation.SetClaimsEmailVerified(b)
	return rtu
}

// SetClaimsGroups sets the "claims_groups" field.
func (rtu *RefreshTokenUpdate) SetClaimsGroups(s []string) *RefreshTokenUpdate {
	rtu.mutation.SetClaimsGroups(s)
	return rtu
}

// ClearClaimsGroups clears the value of the "claims_groups" field.
func (rtu *RefreshTokenUpdate) ClearClaimsGroups() *RefreshTokenUpdate {
	rtu.mutation.ClearClaimsGroups()
	return rtu
}

// SetClaimsPreferredUsername sets the "claims_preferred_username" field.
func (rtu *RefreshTokenUpdate) SetClaimsPreferredUsername(s string) *RefreshTokenUpdate {
	rtu.mutation.SetClaimsPreferredUsername(s)
	return rtu
}

// SetNillableClaimsPreferredUsername sets the "claims_preferred_username" field if the given value is not nil.
func (rtu *RefreshTokenUpdate) SetNillableClaimsPreferredUsername(s *string) *RefreshTokenUpdate {
	if s != nil {
		rtu.SetClaimsPreferredUsername(*s)
	}
	return rtu
}

// SetConnectorID sets the "connector_id" field.
func (rtu *RefreshTokenUpdate) SetConnectorID(s string) *RefreshTokenUpdate {
	rtu.mutation.SetConnectorID(s)
	return rtu
}

// SetConnectorData sets the "connector_data" field.
func (rtu *RefreshTokenUpdate) SetConnectorData(b []byte) *RefreshTokenUpdate {
	rtu.mutation.SetConnectorData(b)
	return rtu
}

// ClearConnectorData clears the value of the "connector_data" field.
func (rtu *RefreshTokenUpdate) ClearConnectorData() *RefreshTokenUpdate {
	rtu.mutation.ClearConnectorData()
	return rtu
}

// SetToken sets the "token" field.
func (rtu *RefreshTokenUpdate) SetToken(s string) *RefreshTokenUpdate {
	rtu.mutation.SetToken(s)
	return rtu
}

// SetNillableToken sets the "token" field if the given value is not nil.
func (rtu *RefreshTokenUpdate) SetNillableToken(s *string) *RefreshTokenUpdate {
	if s != nil {
		rtu.SetToken(*s)
	}
	return rtu
}

// SetObsoleteToken sets the "obsolete_token" field.
func (rtu *RefreshTokenUpdate) SetObsoleteToken(s string) *RefreshTokenUpdate {
	rtu.mutation.SetObsoleteToken(s)
	return rtu
}

// SetNillableObsoleteToken sets the "obsolete_token" field if the given value is not nil.
func (rtu *RefreshTokenUpdate) SetNillableObsoleteToken(s *string) *RefreshTokenUpdate {
	if s != nil {
		rtu.SetObsoleteToken(*s)
	}
	return rtu
}

// SetCreatedAt sets the "created_at" field.
func (rtu *RefreshTokenUpdate) SetCreatedAt(t time.Time) *RefreshTokenUpdate {
	rtu.mutation.SetCreatedAt(t)
	return rtu
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (rtu *RefreshTokenUpdate) SetNillableCreatedAt(t *time.Time) *RefreshTokenUpdate {
	if t != nil {
		rtu.SetCreatedAt(*t)
	}
	return rtu
}

// SetLastUsed sets the "last_used" field.
func (rtu *RefreshTokenUpdate) SetLastUsed(t time.Time) *RefreshTokenUpdate {
	rtu.mutation.SetLastUsed(t)
	return rtu
}

// SetNillableLastUsed sets the "last_used" field if the given value is not nil.
func (rtu *RefreshTokenUpdate) SetNillableLastUsed(t *time.Time) *RefreshTokenUpdate {
	if t != nil {
		rtu.SetLastUsed(*t)
	}
	return rtu
}

// Mutation returns the RefreshTokenMutation object of the builder.
func (rtu *RefreshTokenUpdate) Mutation() *RefreshTokenMutation {
	return rtu.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (rtu *RefreshTokenUpdate) Save(ctx context.Context) (int, error) {
	var (
		err      error
		affected int
	)
	if len(rtu.hooks) == 0 {
		if err = rtu.check(); err != nil {
			return 0, err
		}
		affected, err = rtu.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*RefreshTokenMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = rtu.check(); err != nil {
				return 0, err
			}
			rtu.mutation = mutation
			affected, err = rtu.sqlSave(ctx)
			mutation.done = true
			return affected, err
		})
		for i := len(rtu.hooks) - 1; i >= 0; i-- {
			if rtu.hooks[i] == nil {
				return 0, fmt.Errorf("db: uninitialized hook (forgotten import db/runtime?)")
			}
			mut = rtu.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, rtu.mutation); err != nil {
			return 0, err
		}
	}
	return affected, err
}

// SaveX is like Save, but panics if an error occurs.
func (rtu *RefreshTokenUpdate) SaveX(ctx context.Context) int {
	affected, err := rtu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (rtu *RefreshTokenUpdate) Exec(ctx context.Context) error {
	_, err := rtu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (rtu *RefreshTokenUpdate) ExecX(ctx context.Context) {
	if err := rtu.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (rtu *RefreshTokenUpdate) check() error {
	if v, ok := rtu.mutation.ClientID(); ok {
		if err := refreshtoken.ClientIDValidator(v); err != nil {
			return &ValidationError{Name: "client_id", err: fmt.Errorf("db: validator failed for field \"client_id\": %w", err)}
		}
	}
	if v, ok := rtu.mutation.Nonce(); ok {
		if err := refreshtoken.NonceValidator(v); err != nil {
			return &ValidationError{Name: "nonce", err: fmt.Errorf("db: validator failed for field \"nonce\": %w", err)}
		}
	}
	if v, ok := rtu.mutation.ClaimsUserID(); ok {
		if err := refreshtoken.ClaimsUserIDValidator(v); err != nil {
			return &ValidationError{Name: "claims_user_id", err: fmt.Errorf("db: validator failed for field \"claims_user_id\": %w", err)}
		}
	}
	if v, ok := rtu.mutation.ClaimsUsername(); ok {
		if err := refreshtoken.ClaimsUsernameValidator(v); err != nil {
			return &ValidationError{Name: "claims_username", err: fmt.Errorf("db: validator failed for field \"claims_username\": %w", err)}
		}
	}
	if v, ok := rtu.mutation.ClaimsEmail(); ok {
		if err := refreshtoken.ClaimsEmailValidator(v); err != nil {
			return &ValidationError{Name: "claims_email", err: fmt.Errorf("db: validator failed for field \"claims_email\": %w", err)}
		}
	}
	if v, ok := rtu.mutation.ConnectorID(); ok {
		if err := refreshtoken.ConnectorIDValidator(v); err != nil {
			return &ValidationError{Name: "connector_id", err: fmt.Errorf("db: validator failed for field \"connector_id\": %w", err)}
		}
	}
	return nil
}

func (rtu *RefreshTokenUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   refreshtoken.Table,
			Columns: refreshtoken.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeString,
				Column: refreshtoken.FieldID,
			},
		},
	}
	if ps := rtu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := rtu.mutation.ClientID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldClientID,
		})
	}
	if value, ok := rtu.mutation.Scopes(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Value:  value,
			Column: refreshtoken.FieldScopes,
		})
	}
	if rtu.mutation.ScopesCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Column: refreshtoken.FieldScopes,
		})
	}
	if value, ok := rtu.mutation.Nonce(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldNonce,
		})
	}
	if value, ok := rtu.mutation.ClaimsUserID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldClaimsUserID,
		})
	}
	if value, ok := rtu.mutation.ClaimsUsername(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldClaimsUsername,
		})
	}
	if value, ok := rtu.mutation.ClaimsEmail(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldClaimsEmail,
		})
	}
	if value, ok := rtu.mutation.ClaimsEmailVerified(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: refreshtoken.FieldClaimsEmailVerified,
		})
	}
	if value, ok := rtu.mutation.ClaimsGroups(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Value:  value,
			Column: refreshtoken.FieldClaimsGroups,
		})
	}
	if rtu.mutation.ClaimsGroupsCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Column: refreshtoken.FieldClaimsGroups,
		})
	}
	if value, ok := rtu.mutation.ClaimsPreferredUsername(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldClaimsPreferredUsername,
		})
	}
	if value, ok := rtu.mutation.ConnectorID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldConnectorID,
		})
	}
	if value, ok := rtu.mutation.ConnectorData(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBytes,
			Value:  value,
			Column: refreshtoken.FieldConnectorData,
		})
	}
	if rtu.mutation.ConnectorDataCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeBytes,
			Column: refreshtoken.FieldConnectorData,
		})
	}
	if value, ok := rtu.mutation.Token(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldToken,
		})
	}
	if value, ok := rtu.mutation.ObsoleteToken(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldObsoleteToken,
		})
	}
	if value, ok := rtu.mutation.CreatedAt(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: refreshtoken.FieldCreatedAt,
		})
	}
	if value, ok := rtu.mutation.LastUsed(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: refreshtoken.FieldLastUsed,
		})
	}
	if n, err = sqlgraph.UpdateNodes(ctx, rtu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{refreshtoken.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{err.Error(), err}
		}
		return 0, err
	}
	return n, nil
}

// RefreshTokenUpdateOne is the builder for updating a single RefreshToken entity.
type RefreshTokenUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *RefreshTokenMutation
}

// SetClientID sets the "client_id" field.
func (rtuo *RefreshTokenUpdateOne) SetClientID(s string) *RefreshTokenUpdateOne {
	rtuo.mutation.SetClientID(s)
	return rtuo
}

// SetScopes sets the "scopes" field.
func (rtuo *RefreshTokenUpdateOne) SetScopes(s []string) *RefreshTokenUpdateOne {
	rtuo.mutation.SetScopes(s)
	return rtuo
}

// ClearScopes clears the value of the "scopes" field.
func (rtuo *RefreshTokenUpdateOne) ClearScopes() *RefreshTokenUpdateOne {
	rtuo.mutation.ClearScopes()
	return rtuo
}

// SetNonce sets the "nonce" field.
func (rtuo *RefreshTokenUpdateOne) SetNonce(s string) *RefreshTokenUpdateOne {
	rtuo.mutation.SetNonce(s)
	return rtuo
}

// SetClaimsUserID sets the "claims_user_id" field.
func (rtuo *RefreshTokenUpdateOne) SetClaimsUserID(s string) *RefreshTokenUpdateOne {
	rtuo.mutation.SetClaimsUserID(s)
	return rtuo
}

// SetClaimsUsername sets the "claims_username" field.
func (rtuo *RefreshTokenUpdateOne) SetClaimsUsername(s string) *RefreshTokenUpdateOne {
	rtuo.mutation.SetClaimsUsername(s)
	return rtuo
}

// SetClaimsEmail sets the "claims_email" field.
func (rtuo *RefreshTokenUpdateOne) SetClaimsEmail(s string) *RefreshTokenUpdateOne {
	rtuo.mutation.SetClaimsEmail(s)
	return rtuo
}

// SetClaimsEmailVerified sets the "claims_email_verified" field.
func (rtuo *RefreshTokenUpdateOne) SetClaimsEmailVerified(b bool) *RefreshTokenUpdateOne {
	rtuo.mutation.SetClaimsEmailVerified(b)
	return rtuo
}

// SetClaimsGroups sets the "claims_groups" field.
func (rtuo *RefreshTokenUpdateOne) SetClaimsGroups(s []string) *RefreshTokenUpdateOne {
	rtuo.mutation.SetClaimsGroups(s)
	return rtuo
}

// ClearClaimsGroups clears the value of the "claims_groups" field.
func (rtuo *RefreshTokenUpdateOne) ClearClaimsGroups() *RefreshTokenUpdateOne {
	rtuo.mutation.ClearClaimsGroups()
	return rtuo
}

// SetClaimsPreferredUsername sets the "claims_preferred_username" field.
func (rtuo *RefreshTokenUpdateOne) SetClaimsPreferredUsername(s string) *RefreshTokenUpdateOne {
	rtuo.mutation.SetClaimsPreferredUsername(s)
	return rtuo
}

// SetNillableClaimsPreferredUsername sets the "claims_preferred_username" field if the given value is not nil.
func (rtuo *RefreshTokenUpdateOne) SetNillableClaimsPreferredUsername(s *string) *RefreshTokenUpdateOne {
	if s != nil {
		rtuo.SetClaimsPreferredUsername(*s)
	}
	return rtuo
}

// SetConnectorID sets the "connector_id" field.
func (rtuo *RefreshTokenUpdateOne) SetConnectorID(s string) *RefreshTokenUpdateOne {
	rtuo.mutation.SetConnectorID(s)
	return rtuo
}

// SetConnectorData sets the "connector_data" field.
func (rtuo *RefreshTokenUpdateOne) SetConnectorData(b []byte) *RefreshTokenUpdateOne {
	rtuo.mutation.SetConnectorData(b)
	return rtuo
}

// ClearConnectorData clears the value of the "connector_data" field.
func (rtuo *RefreshTokenUpdateOne) ClearConnectorData() *RefreshTokenUpdateOne {
	rtuo.mutation.ClearConnectorData()
	return rtuo
}

// SetToken sets the "token" field.
func (rtuo *RefreshTokenUpdateOne) SetToken(s string) *RefreshTokenUpdateOne {
	rtuo.mutation.SetToken(s)
	return rtuo
}

// SetNillableToken sets the "token" field if the given value is not nil.
func (rtuo *RefreshTokenUpdateOne) SetNillableToken(s *string) *RefreshTokenUpdateOne {
	if s != nil {
		rtuo.SetToken(*s)
	}
	return rtuo
}

// SetObsoleteToken sets the "obsolete_token" field.
func (rtuo *RefreshTokenUpdateOne) SetObsoleteToken(s string) *RefreshTokenUpdateOne {
	rtuo.mutation.SetObsoleteToken(s)
	return rtuo
}

// SetNillableObsoleteToken sets the "obsolete_token" field if the given value is not nil.
func (rtuo *RefreshTokenUpdateOne) SetNillableObsoleteToken(s *string) *RefreshTokenUpdateOne {
	if s != nil {
		rtuo.SetObsoleteToken(*s)
	}
	return rtuo
}

// SetCreatedAt sets the "created_at" field.
func (rtuo *RefreshTokenUpdateOne) SetCreatedAt(t time.Time) *RefreshTokenUpdateOne {
	rtuo.mutation.SetCreatedAt(t)
	return rtuo
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (rtuo *RefreshTokenUpdateOne) SetNillableCreatedAt(t *time.Time) *RefreshTokenUpdateOne {
	if t != nil {
		rtuo.SetCreatedAt(*t)
	}
	return rtuo
}

// SetLastUsed sets the "last_used" field.
func (rtuo *RefreshTokenUpdateOne) SetLastUsed(t time.Time) *RefreshTokenUpdateOne {
	rtuo.mutation.SetLastUsed(t)
	return rtuo
}

// SetNillableLastUsed sets the "last_used" field if the given value is not nil.
func (rtuo *RefreshTokenUpdateOne) SetNillableLastUsed(t *time.Time) *RefreshTokenUpdateOne {
	if t != nil {
		rtuo.SetLastUsed(*t)
	}
	return rtuo
}

// Mutation returns the RefreshTokenMutation object of the builder.
func (rtuo *RefreshTokenUpdateOne) Mutation() *RefreshTokenMutation {
	return rtuo.mutation
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (rtuo *RefreshTokenUpdateOne) Select(field string, fields ...string) *RefreshTokenUpdateOne {
	rtuo.fields = append([]string{field}, fields...)
	return rtuo
}

// Save executes the query and returns the updated RefreshToken entity.
func (rtuo *RefreshTokenUpdateOne) Save(ctx context.Context) (*RefreshToken, error) {
	var (
		err  error
		node *RefreshToken
	)
	if len(rtuo.hooks) == 0 {
		if err = rtuo.check(); err != nil {
			return nil, err
		}
		node, err = rtuo.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*RefreshTokenMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = rtuo.check(); err != nil {
				return nil, err
			}
			rtuo.mutation = mutation
			node, err = rtuo.sqlSave(ctx)
			mutation.done = true
			return node, err
		})
		for i := len(rtuo.hooks) - 1; i >= 0; i-- {
			if rtuo.hooks[i] == nil {
				return nil, fmt.Errorf("db: uninitialized hook (forgotten import db/runtime?)")
			}
			mut = rtuo.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, rtuo.mutation); err != nil {
			return nil, err
		}
	}
	return node, err
}

// SaveX is like Save, but panics if an error occurs.
func (rtuo *RefreshTokenUpdateOne) SaveX(ctx context.Context) *RefreshToken {
	node, err := rtuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (rtuo *RefreshTokenUpdateOne) Exec(ctx context.Context) error {
	_, err := rtuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (rtuo *RefreshTokenUpdateOne) ExecX(ctx context.Context) {
	if err := rtuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (rtuo *RefreshTokenUpdateOne) check() error {
	if v, ok := rtuo.mutation.ClientID(); ok {
		if err := refreshtoken.ClientIDValidator(v); err != nil {
			return &ValidationError{Name: "client_id", err: fmt.Errorf("db: validator failed for field \"client_id\": %w", err)}
		}
	}
	if v, ok := rtuo.mutation.Nonce(); ok {
		if err := refreshtoken.NonceValidator(v); err != nil {
			return &ValidationError{Name: "nonce", err: fmt.Errorf("db: validator failed for field \"nonce\": %w", err)}
		}
	}
	if v, ok := rtuo.mutation.ClaimsUserID(); ok {
		if err := refreshtoken.ClaimsUserIDValidator(v); err != nil {
			return &ValidationError{Name: "claims_user_id", err: fmt.Errorf("db: validator failed for field \"claims_user_id\": %w", err)}
		}
	}
	if v, ok := rtuo.mutation.ClaimsUsername(); ok {
		if err := refreshtoken.ClaimsUsernameValidator(v); err != nil {
			return &ValidationError{Name: "claims_username", err: fmt.Errorf("db: validator failed for field \"claims_username\": %w", err)}
		}
	}
	if v, ok := rtuo.mutation.ClaimsEmail(); ok {
		if err := refreshtoken.ClaimsEmailValidator(v); err != nil {
			return &ValidationError{Name: "claims_email", err: fmt.Errorf("db: validator failed for field \"claims_email\": %w", err)}
		}
	}
	if v, ok := rtuo.mutation.ConnectorID(); ok {
		if err := refreshtoken.ConnectorIDValidator(v); err != nil {
			return &ValidationError{Name: "connector_id", err: fmt.Errorf("db: validator failed for field \"connector_id\": %w", err)}
		}
	}
	return nil
}

func (rtuo *RefreshTokenUpdateOne) sqlSave(ctx context.Context) (_node *RefreshToken, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   refreshtoken.Table,
			Columns: refreshtoken.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeString,
				Column: refreshtoken.FieldID,
			},
		},
	}
	id, ok := rtuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "ID", err: fmt.Errorf("missing RefreshToken.ID for update")}
	}
	_spec.Node.ID.Value = id
	if fields := rtuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, refreshtoken.FieldID)
		for _, f := range fields {
			if !refreshtoken.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
			}
			if f != refreshtoken.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := rtuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := rtuo.mutation.ClientID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldClientID,
		})
	}
	if value, ok := rtuo.mutation.Scopes(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Value:  value,
			Column: refreshtoken.FieldScopes,
		})
	}
	if rtuo.mutation.ScopesCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Column: refreshtoken.FieldScopes,
		})
	}
	if value, ok := rtuo.mutation.Nonce(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldNonce,
		})
	}
	if value, ok := rtuo.mutation.ClaimsUserID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldClaimsUserID,
		})
	}
	if value, ok := rtuo.mutation.ClaimsUsername(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldClaimsUsername,
		})
	}
	if value, ok := rtuo.mutation.ClaimsEmail(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldClaimsEmail,
		})
	}
	if value, ok := rtuo.mutation.ClaimsEmailVerified(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: refreshtoken.FieldClaimsEmailVerified,
		})
	}
	if value, ok := rtuo.mutation.ClaimsGroups(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Value:  value,
			Column: refreshtoken.FieldClaimsGroups,
		})
	}
	if rtuo.mutation.ClaimsGroupsCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Column: refreshtoken.FieldClaimsGroups,
		})
	}
	if value, ok := rtuo.mutation.ClaimsPreferredUsername(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldClaimsPreferredUsername,
		})
	}
	if value, ok := rtuo.mutation.ConnectorID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldConnectorID,
		})
	}
	if value, ok := rtuo.mutation.ConnectorData(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBytes,
			Value:  value,
			Column: refreshtoken.FieldConnectorData,
		})
	}
	if rtuo.mutation.ConnectorDataCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeBytes,
			Column: refreshtoken.FieldConnectorData,
		})
	}
	if value, ok := rtuo.mutation.Token(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldToken,
		})
	}
	if value, ok := rtuo.mutation.ObsoleteToken(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: refreshtoken.FieldObsoleteToken,
		})
	}
	if value, ok := rtuo.mutation.CreatedAt(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: refreshtoken.FieldCreatedAt,
		})
	}
	if value, ok := rtuo.mutation.LastUsed(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: refreshtoken.FieldLastUsed,
		})
	}
	_node = &RefreshToken{config: rtuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, rtuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{refreshtoken.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{err.Error(), err}
		}
		return nil, err
	}
	return _node, nil
}
