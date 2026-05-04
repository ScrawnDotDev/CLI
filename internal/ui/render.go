package ui

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ScrawnDotDev/scrawn-cli/internal/apperr"
	"github.com/ScrawnDotDev/scrawn-cli/internal/setup"
)

func RenderServerSetupIntro() {
	fmt.Println(sectionStyle.Render("scrawn create server"))
	fmt.Println(mutedStyle.Render("Pull latest backend, configure it, migrate the database, verify startup, and generate a dashboard API key."))
	fmt.Println(subtleRule())
}

func RenderDashboardIntent(target string) {
	fmt.Println(sectionStyle.Render("scrawn create dashboard"))
	fmt.Printf("%s %s\n", labelStyle.Render("target"), valueStyle.Render(formatPath(target)))
	fmt.Println(mutedStyle.Render("Dashboard scaffolding is not implemented yet."))
}

func Step(message string) {
	fmt.Printf("%s %s\n", stepStyle.Render("==>"), message)
}

func MarkOK(label string, detail string) {
	fmt.Printf("%s %s", successStyle.Render("OK"), label)
	if strings.TrimSpace(detail) != "" {
		fmt.Printf(" %s", mutedStyle.Render("["+detail+"]"))
	}
	fmt.Println()
}

func RenderSuccess(result setup.Result, kind string) {
	fmt.Println()
	fmt.Println(successStyle.Render("Setup complete"))
	fmt.Printf("%s %s\n", labelStyle.Render("location"), valueStyle.Render(formatPath(result.TargetPath)))
	fmt.Printf("%s %s\n", labelStyle.Render("package manager"), valueStyle.Render(result.UsedPM))
	fmt.Printf("%s %s\n", labelStyle.Render("dashboard key name"), valueStyle.Render(result.APIKeyName))
	fmt.Printf("%s %s\n", labelStyle.Render("health"), valueStyle.Render(setup.DefaultHTTPURL))
	fmt.Println()
	fmt.Println(appStyle.Render("Dashboard API Key"))
	fmt.Println(valueStyle.Render(result.APIKey))
	fmt.Println()
	fmt.Println(mutedStyle.Render("store this key securely, dashboard should use this to communicate to the server"))
}

func RenderDashboardStub(target string) {
	fmt.Println()
	fmt.Println(warnStyle.Render("Dashboard setup is not implemented yet."))
	fmt.Printf("%s %s\n", labelStyle.Render("server location"), valueStyle.Render(formatPath(target)))
}

func RenderError(err error) {
	var cmdErr *apperr.CommandError
	if errors.As(err, &cmdErr) {
		fmt.Fprintln(os.Stderr, failureStyle.Render("error")+" "+cmdErr.Summary)
		if strings.TrimSpace(cmdErr.Detail) != "" {
			fmt.Fprintln(os.Stderr, mutedStyle.Render(cmdErr.Detail))
		}
		return
	}

	fmt.Fprintln(os.Stderr, failureStyle.Render("error")+" "+err.Error())
}

func subtleRule() string {
	return subtleRuleStyle.Render(strings.Repeat("-", 72))
}

func formatPath(target string) string {
	if strings.TrimSpace(target) == "" {
		return "`.`"
	}
	return "`" + target + "`"
}
