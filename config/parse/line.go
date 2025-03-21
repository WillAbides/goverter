package parse

import (
	"bufio"
	"go/ast"
	"strings"
)

const (
	Prefix    = "goverter"
	Delimiter = ":"
)

func SettingLines(comment string) (lines []string) {
	scanner := bufio.NewScanner(strings.NewReader(comment))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, Prefix+Delimiter) {
			line := strings.TrimPrefix(line, Prefix+Delimiter)
			lines = append(lines, line)
		}
	}
	return lines
}

func CommentGroupSettingLines(group *ast.CommentGroup) []string {
	if group == nil {
		return nil
	}
	var settings []string
	for _, comment := range group.List {
		for _, line := range strings.Split(comment.Text, "\n") {
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "//")
			line = strings.TrimPrefix(line, "/*")
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, Prefix+Delimiter) {
				continue
			}
			settings = append(settings, strings.TrimPrefix(line, Prefix+Delimiter))
		}
	}
	return settings
}
