package signedxml

import (
	"sort"
	"strings"

	"github.com/beevik/etree"
)

// the attribute and attributes structs are used to implement the sort.Interface
type attribute struct {
	prefix, uri, key, value string
}

type attributes []attribute

func (a attributes) Len() int {
	return len(a)
}

// Less is part of the sort.Interface, and is used to order attributes by their
// namespace URIs and then by their keys.
func (a attributes) Less(i, j int) bool {
	if a[i].uri == "" && a[j].uri != "" {
		return true
	}
	if a[j].uri == "" && a[i].uri != "" {
		return false
	}

	iQual := a[i].uri + a[i].key
	jQual := a[j].uri + a[j].key

	return iQual < jQual
}

func (a attributes) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// ExclusiveCanonicalization implements the CanonicalizationAlgorithm
// interface and is used for processing the
// http://www.w3.org/2001/10/xml-exc-c14n# and
// http://www.w3.org/2001/10/xml-exc-c14n#WithComments transform
// algorithms
type ExclusiveCanonicalization struct {
	WithComments                 bool
	inclusiveNamespacePrefixList []string
	namespaces                   map[string]string
}

// Process is called to transfrom the XML using the ExclusiveCanonicalization
// algorithm
func (e ExclusiveCanonicalization) Process(inputXML string,
	transformXML string) (outputXML string, err error) {

	e.namespaces = make(map[string]string)

	doc := etree.NewDocument()
	doc.WriteSettings.CanonicalEndTags = true
	doc.WriteSettings.CanonicalText = true
	doc.WriteSettings.CanonicalAttrVal = true

	err = doc.ReadFromString(inputXML)
	if err != nil {
		return "", err
	}

	e.loadPrefixList(transformXML)
	e.processDocLevelNodes(doc)
	e.processRecursive(doc.Root(), nil, "")

	outputXML, err = doc.WriteToString()
	return outputXML, err
}

func (e *ExclusiveCanonicalization) loadPrefixList(transformXML string) {
	if transformXML != "" {
		tDoc := etree.NewDocument()
		tDoc.ReadFromString(transformXML)
		inclNSNode := tDoc.Root().SelectElement("InclusiveNamespaces")
		if inclNSNode != nil {
			prefixList := inclNSNode.SelectAttrValue("PrefixList", "")
			if prefixList != "" {
				e.inclusiveNamespacePrefixList = strings.Split(prefixList, " ")
			}
		}
	}
}

// process nodes outside of the root element
func (e ExclusiveCanonicalization) processDocLevelNodes(doc *etree.Document) {
	// keep track of the previous node action to manage line returns in CharData
	previousNodeRemoved := false

	for i := 0; i < len(doc.Child); i++ {
		c := doc.Child[i]

		switch c := c.(type) {
		case *etree.Comment:
			if e.WithComments {
				previousNodeRemoved = false
			} else {
				removeTokenFromDocument(c, doc)
				i--
				previousNodeRemoved = true
			}
		case *etree.CharData:
			if isWhitespace(c.Data) {
				if previousNodeRemoved {
					removeTokenFromDocument(c, doc)
					i--
					previousNodeRemoved = true
				} else {
					c.Data = "\n"
				}

			}
		case *etree.Directive:
			removeTokenFromDocument(c, doc)
			i--
			previousNodeRemoved = true
		case *etree.ProcInst:
			// remove declaration, but leave other PI's
			if c.Target == "xml" {
				removeTokenFromDocument(c, doc)
				i--
				previousNodeRemoved = true
			} else {
				previousNodeRemoved = false
			}
		default:
			previousNodeRemoved = false
		}
	}

	// if the last line is CharData whitespace, then remove it
	if c, ok := doc.Child[len(doc.Child)-1].(*etree.CharData); ok {
		if isWhitespace(c.Data) {
			removeTokenFromDocument(c, doc)
		}
	}
}

func (e ExclusiveCanonicalization) processRecursive(node *etree.Element,
	prefixesInScope []string, defaultNS string) {

	newDefaultNS, newPrefixesInScope :=
		e.renderAttributes(node, prefixesInScope, defaultNS)

	for _, child := range node.Child {
		switch child := child.(type) {
		case *etree.Comment:
			if !e.WithComments {
				removeTokenFromElement(etree.Token(child), node)
			}
		case *etree.Element:
			e.processRecursive(child, newPrefixesInScope, newDefaultNS)
		}
	}
}

