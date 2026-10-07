# Release judging settings prototype

**Throwaway.** This never merges. It exists to answer one question for [Custom formats engine and TRaSH Guides import](https://github.com/jatassi/SlipStream/issues/7): can the brainstormed design (Verified Priorities plus Horizon) configure a real library precisely enough? Can its owner set it up in a way they trust?

Open `dist/index.html` by double-clicking it. It is one self-contained file, and settings live only in memory.

## What is real

- **The library.** Every number comes from a snapshot of `/Volumes/Media` taken on 2026-10-06. It holds 1,291 Plex files, plus 296 releases in Downloads matched to library files by byte size.
- **The probes.** All 2,140 video files were probed with MediaInfo 26.10 using a header-only parse (`--ParseSpeed=0`), with 0 errors. On a 6-file check, the header-only parse matched a full parse on HDR format, Dolby Vision profile, HDR10+, Atmos and duration.
- **The TRaSH data.** The pack is TRaSH Guides at commit `e7c97a6` (2026-09-28). It holds 478 custom formats converted to claims with no scores, 28 English profiles, and the quality sizes.
- **The engine.** `src/engine.js` is a pure module covering parsing, claims, verification, trust, gates, judging, upgrades, probation, the Horizon stages and the custom format baseline.

## Variants: how a user sets priority order

- **A. Order a list.** Drag priorities and their ladders into order. A live preview runs on real Items.
- **B. Teach by choosing.** Pick between two real files of the same quality, and the order is solved from your answers.

## Findings from the real library

Numbers below are measured on the snapshot with default settings unless noted.

- **Byte-identical copies.** 196 files are second copies of a file already held for the same Item, totalling 1.19 TB. Under ADR 0002 one file fills every Rendition, so v2 keeps one.
- **Coverage.** Out of 1,024 Items:

  | Rendition | At cutoff | Upgradable | Derivable | Missing |
  |---|---|---|---|---|
  | 2160p HDR | 69 | 301 | n/a | 654 |
  | 1080p | 360 | 68 | 338 | 258 |

  45 held files have no source in either name, so they fill nothing until the owner picks a policy.
- **Names against probes.** One claim was refuted: `The.Devil.Wears.Prada.2…HDR…-POKE` is SDR BT.709. 58 releases undersell their file, mostly with HDR10+ that the name never mentions.
- **Probe limits.** 8 claims can't be checked from a header parse. A DTS-HD MA track can show only its DTS core, and an Atmos marker can be missing.
- **Calibration.** The base pack's terms held up on your files:

  | Claim | Held |
  |---|---|
  | Bare `HDR` | 88 of 89 |
  | `DV` on a UHD remux keeps an HDR10 base layer | 24 of 24 |
  | Atmos | 86 of 86 |

  Without calibration, a medium-confidence term from a group with few probes counts as weak. `DV.HDR` releases were then judged as DV with no fallback.
- **TRaSH conversion.** 451 of 478 custom formats translate. Per app, 7 regexes fail to compile in JavaScript. 85 Radarr and 70 Sonarr regexes use lookaround (57 and 53 formats), which Go RE2 and Rust `regex` reject.
- **TRaSH hazards needed two fixes.**
  - Hazards that only restate a fact (SDR, DV without fallback) became terms that the probe checks. TRaSH's "SDR" means "2160p with no HDR word", and it flagged HDR remuxes renamed by Plex.
  - Severity now comes only from profiles where the format is on by default. Before that, Scene and other optional formats came out as reject.
- **TRaSH on your library.** With the TRaSH pack on, two of its opinions reject held files:
  - "x265 (HD)" rejects 213 files, among them HONE's 1080p H.265 releases.
  - Its LQ list rejects 154 files, including FGT and 4K4U.
- **Sizes.** TRaSH's Bluray-1080p minimum is 50.8 MB/min. 100 of your 102 Bluray-1080p files fall below it.
- **Timing.** There is no Release Log yet, so the prototype uses download finish times as a stand-in.
  - 15 Items got a better download after an earlier one, and the earlier downloads total 102 GB.
  - Going from first web to best web happened 12 times. 75% arrived within 72 hours, with a median of 3 hours.
- **Variant B ran short of real choices.** No single Item in your library has two passing releases of the same quality that trade one priority for another. So B compares real files across titles. Alternating answers produce a conflict that a strict order cannot satisfy.

## Known approximations

- **Sonarr/Radarr baseline.** It reads language and indexer flags from the name only. An unnamed language counts as English, and English is taken as the original language.
- **Source.** WEB-DL versus Blu-ray can't be probed. It comes from the matched release name or the Plex filename.
- **Timing.** Download finish times lag the moment a release appeared, so they are only a stand-in for a Release Log.

## Rebuild

1. Run `tools/build_inventory.py`, `tools/probe.py` and `tools/trash_convert.py` in a scratch folder laid out as `inventory-1/`, `probe-1/`, `trash-pack-1/` and `trash/Guides/` (a sparse clone of `TRaSH-Guides/Guides`, `docs/json`).
2. Then run:

```bash
./build.sh <scratch>
```
