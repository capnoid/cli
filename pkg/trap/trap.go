// Package trap returns a root context for graceful exits.
//
// The package is typically used in main, it's context method
// can be used to return a context that is canceled when an
// os signal is received.
//
// 		func main() {
// 			ctx := trap.Context()
//
// 			db, err := open(ctx, "...")
// 			if err != nil {
// 				log.Fatalf("open: %s", err)
// 			}
//
// 			go work(db)
// 			<-ctx.Done()
// 			db.Close()
//
// 			log.Println("done :)")
// 		}
//
package trap

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	// Timeout is a graceful exit timeout.
	//
	// When reached, `Context()` will call os.Exit(1)
	// and exit the program abruptly.
	//
	// If <= 0, Context() is is ignored.
	Timeout = 1 * time.Minute

	// ForceExit causes trap to exit abruptly
	// on a second signal.
	//
	// When false, a second signal will do nothing.
	ForceExit = true

	// Signals are the signals the package will
	// listen on.
	Signals = []os.Signal{
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}

	// Printf is the logging function to use.
	//
	// Uses the global logger by default.
	Printf = log.Printf

	// Exit is the exit func to use.
	//
	// Used in tests.
	exit = os.Exit
)

// Context returns a new context.
//
// The method listens for os signals that are listed
// in the Signals variable and cancels the returned
// context when a signal is received.
//
// Once the context is canceled, the method will listen on
// for more signals, if one is received and ForceExit is true
// it will force exit with exit code 1.
//
// When Timeout > 0, the method will forcefully exit when
// the timeout is reached.
func Context() context.Context {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, Signals...)
	return newContext(signals)
}

// NewContext returns a new context.
func newContext(sigc <-chan os.Signal) context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sig := <-sigc
		printf("trap: received %q", sig)
		cancel()

		select {
		case sig := <-sigc:
			if ForceExit {
				printf("trap: received %q, forcing exit", sig)
				exit(1)
			}
		case <-time.After(Timeout):
			if Timeout > 0 {
				printf("trap: timeout of %s reached, forcing exit", Timeout)
				exit(1)
			}
		}
	}()

	return ctx
}

// Printf is the logging function.
func printf(format string, args ...interface{}) {
	if Printf != nil {
		Printf(format, args...)
	}
}
