package server

import "testing"

func TestNewTemplates(t *testing.T) {
	var config TemplateConfig
	if _, err := loadTemplates(config); err != nil {
		t.Fatal(err)
	}
}

func TestLoadTemplates(t *testing.T) {
	var config TemplateConfig

	config.Dir = "../web/templates"
}
