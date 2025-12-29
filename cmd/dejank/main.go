package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/thesavant42/dejank/internal/modes"
	"github.com/thesavant42/dejank/internal/ui"
)

var version = "1.0.9"

func main() {
	// Global flags
	verbose := flag.Bool("v", false, "Enable verbose output")
	output := flag.String("o", ".", "Output directory")
	force := flag.Bool("f", false, "Overwrite existing output")
	showVersion := flag.Bool("version", false, "Show version")
	flag.Parse()

	args := flag.Args()

	if *showVersion {
		fmt.Println(ui.Banner(version))
		return
	}

	if len(args) < 1 {
		printHelp()
		return
	}

	command := args[0]
	cmdArgs := args[1:]

	cfg := modes.DefaultConfig()
	cfg.Verbose = *verbose
	cfg.OutputRoot = *output
	cfg.Force = *force

	switch command {
	case "url":
		runURL(cfg, cmdArgs)
	case "single":
		runSingle(cfg, cmdArgs)
	case "local":
		runLocal(cfg, cmdArgs)
	case "help":
		printHelp()
	default:
		fmt.Println(ui.Error(fmt.Sprintf("Unknown command: %s", command)))
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(ui.Banner(version))
	fmt.Println()
	fmt.Println(ui.TextStyle.Render("A surgical tool for unpacking JavaScript bundles using their sourcemaps."))
	fmt.Println()

	fmt.Println(ui.AccentStyle.Render("USAGE"))
	fmt.Printf("  %s\n", ui.FormatUsage("dejank <command> [options] <target>"))
	fmt.Println()

	fmt.Println(ui.AccentStyle.Render("COMMANDS"))
	fmt.Printf("  %s    %s\n", ui.InfoStyle.Render("url"), ui.TextStyle.Render("Crawl webpage, extract sourcemaps from all scripts"))
	fmt.Printf("  %s %s\n", ui.InfoStyle.Render("single"), ui.TextStyle.Render("Extract sourcemap from a single script URL"))
	fmt.Printf("  %s  %s\n", ui.InfoStyle.Render("local"), ui.TextStyle.Render("Process local .js and .map files"))
	fmt.Printf("  %s   %s\n", ui.InfoStyle.Render("help"), ui.TextStyle.Render("Show this help"))
	fmt.Println()

	fmt.Println(ui.AccentStyle.Render("OPTIONS"))
	fmt.Printf("  %s\n", ui.FormatUsage("-v       Verbose output"))
	fmt.Printf("  %s\n", ui.FormatUsage("-f       Force overwrite existing output"))
	fmt.Printf("  %s\n", ui.FormatUsage("-o <dir> Output directory (default: .)"))
	fmt.Println()

	fmt.Println(ui.AccentStyle.Render("EXAMPLES"))
	fmt.Printf("  %s\n", ui.InfoStyle.Render("dejank url https://example.com"))
	fmt.Printf("  %s\n", ui.InfoStyle.Render("dejank single https://example.com/app.js"))
	fmt.Printf("  %s\n", ui.InfoStyle.Render("dejank local ./example.com"))
	fmt.Println()
}

func runURL(cfg *modes.Config, args []string) {
	if len(args) < 1 {
		fmt.Println(ui.Error("Missing URL argument"))
		fmt.Println(ui.DimStyle.Render("Usage: dejank url <webpage-url>"))
		os.Exit(1)
	}

	targetURL := args[0]
	fmt.Println(ui.Banner(version))
	fmt.Println(ui.Target(targetURL))

	var progress *ui.Progress
	cfg.OnProgress = func(event string, data interface{}) {
		switch event {
		case "discovery_complete":
			if m, ok := data.(map[string]int); ok {
				total := m["scripts"]
				if total > 0 && !cfg.Verbose {
					progress = ui.NewProgress(total, "Processing scripts")
				}
			}
		case "processing_script":
			if progress != nil {
				progress.Increment()
			}
		}
	}

	result, err := modes.RunURL(cfg, targetURL)

	if progress != nil {
		progress.Done()
	}

	if err != nil {
		fmt.Println(ui.Error(err.Error()))
		os.Exit(1)
	}

	printURLSummary(result, cfg.Verbose)
}

func runSingle(cfg *modes.Config, args []string) {
	if len(args) < 1 {
		fmt.Println(ui.Error("Missing script URL argument"))
		fmt.Println(ui.DimStyle.Render("Usage: dejank single <script-url>"))
		os.Exit(1)
	}

	scriptURL := args[0]
	fmt.Println(ui.Banner(version))
	fmt.Println(ui.Target(scriptURL))

	result, err := modes.RunSingle(cfg, scriptURL)
	if err != nil {
		fmt.Println(ui.Error(err.Error()))
		os.Exit(1)
	}

	fmt.Println(ui.SummaryHeader())
	fmt.Println(ui.SummaryLine("Sourcemap found:", result.MapFound))
	fmt.Println(ui.SummaryLine("Sources restored:", result.SourcesRestored))

	if len(result.Errors) > 0 {
		fmt.Println(ui.SummaryLine("Errors:", len(result.Errors)))
		if cfg.Verbose {
			for _, e := range result.Errors {
				fmt.Printf("      %s\n", ui.DimStyle.Render(fmt.Sprintf("- %v", e)))
			}
		}
	}
	fmt.Println()
}

func runLocal(cfg *modes.Config, args []string) {
	var target string
	if len(args) > 0 {
		target = args[0]
	}

	fmt.Println(ui.Banner(version))
	if target != "" {
		fmt.Println(ui.Target(target))
	} else {
		fmt.Println(ui.Info(fmt.Sprintf("Processing all domains in: %s", ui.URLStyle.Render(cfg.OutputRoot))))
	}

	result, err := modes.RunLocal(cfg, target)
	if err != nil {
		fmt.Println(ui.Error(err.Error()))
		os.Exit(1)
	}

	fmt.Println(ui.SummaryHeader())
	fmt.Println(ui.SummaryLine("Targets processed:", result.TargetsProcessed))
	fmt.Println(ui.SummaryLine("Maps processed:", result.MapsProcessed))
	fmt.Println(ui.SummaryLine("Sources restored:", result.SourcesRestored))
	fmt.Println(ui.SummaryLine("Assets extracted:", result.AssetsExtracted))

	if len(result.Errors) > 0 {
		fmt.Println(ui.SummaryLine("Errors:", len(result.Errors)))
		if cfg.Verbose {
			for _, e := range result.Errors {
				fmt.Printf("      %s\n", ui.DimStyle.Render(fmt.Sprintf("- %v", e)))
			}
		}
	}
	fmt.Println()
}

func printURLSummary(result *modes.URLResult, verbose bool) {
	fmt.Println(ui.SummaryHeader())
	fmt.Println(ui.SummaryLine("Scripts discovered:", result.ScriptsFound))
	fmt.Println(ui.SummaryLine("Maps discovered:", result.MapsDiscovered))
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
}
