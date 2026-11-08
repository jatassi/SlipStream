// Command record writes a port's fixture from a real exchange or from a
// Prowlarr recording, scrubbed, and writes nothing the scrub check
// refuses. Run it from the repository root.
//
// A live capture runs one call through a Node against the real site,
// with the credentials of an account on it, and adds the exchanges as a
// case:
//
//	go run ./fakes/site/ports/record live -port hdbits -case search -settings ~/hdbits.yaml -search 'mode=movie&imdbid=tt0000001'
//
// The settings file maps each setting's name to its value and stays out
// of the repository. -recent, -fetch <link> and -test [-code <code>] are
// the other calls.
//
// An import rewrites one of Prowlarr's recorded bodies, as
// `git -C <prowlarr> show HEAD:src/NzbDrone.Core.Test/Files/Indexers/PTP/imdbsearch.json`
// prints it, into the port's bodies/:
//
//	go run ./fakes/site/ports/record import -port passthepopcorn -recording imdbsearch.json -body search.json
//
// -title adds a title the body holds that isn't a dotted release name,
// and -replace old=new a literal to swap, such as a key. The exchange
// that answers with the body is written by hand. A live capture rewrites
// fixture.yaml without its comments. Run go test ./fakes/site/ports
// -update afterwards, and read every golden it writes.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/jatassi/slipstream/internal/ports/all"
	"github.com/jatassi/slipstream/internal/ports/port"
)

func main() {
	err := run(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "record:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("name a command: live or import")
	}
	reg := all.Registry()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	switch args[0] {
	case "live":
		c, err := parseLive(args[1:], reg)
		if err != nil {
			return err
		}
		c.native = port.Driver(reg)
		c.transport = http.DefaultTransport
		return live(ctx, c)
	case "import":
		c, err := parseImport(args[1:], reg)
		if err != nil {
			return err
		}
		return importRecording(c)
	}
	return fmt.Errorf("%q is neither live nor import", args[0])
}
