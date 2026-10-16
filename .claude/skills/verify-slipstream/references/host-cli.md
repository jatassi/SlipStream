# Host CLI

The host CLI is what an operator types on a machine that runs a Node. Recipes run the run's own binary copy against a Node's data dir from the manifest, through `fakeworld exec`:

```sh
$FW exec $RUN --proof <f>/<s> -- {runDir}/bin/slipstream config --data-dir {nodes.home.dataDir}
$FW exec $RUN --proof <f>/<s> -- {runDir}/bin/slipstream version
```

`{runDir}/bin/slipstream --help` lists the commands: `run`, `app`, `stop`, `restart`, `status`, `config`, `config set`, `config unset`, `join`, `service`, `version`, `notices`. `config` prints the resolved configuration and the endpoints census as JSON. `config set --data-dir <dir> <key> <value>` and `config unset --data-dir <dir> <key>` change a Node settings key in `config.yaml` and print `<key>: <value>`, ending `, applies after restart` for a restart key on a running Node; a key an environment variable sets exits 3 with `<key>: Set by SLIPSTREAM_<KEY>.`. `restart --data-dir <dir>` prints the Node's web address once its successor answers. `join --data-dir <dir> <code>` pairs a stopped, Unclaimed Node with the Controller a Join Code names and prints `joined <controller> as <name> (node <id>); link up to <address>`.

Use only data dirs the manifest names: they sit in the run's scratch dir, which `fakeworld stop` deletes.

## Machine events

A machine dying or restarting is a world event, played with `fakeworld node`. Each prints the Node's new manifest entry, rewrites the manifest and logs a `node.*` record.

```sh
$FW node stop $RUN seedbox     # slipstream stop on its data dir: a clean shutdown
$FW node kill $RUN seedbox     # kill the recorded PID: the machine losing power
$FW node wipe $RUN seedbox     # a stopped Node's data dir emptied but config.yaml: Unclaimed again
$FW node start $RUN seedbox    # a stopped, killed or exited Node back on its first address and port
$FW node restart $RUN seedbox  # slipstream restart on its data dir: back once its next ready record is in
```

The world follows a restart however it was asked for, the API included: each readiness record after a process's first logs `node.restarted` with the old and new PID, then `node.ready`, and rewrites the manifest entry.

A restarted Node keeps its HTTP, link and torrent ports, so browser URLs, saved sign-ins and Pairing records stay valid. When another process took the HTTP port meanwhile, `start` falls back to one of the next 9 ports, logs a Warn line naming both, and the manifest's `listeners.http` follows; a taken link or torrent port fails `start` with the bind error. Run doctor after any of these; check 3 reports a Node a cue stopped without failing.
