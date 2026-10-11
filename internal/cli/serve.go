package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"bakku.dev/bookist/internal/app"
)

func runServe(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)

	addr := flags.String("addr", envDefault(addrEnvVar, defaultAddr), "HTTP address to listen on (env: "+addrEnvVar+")")
	dataDir, dbPath := storageFlags(flags)

	help := commandHelp{
		name:        "bookist serve",
		usage:       "bookist serve [options]",
		description: "Start the Bookist server",
		details: []string{
			"Runs until interrupted (SIGINT or SIGTERM), then finishes in-flight requests before exiting.",
		},
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}
	if !rejectArgs(flags, "serve", stderr, help) {
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server, err := app.New(ctx, app.Config{Addr: *addr, DataDir: *dataDir, DBPath: *dbPath}, newLogger(stderr))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	if err := server.Run(ctx); err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	return 0
}
