package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zackly20/commit-ai/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Buat file konfigurasi .commit-ai.yaml di repository",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.ProjectPath()
		if _, err := os.Stat(path); err == nil {
			fmt.Println("File konfigurasi sudah ada:", path)
			return nil
		}
		if err := os.WriteFile(path, []byte(config.TemplateYAML()), 0o644); err != nil {
			return fmt.Errorf("tulis %s: %w", path, err)
		}
		fmt.Println("Dibuat:", path)
		fmt.Println("Edit model/endpoint sesuai kebutuhanmu.")
		return nil
	},
}
