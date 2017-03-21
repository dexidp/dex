package etreeutils

import "github.com/beevik/etree"

type SortedAttrs []etree.Attr

func (a SortedAttrs) Len() int {
	return len(a)
}

func (a SortedAttrs) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a SortedAttrs) Less(i, j int) bool {
	// As I understand it: any "xmlns" attribute should come first, followed by any
	// any "xmlns:prefix" attributes, presumably ordered by prefix. Lastly any other
	// attributes in lexicographical order.
	if a[i].Space == defaultPrefix && a[i].Key == xmlnsPrefix {
		return true
	}

	if a[j].Space == defaultPrefix && a[j].Key == xmlnsPrefix {
		return false
	}

	if a[i].Space == xmlnsPrefix {
		if a[j].Space == xmlnsPrefix {
			return a[i].Key < a[j].Key
		}
		return true
	}

	if a[j].Space == xmlnsPrefix {
		return false
	}

	if a[i].Space == a[j].Space {
		return a[i].Key < a[j].Key
	}

	return a[i].Space < a[j].Space
}
