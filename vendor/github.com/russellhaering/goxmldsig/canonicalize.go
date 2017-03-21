package dsig

import (
	"sort"
	"strings"

	"github.com/beevik/etree"
	"github.com/russellhaering/goxmldsig/etreeutils"
)

// Canonicalizer is an implementation of a canonicalization algorithm.
type Canonicalizer interface {
	Canonicalize(el *etree.Element) ([]byte, error)
	Algorithm() AlgorithmID
}

type c14N10ExclusiveCanonicalizer struct {
	InclusiveNamespaces map[string]struct{}
}

// MakeC14N10ExclusiveCanonicalizerWithPrefixList constructs an exclusive Canonicalizer
// from a PrefixList in NMTOKENS format (a white space separated list).
func MakeC14N10ExclusiveCanonicalizerWithPrefixList(prefixList string) Canonicalizer {
	prefixes := strings.Fields(prefixList)
	prefixSet := make(map[string]struct{}, len(prefixes))

	for _, prefix := range prefixes {
		prefixSet[prefix] = struct{}{}
	}

	return &c14N10ExclusiveCanonicalizer{
		InclusiveNamespaces: prefixSet,
	}
}

// Canonicalize transforms the input Element into a serialized XML document in canonical form.
func (c *c14N10ExclusiveCanonicalizer) Canonicalize(el *etree.Element) ([]byte, error) {
	scope := make(map[string]c14nSpace)
	return canonicalSerialize(excCanonicalPrep(el, scope, c.InclusiveNamespaces))
}

func (c *c14N10ExclusiveCanonicalizer) Algorithm() AlgorithmID {
	return CanonicalXML10ExclusiveAlgorithmId
}

type c14N11Canonicalizer struct{}

// MakeC14N11Canonicalizer constructs an inclusive canonicalizer.
func MakeC14N11Canonicalizer() Canonicalizer {
	return &c14N11Canonicalizer{}
}

// Canonicalize transforms the input Element into a serialized XML document in canonical form.
func (c *c14N11Canonicalizer) Canonicalize(el *etree.Element) ([]byte, error) {
	scope := make(map[string]struct{})
	return canonicalSerialize(canonicalPrep(el, scope))
}

func (c *c14N11Canonicalizer) Algorithm() AlgorithmID {
	return CanonicalXML11AlgorithmId
}

func composeAttr(space, key string) string {
	if space != "" {
		return space + ":" + key
	}

	return key
}

type c14nSpace struct {
	a    etree.Attr
	used bool
}

const nsSpace = "xmlns"

// excCanonicalPrep accepts an *etree.Element and recursively transforms it into one
// which is ready for serialization to exclusive canonical form. Specifically this
// entails:
//
// 1. Stripping re-declarations of namespaces
// 2. Stripping unused namespaces
// 3. Sorting attributes into canonical order.
//
// NOTE(russell_h): Currently this function modifies the passed element.
func excCanonicalPrep(el *etree.Element, _nsAlreadyDeclared map[string]c14nSpace, inclusiveNamespaces map[string]struct{}) *etree.Element {
	//Copy alreadyDeclared map (only contains namespaces)
	nsAlreadyDeclared := make(map[string]c14nSpace, len(_nsAlreadyDeclared))
	for k := range _nsAlreadyDeclared {
		nsAlreadyDeclared[k] = _nsAlreadyDeclared[k]
	}

	//Track the namespaces used on the current element
	nsUsedHere := make(map[string]struct{})

	//Make sure to track the element namespace for the case:
	//<foo:bar xmlns:foo="..."/>
	if el.Space != "" {
		nsUsedHere[el.Space] = struct{}{}
	}

	toRemove := make([]string, 0, 0)

	for _, a := range el.Attr {
		switch a.Space {
		case nsSpace:

			//For simplicity, remove all xmlns attribues; to be added in one pass
			//later.  Otherwise, we need another map/set to track xmlns attributes
			//that we left alone.
			toRemove = append(toRemove, a.Space+":"+a.Key)
			if _, ok := nsAlreadyDeclared[a.Key]; !ok {
				//If we're not tracking ancestor state already for this namespace, add
				//it to the map
				nsAlreadyDeclared[a.Key] = c14nSpace{a: a, used: false}
			}

			// This algorithm accepts a set of namespaces which should be treated
			// in an inclusive fashion. Specifically that means we should keep the
			// declaration of that namespace closest to the root of the tree. We can
			// accomplish that be pretending it was used by this element.
			_, inclusive := inclusiveNamespaces[a.Key]
			if inclusive {
				nsUsedHere[a.Key] = struct{}{}
			}

		default:
			//We only track namespaces, so ignore attributes without one.
			if a.Space != "" {
				nsUsedHere[a.Space] = struct{}{}
			}
		}
	}

	//Remove all attributes so that we can add them with much-simpler logic
	for _, attrK := range toRemove {
		el.RemoveAttr(attrK)
	}

	//For all namespaces used on the current element, declare them if they were
	//not declared (and used) in an ancestor.
	for k := range nsUsedHere {
		spc := nsAlreadyDeclared[k]
		//If previously unused, mark as used
		if !spc.used {
			el.Attr = append(el.Attr, spc.a)
			spc.used = true

			//Assignment here is only to update the pre-existing `used` tracking value
			nsAlreadyDeclared[k] = spc
		}
	}

	//Canonicalize all children, passing down the ancestor tracking map
	for _, child := range el.ChildElements() {
		excCanonicalPrep(child, nsAlreadyDeclared, inclusiveNamespaces)
	}

	//Sort attributes lexicographically
	sort.Sort(etreeutils.SortedAttrs(el.Attr))

	return el.Copy()
}

// canonicalPrep accepts an *etree.Element and transforms it into one which is ready
// for serialization into inclusive canonical form. Specifically this
// entails:
//
// 1. Stripping re-declarations of namespaces
// 2. Sorting attributes into canonical order
//
// Inclusive canonicalization does not strip unused namespaces.
//
// TODO(russell_h): This is very similar to excCanonicalPrep - perhaps they should
// be unified into one parameterized function?
func canonicalPrep(el *etree.Element, seenSoFar map[string]struct{}) *etree.Element {
	_seenSoFar := make(map[string]struct{})
	for k, v := range seenSoFar {
		_seenSoFar[k] = v
	}

	ne := el.Copy()
	sort.Sort(etreeutils.SortedAttrs(ne.Attr))
	if len(ne.Attr) != 0 {
		for _, attr := range ne.Attr {
			if attr.Space != nsSpace {
				continue
			}
			key := attr.Space + ":" + attr.Key
			if _, seen := _seenSoFar[key]; seen {
				ne.RemoveAttr(attr.Space + ":" + attr.Key)
			} else {
				_seenSoFar[key] = struct{}{}
			}
		}
	}

	for i, token := range ne.Child {
		childElement, ok := token.(*etree.Element)
		if ok {
			ne.Child[i] = canonicalPrep(childElement, _seenSoFar)
		}
	}

	return ne
}

func canonicalSerialize(el *etree.Element) ([]byte, error) {
	doc := etree.NewDocument()
	doc.SetRoot(el)

	doc.WriteSettings = etree.WriteSettings{
		CanonicalAttrVal: true,
		CanonicalEndTags: true,
		CanonicalText:    true,
	}

	return doc.WriteToBytes()
}
