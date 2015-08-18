package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"
)

var (
	cmdHelp = &command{
		Name:    "help",
		Summary: "Show a list of commands or help for one command",
		Usage:   "[COMMAND]",
		Run:     runHelp,
	}

	globalUsageTemplate  *template.Template
	commandUsageTemplate *template.Template
	templFuncs           = template.FuncMap{
		"descToLines": func(s string) []string {
			// trim leading/trailing whitespace and split into slice of lines
			return strings.Split(strings.Trim(s, "\n\t "), "\n")
		},
		"printOption": func(name, defvalue, usage string) string {
			prefix := "--"
			if len(name) == 1 {
				prefix = "-"
			}
			return fmt.Sprintf("\n\t%s%s=%s\t%s", prefix, name, defvalue, usage)
		},
	}

	tabOut *tabwriter.Writer
)

func init() {
	tabOut = tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)

	commands = append(commands, cmdHelp)

	globalUsageTemplate = template.Must(template.New("global_usage").Funcs(templFuncs).Parse(`
NAME:
{{printf "\t%s - %s" .Executable .Description}}

USAGE: 
{{printf "\t%s" .Executable}} [global options] <command> [command options] [arguments...]

COMMANDS:{{range .Commands}}
{{printf "\t%s\t%s" .Name .Summary}}{{end}}

GLOBAL OPTIONS:{{range .Flags}}{{printOption .Name .DefValue .Usage}}{{end}}

Global options can also be configured via upper-case environment variables prefixed with "DEXCTL_"
For example, "some-flag" => "DEXCTL_SOME_FLAG"

Run "{{.Executable}} help <command>" for more details on a specific command.
`[1:]))
	commandUsageTemplate = template.Must(template.New("command_usage").Funcs(templFuncs).Parse(`
NAME:
{{printf "\t%s - %s" .Cmd.Name .Cmd.Summary}}

USAGE:
{{printf "\t%s %s %s" .Executable .Cmd.Name .Cmd.Usage}}

DESCRIPTION:
{{range $line := descToLines .Cmd.Description}}{{printf "\t%s" $line}}
{{end}}
{{if .CmdFlags}}OPTIONS:{{range .CmdFlags}}
{{printOption .Name .DefValue .Usage}}{{end}}

{{end}}For help on global options run "{{.Executable}} help"
`[1:]))
}

func runHelp(args []string) (exit int) {
	if len(args) < 1 {
		printGlobalUsage()
		return
	}

	var cmd *command
	for _, c := range commands {
		if c.Name == args[0] {
			cmd = c
			break
		}
	}

	if cmd == nil {
		stderr("Unrecognized command: %s", args[0])
		return 1
	}

	printCommandUsage(cmd)
	return
}

func printGlobalUsage() {
	globalUsageTemplate.Execute(tabOut, struct {
		Executable  string
		Commands    []*command
		Flags       []*flag.Flag
		Description string
	}{
		cliName,
		commands,
		getFlags(globalFS),
		cliDescription,
	})
	tabOut.Flush()
}

func printCommandUsage(cmd *command) {
	commandUsageTemplate.Execute(tabOut, struct {
		Executable string
		Cmd        *command
		CmdFlags   []*flag.Flag
	}{
		cliName,
		cmd,
		getFlags(&cmd.Flags),
	})
	tabOut.Flush()
}

func getFlags(flagset *flag.FlagSet) (flags []*flag.Flag) {
	flags = make([]*flag.Flag, 0)
	flagset.VisitAll(func(f *flag.Flag) {
		flags = append(flags, f)
	})
	return
}
