package goverter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_parseArgs(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		tests := []struct {
			args     []string
			contains string
		}{
			{args: []string{}, contains: "Error: invalid args"},
			{args: []string{"goverter"}, contains: "goverter gen [OPTIONS]"},
			{args: []string{"goverter"}, contains: "Error: missing command"},
			{args: []string{"goverter", "-u"}, contains: "Error: flag provided but not defined: -u"},
			{args: []string{"goverter", "test"}, contains: "Error: unknown command test"},
			{args: []string{"goverter", "gen"}, contains: "Error: missing PATTERN"},
			{args: []string{"goverter", "gen", "-u"}, contains: "Error: flag provided but not defined: -u"},
			{args: []string{"goverter", "gen", "-g"}, contains: "Error: flag needs an argument: -g"},
		}

		for _, test := range tests {
			t.Run(strings.Join(test.args, " "), func(t *testing.T) {
				_, err := parseArgs(test.args)
				require.ErrorContains(t, err, test.contains)
			})
		}
	})

	t.Run("help", func(t *testing.T) {
		tests := [][]string{
			{"goverter", "help"},
			{"goverter", "-h"},
			{"goverter", "--help"},
			{"goverter", "gen", "-h"},
			{"goverter", "gen", "--help"},
		}

		for _, test := range tests {
			t.Run(strings.Join(test, " "), func(t *testing.T) {
				cmd, err := parseArgs(test)
				require.NoError(t, err)
				require.IsType(t, &helpCmd{}, cmd)
			})
		}
	})

	t.Run("version", func(t *testing.T) {
		cmd, err := parseArgs([]string{"goverter", "version"})
		require.NoError(t, err)
		require.IsType(t, &versionCmd{}, cmd)
	})

	t.Run("success", func(t *testing.T) {
		actual, err := parseArgs([]string{
			"goverter",
			"gen",
			"-cwd", "file/path",
			"-build-tags", "",
			"-output-constraint", "",
			"-g", "g1",
			"-global", "g2",
			"-g", "g3 oops",
			"pattern1", "pattern2",
		})
		require.NoError(t, err)

		expected := &generateCmd{
			Config: &generateCmdConfig{
				PackagePatterns:       []string{"pattern1", "pattern2"},
				WorkingDir:            "file/path",
				OutputBuildConstraint: "",
				BuildTags:             "",
				EnumTransformers:      map[string]EnumTransformer{},
				Global: rawLines{
					Location: "command line (-g, -global)",
					Lines:    []string{"g1", "g2", "g3 oops"},
				},
			},
		}
		require.Equal(t, expected, actual)
	})

	t.Run("default", func(t *testing.T) {
		actual, err := parseArgs([]string{"goverter", "gen", "pattern"})
		require.NoError(t, err)

		expected := &generateCmd{
			Config: &generateCmdConfig{
				PackagePatterns:       []string{"pattern"},
				WorkingDir:            "",
				OutputBuildConstraint: "!goverter",
				BuildTags:             "goverter",
				EnumTransformers:      map[string]EnumTransformer{},
				Global: rawLines{
					Location: "command line (-g, -global)",
					Lines:    nil,
				},
			},
		}
		require.Equal(t, expected, actual)
	})
}
