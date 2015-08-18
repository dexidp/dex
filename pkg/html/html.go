package html

import (
	"io"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// FormValues will return the values of a form on an html document.
func FormValues(formSelector string, body io.Reader) (url.Values, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	form := doc.Find(formSelector)
	inputs := form.Find("input")
	for _, input := range inputs.Nodes {
		inputName, ok := attrValue(input.Attr, "name")
		if !ok {
			continue
		}
		val, ok := attrValue(input.Attr, "value")
		if !ok {
			continue
		}

		values.Add(inputName, val)
	}
	return values, nil
}

func attrValue(attrs []html.Attribute, name string) (string, bool) {
	for _, attr := range attrs {
		if attr.Key == name {
			return attr.Val, true
		}
	}
	return "", false
}
