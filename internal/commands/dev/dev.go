package devcmd

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
	apperr "github.com/ScrawnDotDev/scrawn-cli/internal/apperr"
	"github.com/ScrawnDotDev/scrawn-cli/internal/cmd"
)

func init() {
	cmd.Register("dev", func() cmd.Command { return &DevCommand{} })
}

var heading = lipgloss.NewStyle().Foreground(lipgloss.Color("221"))
var muted = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
var success = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
var step = lipgloss.NewStyle().Foreground(lipgloss.Color("221"))

type DevCommand struct{}

func (c *DevCommand) Name() string     { return "dev" }
func (c *DevCommand) Description() string { return "start local development environment" }

func (c *DevCommand) Run(ctx *cmd.Context, args []string) error {
	flags := parseFlags(args)
	if flags == nil {
		return nil
	}

	if flags.stop {
		return runStop()
	}

	if flags.detach {
		return runDetached(flags)
	}

	return runInteractive()
}

type devFlags struct {
	detach bool
	stop   bool
	help   bool
}

func parseFlags(args []string) *devFlags {
	flags := &devFlags{}

	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	fs.BoolVar(&flags.help, "h", false, "help")
	fs.BoolVar(&flags.help, "help", false, "help")
	fs.BoolVar(&flags.detach, "detach", false, "run detached")
	fs.BoolVar(&flags.stop, "stop", false, "stop containers")

	fs.SetOutput(os.NewFile(1, "/dev/null"))
	if err := fs.Parse(args); err != nil {
		return nil
	}

	if flags.help {
		printHelp()
		return nil
	}

	return flags
}

func printHelp() {
	fmt.Println()
	fmt.Println(heading.Render("Usage:") + " scrawn dev [flags]")
	fmt.Println()
	fmt.Println("Start local development environment (postgres + redis + server)")
	fmt.Println()
	fmt.Println(heading.Render("Flags:"))
	fmt.Println("  --detach    Start containers in background, no logs")
	fmt.Println("  --stop      Stop all containers")
	fmt.Println("  -h, --help  Show this help")
	fmt.Println()
	fmt.Println(heading.Render("Examples:"))
	fmt.Println("  scrawn dev              # Start with live logs")
	fmt.Println("  scrawn dev --detach    # Start in background")
	fmt.Println("  scrawn dev --stop     # Stop all containers")
}

func runStop() error {
	fmt.Println()
	fmt.Println(step.Render("==>"), "Stopping containers...")

	cmd := exec.Command("docker", "compose", "down", "--volumes")
	cmd.Dir = "."
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Println(muted.Render(string(output)))
		return &apperr.CommandError{
			Summary: "failed to stop containers",
			Detail:  err.Error(),
		}
	}

	fmt.Println(success.Render("✓"), "All containers stopped")
	return nil
}

func runDetached(flags *devFlags) error {
	fmt.Println()
	fmt.Println(step.Render("==>"), "Starting db and redis...")

	checkDeps()

	cmd := exec.Command("docker", "compose", "up", "-d", "db", "redis")
	cmd.Dir = "."
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Println(muted.Render(string(output)))
		return &apperr.CommandError{
			Summary: "failed to start containers",
			Detail:  err.Error(),
		}
	}

	fmt.Println(success.Render("✓"), "Containers started in background")
	fmt.Println()
	fmt.Println(muted.Render("Waiting for containers to be healthy..."))

	waitForContainer("db")
	waitForContainer("redis")

	fmt.Println()

	switchEnvIfNeeded()

	fmt.Println()
	fmt.Println(step.Render("==>"), "Starting server...")

	cmd = exec.Command("docker", "compose", "up", "server")
	cmd.Dir = "."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return &apperr.CommandError{
			Summary: "container process ended",
			Detail:  err.Error(),
		}
	}

	return nil
}

