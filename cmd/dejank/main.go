package main

import (
	"fmt"
	"os"

	"github.com/thesavant42/dejank/internal/modes"
	"github.com/thesavant42/dejank/internal/ui"
	"github.com/spf13/cobra"
)

var (
	version = "2.0.0"
	verbose bool
	output  string
	force   bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "dejank",
		Short: "Skeleton key for sourcemap extraction",
		Long: `dejank by thesavant42

A surgical tool for unpacking JavaScript bundles using their sourcemaps.

It can restore original folder structures, extract inline sourcemaps, 
and decode embedded base64 assets (images, fonts, SVGs).`,
		Version: version,
	}

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", ".", "Output directory (domain folder created here)")
	rootCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "Overwrite existing output directory")

	// Add subcommands
	rootCmd.AddCommand(urlCmd())
	rootCmd.AddCommand(singleCmd())
	rootCmd.AddCommand(localCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func urlCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "url <webpage-url>",
		Short: "Crawl a webpage and extract sourcemaps from all scripts",
		Long: `Downloads all scripts from a webpage, finds their sourcemaps,
and restores the original source files.

Example:
  dejank url https://example.com`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := modes.DefaultConfig()
			cfg.Verbose = verbose
			cfg.OutputRoot = output
			cfg.Force = force

			targetURL := args[0]
			fmt.Print(ui.Banner(version))
			fmt.Print(ui.Target(targetURL))

			// Set up progress tracking
			var progress *ui.Progress
			cfg.OnProgress = func(event string, data interface{}) {
				switch event {
				case "scripts_found":
					total := data.(int)
					if total > 0 && !verbose {
						progress = ui.NewProgress(total, "Processing scripts")
					}
				case "processing_script":
					if progress != nil {
						progress.Increment()
					}
				}
			}

			result, err := modes.RunURL(cfg, targetURL)
			
			// Clean up progress bar
			if progress != nil {
				progress.Done()
			}

			if err != nil {
				fmt.Println(ui.Error(err.Error()))
				return err
			}

			fmt.Println(ui.SummaryHeader())
			fmt.Println(ui.SummaryLine("Scripts found:", result.ScriptsFound))
			fmt.Println(ui.SummaryLine("Scripts processed:", result.ScriptsProcessed))
			fmt.Println(ui.SummaryLine("Sources restored:", result.SourcesRestored))
			fmt.Println(ui.SummaryLine("Assets extracted:", result.AssetsExtracted))

			if len(result.Errors) > 0 {
				fmt.Println(ui.SummaryLine("Errors:", len(result.Errors)))
				if verbose {
					for _, e := range result.Errors {
						fmt.Printf("      %s\n", ui.DimStyle.Render(fmt.Sprintf("- %v", e)))
					}
				}
			}

			fmt.Println()
			return nil
		},
	}
}

func singleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "single <script-url>",
		Short: "Extract sourcemap from a single script URL",
		Long: `Downloads a single JavaScript file, finds its sourcemap,
and restores the original source files.

Example:
  dejank single https://example.com/static/app.bundle.js`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := modes.DefaultConfig()
			cfg.Verbose = verbose
			cfg.OutputRoot = output
			cfg.Force = force

			scriptURL := args[0]
			fmt.Print(ui.Banner(version))
			fmt.Print(ui.Target(scriptURL))

			result, err := modes.RunSingle(cfg, scriptURL)
			if err != nil {
				fmt.Println(ui.Error(err.Error()))
				return err
			}

			fmt.Println(ui.SummaryHeader())
			fmt.Println(ui.SummaryLine("Sourcemap found:", result.MapFound))
			fmt.Println(ui.SummaryLine("Sources restored:", result.SourcesRestored))

			if len(result.Errors) > 0 {
				fmt.Println(ui.SummaryLine("Errors:", len(result.Errors)))
				if verbose {
					for _, e := range result.Errors {
						fmt.Printf("      %s\n", ui.DimStyle.Render(fmt.Sprintf("- %v", e)))
					}
				}
			}

			fmt.Println()
			return nil
		},
	}
}

func localCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "local [target-directory]",
		Short: "Process local .js and .map files",
		Long: `Processes downloaded JavaScript and sourcemap files from a local directory.

If no target is specified, processes all domain directories under the output folder.

Examples:
  dejank local                           # Process all domains in current directory
  dejank local ./example.com             # Process specific domain folder`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := modes.DefaultConfig()
			cfg.Verbose = verbose
			cfg.OutputRoot = output
			cfg.Force = force

			var target string
			if len(args) > 0 {
				target = args[0]
			}

			fmt.Print(ui.Banner(version))
			if target != "" {
				fmt.Print(ui.Target(target))
			} else {
				fmt.Println(ui.Info(fmt.Sprintf("Processing all domains in: %s\n", ui.URLStyle.Render(output))))
			}

			result, err := modes.RunLocal(cfg, target)
			if err != nil {
				fmt.Println(ui.Error(err.Error()))
				return err
			}

			fmt.Println(ui.SummaryHeader())
			fmt.Println(ui.SummaryLine("Targets processed:", result.TargetsProcessed))
			fmt.Println(ui.SummaryLine("Maps processed:", result.MapsProcessed))
			fmt.Println(ui.SummaryLine("Sources restored:", result.SourcesRestored))
			fmt.Println(ui.SummaryLine("Assets extracted:", result.AssetsExtracted))

			if len(result.Errors) > 0 {
				fmt.Println(ui.SummaryLine("Errors:", len(result.Errors)))
				if verbose {
					for _, e := range result.Errors {
						fmt.Printf("      %s\n", ui.DimStyle.Render(fmt.Sprintf("- %v", e)))
					}
				}
			}

			fmt.Println()
			return nil
		},
	}
}
