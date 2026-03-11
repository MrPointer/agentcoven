---
name: developing-cli-apps
description: Develop CLI applications in Go. Use when creating or modifying CLI commands, adding flags or arguments, implementing command workflows, building interactive prompts, handling signals and exit codes, or working with stdin/stdout/stderr. Currently uses Cobra for command structure and Huh for interactive UI.
---

# CLI Application Development

Standards for building CLI applications in Go. Currently uses [Cobra](https://github.com/spf13/cobra) for command structure and [Huh](https://github.com/charmbracelet/huh) for interactive UI.

**Interactive UI patterns:** See [Interactive UI Reference](references/interactive-ui.md)

## Command Organization

- One file per command in `cmd/`, file name matches command name (camelCase)
- All commands registered in their own `init()` function via `rootCmd.AddCommand()`
- See `cmd/root.go` for the root command structure and initialization chain
- See any existing command file (e.g., `cmd/version.go`) for a minimal example

## Adding a New Command

1. Create a new file in `cmd/` (camelCase name matching the command)
2. Define a `cobra.Command` variable with `Use` and `Short` fields
3. In `init()`: register with `rootCmd.AddCommand()`, define flags, bind to Viper
4. Suppress the init lint: `//nolint:gochecknoinits // Cobra requires an init function to set up the command structure.`

## Flag Conventions

| Scope | Method |
|-------|--------|
| Global (all commands) | `rootCmd.PersistentFlags()` |
| Local (one command) | `cmd.Flags()` |

- Use `StringVar`/`BoolVar`/`CountVarP` (pointer-binding) for all flags
- Bind every flag to Viper: `viper.BindPFlag("name", cmd.Flags().Lookup("name"))`
- Use kebab-case for flag names: `--git-clone-protocol`, not `--gitCloneProtocol`
- Provide meaningful defaults and descriptions

## Initialization Chain

Global dependencies are initialized via `cobra.OnInitialize()` in `root.go`. Each initializer sets a package-level global. Order matters — later initializers may depend on earlier ones. Read `root.go` for the current chain.

## Error Handling in Commands

- Use `RunE` (not `Run`) — return errors from the function; Cobra handles display and exit
- Set `SilenceUsage: true` on the root command so runtime errors don't print usage
- Keep `RunE` functions thin — delegate to business logic packages

## Signal Handling and Cleanup

- Register signal handlers (e.g., `os.Interrupt`, `syscall.SIGTERM`) early in the root command initialization
- Use `PersistentPostRun` on the root command for successful completion cleanup
- Provide a dedicated cleanup function for error and signal exit paths
- Always clean up resources (loggers, temp files) on all exit paths

## Key Rules

- Commands should not import each other; share state via package-level variables in `cmd/`
- Use `fmt.Fprint(os.Stderr, ...)` for error output
