package config

import (
	"go/ast"
	"strings"
)

const DirectivePrefix = "goverter:"

func CommentGroupSettingLines(groups []*ast.CommentGroup) []string {
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
				if !strings.HasPrefix(line, DirectivePrefix) {
					continue
				}
				settings = append(settings, strings.TrimPrefix(line, DirectivePrefix))
			}
		}
	}
	return settings
}
