package main

import (
	"fmt"
	"os"
	"text/template"
)

func (inst *Instance) render_template(template_file string, dst_filename string) error {
	data, err := os.ReadFile(template_file)
	if err != nil {
		return fmt.Errorf("Template error : %w", err)
	}
	tmpl, err := template.New("void").Parse(string(data))
	if err != nil {
		return fmt.Errorf("Template error : %w", err)
	}
	file, err := os.OpenFile(dst_filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Template error : %w", err)
	}
	err = tmpl.Execute(file, inst)
	if err != nil {
		file.Close()
		return fmt.Errorf("Template error : %w", err)
	}
	file.Close()
	return nil
}
