package cmd

import (
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func Test_Execute(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		errPrefix string
	}{
		{
			name:      "help",
			args:      []string{"help"},
			errPrefix: "",
		},
		{
			name:      "completion",
			args:      []string{"completion"},
			errPrefix: "",
		},
		{
			name:      "completion bash",
			args:      []string{"completion", "bash"},
			errPrefix: "",
		},
		{
			name:      "__test__",
			args:      []string{"__test__"},
			errPrefix: "required flag(s) \"api-key\"",
		},
	}

	origOut := rootCmd.OutOrStdout()
	origErr := rootCmd.ErrOrStderr()
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	testCommand := &cobra.Command{
		Use: "__test__",
		Run: func(_ *cobra.Command, _ []string) {},
	}
	rootCmd.AddCommand(testCommand)

	defer func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(origOut)
		rootCmd.SetErr(origErr)
		rootCmd.RemoveCommand(testCommand)
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			defer func() {
				rootCmd.MarkPersistentFlagRequired("api-key")
			}()

			err := rootCmd.Execute()
			if tt.errPrefix != "" {
				if err == nil {
					t.Errorf("err = nil, want \"%s...\"", tt.errPrefix)
				} else if !strings.HasPrefix(err.Error(), tt.errPrefix) {
					t.Errorf("err = %v, want \"%s...\"", err, tt.errPrefix)
				}
			} else if err != nil {
				t.Errorf("err = %v, want nil", err)
			}
		})
	}
}
