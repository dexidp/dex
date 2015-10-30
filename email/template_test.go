package email

import (
	htmltemplate "html/template"
	"testing"
	"text/template"
)

const (
	textTemplateString = `{{define "T1.txt"}}{{.gift}} from {{.from}} to {{.to}}.{{end}}
{{define "T3.txt"}}Hello there, {{.name}}!{{end}}
{{define "T4.txt"}}Hello there, {{.name}}! Welcome to {{.planet}}!{{end}}
`
	htmlTemplateString = `{{define "T1.html"}}<html><body>{{.gift}} from {{.from}} to {{.to}}.</body></html>{{end}}
{{define "T2.html"}}<html><body>Hello, {{.name}}!</body></html>{{end}}
{{define "T4.html"}}<html><body>Hello there, {{.name}}! Welcome to {{.planet}}!</body></html>{{end}}
`
)

type testEmailer struct {
	from, subject, text, html string
	to                        []string
}

func (t *testEmailer) SendMail(from, subject, text, html string, to ...string) error {
	t.from = from
	t.subject = subject
	t.text = text
	t.html = html
	t.to = to
	return nil
}

func TestTemplatizedEmailSendMail(t *testing.T) {
	textTemplates := template.New("text")
	_, err := textTemplates.Parse(textTemplateString)
	if err != nil {
		t.Fatalf("error parsing text templates: %v", err)
	}

	htmlTemplates := htmltemplate.New("html")
	_, err = htmlTemplates.Parse(htmlTemplateString)
	if err != nil {
		t.Fatalf("error parsing html templates: %v", err)
	}

	htmlStart := "<html><body>"
	htmlEnd := "</body></html>"
	tests := []struct {
		tplName           string
		from, to, subject string
		data              map[string]interface{}
		wantText          string
		wantHtml          string
		wantErr           bool
		ctx               map[string]interface{}
	}{
		{
			tplName: "T1",
			from:    "bob@example.com",
			to:      "alice@example.com",
			subject: "hello there",
			data: map[string]interface{}{
				"gift": "Greetings",
			},
			wantText: "Greetings from bob@example.com to alice@example.com.",
			wantHtml: htmlStart + "Greetings from bob@example.com to alice@example.com." + htmlEnd,
		},
		{
			tplName: "T2",
			from:    "bob@example.com",
			to:      "alice@example.com",
			subject: "hello there",
			data: map[string]interface{}{
				"name": "Alice",
			},
			wantText: "",
			wantHtml: htmlStart + "Hello, Alice!" + htmlEnd,
		},
		{
			tplName: "T3",
			from:    "bob@example.com",
			to:      "alice@example.com",
			subject: "hello there",
			data: map[string]interface{}{
				"name": "Alice",
			},
			wantText: "Hello there, Alice!",
			wantHtml: "",
		},
		{
			// make sure HTML stuff gets escaped.
			tplName: "T2",
			from:    "bob@example.com",
			to:      "alice@example.com",
			subject: "hello there",
			data: map[string]interface{}{
				"name": "Alice<script>alert('hacked!')</script>",
			},
			wantText: "",
			wantHtml: htmlStart + "Hello, Alice&lt;script&gt;alert(&#39;hacked!&#39;)&lt;/script&gt;!" + htmlEnd,
		},
		{
			tplName: "T4",
			from:    "bob@example.com",
			to:      "alice@example.com",
			subject: "hello there",
			data: map[string]interface{}{
				"name": "Alice",
			},
			wantText: "Hello there, Alice! Welcome to Mars!",
			ctx: map[string]interface{}{
				"planet": "Mars",
			},
			wantHtml: "<html><body>Hello there, Alice! Welcome to Mars!</body></html>",
		},
	}

	for i, tt := range tests {
		emailer := &testEmailer{}
		templatizer := NewTemplatizedEmailerFromTemplates(textTemplates, htmlTemplates, emailer)
		if tt.ctx != nil {
			templatizer.SetGlobalContext(tt.ctx)
		}

		err := templatizer.SendMail(tt.from, tt.subject, tt.tplName, tt.data, tt.to)
		if tt.wantErr {
			if err == nil {
				t.Errorf("case %d: err == nil, want non-nil err", i)
			}
			continue
		}

		if emailer.from != tt.from {
			t.Errorf("case %d: want=%q, got=%q", i, tt.from, emailer.from)
		}
		if emailer.subject != tt.subject {
			t.Errorf("case %d: want=%q, got=%q", i, tt.subject, emailer.subject)
		}
		if emailer.text != tt.wantText {
			t.Errorf("case %d: want=%q, got=%q", i, tt.wantText, emailer.text)
		}
		if emailer.html != tt.wantHtml {
			t.Errorf("case %d: want=%q, got=%q", i, tt.wantHtml, emailer.html)
		}
		if len(emailer.to) != 1 {
			t.Errorf("case %d: want=1, got=%d", i, len(emailer.to))
		}
		if emailer.to[0] != tt.to {
			t.Errorf("case %d: want=%q, got=%q", i, tt.to, emailer.to[0])
		}

	}

}
