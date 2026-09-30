package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zackly20/commit-ai/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Lihat atau ubah konfigurasi",
	Args:  cobra.NoArgs,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Tampilkan konfigurasi efektif (gabungan semua layer)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := resolvedConfig()
		if err != nil {
			return err
		}
		printConfig(cfg)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Tampilkan nilai satu key konfigurasi",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := resolvedConfig()
		if err != nil {
			return err
		}
		v, ok := configValue(cfg, args[0])
		if !ok {
			return fmt.Errorf("key %q tidak dikenal; gunakan: provider, endpoint, model, format, language, max_diff_chars, subject_max_length, confirm_before_commit", args[0])
		}
		fmt.Println(v)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Simpan nilai konfigurasi ke file (project jika ada, atau global)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		// Target file: project config jika ada, kalau tidak global.
		target := config.ProjectPath()
		if _, err := os.Stat(target); err != nil {
			gp, gerr := config.GlobalPath()
			if gerr != nil {
				return gerr
			}
			target = gp
		}

		cfg, err := config.LoadPath(target)
		if err != nil {
			return err
		}
		if err := setConfigValue(&cfg, key, value); err != nil {
			return err
		}
		if err := config.Validate(cfg); err != nil {
			return err
		}
		if err := config.Save(target, cfg); err != nil {
			return err
		}
		fmt.Printf("%s = %s (disimpan di %s)\n", key, value, target)
		return nil
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Kembalikan project config ke default bawaan",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.ProjectPath()
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("tidak ada %s di project ini", path)
		}
		if err := config.Save(path, config.Defaults()); err != nil {
			return err
		}
		fmt.Println("Project config dikembalikan ke default:", path)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd, configGetCmd, configSetCmd, configResetCmd)
}

func configValue(cfg config.Config, key string) (string, bool) {
	switch strings.ToLower(key) {
	case "provider":
		return cfg.Provider, true
	case "endpoint":
		return cfg.Endpoint, true
	case "model":
		return cfg.Model, true
	case "format":
		return cfg.Format, true
	case "language":
		return cfg.Language, true
	case "max_diff_chars":
		return fmt.Sprintf("%d", cfg.MaxDiffChars), true
	case "subject_max_length":
		return fmt.Sprintf("%d", cfg.SubjectMaxLength), true
	case "confirm_before_commit":
		return fmt.Sprintf("%t", cfg.ConfirmBeforeCommit), true
	default:
		return "", false
	}
}

func setConfigValue(cfg *config.Config, key, value string) error {
	switch strings.ToLower(key) {
	case "provider":
		cfg.Provider = value
	case "endpoint":
		cfg.Endpoint = value
	case "model":
		cfg.Model = value
	case "format":
		cfg.Format = value
	case "language":
		cfg.Language = value
	case "max_diff_chars":
		return setInt(&cfg.MaxDiffChars, key, value)
	case "subject_max_length":
		return setInt(&cfg.SubjectMaxLength, key, value)
	case "confirm_before_commit":
		switch strings.ToLower(value) {
		case "true", "1", "yes":
			cfg.ConfirmBeforeCommit = true
		case "false", "0", "no":
			cfg.ConfirmBeforeCommit = false
		default:
			return fmt.Errorf("nilai boolean tidak valid: %q", value)
		}
	default:
		return fmt.Errorf("key %q tidak dikenal", key)
	}
	return nil
}

func setInt(dst *int, key, value string) error {
	var n int
	if _, err := fmt.Sscanf(value, "%d", &n); err != nil {
		return fmt.Errorf("nilai %s harus angka: %q", key, value)
	}
	*dst = n
	return nil
}

func printConfig(cfg config.Config) {
	fmt.Println("provider:", cfg.Provider)
	fmt.Println("endpoint:", cfg.Endpoint)
	fmt.Println("model:", cfg.Model)
	fmt.Println("format:", cfg.Format)
	fmt.Println("language:", cfg.Language)
	fmt.Println("max_diff_chars:", cfg.MaxDiffChars)
	fmt.Println("subject_max_length:", cfg.SubjectMaxLength)
	fmt.Println("confirm_before_commit:", cfg.ConfirmBeforeCommit)
}
