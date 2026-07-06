package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"leftoff/internal/leftoff"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer, _ io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}

	switch args[0] {
	case "init":
		return runInit(args[1:], stdout)
	case "capture":
		return runCapture(args[1:], stdout)
	case "now":
		return runNow(args[1:], stdout)
	case "scan":
		return runScan(args[1:], stdout)
	case "resume":
		return runResume(args[1:], stdout)
	case "remember-why":
		return runRememberWhy(args[1:], stdout)
	case "save-solution":
		return runSaveSolution(args[1:], stdout)
	case "review-week":
		return runReviewWeek(args[1:], stdout)
	case "friction":
		return runFriction(args[1:], stdout)
	case "clean-up":
		return runCleanUp(args[1:], stdout)
	case "github":
		return runGitHub(args[1:], stdout)
	case "workspace":
		return runWorkspace(args[1:], stdout)
	case "compat":
		return runCompat(args[1:], stdout)
	case "export":
		return runExport(args[1:], stdout)
	case "import":
		return runImport(args[1:], stdout)
	case "delete-data":
		return runDeleteData(args[1:], stdout)
	case "version":
		fmt.Fprintf(stdout, "leftoff %s\n", leftoff.LeftoffVersion)
		return nil
	case "validate":
		return runValidate(args[1:], stdout)
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "leftoff: local records for unfinished developer work")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  init                         initialize the local store")
	fmt.Fprintln(w, "  capture [flags] <text>        capture a task, decision, problem, solution, or follow-up")
	fmt.Fprintln(w, "  now [flags]                   recommend the best next local task")
	fmt.Fprintln(w, "  scan [flags]                  save compact read-only Git state")
	fmt.Fprintln(w, "  resume [flags]                reconstruct project context")
	fmt.Fprintln(w, "  remember-why [flags] <query>  recall decisions and solved problems")
	fmt.Fprintln(w, "  save-solution [flags]         preview or explicitly save a solved-problem record")
	fmt.Fprintln(w, "  review-week [flags]           produce a weekly local review")
	fmt.Fprintln(w, "  friction [flags]              find recurring local friction patterns")
	fmt.Fprintln(w, "  clean-up [flags]              report cleanup opportunities")
	fmt.Fprintln(w, "  github [flags]                opt-in GitHub CLI metadata cache")
	fmt.Fprintln(w, "  workspace <add|scan|list>     track unfinished work across repositories")
	fmt.Fprintln(w, "  compat [flags]                print agent skill compatibility information")
	fmt.Fprintln(w, "  export [flags]                export the local store to a zip archive")
	fmt.Fprintln(w, "  import [flags]                import a zip archive into the local store")
	fmt.Fprintln(w, "  delete-data [flags]           delete the marked local store with confirmation")
	fmt.Fprintln(w, "  version                       print the leftoff version")
	fmt.Fprintln(w, "  validate [flags]              validate and optionally repair the store")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Common flags:")
	fmt.Fprintln(w, "  --store <path>                override ~/.leftoff")
}

