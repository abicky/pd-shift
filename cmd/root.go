package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	toolName = "pd-shift"
)

// These variables should be overwritten by -ldflags
var (
	version  = "dev"
	revision = "HEAD"
)

var vipers = make(map[*cobra.Command]*viper.Viper)

var (
	defaultCommandGroup = &cobra.Group{
		ID:    "default",
		Title: "Commands:",
	}
	auxiliaryCommandGroup = &cobra.Group{
		ID:    "auxiliary",
		Title: "Auxiliary Commands:",
	}
)

var rootCmd = &cobra.Command{
	Use:     toolName,
	Short:   "A CLI tool for managing PagerDuty on-call shifts",
	Long:    "A CLI tool for managing PagerDuty on-call shifts",
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Workaround for https://github.com/spf13/cobra/issues/1918
		for c := cmd; c != nil; c = c.Parent() {
			if c.GroupID == auxiliaryCommandGroup.ID {
				c.Root().PersistentFlags().Lookup("api-key").Annotations[cobra.BashCompOneRequiredFlag] = []string{"false"}
				break
			}
		}
		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.SetVersionTemplate(fmt.Sprintf(
		`{{with .Name}}{{printf "%%s " .}}{{end}}{{printf "version %%s" .Version}} (revision %s)
`, revision))

	rootCmd.AddGroup(defaultCommandGroup, auxiliaryCommandGroup)
	rootCmd.SetHelpCommandGroupID(auxiliaryCommandGroup.ID)
	rootCmd.SetCompletionCommandGroupID(auxiliaryCommandGroup.ID)

	rootCmd.PersistentFlags().String("config", "", "Path to config file")
	rootCmd.PersistentFlags().String("api-key", "", "PagerDuty API key")
	rootCmd.MarkPersistentFlagRequired("api-key")
}

func initConfig() {
	// Allow using "PD_SHIFT_SOME_FLAG" environment variable for "some-flag" flag
	// and "PD_SHIFT_SOME_NESTED_FLAG" environment variable for "some.nested-flag" flag
	viper.SetEnvPrefix(toolName)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	// Bind persistent flags to reflect the config flag
	cobra.CheckErr(viper.BindPFlags(rootCmd.PersistentFlags()))

	if cfgFile := viper.GetString("config"); cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		cobra.CheckErr(viper.ReadInConfig())
	} else {
		viper.SetConfigName("config")

		// Add config paths according to https://specifications.freedesktop.org/basedir-spec/0.8/
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			home, err := os.UserHomeDir()
			cobra.CheckErr(err)
			configHome = filepath.Join(home, ".config")
		}
		viper.AddConfigPath(filepath.Join(configHome, toolName))

		configDirs := os.Getenv("XDG_CONFIG_DIRS")
		if configDirs == "" {
			configDirs = "/etc/xdg"
		}
		for _, v := range strings.Split(configDirs, ":") {
			viper.AddConfigPath(filepath.Join(v, toolName))
		}

		viper.ReadInConfig()
	}

	cobra.CheckErr(bindPFlags(viper.GetViper(), rootCmd))
}

func bindPFlags(v *viper.Viper, cmd *cobra.Command) error {
	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return err
	}
	for _, c := range cmd.Commands() {
		name := strings.Split(c.Use, " ")[0]
		// Set the default value to prevent Viper.Sub from returning nil
		v.SetDefault(name, make(map[string]any))
		subv := v.Sub(name)
		if err := bindPFlags(subv, c); err != nil {
			return err
		}
		vipers[c] = subv
	}

	// Workaround for MarkFlagRequired
	// cf. https://github.com/spf13/viper/issues/397#issuecomment-544272457
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if v.IsSet(f.Name) {
			isSet := false
			cmd.Flags().Visit(func(flag *pflag.Flag) {
				if flag.Name == f.Name {
					isSet = true
					return
				}
			})
			if isSet {
				return
			}

			if value := v.GetString(f.Name); value != "" {
				cmd.Flags().Set(f.Name, value)
			} else if value := v.GetStringSlice(f.Name); len(value) > 0 {
				cmd.Flags().Set(f.Name, strings.Join(value, ","))
			}
		}
	})

	return nil
}
