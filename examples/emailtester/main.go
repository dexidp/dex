package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/coreos/dex/email"
	"github.com/coreos/dex/pkg/log"
)

func getEmailer(emailerConfigFile, emailTemplateDir string) (*email.TemplatizedEmailer, error) {
	cfg, err := email.NewEmailerConfigFromFile(emailerConfigFile)
	if err != nil {
		return nil, err
	}

	emailer, err := cfg.Emailer()
	if err != nil {
		return nil, err
	}

	tMailer, err := email.NewTemplatizedEmailerFromGlobs(emailTemplateDir+"/*.txt", emailTemplateDir+"/*.html", emailer)
	if err != nil {
		return nil, err
	}

	return tMailer, nil
}

var (
	tpls = map[string]map[string]interface{}{
		"verify-email": map[string]interface{}{
			"email": "test@example.com",
			"link":  "http://text.example.com",
		},
	}
)

func stderr(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format = format + "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
}

func die(format string, args ...interface{}) {
	stderr(format, args...)
	os.Exit(1)
}

func main() {
	log.EnableDebug()

	emailTemplates := flag.String("templates-dir", "./static/email", "directory of email template files")
	emailFrom := flag.String("from", "no-reply@example.com", "")
	emailTo := flag.String("to", "", "")
	emailConfig := flag.String("cfg", "./static/fixtures/emailer.json", "configures emailer.")
	tplName := flag.String("template", "verify-email", "which email template to use.")
	flag.Parse()
	emailer, err := getEmailer(*emailConfig, *emailTemplates)
	if err != nil {
		die("Error getting emailer: %v", err)
	}

	data, ok := tpls[*tplName]
	if !ok {
		die("no such template.")
	}

	if *emailTo == "" {
		die("--email-to is required")
	}
	err = emailer.SendMail(*emailFrom, "TEST EMAIL", *tplName, data, *emailTo)
	if err != nil {
		die("err: %v", err)
	}

}
