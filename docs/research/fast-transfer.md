# Fast transfer between Nodes

Research for [#13](https://github.com/jatassi/SlipStream/issues/13) on the SlipStream v2 map ([#1](https://github.com/jatassi/SlipStream/issues/1)). Facts and tradeoffs only. The decision belongs to the ticket this feeds.

## Question and context

How do the fastest tools move large completed downloads from a seedbox to a home server and saturate the link? The ticket names segmented parallel transfer, multi-stream TCP, QUIC and congestion control such as BBR. It also asks about resumability, integrity checks, and leaving the original in place for seeding.

The charting decisions in [#2](https://github.com/jatassi/SlipStream/issues/2) frame it. Q9 says one Controller owns the library and other Nodes run the same binary with assigned Roles. Q11 says file sync is a one-way push of completed downloads to the importing Node, while the seedbox keeps seeding. Q13 says the torrent client is built in and private-tracker grade. So both ends of the transfer are SlipStream Nodes, and the seedbox Node holds the torrent metadata for every file it sends.

v1 has no transfer feature to carry forward. Its backend inventory lists "remote path mappings for download clients" as absent ([v1 backend inventory, section "Absent in code"](https://github.com/jatassi/SlipStream/blob/research/v1-inventory/docs/research/v1-inventory/backend.md)).

## Short answer

- A single stream is capped by its window divided by the round-trip time, so long paths need either big windows or many streams. Default SSH channel windows and SFTP request pipelines cap one file at roughly 40 to 170 Mbit/s at 100 ms RTT (computed below).
- The fast tools split work across many TCP connections. lftp `pget`, rclone multi-thread and aria2 split one file into byte ranges. lftp `mirror --parallel` and rclone `--transfers` run several files at once.
- Segmenting one file is a pull operation in every tool examined. lftp only uses `pget` when the target is local. rclone needs a capable destination backend, and SFTP is not one. The IETF resumable upload draft explicitly rules out parallel upload of one representation.
- BBR is a sender-side change that helps on lossy and shallow-buffered paths. On Linux an unprivileged process can only pick it per socket if the admin allowed it.
- QUIC streams in one connection share one congestion controller, so QUIC alone does not buy the multi-flow share that N TCP connections get.
- Resume and integrity are solved by range requests plus per-chunk hashes. The seedbox already has per-piece hashes in the torrent metadata, which can serve as the integrity reference.

## 1. What limits one stream

A reliable transport can have at most one window of unacknowledged data in flight per round trip. Throughput per stream is therefore at most window / RTT. The bandwidth-delay product (BDP) is the window a path needs to fill. A 1 Gbit/s path at 100 ms RTT needs 12.5 MB in flight, and 10 Gbit/s at 100 ms needs 125 MB (computed).

Several layers impose a window, and the smallest one wins.

| Limit | Default value | Source | Ceiling at 20 / 50 / 100 / 200 ms RTT (computed) |
|---|---|---|---|
| OpenSSH channel window | `CHAN_SES_WINDOW_DEFAULT (64*CHAN_SES_PACKET_DEFAULT)` with 32 KiB packets, so 2 MiB | [openssh-portable `channels.h`](https://github.com/openssh/openssh-portable/blob/master/channels.h) | 839 / 336 / 168 / 84 Mbit/s |
| Go `x/crypto/ssh` channel window (used by Go SSH clients such as rclone's SFTP backend, inferred) | `channelWindowSize = 64 * channelMaxPacket`, commented "We follow OpenSSH here", so 2 MiB | [golang/crypto `ssh/channel.go`](https://github.com/golang/crypto/blob/master/ssh/channel.go) | same as above |
| rclone SFTP pipeline | 64 outstanding requests (`--sftp-concurrency`) of 32 KiB (`--sftp-chunk-size`), so 2 MiB | [rclone SFTP docs](https://rclone.org/sftp/) | 839 / 336 / 168 / 84 Mbit/s |
| lftp SFTP pipeline | `sftp:max-packets-in-flight` default 16, `sftp:size-read` default 0x8000, so 512 KiB | [lftp man page](https://lftp.yar.ru/lftp-man.html) | 210 / 84 / 42 / 21 Mbit/s |
| Linux TCP send buffer autotuning max | `tcp_wmem` max "between 64K and 4MB, depending on RAM size" | [kernel ip-sysctl docs](https://docs.kernel.org/networking/ip-sysctl.html) | up to 1678 / 671 / 336 / 168 Mbit/s for 4 MiB |
| Linux TCP receive buffer autotuning max | `tcp_rmem` max "between 131072 and 32MB, depending on RAM size" | same | up to 2684 Mbit/s at 100 ms for 32 MiB |

The SSH window mechanism is defined in [RFC 4254 section 5.2](https://www.rfc-editor.org/rfc/rfc4254#section-5.2). The receiver grants window with `SSH_MSG_CHANNEL_WINDOW_ADJUST`, and the sender stops when it runs out. The ceilings above assume no loss and treat the buffer size as the usable window. Real ceilings are lower because kernel buffers also hold overhead (inferred).

A trap for tuning. The kernel docs say calling `setsockopt()` with `SO_RCVBUF` or `SO_SNDBUF` disables autotuning for that socket ([ip-sysctl](https://docs.kernel.org/networking/ip-sysctl.html)). lftp's `net:socket-buffer` sets exactly those options ([lftp man page](https://lftp.yar.ru/lftp-man.html)), so a small fixed value can make a long path slower.

Loss is the second limit. The BBR draft quotes RFC 9438. For CUBIC, "sustaining 10Gbps over 100ms RTT needs a packet loss rate below 0.000003%", and at 1% loss on a 100 ms path CUBIC "can only sustain at most 3 Mbps" ([draft-ietf-ccwg-bbr-06, Introduction](https://datatracker.ietf.org/doc/html/draft-ietf-ccwg-bbr)). Loss-based algorithms cut the sending rate "by 50% in Reno [RFC5681] or 30% in CUBIC [RFC9438]" on each loss event (same source).

## 2. Techniques the fast tools use

### 2.1 Segmented parallel transfer of one file

| Tool | Mechanism | Direction | Source |
|---|---|---|---|
| lftp `pget -n N` | Gets one file over N connections. The man page warns it "loads the net and server heavily impacting other users". | Download only. `mirror --use-pget-n` applies only when the target is local. | [lftp man page](https://lftp.yar.ru/lftp-man.html); [`MirrorJob.cc`](https://github.com/lavv17/lftp/blob/master/src/MirrorJob.cc) has `bool use_pget=(pget_n>1) && target_is_local;` |
| rclone multi-thread | Files above `--multi-thread-cutoff` (default 256M) use `--multi-thread-streams` (default 4) streams. Local destination is preallocated with `fallocate` and each thread writes its own range. | "only works with supported backends as the destination but will work with any backend as the source". SFTP is marked without MultithreadUpload, local and SMB have it. | [rclone docs](https://rclone.org/docs/); [rclone overview, Optional Features table](https://rclone.org/overview/) |
| aria2 | `--split` connections per file (default 5), capped by `--max-connection-per-server` (default 1). `--min-split-size` default 20M. | Download. The split options are documented for HTTP(S)/FTP. | [aria2c manual](https://aria2.github.io/manual/en/html/aria2c.html) |

So segmenting a single file is a pull by the receiving side in all three tools. Pushing one file as parallel segments needs a server that accepts out-of-order writes at offsets. rclone does this only for backends that implement `OpenWriterAt` or `OpenChunkWriter` ([rclone docs](https://rclone.org/docs/)). tus offers it through its Concatenation extension, where partial uploads are later joined ([tus protocol](https://tus.io/protocols/resumable-upload)). The IETF HTTP resumable upload draft says "Unlike ranged downloads, this protocol does not support transferring an upload as multiple requests in parallel" ([draft-ietf-httpbis-resumable-upload-12](https://datatracker.ietf.org/doc/html/draft-ietf-httpbis-resumable-upload)).

### 2.2 Several files in parallel

lftp `mirror -P N` downloads N files in parallel, and `mirror -R` reverses direction to put files ([lftp man page](https://lftp.yar.ru/lftp-man.html)). rclone `--transfers` runs 4 file transfers in parallel by default ([rclone docs](https://rclone.org/docs/)). File-level parallelism works for push as well as pull. It does not help a season pack that is one large file, or a single movie (inferred).

### 2.3 Multi-stream TCP

N TCP connections through a shared bottleneck each get a congestion window, so together they grab a larger share and recover from a single loss faster than one connection (inferred from the per-flow window and loss limits in section 1). The IETF has long flagged this. RFC 2914 calls opening "multiple connections to the same place" a "well-known way to achieve more aggressive performance without even changing the transport protocol" ([RFC 2914 section 3.2](https://www.rfc-editor.org/rfc/rfc2914#section-3.2)).

Syncthing is the closest open source design to Node-to-Node sync. Since version 1.25.0 it can hold several connections to one device. Both sides must opt in, one connection carries metadata, and block requests spread randomly over the rest. Its docs say returns are likely "negligible above about 10 to 20 connections" and cap the count at 128. Rate limits apply to the sum of all connections ([Syncthing numConnections](https://docs.syncthing.net/advanced/device-numconnections.html)).

### 2.4 Congestion control and BBR

Congestion control runs at the sender. RFC 9002 calls its algorithm "a sender-side congestion controller" and says "A sender can unilaterally choose a different algorithm" ([RFC 9002 section 7](https://www.rfc-editor.org/rfc/rfc9002#section-7)). Data flows from the seedbox to the importing Node in both push and pull, so the seedbox's choice is the one that matters (inferred).

BBR builds "an explicit model of the network path" from delivery rate and RTT. It "offers substantially higher throughput for bottlenecks with shallow buffers or random losses" than Reno or CUBIC ([draft-ietf-ccwg-bbr-06, Abstract](https://datatracker.ietf.org/doc/html/draft-ietf-ccwg-bbr)). The draft describes BBRv3. Mainline Linux `net/ipv4/tcp_bbr.c` cites the 2016 ACM Queue paper, and Google publishes v3 as a separate branch "intended to enable research collaboration and wider testing" ([Linux `tcp_bbr.c`](https://github.com/torvalds/linux/blob/master/net/ipv4/tcp_bbr.c); [google/bbr v3 README](https://github.com/google/bbr/blob/v3/README.md)). So mainline is BBRv1 (inferred from those two sources).

How a process gets BBR on Linux:

- System-wide with `net.ipv4.tcp_congestion_control` ([ip-sysctl](https://docs.kernel.org/networking/ip-sysctl.html)).
- Per socket with `setsockopt(TCP_CONGESTION)`. "Unprivileged processes are restricted to choosing one of the algorithms in tcp_allowed_congestion_control", whose default is "reno" plus the system default ([tcp(7)](https://man7.org/linux/man-pages/man7/tcp.7.html)). A SlipStream Node on a rented seedbox without root cannot switch itself to BBR unless the host already allows it (inferred).
- The kernel source notes BBR "might be used with the fq qdisc" with pacing. Otherwise the stack falls back to internal pacing that "may use more resources" ([Linux `tcp_bbr.c`](https://github.com/torvalds/linux/blob/master/net/ipv4/tcp_bbr.c)).

### 2.5 QUIC

RFC 9002 says "The congestion controller is per path" ([RFC 9002 section 7](https://www.rfc-editor.org/rfc/rfc9002#section-7)). All streams in one QUIC connection therefore share one window. QUIC streams remove head-of-line blocking between streams, but they do not add flows. Getting the multi-flow effect of section 2.3 needs several QUIC connections (inferred from the per-path controller).

QUIC runs in user space, so the congestion algorithm ships with the library, not the kernel. That sidesteps the `tcp_allowed_congestion_control` restriction (inferred). What the libraries offer:

| Library | Congestion control | Source |
|---|---|---|
| quic-go (Go) | `internal/congestion` holds Cubic, hybrid slow start and a pacer. No BBR file. | [quic-go `internal/congestion`](https://github.com/quic-go/quic-go/tree/master/internal/congestion) |
| quinn (Rust) | Cubic, NewReno, and `Bbr` documented as "Experimental! Use at your own risk." | [quinn-proto `congestion`](https://docs.rs/quinn-proto/latest/quinn_proto/congestion/index.html) |
| msquic (C) | `CongestionControlAlgorithm` setting, default 0 (Cubic) | [msquic Settings.md](https://github.com/microsoft/msquic/blob/main/docs/Settings.md) |

UDP has its own tuning cost. quic-go's docs say "QUIC transfers on high-bandwidth connections can be limited by the size of the UDP receive and send buffer" and recommend raising `net.core.rmem_max` and `net.core.wmem_max` to 7340032. GSO batching needs Linux 4.18 or later ([quic-go optimizations](https://quic-go.net/docs/quic/optimizations/)). Raising those sysctls also needs root (inferred).

Syncthing ships QUIC alongside TCP, and its multi-connection mode supports both ([Syncthing config](https://docs.syncthing.net/users/config.html); [numConnections](https://docs.syncthing.net/advanced/device-numconnections.html)).

### 2.6 Proprietary UDP transports

IBM Aspera FASP is described as a "patented technology" ([IBM Aspera](https://www.ibm.com/products/aspera/technology)). It is not an option for an open source binary. It is listed only because it is the commercial reference point.

## 3. Resumability

| Tool or protocol | How it resumes | Source |
|---|---|---|
| lftp `pget -c` | Requires a `lfile.lftp-pget-status` file. `pget:save-status` sets how often it is written. | [lftp man page](https://lftp.yar.ru/lftp-man.html) |
| lftp `mirror -c` | "continue a mirror job if possible" | same |
| aria2 | `--continue` for HTTP(S)/FTP. A control file (`*.aria2`) records progress and is saved every `--auto-save-interval` seconds and on stop. | [aria2c manual](https://aria2.github.io/manual/en/html/aria2c.html) |
| rsync `--partial`, `--partial-dir` | Keeps the partial file so "a subsequent transfer of the rest of the file much faster". Default is to delete it on interrupt. | [rsync(1)](https://download.samba.org/pub/rsync/rsync.1) |
| rsync `--append-verify` | Appends to a shorter destination and includes all data in the checksum check | same |
| rclone | Uploads to `name.XXXXXX.partial` then renames. "If the upload fails then the .partial file will be deleted." The docs examined describe no resume of a partial file. | [rclone docs](https://rclone.org/docs/) |
| HTTP | Range requests exist so a client with a partial copy can "request the remainder" ([RFC 9110 section 14](https://www.rfc-editor.org/rfc/rfc9110#section-14)) | RFC 9110 |
| HTTP upload | tus and the IETF draft resume by asking the server its offset, then appending from `Upload-Offset` | [tus](https://tus.io/protocols/resumable-upload); [IETF draft](https://datatracker.ietf.org/doc/html/draft-ietf-httpbis-resumable-upload) |

Segmented tools resume at chunk granularity because each chunk is an independent range (inferred from the lftp and aria2 status files). A resume that trusts existing bytes needs a hash check on them, or it can carry forward corruption. rsync's `--append` "updates these growing file in-place without verifying any of the existing content", which is why `--append-verify` exists ([rsync(1)](https://download.samba.org/pub/rsync/rsync.1)).

## 4. Integrity checks

| Approach | Facts | Source |
|---|---|---|
| rsync built-in | "rsync normally verifies that each transferred file was correctly reconstructed on the receiving side by checking a whole-file checksum that is generated as the file is transferred". Algorithms include xxh128, xxh3, xxh64, md5, md4, sha1. | [rsync(1)](https://download.samba.org/pub/rsync/rsync.1) |
| rclone | Checks transferred checksums and errors "corrupted on transfer" on mismatch. Over SFTP this needs shell access to run `md5sum`, `sha1sum` or similar on the remote. | [rclone docs](https://rclone.org/docs/); [rclone SFTP](https://rclone.org/sftp/) |
| lftp | `xfer:verify` runs `xfer:verify-command` after a transfer. A zero exit code means the file is correct. | [lftp man page](https://lftp.yar.ru/lftp-man.html) |
| Syncthing | Files are split into blocks of 128 KiB to 16 MiB, and each block carries its size and hash | [Syncthing BEP v1](https://docs.syncthing.net/specs/bep-v1.html) |
| BitTorrent v1 | `pieces` is a list of SHA-1 hashes, one per piece. Multi-file torrents are hashed as if the files were concatenated, so a piece can span two files. | [BEP 3](https://www.bittorrent.org/beps/bep_0003.html) |
| BitTorrent v2 | Each file has a `pieces root`, a SHA-256 merkle tree over 16 KiB blocks. Every non-empty file is aligned to a piece boundary. | [BEP 52](https://www.bittorrent.org/beps/bep_0052.html) |

Hash speed decides whether verification costs throughput. The xxHash README reports XXH3 at 59.4 GB/s and XXH128 at 57.9 GB/s against SHA1 at 0.8 GB/s and MD5 at 0.6 GB/s on an i7-9700K. It also warns xxHash "is not a cryptographic hash function" ([xxHash README](https://github.com/Cyan4973/xxHash)). BLAKE3 is cryptographic, "Highly parallelizable across any number of threads and SIMD lanes, because it's a Merkle tree on the inside", and "Capable of verified streaming" through Bao ([BLAKE3 README](https://github.com/BLAKE3-team/BLAKE3)). These benchmark numbers are the projects' own and were not reproduced here.

The torrent hashes are already on the seedbox Node for every completed download. For a v2 torrent, the per-file merkle root lets the receiver verify each file and any 16 KiB-aligned chunk on its own. For a v1 torrent, a piece that spans a file boundary cannot be checked without the neighbouring file's bytes (inferred from BEP 3 and BEP 52). Hashing the file again at the source would be needed in that case.

## 5. Leaving the original in place for seeding

- **Copy, never move.** lftp `--Remove-source-files`, `--Remove-source-dirs` and `--Move` delete sources ([lftp man page](https://lftp.yar.ru/lftp-man.html)). rsync `--remove-source-files` does the same ([rsync(1)](https://download.samba.org/pub/rsync/rsync.1)). rclone `move` deletes sources and `copy` keeps them ([rclone docs](https://rclone.org/docs/)).
- **Only send quiescent files.** rsync warns to use `--remove-source-files` "only ... on source files that are quiescent" ([rsync(1)](https://download.samba.org/pub/rsync/rsync.1)). The built-in client knows when a torrent is complete, so the trigger can be the completion event, not a directory scan (inferred).
- **Write to a temp name, then rename.** This keeps a half-written file away from the importer. rclone does it by default with `.partial` ([rclone docs](https://rclone.org/docs/)). lftp does it with `xfer:use-temp-file` ([lftp man page](https://lftp.yar.ru/lftp-man.html)).
- **Same machine, no transfer.** When the downloading and importing Roles share a Node, a hard link or reflink keeps one copy on disk. Both fail with `EXDEV` across filesystems ([link(2)](https://man7.org/linux/man-pages/man2/link.2.html); [ioctl_ficlone(2)](https://man7.org/linux/man-pages/man2/ioctl_ficlone.2.html)).
- **Bandwidth contention.** The transfer leaves the seedbox on the same uplink that serves seeding peers. More streams and BBR take a larger share of that uplink, which lowers seeding upload while the transfer runs (inferred). The tools expose caps for this, such as lftp `net:limit-rate` and Syncthing's per-device rate limit that applies to all connections together ([lftp man page](https://lftp.yar.ru/lftp-man.html); [numConnections](https://docs.syncthing.net/advanced/device-numconnections.html)).
- **BitTorrent between Nodes is constrained.** A client with a `private=1` torrent "MUST ONLY initiate connections to peers returned from the private tracker" ([BEP 27](https://www.bittorrent.org/beps/bep_0027.html)). Adding the home Node as a direct peer for a private torrent contradicts that rule.

## 6. Push versus pull

Q11 fixes the data direction, seedbox to importing Node. It does not fix which side opens the connection (inferred from the wording in #2).

| | Seedbox initiates (push) | Importing Node initiates (pull) |
|---|---|---|
| Segmented single file | Needs a receiver that accepts offset writes or partial-upload concatenation (rclone `OpenWriterAt`/`OpenChunkWriter`, tus Concatenation). Not plain SFTP, not the IETF resumable upload draft. | Native in lftp, rclone and aria2 via range reads ([section 2.1](#21-segmented-parallel-transfer-of-one-file)) |
| Reachability | The importing Node must accept inbound connections. A home server behind NAT needs a port forward or a relay. | The seedbox must accept inbound connections, which a rented server normally does (inferred) |
| Who chooses congestion control | The seedbox, as sender | Still the seedbox, as sender |
| Resume state | Sender asks the receiver for its offset (tus, IETF draft) | Receiver keeps a status or control file (lftp, aria2) |

A relaying Role exists in the glossary, and Q10 puts a forwarder in front of the UI and Media Server. Whether that forwarder also carries Node-to-Node file traffic is not settled in #2.

## 7. Options for the decision

Each option is a shape the transfer could take. None is recommended here.

| Option | What it is | For | Against |
|---|---|---|---|
| A. Drive an external tool | Shell out to rclone or lftp over SFTP | Proven segmenting and resume. Little code. | Segmenting works only as a pull. A second binary to ship, configure and keep in step with Q24 packaging. SFTP pipelines cap at 512 KiB to 2 MiB per file without tuning. rclone hash checks over SFTP need a remote shell. |
| B. HTTP range pull, built in | The importing Node fetches ranges in parallel from the seedbox Node over HTTPS | Standard semantics (RFC 9110). Chunk resume and per-chunk hashes are straightforward. Reuses the TLS setup Q10 already needs. | Inverts the "push" wording of Q11 in initiation. The seedbox must be reachable. Congestion control is whatever the seedbox kernel allows. |
| C. Custom chunk protocol over N TCP connections | Node-to-Node protocol in the style of Syncthing: block list with hashes, requests spread over several connections | Works in either initiation direction. Can reuse torrent piece hashes. Rate cap across all connections is easy. | Most code to own. Fairness to seeding traffic is on SlipStream. |
| D. Same as C, over QUIC | Several QUIC connections, congestion control chosen in the library | Congestion control does not depend on kernel permissions. One port, built-in TLS. | One connection is one window, so parallelism still needs several connections. UDP buffer limits need root to raise. In Go, quic-go ships Cubic only. |
| E. BitTorrent between Nodes | Home Node joins the swarm as a peer of the seedbox | Integrity and resume come free with the protocol | Breaks BEP 27 for private torrents, and Q13 makes private trackers the target |

## 8. Not verified here

- No throughput was measured. Every Mbit/s figure above is arithmetic from documented defaults, not a benchmark.
- Whether OpenSSH grows the channel window past `CHAN_SES_WINDOW_DEFAULT` at runtime was not traced in source.
- How common BBR or a permissive `tcp_allowed_congestion_control` is on rented seedboxes was not surveyed.
- Whether rclone resumes multi-thread downloads after a crash was not found in its docs. The docs only describe deleting the `.partial` file on failure.
