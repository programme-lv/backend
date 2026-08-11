package mail

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"
	texttemplate "text/template"
)

//go:embed templates/*
var templateFS embed.FS

type TemplateData struct {
	Username   string
	ActionURL  string
	ExpiryNote string
}

type RenderedEmail struct {
	Subject  string
	TextBody string
	HTMLBody string
}

func RenderPasswordReset(data TemplateData) (RenderedEmail, error) {
	return render("password_reset", "Paroles atjaunošana — programme.lv", data)
}

func RenderEmailVerify(data TemplateData) (RenderedEmail, error) {
	return render("email_verify", "Apstipriniet e-pastu — programme.lv", data)
}

func render(name, subject string, data TemplateData) (RenderedEmail, error) {
	textTpl, err := texttemplate.ParseFS(templateFS, "templates/"+name+".txt")
	if err != nil {
		return RenderedEmail{}, fmt.Errorf("parse text template %s: %w", name, err)
	}
	htmlTpl, err := template.ParseFS(templateFS, "templates/"+name+".html")
	if err != nil {
		return RenderedEmail{}, fmt.Errorf("parse html template %s: %w", name, err)
	}

	var textBuf, htmlBuf bytes.Buffer
	if err := textTpl.Execute(&textBuf, data); err != nil {
		return RenderedEmail{}, fmt.Errorf("execute text template %s: %w", name, err)
	}
	if err := htmlTpl.Execute(&htmlBuf, data); err != nil {
		return RenderedEmail{}, fmt.Errorf("execute html template %s: %w", name, err)
	}

	return RenderedEmail{
		Subject:  subject,
		TextBody: strings.TrimSpace(textBuf.String()),
		HTMLBody: htmlBuf.String(),
	}, nil
}
