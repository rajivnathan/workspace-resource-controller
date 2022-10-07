package templates

import (
	"bytes"
	"text/template"
)

type TemplateData struct {
	Content []byte
	Args    any
}

// RenderResources renders the resources from the provided template
func RenderResources(tmplContent []byte, tmplArgs any) ([]byte, error) {
	// templateBytes, err := embeddedResources.ReadFile("resources.yaml")
	// if err != nil {
	// 	return nil, err
	// }
	tmpl, err := template.New("resourceTemplate").Parse(string(tmplContent))
	if err != nil {
		return nil, err
	}
	buffer := bytes.NewBuffer([]byte{})
	err = tmpl.Execute(buffer, tmplArgs)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
