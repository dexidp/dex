// Package gendoc generates documentation for REST APIs.
package gendoc

import (
	"encoding/json"
	"io"
	"path"
	"sort"
	"strings"
)

func ParseGoogleAPI(r io.Reader) (Document, error) {
	var d doc
	if err := json.NewDecoder(r).Decode(&d); err != nil {
		return Document{}, err
	}
	return d.toDocument(), nil
}

// doc represents a Google API specification document. It is NOT intended to encompass all
// options provided by the spec, only the minimal fields needed to convert dex's API
// definitions into documentation.
type doc struct {
	Name              string             `json:"name"`
	Version           string             `json:"version"`
	Title             string             `json:"title"`
	Description       string             `json:"description"`
	DocumentationLink string             `json:"documentationLink"`
	Protocol          string             `json:"protocol"`
	BasePath          string             `json:"basePath"`
	Schemas           map[string]schema  `json:"schemas"`
	Resources         map[string]methods `json:"resources"`
}

type methods struct {
	Methods map[string]resource `json:"methods"`
}

type param struct {
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Location string `json:"location"`
}

type schema struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Items       *schema `json:"items"`
	Format      string  `json:"format"`
	Properties  map[string]schema
	Ref         string `json:"$ref"`
}

type resource struct {
	Description string           `json:"description"`
	Method      string           `json:"httpMethod"`
	Path        string           `json:"path"`
	Parameters  map[string]param `json:"parameters"`
	Request     *ref             `json:"request"`
	Response    *ref             `json:"response"`
}

type ref struct {
	Ref string `json:"$ref"`
}

func (d doc) toDocument() Document {
	gDoc := Document{
		Title:       d.Title,
		Description: d.Description,
		Version:     d.Version,
	}
	for name, s := range d.Schemas {
		s.ID = name
		gDoc.Models = append(gDoc.Models, s.toSchema())
	}

	for object, methods := range d.Resources {
		for action, r := range methods.Methods {
			gDoc.Paths = append(gDoc.Paths, r.toPath(d, object, action))
		}
	}

	sort.Sort(byPath(gDoc.Paths))
	sort.Sort(byName(gDoc.Models))
	return gDoc
}

func (s schema) toSchema() Schema {
	sch := Schema{
		Name:        s.ID,
		Type:        s.Type,
		Description: s.Description,
		Ref:         s.Ref,
	}
	for name, prop := range s.Properties {
		c := prop.toSchema()
		c.Name = name
		sch.Children = append(sch.Children, c)
	}
	if s.Items != nil {
		sch.Children = []Schema{s.Items.toSchema()}
	}
	sort.Sort(byName(sch.Children))
	return sch
}

func (r resource) toPath(d doc, object, action string) Path {
	p := Path{
		Method:      r.Method,
		Path:        path.Join("/", r.Path),
		Summary:     strings.TrimSpace(action + " " + object),
		Description: r.Description,
	}
	for name, param := range r.Parameters {
		p.Parameters = append(p.Parameters, Parameter{
			Name:      name,
			LocatedIn: param.Location,
			Required:  param.Required,
			Type:      param.Type,
		})
	}
	if r.Request != nil {
		ref := r.Request.Ref
		p.Parameters = append(p.Parameters, Parameter{
			LocatedIn: "body",
			Required:  true,
			Type:      ref,
		})
	}
	if r.Response != nil {
		p.Responses = append(p.Responses, Response{
			Code: 200,
			Type: r.Response.Ref,
		})
	}
	p.Responses = append(p.Responses, Response{
		Code:        CodeDefault,
		Description: "Unexpected error",
	})
	return p
}
