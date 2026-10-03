// Command infoboard-host is the InfoBoard Native Messaging host.
//
// It is started by Chrome or Edge, speaks only the browser-provided framed
// stdin/stdout channel, writes bounded redacted diagnostics to stderr, and
// never opens a listening socket, dials a remote endpoint, spawns a child
// process, or writes a non-frame byte to stdout. It contains no business logic:
// process bootstrap, lifecycle, and diagnostics live in
// host/internal/runtime.
package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ryantr-statinops/InfoBoard/host/internal/runtime"
)

var (
	// hostVersion is stamped by the packaging step through -ldflags.
	hostVersion = "0.0.0-dev"
	// registeredOrigins is stamped by the installer through -ldflags. An empty
	// value fails closed as a registration problem instead of trusting every
	// extension that can launch the host.
	registeredOrigins = ""
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, input *os.File, output *os.File, errOutput *os.File) int {
	limits := runtime.DefaultLimits()
	config := runtime.DefaultConfig(hostVersion, runtime.NewBoundedSink(errOutput, limits.MaxDiagnosticBytes), limits)
	config.AllowedOrigins = configuredOrigins(registeredOrigins)

	invocation, err := runtime.ParseInvocation(args)
	if err != nil {
		return runtime.ExitStartupFailure
	}
	session, err := runtime.Bootstrap(config, invocation, input, output, runtime.Dependencies{})
	if err != nil {
		return runtime.ExitStartupFailure
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return session.Run(ctx).ExitCode
}

func configuredOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