func (e ExclusiveCanonicalization) renderAttributes(node *etree.Element,
	prefixesInScope []string, defaultNS string) (newDefaultNS string,
	newPrefixesInScope []string) {

	currentNS := node.SelectAttrValue("xmlns", defaultNS)
	elementAttributes := []etree.Attr{}
	nsListToRender := make(map[string]string)
	attrListToRender := attributes{}

	// load map with for prefix -> uri lookup
	for _, attr := range node.Attr {
		if attr.Space == "xmlns" {
			e.namespaces[attr.Key] = attr.Value
		}
	}

	// handle the namespace of the node itself
	if node.Space != "" {
		if !contains(prefixesInScope, node.Space) {
			nsListToRender["xmlns:"+node.Space] = e.namespaces[node.Space]
			prefixesInScope = append(prefixesInScope, node.Space)
		}
	} else if defaultNS != currentNS {
		newDefaultNS = currentNS
		elementAttributes = append(elementAttributes,
			etree.Attr{Key: "xmlns", Value: currentNS})
	}

	for _, attr := range node.Attr {
		// include the namespaces if they are in the inclusiveNamespacePrefixList
		if attr.Space == "xmlns" {
			if !contains(prefixesInScope, attr.Key) &&
				contains(e.inclusiveNamespacePrefixList, attr.Key) {

				nsListToRender["xmlns:"+attr.Key] = attr.Value
				prefixesInScope = append(prefixesInScope, attr.Key)
			}
		}

		// include namespaces for qualfied attributes
		if attr.Space != "" &&
			attr.Space != "xmlns" &&
			!contains(prefixesInScope, attr.Space) {

			nsListToRender["xmlns:"+attr.Space] = e.namespaces[attr.Space]
			prefixesInScope = append(prefixesInScope, attr.Space)
		}

		// inclued all non-namespace attributes
		if attr.Space != "xmlns" && attr.Key != "xmlns" {
			attrListToRender = append(attrListToRender,
				attribute{
					prefix: attr.Space,
					uri:    e.namespaces[attr.Space],
					key:    attr.Key,
					value:  attr.Value,
				})
		}
	}

	// sort and add the namespace attributes first
	sortedNSList := getSortedNamespaces(nsListToRender)
	elementAttributes = append(elementAttributes, sortedNSList...)
	// then sort and add the non-namespace attributes
	sortedAttributes := getSortedAttributes(attrListToRender)
	elementAttributes = append(elementAttributes, sortedAttributes...)
	// replace the nodes attributes with the sorted copy
	node.Attr = elementAttributes
	return currentNS, prefixesInScope
}

func contains(slice []string, value string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}

// getSortedNamespaces sorts the namespace attributes by their prefix
func getSortedNamespaces(list map[string]string) []etree.Attr {
	var keys []string
	for k := range list {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	elem := etree.Element{}
	for _, k := range keys {
		elem.CreateAttr(k, list[k])
	}

	return elem.Attr
}

// getSortedAttributes sorts attributes by their namespace URIs
func getSortedAttributes(list attributes) []etree.Attr {
	sort.Sort(list)
	attrs := make([]etree.Attr, len(list))
	for i, a := range list {
		attrs[i] = etree.Attr{
			Space: a.prefix,
			Key:   a.key,
			Value: a.value,
		}
	}
	return attrs
}

func removeTokenFromElement(token etree.Token, e *etree.Element) *etree.Token {
	for i, t := range e.Child {
		if t == token {
			e.Child = append(e.Child[0:i], e.Child[i+1:]...)
			return &t
		}
	}
	return nil
}

func removeTokenFromDocument(token etree.Token, d *etree.Document) *etree.Token {
	for i, t := range d.Child {
		if t == token {
			d.Child = append(d.Child[0:i], d.Child[i+1:]...)
			return &t
		}
	}
	return nil
}

// isWhitespace returns true if the byte slice contains only
// whitespace characters.
func isWhitespace(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i]; c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return false
		}
	}
	return true
}
