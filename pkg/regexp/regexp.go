package regexp

import (
	"regexp/syntax"
	"slices"
)

// isNegatedOrBroadClass determines whether a character class (e.g., [0-9], [^/], [a-z])
// allows a character range wide enough to act as an unrestricted wildcard.
//
// In Go's regex AST (regexp/syntax), character classes (syntax.OpCharClass) are represented
// in the 'Rune' slice as a sequence of inclusive range pairs: [start1, end1, start2, end2, ...].
//
// For example:
//   - [0-9]      is encoded as: ['0', '9']                      -> 10 total characters
//   - [a-zA-Z]   is encoded as: ['A', 'Z', 'a', 'z']             -> 52 total characters
//   - [^/]       (negated class) expands into two vast ranges:
//     [UnicodeMin, '/'-1, '/'+1, UnicodeMax]          -> >1,100,000 characters
//
// If the total number of allowed characters exceeds the threshold (100), the class is
// considered either a negated class (like [^/]) or overly permissive, and thus classified
// as an unrestricted wildcard.
func isNegatedOrBroadClass(runes []rune) bool {
	var totalChars int32

	// Rune ranges are always stored in start/end pairs at even/odd indices:
	// runes[i]   = range start
	// runes[i+1] = range end
	for i := 0; i < len(runes); i += 2 {
		totalChars += (runes[i+1] - runes[i] + 1)
	}

	// Safety threshold: Allows common URL character sets like [a-zA-Z0-9_-] (~65 chars),
	// while flagging negated classes or broad ranges that could match arbitrary subdomains.
	return totalChars > 100
}

func inspectForWildcardSegments(node *syntax.Regexp) bool {
	// Inspect for child elements first
	// Any child element has an unrestricted wildcard
	if slices.ContainsFunc(node.Sub, inspectForWildcardSegments) {
		return true
	}

	switch node.Op {
	// Unrestricter char wildcard (AKA. the dot)
	case syntax.OpAnyChar, syntax.OpAnyCharNotNL:
		return true

	// Check for negated class chars with a wide allowed chars, marked as unrestricted wildcard.
	// Ex: [^/], [^#], many others...
	case syntax.OpCharClass:
		if isNegatedOrBroadClass(node.Rune) {
			return true
		}
	}

	return false
}

// HasArbitraryWildcard parses the regex pattern to check if it contains unrestricted wildcards.
//
// An unrestricted wildcard includes constructs like '.', '.*', '.+', or broad/negated character
// classes like '[^/]+'.
//
// NOTE: Unescaped literal dots in domain names (e.g., "example.com" instead of "example\.com")
// are parsed as syntax.OpAnyCharNotNL and will be flagged as wildcards. This prevents subtle
// open-redirect vulnerabilities where "pr-123.example.com" could match "pr-123xexample.com"
func HasArbitraryWildcard(pattern string) (bool, error) {
	ast, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return false, err
	}

	return inspectForWildcardSegments(ast), nil
}

func SurroundRedirectURIRegexp(uri string) string {
	return `\A(?:` + uri + `)\z`
}
