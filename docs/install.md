# Installing SlipStream

Every release on the [releases page](https://github.com/jatassi/SlipStream/releases) carries a package for each system below. Each package holds the same SlipStream program, with ffmpeg, ffprobe and dovi_tool beside it, so conversions work with no internet connection.

The first time SlipStream starts, it is Unclaimed. The setup page asks for the Setup code that the Node prints at start. A desktop launch opens the setup page with the code already filled in.

## Supported systems

| System | Oldest supported | Packages |
|---|---|---|
| Windows | Windows 10, 64-bit x86 | `SlipStream-<v>-windows-amd64.msi`, `SlipStream-<v>-windows-amd64.zip` |
| macOS | macOS 13.0 on Apple silicon | `SlipStream-<v>-darwin-arm64.dmg` |
| Linux | glibc 2.28 and kernel 4.18, x86-64 or arm64 | `slipstream_<v>_<arch>.deb`, `slipstream-<v>-1.<arch>.rpm`, `slipstream-<v>-linux-<arch>.tar.gz`, `SlipStream-<v>-x86_64.AppImage` |
| Docker | Linux amd64 or arm64 | `ghcr.io/jatassi/slipstream` |

The MSI, the deb and the rpm refuse an older system and name the minimum. The tarball and the AppImage have no installer, so SlipStream logs a warning at start instead.

Each Node keeps its database, settings and logs in one data dir. Removing or uninstalling a package never deletes it.

## Windows

### Installer (MSI)

1. Download `SlipStream-<v>-windows-amd64.msi` and open it. It installs for your account alone and never asks for an administrator.
2. If Windows shows **Windows protected your PC**, choose **More info**, then **Run anyway**. Windows shows this prompt because the release is not yet code-signed.
3. Keep **Launch SlipStream** ticked on the last page. SlipStream appears in the tray, and your browser opens at the setup page.

The installer puts SlipStream in `%LOCALAPPDATA%\Programs\SlipStream` and adds that folder to your PATH. It adds a Start menu entry, and a Desktop shortcut only if you tick it. SlipStream starts when you sign in. To stop that, untick **Start at login** in the tray menu; upgrades keep that choice.

Your data lives in `%LOCALAPPDATA%\SlipStream2`, which never overlaps SlipStream v1's folder. Uninstalling from **Settings > Apps** removes the program, its shortcuts and its sign-in entry, and keeps that folder.

Host commands such as `slipstream join`, `slipstream status` and `slipstream stop` work in cmd and PowerShell, in any new terminal. [Host commands](#host-commands) lists them.

### Portable ZIP

Extract `SlipStream-<v>-windows-amd64.zip` into any folder you can write to, then open `slipstream-app.exe` in the `SlipStream` folder. Data lives in the `data` folder beside it. The ZIP never starts SlipStream at sign-in by itself.

## macOS

1. Open `SlipStream-<v>-darwin-arm64.dmg` and drag **SlipStream** into **Applications**.
2. Open SlipStream from Applications. macOS refuses the first time, because the app is not notarized. Choose **Done**, not **Move to Trash**.
3. Open **System Settings > Privacy & Security**. Scroll to **Security**, where macOS names SlipStream as blocked, and choose **Open Anyway**. Confirm with your password, then choose **Open Anyway** once more in the dialog that follows.

   > **Screenshot placeholder:** System Settings > Privacy & Security, with the "SlipStream was blocked to protect your Mac" line and its **Open Anyway** button highlighted. Add it as `docs/images/macos-open-anyway.png`.

4. SlipStream appears in the menu bar, with no Dock icon, and your browser opens at the setup page.

macOS asks only once per installed copy. SlipStream starts at login after its first launch; untick **Start at login** in its menu to stop that. Data lives in `~/Library/Application Support/SlipStream2`.

## Linux

### Debian, Ubuntu (deb)

```sh
sudo apt install ./slipstream_<v>_amd64.deb
```

### Fedora, RHEL, Rocky, Alma (rpm)

```sh
sudo dnf install ./slipstream-<v>-1.x86_64.rpm
```

Use the arm64 or aarch64 file on an arm64 machine.

Either package creates a `slipstream` system user in the `render` and `video` groups, and starts the `slipstream` service, which is enabled at boot. Its data lives in `/var/lib/slipstream`.

The Setup code and the setup page's URL are in the service's log:

```sh
sudo journalctl -u slipstream
```

Open that URL from any machine's browser, with `localhost` replaced by this machine's address.

Run host commands as the service user, so the files they write stay readable by the service:

```sh
sudo -u slipstream slipstream join <code>
sudo -u slipstream slipstream status
sudo -u slipstream slipstream admin reset <email>
```

`admin reset` prints a Reset link for the Account with that email, for when no Admin can sign in. It works whether SlipStream is running or stopped.

SlipStream refuses a host command run as another user and prints this hint.

Removing the package (`sudo apt remove slipstream` or `sudo dnf remove slipstream`) stops the service and keeps `/var/lib/slipstream`. Purging keeps it too.

### Without root: tarball

The tarball suits a seedbox, or any machine where you cannot install packages:

```sh
tar -xzf slipstream-<v>-linux-amd64.tar.gz
cd slipstream-<v>
./slipstream join <code>
./slipstream service install --user
```

`service install --user` writes a `systemctl --user` unit, starts it, and turns on lingering, so the Node keeps running after you log out. It prints `slipstream service running (PID n) on <dir>` once the Node is up. Running it again changes nothing. Data lives in `~/.local/share/slipstream2`, or under `$XDG_DATA_HOME` when you set it.

To leave, run `./slipstream service uninstall --user`. It stops and removes the unit, and leaves lingering on.

### Desktop: AppImage

```sh
chmod +x SlipStream-<v>-x86_64.AppImage
./SlipStream-<v>-x86_64.AppImage
```

Double-clicking the file does the same, and opens your browser at the setup page. The AppImage mounts itself through FUSE. If it refuses to start, install `fuse3` (or `libfuse2` on older distributions). Linux has no tray icon.

To run it as a service, run `./SlipStream-<v>-x86_64.AppImage service install --user`. The unit starts the AppImage from where it sits, so move it to its final place, such as `~/Applications`, first.

## Docker

```sh
docker run -d --name slipstream \
  -e PUID=1000 -e PGID=1000 \
  -p 8484:8484 -p 8485:8485 \
  -v /srv/slipstream/config:/config \
  -v /srv/media:/data \
  --device /dev/dri \
  ghcr.io/jatassi/slipstream:latest
```

- `/config` holds the Node's data dir. `/data` is where your media lives.
- `PUID` and `PGID` set the user and group the Node runs as. Both default to 1000.
- Port 8484 serves the web UI. Port 8485 is where other Nodes reach this one.
- The image tries only its configured web port, so the port inside the container always matches your `-p` mapping. If that port is taken, the Node exits and names it. Other installs try the next 9 ports first. `http.ports_to_try` sets how many ports a Node tries.
- `--device /dev/dri` gives the container hardware transcoding. The entrypoint joins the group that owns each render device. Leave the line out on a machine with no GPU.
- `docker logs slipstream` shows the Setup code and the setup page's URL.

Run host commands as your `PUID`, so `/config` keeps its owner:

```sh
docker exec -u 1000 slipstream slipstream join <code>
docker exec -u 1000 slipstream slipstream admin reset <email>
```

SlipStream refuses a `docker exec` run as root and prints this form.

Tags:

| Tag | Follows |
|---|---|
| `X.Y.Z` | one release, never moved |
| `X.Y`, `X` | the newest release in that line |
| `latest` | the newest release |
| `beta` | the newest pre-release |

The image never updates itself. Pull a newer tag and recreate the container.

## Host commands

Host commands act on the Node whose data dir is `--data-dir`, else `SLIPSTREAM_DATA_DIR`, else the install's default. Run them as the user the sections above name for each install.

| Command | What it does |
|---|---|
| `slipstream status` | Prints the running Node's status as JSON. |
| `slipstream stop` | Stops the Node and waits until it releases its data dir. |
| `slipstream restart` | Restarts the Node and prints its web address once it answers again. |
| `slipstream config` | Prints the resolved configuration as JSON. |
| `slipstream config set <key> <value>` | Saves a Node setting, such as `log.level`, in `config.yaml`. |
| `slipstream config unset <key>` | Removes a Node setting from `config.yaml`, so its default applies. |
| `slipstream join [--node-port <port>] <code>` | Joins an Unclaimed Node, stopped or running, to the Controller a Join Code names. `--node-port` sets the port of `link.listen` first. |
| `slipstream node-address` | Prints the Node address, `SLN1-…`, that a Controller dials this Node at. |
| `slipstream version` | Prints the build stamp. |

A Node that can't reach the Controller a Join Code names prints its Node address and waits up to 10 minutes. Paste the address into the Add a Node dialog on the Controller, which then dials this Node and keeps dialing it. Forward the port `--node-port` sets to this machine, or the Controller can't reach it either. A running Node binds a new `--node-port` only after a restart.

`config set` and `config unset` print the key's value as the Node now resolves it. A running Node applies a live setting, such as `log.level`, at once. A setting such as `http.listen` waits for a restart, and the line ends `, applies after restart`. A setting an environment variable sets is refused.
