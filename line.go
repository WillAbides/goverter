package goverter

import (
	"go/ast"
	"strings"
)

const directivePrefix = "goverter:"

func commentGroupSettingLines(groups []*ast.CommentGroup) []string {
	var settings []string
	for _, group := range groups {
		if group == nil {
			return nil
		}
		for _, comment := range group.List {
			for _, line := range strings.Split(comment.Text, "\n") {
				line = strings.TrimSpace(line)
				line = strings.TrimPrefix(line, "//")
				line = strings.TrimPrefix(line, "/*")
				line = strings.TrimSpace(line)
				if !strings.HasPrefix(line, directivePrefix) {
					continue
				}
				settings = append(settings, strings.TrimPrefix(line, directivePrefix))
			}
		}
	}
	return settings
}
