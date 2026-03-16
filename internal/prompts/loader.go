package prompts

import (
	_ "embed"
	"html/template"
	"sync"
)

//go:embed templates/v1/system.tmpl
var v1SystemTmpl string

//go:embed templates/v1/react_cot.tmpl
var v1ReactCoTTmpl string

//go:embed templates/v1/output_format.tmpl
var v1OutputFormatTmpl string

var (
	loadOnce sync.Once
	v1Tmpls  *template.Template
)

// loadV1Templates parses all v1 templates. Safe to call multiple times.
func loadV1Templates() (*template.Template, error) {
	var err error
	loadOnce.Do(func() {
		v1Tmpls, err = template.New("v1").Parse(v1SystemTmpl)
		if err != nil {
			return
		}
		_, err = v1Tmpls.New("react_cot").Parse(v1ReactCoTTmpl)
		if err != nil {
			return
		}
		_, err = v1Tmpls.New("output_format").Parse(v1OutputFormatTmpl)
	})
	return v1Tmpls, err
}