func runInteractive() error {
	fmt.Println()
	fmt.Println(step.Render("==>"), "Checking dependencies...")

	checkDeps()

	fmt.Println(step.Render("==>"), "Starting db and redis...")

	cmd := exec.Command("docker", "compose", "up", "-d", "db", "redis")
	cmd.Dir = "."
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Println(muted.Render(string(output)))
		return &apperr.CommandError{
			Summary: "failed to start containers",
			Detail:  err.Error(),
		}
	}

	fmt.Println(success.Render("✓"), "db and redis started")
	fmt.Println()
	fmt.Println(muted.Render("Waiting for containers to be healthy..."))

	waitForContainer("db")
	waitForContainer("redis")

	fmt.Println()

	switchEnvIfNeeded()

	fmt.Println()
	fmt.Println(step.Render("==>"), "Starting server...")

	cmd = exec.Command("docker", "compose", "up", "server")
	cmd.Dir = "."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	in, _ := os.Open("/dev/stdin")
	cmd.Stdin = in

	if err := cmd.Run(); err != nil {
		return &apperr.CommandError{
			Summary: "container process ended",
			Detail:  err.Error(),
		}
	}

	return nil
}

func waitForContainer(name string) {
	for i := 0; i < 30; i++ {
		cmd := exec.Command("docker", "inspect", "--format={{.State.Health.Status}}", fmt.Sprintf("scrawn-%s-1", name))
		out, _ := cmd.CombinedOutput()
		if strings.Contains(string(out), "healthy") {
			fmt.Println(success.Render("✓"), fmt.Sprintf("%s is ready", name))
			return
		}
		fmt.Print(".")
	}
}

func checkDeps() {
	fmt.Println(muted.Render("Checking docker compose..."))

	cmd := exec.Command("docker", "compose", "version")
	if err := cmd.Run(); err != nil {
		fmt.Println(muted.Render("Make sure docker and docker compose are installed"))
		os.Exit(1)
	}
}

func switchEnvIfNeeded() {
	envPath := ".env.local"
	fileInfo, err := os.Stat(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println(heading.Render("No .env.local found - using docker instances by default"))
			writeDockerEnv()
			return
		}
		return
	}

	if fileInfo.IsDir() {
		return
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		return
	}

	hasDbUrl := strings.Contains(string(content), "DATABASE_URL=")
	hasRedisUrl := strings.Contains(string(content), "REDIS_URL=")

	if !hasDbUrl && !hasRedisUrl {
		fmt.Println(heading.Render("No existing database config found - using docker instances"))
		writeDockerEnv()
		return
	}

	fmt.Println(heading.Render("Environment Switch"))
	fmt.Println(muted.Render("We found existing database config in .env.local."))
	fmt.Println(muted.Render("Switch to docker instances?"))
	fmt.Println()
	fmt.Printf("  New DATABASE_URL: %s\n", success.Render("postgresql://postgres:postgres@localhost:5432/scrawn"))
	fmt.Printf("  New REDIS_URL: %s\n", success.Render("redis://localhost:6379"))
	fmt.Println()
	fmt.Print(muted.Render("Switch? [y/N]: "))

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		fmt.Println(muted.Render("Skipped - keeping existing config"))
		return
	}

	rewriteEnvFile(string(content))
	fmt.Println()
	fmt.Println(success.Render("✓"), "Switched to docker instances")
	fmt.Println(muted.Render("Server will use docker postgres + redis"))
}

func writeDockerEnv() {
	envPath := ".env.local"
	newContent := `DATABASE_URL="postgresql://postgres:postgres@localhost:5432/scrawn"
REDIS_URL="redis://localhost:6379"
`

	if err := os.WriteFile(envPath, []byte(newContent), 0644); err != nil {
		return
	}

	fmt.Println(success.Render("✓"), "Written docker config to .env.local")
}

func rewriteEnvFile(original string) {
	var lines []string
	for _, line := range strings.Split(original, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "DATABASE_URL=") && !strings.HasPrefix(line, "#") {
			lines = append(lines, "# "+line)
			lines = append(lines, `DATABASE_URL="postgresql://postgres:postgres@localhost:5432/scrawn"`)
		} else if strings.HasPrefix(line, "REDIS_URL=") && !strings.HasPrefix(line, "#") {
			lines = append(lines, "# "+line)
			lines = append(lines, `REDIS_URL="redis://localhost:6379"`)
		} else if strings.HasPrefix(line, "# ") && (strings.Contains(line, "DATABASE_URL=") || strings.Contains(line, "REDIS_URL=")) {
			lines = append(lines, line)
		} else {
			lines = append(lines, line)
		}
	}

	updated := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(".env.local", []byte(updated), 0644); err != nil {
		return
	}
}