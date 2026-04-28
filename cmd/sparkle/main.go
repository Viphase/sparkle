package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/viphase/sparkle/internal/config"
	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/storage/markdown"
	"github.com/viphase/sparkle/internal/tui"
	"github.com/viphase/sparkle/internal/workspace"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	command, args := commandFromArgs(args)
	root, err := workspacePath(args)
	if err != nil {
		return fmt.Errorf("resolve workspace: %w", err)
	}
	if command == "sample-data" {
		result, err := createSampleData(root)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "sample-data: created %d, skipped %d in %s\n", result.Created, result.Skipped, root)
		return nil
	}
	if command != "" {
		return fmt.Errorf("unknown command %q", command)
	}

	ws, err := workspace.Open(root)
	if err != nil {
		return fmt.Errorf("open workspace %s: %w", root, err)
	}
	cfg, err := config.Ensure(ws.Root)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	store := markdown.NewStore(ws.Root)

	p := tea.NewProgram(tui.NewRootWithConfig(ws, store, cfg), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func commandFromArgs(args []string) (string, []string) {
	for i, arg := range args {
		if arg == "sample-data" {
			out := make([]string, 0, len(args)-1)
			out = append(out, args[:i]...)
			out = append(out, args[i+1:]...)
			return arg, out
		}
	}
	return "", args
}

func workspacePath(args []string) (string, error) {
	fs := flag.NewFlagSet("sparkle", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	workspaceFlag := fs.String("workspace", "", "workspace root")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() > 0 {
		return "", fmt.Errorf("unknown argument %q", fs.Arg(0))
	}
	if *workspaceFlag == "" {
		return workspace.DefaultPath()
	}
	root, err := filepath.Abs(*workspaceFlag)
	if err != nil {
		return "", err
	}
	return root, nil
}

type sampleDataResult struct {
	Created int
	Skipped int
}

func createSampleData(root string) (sampleDataResult, error) {
	ws, err := workspace.Open(root)
	if err != nil {
		return sampleDataResult{}, fmt.Errorf("open workspace %s: %w", root, err)
	}
	if _, err := config.Ensure(ws.Root); err != nil {
		return sampleDataResult{}, fmt.Errorf("load config: %w", err)
	}
	store := markdown.NewStore(ws.Root)

	var result sampleDataResult
	for _, sp := range sampleSparks() {
		if _, err := os.Stat(store.SparkPath(sp.ID)); err == nil {
			result.Skipped++
			continue
		} else if !os.IsNotExist(err) {
			return sampleDataResult{}, fmt.Errorf("stat sample spark %s: %w", sp.ID, err)
		}
		if err := store.SaveSpark(sp); err != nil {
			return sampleDataResult{}, fmt.Errorf("save sample spark %s: %w", sp.ID, err)
		}
		result.Created++
	}
	for _, p := range sampleProjects() {
		if _, err := os.Stat(store.ProjectPath(p.ID)); err == nil {
			result.Skipped++
			continue
		} else if !os.IsNotExist(err) {
			return sampleDataResult{}, fmt.Errorf("stat sample project %s: %w", p.ID, err)
		}
		if err := store.SaveProject(p); err != nil {
			return sampleDataResult{}, fmt.Errorf("save sample project %s: %w", p.ID, err)
		}
		result.Created++
	}
	return result, nil
}

func sampleProjects() []domain.Project {
	base := time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC)
	return []domain.Project{
		{
			ID:             "project_sparkle_20260420_sample",
			Title:          "Sparkle",
			Status:         domain.ProjectStatusActive,
			GitHubURL:      "https://github.com/viphase/sparkle",
			TargetAudience: "developers and writers who manage personal projects",
			Tags:           []string{"tui", "go", "productivity"},
			CreatedAt:      base,
			UpdatedAt:      base.Add(8 * 24 * time.Hour),
		},
		{
			ID:             "project_novel_tracker_20260420_sample",
			Title:          "Novel Tracker",
			Status:         domain.ProjectStatusDraft,
			GitHubURL:      "",
			TargetAudience: "fiction writers tracking daily word counts",
			Tags:           []string{"writing", "tracking"},
			CreatedAt:      base.Add(2 * 24 * time.Hour),
			UpdatedAt:      base.Add(3 * 24 * time.Hour),
		},
	}
}

func sampleSparks() []domain.Spark {
	base := time.Date(2026, 4, 27, 9, 0, 0, 0, time.UTC)
	return []domain.Spark{
		{
			ID:          "spark_20260427_090000_sample_tui",
			Title:       "Make the TUI feel solid",
			Description: "Ensure the app paints a complete background, keeps tab focus readable, and avoids terminal transparency artifacts.",
			Status:      domain.SparkStatusQuestioning,
			Tags:        []string{"tui", "polish"},
			CreatedAt:   base,
			UpdatedAt:   base.Add(2 * time.Hour),
		},
		{
			ID:          "spark_20260427_100000_sample_tracking",
			Title:       "Track weekly project momentum",
			Description: "Turn spark and project updates into a small dashboard graph once tracker events land.",
			Status:      domain.SparkStatusNew,
			Tags:        []string{"tracking", "dashboard"},
			CreatedAt:   base.Add(time.Hour),
			UpdatedAt:   base.Add(25 * time.Hour),
		},
		{
			ID:          "spark_20260427_110000_sample_archive",
			Title:       "Archived reference idea",
			Description: "A sample archived spark so the Sparks tab can demonstrate show/hide behavior without user data.",
			Status:      domain.SparkStatusArchived,
			Tags:        []string{"example"},
			CreatedAt:   base.Add(2 * time.Hour),
			UpdatedAt:   base.Add(3 * time.Hour),
		},
	}
}
