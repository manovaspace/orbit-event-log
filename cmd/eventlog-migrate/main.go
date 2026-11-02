// Command eventlog-migrate applies the embedded event-log migrations without
// starting a consumer. eventlog.Open applies them automatically; this command
// exists for an operator who needs an explicit up, down or status.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/manovaspace/orbit-event-log/internal/infrastructure/postgres/migrations"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "eventlog-migrate:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	action := "up"
	if len(args) > 0 {
		action = args[0]
	}
	m, err := migrations.Open(dsn)
	if err != nil {
		return err
	}
	defer func() { _ = m.Close() }()

	switch action {
	case "up":
		results, err := m.Up(ctx)
		if err != nil {
			return err
		}
		for _, result := range results {
			fmt.Printf("applied %d %s\n", result.Source.Version, result.Source.Path)
		}
	case "down":
		result, err := m.Down(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("rolled back %d %s\n", result.Source.Version, result.Source.Path)
	case "status":
		statuses, err := m.Status(ctx)
		if err != nil {
			return err
		}
		for _, status := range statuses {
			fmt.Printf("%d %s applied=%t\n", status.Source.Version, status.Source.Path, status.State == "applied")
		}
		current, target, err := m.Version(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("version current=%d latest=%d\n", current, target)
	default:
		return fmt.Errorf("unknown action %q (want up, down or status)", action)
	}
	return nil
}
