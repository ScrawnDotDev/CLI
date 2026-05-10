package tagcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ScrawnDotDev/scrawn-cli/internal/cmd"
	"github.com/ScrawnDotDev/scrawn-cli/internal/ui"
)

func init() {
	cmd.Register("tag", func() cmd.Command { return &TagCommand{} })
}

// ---- types ----

type scrawnConfigData struct {
	Directory string `json:"directory"`
	ServerURL string `json:"serverUrl"`
}

type projectConfigFile struct {
	Scrawn scrawnConfigData `json:"scrawn"`
}

type scrawnConfig struct {
	Directory  string
	ServerURL  string
	ProjectDir string
}

type tagsResponseBody struct {
	Tags []string `json:"tags"`
}

func defaultScrawnConfig() scrawnConfig {
	return scrawnConfig{
		Directory:  "scrawn",
		ServerURL:  "http://localhost:8070",
		ProjectDir: ".",
	}
}

// ---- command ----

type TagCommand struct{}

func (c *TagCommand) Name() string        { return "tag" }
func (c *TagCommand) Description() string { return "manage Scrawn tags" }

func (c *TagCommand) Run(ctx *cmd.Context, args []string) error {
	if len(args) == 0 {
		return &cmd.CommandError{
			Summary: "missing subcommand",
			Detail:  "available: sync",
		}
	}

	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "sync":
		return runSync(args[1:])
	case "-h", "--help":
		showTagHelp()
		return nil
	default:
		return &cmd.CommandError{
			Summary: "invalid tag subcommand",
			Detail:  fmt.Sprintf("unknown subcommand: %s. available: sync", args[0]),
		}
	}
}

func showTagHelp() {
	fmt.Println()
	fmt.Println(ui.Heading.Render("Usage:") + " scrawn tag <subcommand> [flags]")
	fmt.Println()
	fmt.Println(ui.Heading.Render("Subcommands:"))
	fmt.Println("  sync    Pull tags from the server and generate type-safe definitions")
	fmt.Println()
	fmt.Println(ui.Heading.Render("Examples:"))
	fmt.Println("  scrawn tag sync")
}

func showSyncHelp() {
	fmt.Println()
	fmt.Println(ui.Heading.Render("Usage:") + " scrawn tag sync")
	fmt.Println()
	fmt.Println("Pull tags from the Scrawn server and generate scrawn/tags.ts + scrawn/biller.ts")
	fmt.Println()
	fmt.Println(ui.Heading.Render("Configuration:"))
	fmt.Println("  Reads scrawn.config.json from the project root (or uses defaults)")
	fmt.Println("  Reads SCRAWN_KEY from the OS environment")
}

// ---- sync implementation ----

func runSync(args []string) error {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			showSyncHelp()
			return nil
		}
	}

	cfg, err := loadScrawnConfig()
	if err != nil {
		return &cmd.CommandError{
			Summary: "failed to load config",
			Detail:  err.Error(),
		}
	}

	apiKey := os.Getenv("SCRAWN_KEY")
	if apiKey == "" {
		return &cmd.CommandError{
			Summary: "API key not found",
			Detail:  "SCRAWN_KEY is not set in the environment",
		}
	}

	tags, err := fetchTagsFromServer(cfg.ServerURL, apiKey)
	if err != nil {
		return &cmd.CommandError{
			Summary: "failed to fetch tags",
			Detail:  err.Error(),
		}
	}

	outputDir := filepath.Join(cfg.ProjectDir, cfg.Directory)

	// Generate tags.ts
	tagsPath := filepath.Join(outputDir, "tags.ts")
	if err := writeTagsFile(tagsPath, tags); err != nil {
		return &cmd.CommandError{
			Summary: "failed to generate tags file",
			Detail:  err.Error(),
		}
	}

	// Generate biller.ts
	billerPath := filepath.Join(outputDir, "biller.ts")
	if err := writeBillerFile(billerPath, cfg.ServerURL); err != nil {
		return &cmd.CommandError{
			Summary: "failed to generate biller file",
			Detail:  err.Error(),
		}
	}

	fmt.Println()
	ui.MarkOK("tags synced", fmt.Sprintf("%s (%d tags)", tagsPath, len(tags)))
	return nil
}

// ---- config ----

func loadScrawnConfig() (scrawnConfig, error) {
	cfg := defaultScrawnConfig()

	configPath := findConfigFile()
	if configPath == "" {
		return cfg, nil
	}

	cfg.ProjectDir = filepath.Dir(configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return cfg, fmt.Errorf("could not read %s: %w", configPath, err)
	}

	var fileCfg projectConfigFile
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return cfg, fmt.Errorf("invalid config in %s: %w", configPath, err)
	}

	if fileCfg.Scrawn.Directory != "" {
		cfg.Directory = fileCfg.Scrawn.Directory
	}
	if fileCfg.Scrawn.ServerURL != "" {
		cfg.ServerURL = fileCfg.Scrawn.ServerURL
	}

	return cfg, nil
}

func findConfigFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		configPath := filepath.Join(dir, "scrawn.config.json")
		if _, statErr := os.Stat(configPath); statErr == nil {
			return configPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

// ---- HTTP client ----

func fetchTagsFromServer(serverURL string, apiKey string) ([]string, error) {
	url := strings.TrimRight(serverURL, "/") + "/api/v1/tags"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach server at %s: %w", serverURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid API key (server returned 401)")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result tagsResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("could not parse server response: %w", err)
	}

	return result.Tags, nil
}

// ---- file generators ----

func writeTagsFile(filePath string, tags []string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", dir, err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("could not create file %s: %w", filePath, err)
	}
	defer f.Close()

	return writeTagsContent(f, tags)
}

func writeTagsContent(w io.Writer, tags []string) error {
	lines := []string{
		"// Generated by scrawn tag sync. Do not edit manually.",
		"",
	}

	if len(tags) > 0 {
		quoted := make([]string, len(tags))
		for i, t := range tags {
			quoted[i] = fmt.Sprintf(`"%s"`, t)
		}
		lines = append(lines,
			fmt.Sprintf("export const TAGS = [%s] as const;", strings.Join(quoted, ", ")),
		)
	} else {
		lines = append(lines, "export const TAGS = [] as const;")
	}

	lines = append(lines,
		"export type ScrawnTag = (typeof TAGS)[number];",
		"",
	)

	_, err := io.WriteString(w, strings.Join(lines, "\n"))
	return err
}

func writeBillerFile(filePath string, serverURL string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", dir, err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("could not create file %s: %w", filePath, err)
	}
	defer f.Close()

	return writeBillerContent(f, serverURL)
}

func writeBillerContent(w io.Writer, serverURL string) error {
	lines := []string{
		"// Generated by scrawn tag sync. Do not edit manually.",
		"import { createScrawn } from \"@scrawn/core\";",
		"import { TAGS } from \"./tags\";",
		"",
		"export const biller = createScrawn({",
		"  apiKey: (process.env.SCRAWN_KEY || process.env.SCRAWN_API_KEY) as string,",
		fmt.Sprintf("  baseURL: (process.env.SCRAWN_BASE_URL || %q) as string,", serverURL),
		"  tags: TAGS,",
		"});",
		"",
	}

	_, err := io.WriteString(w, strings.Join(lines, "\n"))
	return err
}
