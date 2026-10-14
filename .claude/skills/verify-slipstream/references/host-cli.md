# Host CLI

The host CLI is what an operator types on a machine that runs a Node. Recipes run the run's own binary copy against a Node's data dir from the manifest, through `fakeworld exec`:

```sh
$FW exec $RUN --proof <f>/<s> -- {runDir}/bin/slipstream config --data-dir {nodes.home.dataDir}
$FW exec $RUN --proof <f>/<s> -- {runDir}/bin/slipstream version
```

`{runDir}/bin/slipstream --help` lists the commands: `run`, `stop`, `config`, `join`, `version`, `notices`. `config` prints the resolved configuration and the endpoints census as JSON. `join --data-dir <dir> <code>` pairs a stopped, Unclaimed Node with the Controller a Join Code names and prints `joined <controller> as <name> (node <id>); link up to <address>`.

Use only data dirs the manifest names: they sit in the run's scratch dir, which `fakeworld stop` deletes.

## Machine events

A machine dying or restarting is a world event, played with `fakeworld node`. Each prints the Node's new manifest entry, rewrites the manifest and logs a `node.*` record.

```sh
$FW node stop $RUN seedbox     # slipstream stop on its data dir: a clean shutdown
$FW node kill $RUN seedbox     # kill the recorded PID: the machine losing power
$FW node wipe $RUN seedbox     # a stopped Node's data dir emptied but config.yaml: Unclaimed again
$FW node start $RUN seedbox    # a stopped, killed or exited Node back on its first address and port
```

A restarted Node keeps its HTTP, link and torrent ports, so browser URLs, saved sign-ins and Pairing records stay valid. `start` fails with the bind error when another process took the port meanwhile. Run doctor after any of these; check 3 reports a Node a cue stopped without failing.
