package xmlsig

import "errors"

// Taken largely from an example in "Programming In Go"
// Keeping it separate from my stuff
type stack []interface{}

func (s *stack) Len() int {
	return len(*s)
}

func (s *stack) Push(x interface{}) {
	*s = append(*s, x)
}

func (s *stack) Top() (interface{}, error) {
	if len(*s) == 0 {
		return nil, errors.New("empty stack")
	}
	return (*s)[s.Len()-1], nil
}

func (s *stack) Pop() (interface{}, error) {
	theStack := *s
	if len(theStack) == 0 {
		return nil, errors.New("empty stack")
	}
	x := theStack[len(theStack)-1]
	*s = theStack[:len(theStack)-1]
	return x, nil
}