func runInit(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	storePath := fs.String("store", "", "store path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := leftoff.NewStore(*storePath)
	if err != nil {
		return err
	}
	if err := store.Init(); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Initialized leftoff store: %s\n", store.Root)
	return nil
}

func runNow(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("now", flag.ContinueOnError)
	storePath := fs.String("store", "", "store path")
	project := fs.String("project", "", "project name or slug")
	focus := fs.String("focus", "", "temporary focus terms")
	minutes := fs.Int("minutes", 0, "available minutes")
	limit := fs.Int("limit", 2, "number of alternatives")
	all := fs.Bool("all", false, "rank across all projects and registered workspace repositories")
	jsonOutput := fs.Bool("json", false, "write structured JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := leftoff.NewStore(*storePath)
	if err != nil {
		return err
	}
	result, err := store.Now(leftoff.NowRequest{
		Project: *project,
		Focus:   *focus,
		Minutes: *minutes,
		Limit:   *limit,
		All:     *all,
	})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSON(stdout, result)
	}
	fmt.Fprint(stdout, result.Output)
	return nil
}

func runCapture(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("capture", flag.ContinueOnError)
	storePath := fs.String("store", "", "store path")
	kind := fs.String("kind", "", "record type")
	project := fs.String("project", "", "project name or slug")
	repo := fs.String("repo", "", "repository path used to infer project identity")
	if err := fs.Parse(args); err != nil {
		return err
	}

	text := strings.TrimSpace(strings.Join(fs.Args(), " "))
	store, err := leftoff.NewStore(*storePath)
	if err != nil {
		return err
	}
	result, err := store.Capture(context.Background(), leftoff.CaptureRequest{
		Kind:     *kind,
		Text:     text,
		Project:  *project,
		RepoPath: *repo,
		Source:   "cli",
	})
	if err != nil {
		if errors.Is(err, leftoff.ErrSecretCapture) {
			return fmt.Errorf("%w; nothing was saved", err)
		}
		return err
	}

	fmt.Fprintf(stdout, "Saved %s (%s)\n", result.ID, result.Type)
	fmt.Fprintf(stdout, "Destination: %s\n", result.Path)
	if result.ProjectSlug != "" {
		fmt.Fprintf(stdout, "Project: %s\n", result.ProjectSlug)
	} else {
		fmt.Fprintf(stdout, "Project: inbox\n")
	}
	fmt.Fprintf(stdout, "Evidence: User capture.\n")
	return nil
}

func runScan(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	storePath := fs.String("store", "", "store path")
	repo := fs.String("repo", ".", "repository path")
	jsonOutput := fs.Bool("json", false, "write structured JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := leftoff.NewStore(*storePath)
	if err != nil {
		return err
	}
	snapshot := leftoff.InspectRepository(context.Background(), *repo, store.Clock)
	if !snapshot.IsRepo {
		if *jsonOutput {
			_ = writeJSON(stdout, leftoff.ScanResult{Snapshot: snapshot})
			return errors.New("no Git state was saved")
		}
		for _, note := range snapshot.HealthNotes {
			fmt.Fprintf(stdout, "- %s\n", note)
		}
		return errors.New("no Git state was saved")
	}

	path, err := store.SaveGitState(snapshot)
	if err != nil {
		return err
	}

	result := leftoff.ScanResult{
		Path:     path,
		Snapshot: snapshot,
	}
	if *jsonOutput {
		return writeJSON(stdout, result)
	}
	fmt.Fprintf(stdout, "Saved compact Git state: %s\n", path)
	fmt.Fprintf(stdout, "Repository: %s\n", snapshot.RepoName)
	fmt.Fprintf(stdout, "Branch: %s\n", snapshot.Branch)
	fmt.Fprintf(stdout, "Head: %s\n", snapshot.Head)
	fmt.Fprintf(stdout, "Changed paths: %d\n", len(snapshot.ChangedFiles))
	fmt.Fprintf(stdout, "Ahead/behind: %d/%d\n", snapshot.Ahead, snapshot.Behind)
	fmt.Fprintf(stdout, "Unpushed commits: %d\n", snapshot.UnpushedCommits)
	fmt.Fprintf(stdout, "Evidence: User-invoked read-only Git scan.\n")
	return nil
}

func runResume(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("resume", flag.ContinueOnError)
	storePath := fs.String("store", "", "store path")
	repo := fs.String("repo", "", "repository path")
	project := fs.String("project", "", "project name or slug")
	saveState := fs.Bool("save-state", false, "save compact Git state while resuming")
	jsonOutput := fs.Bool("json", false, "write structured JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *project == "" && fs.NArg() > 0 {
		*project = strings.Join(fs.Args(), " ")
	}

	store, err := leftoff.NewStore(*storePath)
	if err != nil {
		return err
	}
	result, err := store.Resume(context.Background(), leftoff.ResumeRequest{
		Project:   *project,
		RepoPath:  *repo,
		SaveState: *saveState,
	})
	if err != nil {
		return err
	}

	if *jsonOutput {
		return writeJSON(stdout, result)
	}
	fmt.Fprint(stdout, result.Output)
	return nil
}

func runRememberWhy(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("remember-why", flag.ContinueOnError)
	storePath := fs.String("store", "", "store path")
	project := fs.String("project", "", "project name or slug")
	limit := fs.Int("limit", 5, "maximum matches per section")
	if err := fs.Parse(args); err != nil {
		return err
	}

	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	store, err := leftoff.NewStore(*storePath)
	if err != nil {
		return err
	}
	result, err := store.RememberWhy(leftoff.RememberRequest{
		Query:   query,
		Project: *project,
		Limit:   *limit,
	})
	if err != nil {
		return err
	}
	fmt.Fprint(stdout, result.Output)
	return nil
}

func runSaveSolution(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("save-solution", flag.ContinueOnError)
	storePath := fs.String("store", "", "store path")
	project := fs.String("project", "", "project name or slug")
	problem := fs.String("problem", "", "problem summary")
	solution := fs.String("solution", "", "solution summary")
	confirm := fs.Bool("confirm", false, "explicitly save the solution")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *solution == "" && fs.NArg() > 0 {
		*solution = strings.Join(fs.Args(), " ")
	}

	store, err := leftoff.NewStore(*storePath)
	if err != nil {
		return err
	}
	result, err := store.SaveSolutionCandidate(context.Background(), leftoff.SaveSolutionRequest{
		Project:  *project,
		Problem:  *problem,
		Solution: *solution,
		Confirm:  *confirm,
	})
	if err != nil {
		return err
	}
	fmt.Fprint(stdout, result.Output)
	return nil
}

func runReviewWeek(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("review-week", flag.ContinueOnError)
	storePath := fs.String("store", "", "store path")
	project := fs.String("project", "", "project name or slug")
	week := fs.String("week", "", "ISO week such as 2026-W28")
	write := fs.Bool("write", false, "write report to weekly/YYYY-Www.md")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := leftoff.NewStore(*storePath)
	if err != nil {
		return err
	}
	result, err := store.ReviewWeek(leftoff.ReviewWeekRequest{
		Project: *project,
		Week:    *week,
		Write:   *write,
	})
	if err != nil {
		return err
	}
	fmt.Fprint(stdout, result.Output)
	if result.Path != "" {
		fmt.Fprintf(stdout, "\nReport written: %s\n", result.Path)
	}
	return nil
}

func runFriction(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("friction", flag.ContinueOnError)
	storePath := fs.String("store", "", "store path")
	project := fs.String("project", "", "project name or slug")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := leftoff.NewStore(*storePath)
	if err != nil {
		return err
	}
	result, err := store.Friction(leftoff.FrictionRequest{Project: *project})
	if err != nil {
		return err
	}
	fmt.Fprint(stdout, result.Output)
	return nil
}

func runCleanUp(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("clean-up", flag.ContinueOnError)
	storePath := fs.String("store", "", "store path")
	project := fs.String("project", "", "project name or slug")
	repo := fs.String("repo", "", "repository path for read-only Git cleanup context")
	action := fs.String("action", "", "record-maintenance action, such as dedupe-activity")
	apply := fs.Bool("apply", false, "apply an explicitly supported low-risk cleanup action")
	confirm := fs.Bool("confirm", false, "confirm an apply action")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := leftoff.NewStore(*storePath)
	if err != nil {
		return err
	}
	result, err := store.CleanUp(context.Background(), leftoff.CleanUpRequest{
		Project:  *project,
		RepoPath: *repo,
		Action:   *action,
		Apply:    *apply,
		Confirm:  *confirm,
	})
	if err != nil {
		return err
	}
	fmt.Fprint(stdout, result.Output)
	return nil
}

func runGitHub(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("github", flag.ContinueOnError)
	storePath := fs.String("store", "", "store path")
	project := fs.String("project", "", "project name or slug")
	repo := fs.String("repo", "", "repository path")
	refresh := fs.Bool("refresh", false, "opt in to read-only gh queries")
	forget := fs.Bool("forget-cache", false, "forget cached GitHub metadata")
	retention := fs.Int("retention-days", 14, "cache retention in days")
	jsonOutput := fs.Bool("json", false, "write structured JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := leftoff.NewStore(*storePath)
	if err != nil {
		return err
	}
	result, err := store.GitHub(context.Background(), leftoff.GitHubRequest{
		Project:       *project,
		RepoPath:      *repo,
		Refresh:       *refresh,
		ForgetCache:   *forget,
		RetentionDays: *retention,
	})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSON(stdout, result)
	}
	fmt.Fprint(stdout, result.Output)
	return nil
}

func runWorkspace(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("workspace requires a subcommand: add, scan, or list")
	}

	switch args[0] {
	case "add":
		fs := flag.NewFlagSet("workspace add", flag.ContinueOnError)
		storePath := fs.String("store", "", "store path")
		jsonOutput := fs.Bool("json", false, "write structured JSON output")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return errors.New("workspace add requires exactly one repository path")
		}
		store, err := leftoff.NewStore(*storePath)
		if err != nil {
			return err
		}
		result, err := store.AddWorkspaceRepo(context.Background(), leftoff.WorkspaceAddRequest{RepoPath: fs.Arg(0)})
		if err != nil {
			return err
		}
		if *jsonOutput {
			return writeJSON(stdout, result)
		}
		fmt.Fprint(stdout, result.Output)
		return nil
	case "scan":
		fs := flag.NewFlagSet("workspace scan", flag.ContinueOnError)
		storePath := fs.String("store", "", "store path")
		jsonOutput := fs.Bool("json", false, "write structured JSON output")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		store, err := leftoff.NewStore(*storePath)
		if err != nil {
			return err
		}
		result, err := store.ScanWorkspace(context.Background())
		if err != nil {
			return err
		}
		if *jsonOutput {
			return writeJSON(stdout, result)
		}
		fmt.Fprint(stdout, result.Output)
		return nil
	case "list":
		fs := flag.NewFlagSet("workspace list", flag.ContinueOnError)
		storePath := fs.String("store", "", "store path")
		jsonOutput := fs.Bool("json", false, "write structured JSON output")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		store, err := leftoff.NewStore(*storePath)
		if err != nil {
			return err
		}
		result, err := store.ListWorkspace()
		if err != nil {
			return err
		}
		if *jsonOutput {
			return writeJSON(stdout, result)
		}
		fmt.Fprint(stdout, result.Output)
		return nil
	default:
		return fmt.Errorf("unknown workspace subcommand %q", args[0])
	}
}

