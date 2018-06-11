package system

import (
	"bytes"
	"encoding/base64"
	"text/template"
)

func (s System) configs() ([]config, error) {
	configs := []config{}
	for path, content := range defaultTemplates {
		template, err := template.New(path).Funcs(template.FuncMap{"b64dec": base64decode}).Parse(content)
		if err != nil {
			return configs, err
		}
		configs = append(configs, config{
			template:   template,
			path:       path,
			filesystem: s.Filesystem,
		})
	}
	return configs, nil
}

func base64decode(v string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type config struct {
	template   *template.Template
	path       string
	filesystem Filesystem
}

func (c config) Write(data interface{}) error {
	var buff bytes.Buffer
	err := c.template.Execute(&buff, data)
	if err != nil {
		return err
	}
	return c.filesystem.Sync(&buff, c.path, 0640)
}
