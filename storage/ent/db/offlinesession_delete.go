// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/dexidp/dex/storage/ent/db/offlinesession"
	"github.com/dexidp/dex/storage/ent/db/predicate"
)

// OfflineSessionDelete is the builder for deleting a OfflineSession entity.
type OfflineSessionDelete struct {
	config
	hooks    []Hook
	mutation *OfflineSessionMutation
}

// Where appends a list predicates to the OfflineSessionDelete builder.
func (osd *OfflineSessionDelete) Where(ps ...predicate.OfflineSession) *OfflineSessionDelete {
	osd.mutation.Where(ps...)
	return osd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (osd *OfflineSessionDelete) Exec(ctx context.Context) (int, error) {
	var (
		err      error
		affected int
	)
	if len(osd.hooks) == 0 {
		affected, err = osd.sqlExec(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*OfflineSessionMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			osd.mutation = mutation
			affected, err = osd.sqlExec(ctx)
			mutation.done = true
			return affected, err
		})
		for i := len(osd.hooks) - 1; i >= 0; i-- {
			if osd.hooks[i] == nil {
				return 0, fmt.Errorf("db: uninitialized hook (forgotten import db/runtime?)")
			}
			mut = osd.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, osd.mutation); err != nil {
			return 0, err
		}
	}
	return affected, err
}

// ExecX is like Exec, but panics if an error occurs.
func (osd *OfflineSessionDelete) ExecX(ctx context.Context) int {
	n, err := osd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (osd *OfflineSessionDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := &sqlgraph.DeleteSpec{
		Node: &sqlgraph.NodeSpec{
			Table: offlinesession.Table,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeString,
				Column: offlinesession.FieldID,
			},
		},
	}
	if ps := osd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, osd.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	return affected, err
}

// OfflineSessionDeleteOne is the builder for deleting a single OfflineSession entity.
type OfflineSessionDeleteOne struct {
	osd *OfflineSessionDelete
}

// Exec executes the deletion query.
func (osdo *OfflineSessionDeleteOne) Exec(ctx context.Context) error {
	n, err := osdo.osd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{offlinesession.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (osdo *OfflineSessionDeleteOne) ExecX(ctx context.Context) {
	osdo.osd.ExecX(ctx)
}
