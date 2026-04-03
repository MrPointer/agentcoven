// Package status implements the orchestration logic for displaying cova subscription status.
package status

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/MrPointer/agentcoven/cova/config"
	"github.com/MrPointer/agentcoven/cova/state"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

// Deps holds the injected service dependencies for the status operation.
type Deps struct {
	Logger      logger.Logger
	FileSystem  utils.FileSystem
	BlockStore  state.BlockStore
	EnvManager  osmanager.EnvironmentManager
	UserManager osmanager.UserManager
	Out         io.Writer
}

const (
	// sourceMinComponents is the minimum number of "/" components a valid Source must have.
	sourceMinComponents = 2
	// sourceSplitN is the maximum number of substrings SplitN produces when parsing a Source.
	sourceSplitN = 3
)

// blockKey uniquely identifies a block within a subscription.
type blockKey struct {
	subscription string
	blockType    string
	blockName    string
}

// Run orchestrates the status command: reads config and state, then prints a summary
// of all subscriptions, applied blocks, and configured agents.
func Run(ctx context.Context, deps Deps, verbose bool) error {
	configPath, err := config.DefaultPath(deps.EnvManager, deps.UserManager)
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}

	cfg, err := config.Load(deps.FileSystem, configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Subscriptions) == 0 {
		fmt.Fprintln(deps.Out, "No subscriptions")
		return nil
	}

	// Collect blocks per subscription from state.
	blocksBySubscription := make(map[string]map[string][]string, len(cfg.Subscriptions))

	for _, sub := range cfg.Subscriptions {
		blocksBySubscription[sub.Name] = make(map[string][]string)
	}

	var totalBlocks int

	typeCounts := make(map[string]int)

	if deps.BlockStore != nil {
		for _, sub := range cfg.Subscriptions {
			records, queryErr := deps.BlockStore.QueryBySubscription(ctx, sub.Name)
			if queryErr != nil {
				deps.Logger.Warning("failed to query blocks for subscription %q: %s", sub.Name, queryErr)
				continue
			}

			seen := make(map[blockKey]struct{})

			for _, r := range records {
				parts := strings.SplitN(r.Source, "/", sourceSplitN)
				if len(parts) < sourceMinComponents {
					deps.Logger.Warning("malformed source value %q for subscription %q — skipping", r.Source, sub.Name)
					continue
				}

				blockType := parts[0]
				blockName := parts[1]
				key := blockKey{subscription: sub.Name, blockType: blockType, blockName: blockName}

				if _, alreadySeen := seen[key]; alreadySeen {
					continue
				}

				seen[key] = struct{}{}

				blocksBySubscription[sub.Name][blockType] = append(blocksBySubscription[sub.Name][blockType], blockName)
				totalBlocks++
				typeCounts[blockType]++
			}
		}
	}

	printSubscriptions(deps.Out, cfg.Subscriptions)

	if verbose {
		fmt.Fprintln(deps.Out)
		printVerboseBlocks(deps.Out, cfg.Subscriptions, blocksBySubscription)
	}

	printSummaryLine(deps.Out, totalBlocks, typeCounts)
	printAgentsLine(deps.Out, cfg.Agents)

	return nil
}

// printSubscriptions writes the subscriptions section to w.
func printSubscriptions(w io.Writer, subs []config.Subscription) {
	fmt.Fprintln(w, "Subscriptions:")

	for _, sub := range subs {
		line := fmt.Sprintf("  %-16s  %s", sub.Name, sub.Repo)

		if sub.Path != "" {
			line += "  " + sub.Path
		}

		if sub.Ref != "" {
			line += "  @ " + sub.Ref
		}

		fmt.Fprintln(w, line)
	}
}

// printVerboseBlocks writes the per-subscription block breakdown to w.
func printVerboseBlocks(w io.Writer, subs []config.Subscription, blocksBySubscription map[string]map[string][]string) {
	for _, sub := range subs {
		blocks := blocksBySubscription[sub.Name]

		blockCount := countBlocks(blocks)

		fmt.Fprintf(w, "%s (%d blocks):\n", sub.Name, blockCount)

		if blockCount == 0 {
			fmt.Fprintln(w, "  No blocks applied")
			fmt.Fprintln(w)

			continue
		}

		blockTypes := make([]string, 0, len(blocks))
		for bt := range blocks {
			blockTypes = append(blockTypes, bt)
		}

		slices.Sort(blockTypes)

		for _, bt := range blockTypes {
			names := blocks[bt]
			slices.Sort(names)

			fmt.Fprintf(w, "  %s:\n", bt)

			for _, name := range names {
				fmt.Fprintf(w, "    %s\n", name)
			}
		}

		fmt.Fprintln(w)
	}
}

// countBlocks returns the total number of unique blocks across all block types.
func countBlocks(blocks map[string][]string) int {
	total := 0

	for _, names := range blocks {
		total += len(names)
	}

	return total
}

// printSummaryLine writes the "Applied: N blocks (...)" line to w.
func printSummaryLine(w io.Writer, totalBlocks int, typeCounts map[string]int) {
	if totalBlocks == 0 {
		fmt.Fprint(w, "\nApplied: 0 blocks\n")
		return
	}

	types := make([]string, 0, len(typeCounts))
	for t := range typeCounts {
		types = append(types, t)
	}

	slices.Sort(types)

	parts := make([]string, 0, len(types))

	for _, t := range types {
		n := typeCounts[t]
		parts = append(parts, fmt.Sprintf("%d %s", n, t))
	}

	fmt.Fprintf(w, "\nApplied: %d blocks (%s)\n", totalBlocks, strings.Join(parts, ", "))
}

// printAgentsLine writes the "Agents: ..." line to w.
func printAgentsLine(w io.Writer, agents []string) {
	if len(agents) == 0 {
		return
	}

	fmt.Fprintf(w, "Agents: %s\n", strings.Join(agents, ", "))
}
