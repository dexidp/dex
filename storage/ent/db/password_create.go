// Code generated by entc, DO NOT EDIT.

package db

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/dexidp/dex/storage/ent/db/password"
)

// PasswordCreate is the builder for creating a Password entity.
type PasswordCreate struct {
	config
	mutation *PasswordMutation
	hooks    []Hook
}

// SetEmail sets the "email" field.
func (pc *PasswordCreate) SetEmail(s string) *PasswordCreate {
	pc.mutation.SetEmail(s)
	return pc
}

// SetHash sets the "hash" field.
func (pc *PasswordCreate) SetHash(b []byte) *PasswordCreate {
	pc.mutation.SetHash(b)
	return pc
}

// SetUsername sets the "username" field.
func (pc *PasswordCreate) SetUsername(s string) *PasswordCreate {
	pc.mutation.SetUsername(s)
	return pc
}

// SetUserID sets the "user_id" field.
func (pc *PasswordCreate) SetUserID(s string) *PasswordCreate {
	pc.mutation.SetUserID(s)
	return pc
}

// Mutation returns the PasswordMutation object of the builder.
func (pc *PasswordCreate) Mutation() *PasswordMutation {
	return pc.mutation
}

// Save creates the Password in the database.
func (pc *PasswordCreate) Save(ctx context.Context) (*Password, error) {
	var (
		err  error
		node *Password
	)
	if len(pc.hooks) == 0 {
		if err = pc.check(); err != nil {
			return nil, err
		}
		node, err = pc.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*PasswordMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = pc.check(); err != nil {
				return nil, err
			}
			pc.mutation = mutation
			node, err = pc.sqlSave(ctx)
			mutation.done = true
			return node, err
		})
		for i := len(pc.hooks) - 1; i >= 0; i-- {
			mut = pc.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, pc.mutation); err != nil {
			return nil, err
		}
	}
	return node, err
}

// SaveX calls Save and panics if Save returns an error.
func (pc *PasswordCreate) SaveX(ctx context.Context) *Password {
	v, err := pc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// check runs all checks and user-defined validators on the builder.
func (pc *PasswordCreate) check() error {
	if _, ok := pc.mutation.Email(); !ok {
		return &ValidationError{Name: "email", err: errors.New("db: missing required field \"email\"")}
	}
	if v, ok := pc.mutation.Email(); ok {
		if err := password.EmailValidator(v); err != nil {
			return &ValidationError{Name: "email", err: fmt.Errorf("db: validator failed for field \"email\": %w", err)}
		}
	}
	if _, ok := pc.mutation.Hash(); !ok {
		return &ValidationError{Name: "hash", err: errors.New("db: missing required field \"hash\"")}
	}
	if _, ok := pc.mutation.Username(); !ok {
		return &ValidationError{Name: "username", err: errors.New("db: missing required field \"username\"")}
	}
	if v, ok := pc.mutation.Username(); ok {
		if err := password.UsernameValidator(v); err != nil {
			return &ValidationError{Name: "username", err: fmt.Errorf("db: validator failed for field \"username\": %w", err)}
		}
	}
	if _, ok := pc.mutation.UserID(); !ok {
		return &ValidationError{Name: "user_id", err: errors.New("db: missing required field \"user_id\"")}
	}
	if v, ok := pc.mutation.UserID(); ok {
		if err := password.UserIDValidator(v); err != nil {
			return &ValidationError{Name: "user_id", err: fmt.Errorf("db: validator failed for field \"user_id\": %w", err)}
		}
	}
	return nil
}

func (pc *PasswordCreate) sqlSave(ctx context.Context) (*Password, error) {
	_node, _spec := pc.createSpec()
	if err := sqlgraph.CreateNode(ctx, pc.driver, _spec); err != nil {
		if cerr, ok := isSQLConstraintError(err); ok {
			err = cerr
		}
		return nil, err
	}
	id := _spec.ID.Value.(int64)
	_node.ID = int(id)
	return _node, nil
}

func (pc *PasswordCreate) createSpec() (*Password, *sqlgraph.CreateSpec) {
	var (
		_node = &Password{config: pc.config}
		_spec = &sqlgraph.CreateSpec{
			Table: password.Table,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeInt,
				Column: password.FieldID,
			},
		}
	)
	if value, ok := pc.mutation.Email(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: password.FieldEmail,
		})
		_node.Email = value
	}
	if value, ok := pc.mutation.Hash(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeBytes,
			Value:  value,
			Column: password.FieldHash,
		})
		_node.Hash = value
	}
	if value, ok := pc.mutation.Username(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: password.FieldUsername,
		})
		_node.Username = value
	}
	if value, ok := pc.mutation.UserID(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: password.FieldUserID,
		})
		_node.UserID = value
	}
	return _node, _spec
}

// PasswordCreateBulk is the builder for creating many Password entities in bulk.
type PasswordCreateBulk struct {
	config
	builders []*PasswordCreate
}

// Save creates the Password entities in the database.
func (pcb *PasswordCreateBulk) Save(ctx context.Context) ([]*Password, error) {
	specs := make([]*sqlgraph.CreateSpec, len(pcb.builders))
	nodes := make([]*Password, len(pcb.builders))
	mutators := make([]Mutator, len(pcb.builders))
	for i := range pcb.builders {
		func(i int, root context.Context) {
			builder := pcb.builders[i]
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*PasswordMutation)
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}
				if err := builder.check(); err != nil {
					return nil, err
				}
				builder.mutation = mutation
				nodes[i], specs[i] = builder.createSpec()
				var err error
				if i < len(mutators)-1 {
					_, err = mutators[i+1].Mutate(root, pcb.builders[i+1].mutation)
				} else {
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, pcb.driver, &sqlgraph.BatchCreateSpec{Nodes: specs}); err != nil {
						if cerr, ok := isSQLConstraintError(err); ok {
							err = cerr
						}
					}
				}
				mutation.done = true
				if err != nil {
					return nil, err
				}
				id := specs[i].ID.Value.(int64)
				nodes[i].ID = int(id)
				return nodes[i], nil
			})
			for i := len(builder.hooks) - 1; i >= 0; i-- {
				mut = builder.hooks[i](mut)
			}
			mutators[i] = mut
		}(i, ctx)
	}
	if len(mutators) > 0 {
		if _, err := mutators[0].Mutate(ctx, pcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (pcb *PasswordCreateBulk) SaveX(ctx context.Context) []*Password {
	v, err := pcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}
