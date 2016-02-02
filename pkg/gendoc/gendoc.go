package gendoc

type Document struct {
	Title       string
	Description string
	Version     string
	Paths       []Path
	Models      []Schema
}

type Path struct {
	Method      string
	Path        string
	Summary     string
	Description string
	Parameters  []Parameter
	Responses   []Response
}

type byPath []Path

func (p byPath) Len() int      { return len(p) }
func (p byPath) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p byPath) Less(i, j int) bool {
	if p[i].Path == p[j].Path {
		return p[i].Method < p[j].Method
	}
	return p[i].Path < p[j].Path
}

type Parameter struct {
	Name        string
	LocatedIn   string
	Description string
	Required    bool
	Type        string
}

const (
	TypeArray  = "array"
	TypeBool   = "boolean"
	TypeFloat  = "float"
	TypeInt    = "integer"
	TypeObject = "object"
	TypeString = "string"
)

type Schema struct {
	Name        string
	Type        string
	Description string
	Children    []Schema
	Ref         string
}

type byName []Schema

func (n byName) Len() int           { return len(n) }
func (n byName) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n byName) Less(i, j int) bool { return n[i].Name < n[j].Name }

const CodeDefault = 0

type Response struct {
	Code        int // 0 means "Default"
	Description string
	Type        string
}
