package etreeutils

import (
	"errors"

	"fmt"

	"sort"

	"github.com/beevik/etree"
)

const (
	defaultPrefix = ""
	xmlnsPrefix   = "xmlns"
	xmlPrefix     = "xml"

	XMLNamespace   = "http://www.w3.org/XML/1998/namespace"
	XMLNSNamespace = "http://www.w3.org/2000/xmlns/"
)

var (
	DefaultNSContext = NSContext{
		prefixes: map[string]string{
			defaultPrefix: XMLNamespace,
			xmlPrefix:     XMLNamespace,
			xmlnsPrefix:   XMLNSNamespace,
		},
	}

	EmptyNSContext = NSContext{}

	ErrReservedNamespace       = errors.New("disallowed declaration of reserved namespace")
	ErrInvalidDefaultNamespace = errors.New("invalid default namespace declaration")
	ErrTraversalHalted         = errors.New("traversal halted")
)

type ErrUndeclaredNSPrefix struct {
	Prefix string
}

func (e ErrUndeclaredNSPrefix) Error() string {
	return fmt.Sprintf("undeclared namespace prefix: '%s'", e.Prefix)
}

type NSContext struct {
	prefixes map[string]string
}

func (ctx NSContext) SubContext(el *etree.Element) (NSContext, error) {
	// The subcontext should inherit existing declared prefixes
	prefixes := make(map[string]string, len(ctx.prefixes)+4)
	for k, v := range ctx.prefixes {
		prefixes[k] = v
	}

	// Merge new namespace declarations on top of existing ones.
	for _, attr := range el.Attr {
		if attr.Space == xmlnsPrefix {
			// This attribute is a namespace declaration of the form "xmlns:<prefix>"

			// The 'xml' namespace may only be re-declared with the name 'http://www.w3.org/XML/1998/namespace'
			if attr.Key == xmlPrefix && attr.Value != XMLNamespace {
				return ctx, ErrReservedNamespace
			}

			// The 'xmlns' namespace may not be re-declared
			if attr.Key == xmlnsPrefix {
				return ctx, ErrReservedNamespace
			}

			prefixes[attr.Key] = attr.Value
		} else if attr.Space == defaultPrefix && attr.Key == xmlnsPrefix {
			// This attribute is a default namespace declaration

			// The xmlns namespace value may not be declared as the default namespace
			if attr.Value == XMLNSNamespace {
				return ctx, ErrInvalidDefaultNamespace
			}

			prefixes[defaultPrefix] = attr.Value
		}
	}

	return NSContext{prefixes: prefixes}, nil
}

// LookupPrefix attempts to find a declared namespace for the specified prefix. If the prefix
// is an empty string this will be the default namespace for this context. If the prefix is
// undeclared in this context an ErrUndeclaredNSPrefix will be returned.
func (ctx NSContext) LookupPrefix(prefix string) (string, error) {
	if namespace, ok := ctx.prefixes[prefix]; ok {
		return namespace, nil
	}

	return "", ErrUndeclaredNSPrefix{
		Prefix: prefix,
	}
}

func nsTraverse(ctx NSContext, el *etree.Element, handle func(NSContext, *etree.Element) error) error {
	ctx, err := ctx.SubContext(el)
	if err != nil {
		return err
	}

	err = handle(ctx, el)
	if err != nil {
		return err
	}

	// Recursively traverse child elements.
	for _, child := range el.ChildElements() {
		err := nsTraverse(ctx, child, handle)
		if err != nil {
			return err
		}
	}

	return nil
}

func detachWithNamespaces(ctx NSContext, el *etree.Element) (*etree.Element, error) {
	ctx, err := ctx.SubContext(el)
	if err != nil {
		return nil, err
	}

	el = el.Copy()

	// Build a new attribute list
	attrs := make([]etree.Attr, 0, len(el.Attr))

	// First copy over anything that isn't a namespace declaration
	for _, attr := range el.Attr {
		if attr.Space == xmlnsPrefix {
			continue
		}

		if attr.Space == defaultPrefix && attr.Key == xmlnsPrefix {
			continue
		}

		attrs = append(attrs, attr)
	}

	// Append all in-context namespace declarations
	for prefix, namespace := range ctx.prefixes {
		// Skip the implicit "xml" and "xmlns" prefix declarations
		if prefix == xmlnsPrefix || prefix == xmlPrefix {
			continue
		}

		// Also skip declararing the default namespace as XMLNamespace
		if prefix == defaultPrefix && namespace == XMLNamespace {
			continue
		}

		if prefix != defaultPrefix {
			attrs = append(attrs, etree.Attr{
				Space: xmlnsPrefix,
				Key:   prefix,
				Value: namespace,
			})
		} else {
			attrs = append(attrs, etree.Attr{
				Key:   xmlnsPrefix,
				Value: namespace,
			})
		}
	}

	sort.Sort(SortedAttrs(attrs))

	el.Attr = attrs

	return el, nil
}

// NSSelectOne conducts a depth-first search for an element with the specified namespace
// and tag. If such an element is found, a new *etree.Element is returned which is a
// copy of the found element, but with all in-context namespace declarations attached
// to the element as attributes.
func NSSelectOne(el *etree.Element, namespace, tag string) (*etree.Element, error) {
	var found *etree.Element

	err := nsTraverse(DefaultNSContext, el, func(ctx NSContext, el *etree.Element) error {
		currentNS, err := ctx.LookupPrefix(el.Space)
		if err != nil {
			return err
		}

		// Base case, el is the sought after element.
		if currentNS == namespace && el.Tag == tag {
			found, err = detachWithNamespaces(ctx, el)
			return ErrTraversalHalted
		}

		return nil
	})

	if err != nil && err != ErrTraversalHalted {
		return nil, err
	}

	return found, nil
}

// NSFindIterate conducts a depth-first traversal searching for elements with the
// specified tag in the specified namespace. For each such element, the passed
// handler function is invoked. If the handler function returns an error
// traversal is immediately halted. If the error returned by the handler is
// ErrTraversalHalted then nil will be returned by NSFindIterate. If any other
// error is returned by the handler, that error will be returned by NSFindIterate.
func NSFindIterate(el *etree.Element, namespace, tag string, handle func(*etree.Element) error) error {
	err := nsTraverse(DefaultNSContext, el, func(ctx NSContext, el *etree.Element) error {
		currentNS, err := ctx.LookupPrefix(el.Space)
		if err != nil {
			return err
		}

		// Base case, el is the sought after element.
		if currentNS == namespace && el.Tag == tag {
			return handle(el)
		}

		return nil
	})

	if err != nil && err != ErrTraversalHalted {
		return err
	}

	return nil
}

// NSFindOne conducts a depth-first search for the specified element. If such an element
// is found a reference to it is returned.
func NSFindOne(el *etree.Element, namespace, tag string) (*etree.Element, error) {
	var found *etree.Element

	err := NSFindIterate(el, namespace, tag, func(el *etree.Element) error {
		found = el
		return ErrTraversalHalted
	})

	if err != nil {
		return nil, err
	}

	return found, nil
}