func runCompat(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("compat", flag.ContinueOnError)
	storePath := fs.String("store", "", "store path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result := leftoff.CompatibilityReport(*storePath)
	fmt.Fprint(stdout, result.Output)
	return nil
}

func runExport(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	storePath := fs.String("store", "", "store path")
	out := fs.String("out", "", "zip archive output path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := leftoff.NewStore(*storePath)
	if err != nil {
		return err
	}
	result, err := store.Export(leftoff.ExportRequest{Out: *out})
	if err != nil {
		return err
	}
	fmt.Fprint(stdout, result.Output)
	return nil
}

func runImport(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	storePath := fs.String("store", "", "store path")
	from := fs.String("from", "", "zip archive input path")
	confirm := fs.Bool("confirm", false, "confirm import")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := leftoff.NewStore(*storePath)
	if err != nil {
		return err
	}
	result, err := store.Import(leftoff.ImportRequest{From: *from, Confirm: *confirm})
	if err != nil {
		return err
	}
	fmt.Fprint(stdout, result.Output)
	return nil
}

func runDeleteData(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("delete-data", flag.ContinueOnError)
	storePath := fs.String("store", "", "store path")
	confirm := fs.Bool("confirm", false, "confirm deletion")
	dryRun := fs.Bool("dry-run", false, "preview deletion")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := leftoff.NewStore(*storePath)
	if err != nil {
		return err
	}
	result, err := store.DeleteData(leftoff.DeleteDataRequest{Confirm: *confirm, DryRun: *dryRun})
	if err != nil {
		return err
	}
	fmt.Fprint(stdout, result.Output)
	return nil
}

func runValidate(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	storePath := fs.String("store", "", "store path")
	repair := fs.Bool("repair", false, "repair missing or malformed store files")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := leftoff.NewStore(*storePath)
	if err != nil {
		return err
	}
	issues, err := store.Validate(leftoff.ValidateOptions{Repair: *repair})
	if err != nil {
		return err
	}
	if len(issues) == 0 {
		fmt.Fprintln(stdout, "Store is valid.")
		return nil
	}
	for _, issue := range issues {
		status := "reported"
		if issue.Repaired {
			status = "repaired"
		}
		fmt.Fprintf(stdout, "- %s: %s (%s)\n", issue.Path, issue.Problem, status)
		if issue.BackupPath != "" {
			fmt.Fprintf(stdout, "  backup: %s\n", issue.BackupPath)
		}
	}
	return nil
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
