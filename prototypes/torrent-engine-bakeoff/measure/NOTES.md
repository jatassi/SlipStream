# measure

Python 3 stdlib only.

- `sample.py --pid PID --interval 5 --out FILE.jsonl`: one line per sample `{t, rss_kb, pss_kb, cpu_ticks (utime+stime of the whole process, so all threads), num_threads, fds}`. Stops on process exit, zombie, SIGTERM or SIGINT. Reads /proc/PID/{status,smaps_rollup,stat,fd}; needs same-user or root for smaps_rollup/fd. Convert ticks with `os.sysconf("SC_CLK_TCK")`.
- `announces.py LOG [--since T] [--json]`: summary of a tracker log. Gap stats use only accepted-or-not no-event announces per (infohash, peer_id) pair, rejected ones excluded; "per-infohash" pools gaps from all peer_ids on that infohash. Text mode lists the 5 infohashes with the smallest min gap.
- `cpu.py SAMPLES.jsonl --from T0 --to T1 [--json]`: CPU % of one core = delta cpu_ticks / CLK_TCK / delta t between first and last sample in the window; RSS/PSS median and max in MiB. T0/T1 are unix seconds.
- Memory fields: `rss_kb` (VmRSS) includes file-backed mmap pages (libtorrent reads/writes data via mmap: ~2.0 GB RssFile vs ~8 MB RssAnon for 20x4GiB seeds). Compare engines on `rss_anon_kb` (RssAnon) and `pss_kb`; `rss_file_kb` and `rss_shmem_kb` are sampled separately. cpu.py reports median/max for rss, rss_anon, rss_file and pss.
