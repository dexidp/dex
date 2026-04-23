package library

import (
	"path"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

// Groups provides group-related CEL functions.
//
// Functions (V1):
//
//	dex.groupMatches(groups: list(string), pattern: string) -> list(string)
//	  Returns groups matching a glob pattern.
//	  Example: dex.groupMatches(["team:dev", "team:ops", "admin"], "team:*")
//
//	dex.groupFilter(groups: list(string), allowed: list(string)) -> list(string)
//	  Returns only groups present in the allowed list.
//	  Example: dex.groupFilter(["admin", "dev", "ops"], ["admin", "ops"])
type Groups struct{}

func (Groups) CompileOptions() []cel.EnvOption {
	return []cel.EnvOption{
		cel.Function("dex.groupMatches",
			cel.Overload("dex_group_matches_list_string",
				[]*cel.Type{cel.ListType(cel.StringType), cel.StringType},
				cel.ListType(cel.StringType),
				cel.BinaryBinding(groupMatchesImpl),
			),
		),
		cel.Function("dex.groupFilter",
			cel.Overload("dex_group_filter_list_list",
				[]*cel.Type{cel.ListType(cel.StringType), cel.ListType(cel.StringType)},
				cel.ListType(cel.StringType),
				cel.BinaryBinding(groupFilterImpl),
			),
		),
	}
}

func (Groups) ProgramOptions() []cel.ProgramOption {
	return nil
}

func groupMatchesImpl(lhs, rhs ref.Val) ref.Val {
	groupList, ok := lhs.(traits.Lister)
	if !ok {
		return types.NewErr("dex.groupMatches: expected list(string) as first argument")
	}

	pattern, ok := rhs.Value().(string)
	if !ok {
		return types.NewErr("dex.groupMatches: expected string pattern as second argument")
	}

	iter := groupList.Iterator()
	var matched []ref.Val

	for iter.HasNext() == types.True {
		item := iter.Next()

		group, ok := item.Value().(string)
		if !ok {
			continue
		}

		ok, err := path.Match(pattern, group)
		if err != nil {
			return types.NewErr("dex.groupMatches: invalid pattern %q: %v", pattern, err)
		}
		if ok {
			matched = append(matched, types.String(group))
		}
	}

	return types.NewRefValList(types.DefaultTypeAdapter, matched)
}

func groupFilterImpl(lhs, rhs ref.Val) ref.Val {
	groupList, ok := lhs.(traits.Lister)
	if !ok {
		return types.NewErr("dex.groupFilter: expected list(string) as first argument")
	}

	allowedList, ok := rhs.(traits.Lister)
	if !ok {
		return types.NewErr("dex.groupFilter: expected list(string) as second argument")
	}

	allowed := make(map[string]struct{})
	iter := allowedList.Iterator()
	for iter.HasNext() == types.True {
		item := iter.Next()

		s, ok := item.Value().(string)
		if !ok {
			continue
		}

		allowed[s] = struct{}{}
	}

	var filtered []ref.Val
	iter = groupList.Iterator()

	for iter.HasNext() == types.True {
		item := iter.Next()

		group, ok := item.Value().(string)
		if !ok {
			continue
		}

		if _, exists := allowed[group]; exists {
			filtered = append(filtered, types.String(group))
		}
	}

	return types.NewRefValList(types.DefaultTypeAdapter, filtered)
}
