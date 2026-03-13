package library

import (
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// Email provides email-related CEL functions.
//
// Functions (V1):
//
//	dex.emailDomain(email: string) -> string
//	  Returns the domain portion of an email address.
//	  Example: dex.emailDomain("user@example.com") == "example.com"
//
//	dex.emailLocalPart(email: string) -> string
//	  Returns the local part of an email address.
//	  Example: dex.emailLocalPart("user@example.com") == "user"
type Email struct{}

func (Email) CompileOptions() []cel.EnvOption {
	return []cel.EnvOption{
		cel.Function("dex.emailDomain",
			cel.Overload("dex_email_domain_string",
				[]*cel.Type{cel.StringType},
				cel.StringType,
				cel.UnaryBinding(emailDomainImpl),
			),
		),
		cel.Function("dex.emailLocalPart",
			cel.Overload("dex_email_local_part_string",
				[]*cel.Type{cel.StringType},
				cel.StringType,
				cel.UnaryBinding(emailLocalPartImpl),
			),
		),
	}
}

func (Email) ProgramOptions() []cel.ProgramOption {
	return nil
}

func emailDomainImpl(arg ref.Val) ref.Val {
	email, ok := arg.Value().(string)
	if !ok {
		return types.NewErr("dex.emailDomain: expected string argument")
	}

	_, domain, found := strings.Cut(email, "@")
	if !found {
		return types.String("")
	}

	return types.String(domain)
}

func emailLocalPartImpl(arg ref.Val) ref.Val {
	email, ok := arg.Value().(string)
	if !ok {
		return types.NewErr("dex.emailLocalPart: expected string argument")
	}

	localPart, _, found := strings.Cut(email, "@")
	if !found {
		return types.String(email)
	}

	return types.String(localPart)
}
