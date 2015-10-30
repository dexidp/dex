package email

import (
	"bytes"
	"errors"
	htmltemplate "html/template"
	"text/template"
)

// NewTemplatizedEmailerFromGlobs creates a new TemplatizedEmailer, parsing the templates found in the given filepattern globs.
func NewTemplatizedEmailerFromGlobs(textGlob, htmlGlob string, emailer Emailer) (*TemplatizedEmailer, error) {
	textTemplates, err := template.ParseGlob(textGlob)
	if err != nil {
		return nil, err
	}

	htmlTemplates, err := htmltemplate.ParseGlob(htmlGlob)
	if err != nil {
		return nil, err
	}

	return NewTemplatizedEmailerFromTemplates(textTemplates, htmlTemplates, emailer), nil
}

// NewTemplatizedEmailerFromTemplates creates a new TemplatizedEmailer, given root text and html templates.
func NewTemplatizedEmailerFromTemplates(textTemplates *template.Template, htmlTemplates *htmltemplate.Template, emailer Emailer) *TemplatizedEmailer {
	return &TemplatizedEmailer{
		emailer:       emailer,
		textTemplates: textTemplates,
		htmlTemplates: htmlTemplates,
	}
}

// TemplatizedEmailer sends emails based on templates, using any Emailer.
type TemplatizedEmailer struct {
	textTemplates *template.Template
	htmlTemplates *htmltemplate.Template
	emailer       Emailer
	globalCtx     map[string]interface{}
}

func (t *TemplatizedEmailer) SetGlobalContext(ctx map[string]interface{}) {
	t.globalCtx = ctx
}

// SendMail queues an email to be sent to a recipient.
// SendMail has similar semantics to Emailer.SendMail, except that you provide
// the template names you want to base the message on instead of the actual
// text. "to", "from" and "subject" will be added into the data map regardless
// of if they are used.
func (t *TemplatizedEmailer) SendMail(from, subject, tplName string, data map[string]interface{}, to string) error {
	if tplName == "" {
		return errors.New("Must provide a template name")
	}

	textTpl := t.textTemplates.Lookup(tplName + ".txt")
	htmlTpl := t.htmlTemplates.Lookup(tplName + ".html")

	if textTpl == nil && htmlTpl == nil {
		return ErrorNoTemplate
	}

	data["to"] = to
	data["from"] = from
	data["subject"] = subject

	for k, v := range t.globalCtx {
		data[k] = v
	}

	var textBuffer bytes.Buffer
	if textTpl != nil {
		err := textTpl.Execute(&textBuffer, data)
		if err != nil {
			return err
		}
	}

	var htmlBuffer bytes.Buffer
	if htmlTpl != nil {
		err := htmlTpl.Execute(&htmlBuffer, data)
		if err != nil {
			return err
		}
	}

	return t.emailer.SendMail(from, subject, textBuffer.String(), htmlBuffer.String(), to)
}
