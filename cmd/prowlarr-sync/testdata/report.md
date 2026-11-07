# Prowlarr sync report

Prowlarr at `a69a7eae0ad82aff9a040ad0283271e4fa61a630` (2026-01-09). 7 change sets, 2 units up to date.

## Ports

### port:hdbits@c6e3f94

Synced to `8b635bb`. Run `prowlarr-sync bump --unit port:hdbits --to c6e3f94a90386f9527927100023b7a53c880cd03` once the diff is ported.

| Commit | Date | Subject |
|---|---|---|
| `c6e3f94` | 2026-01-03 | Fixed: (HDBits) Parse freeleech from the API |

| Upstream file | Change | Go files |
|---|---|---|
| `src/NzbDrone.Core/Indexers/Definitions/HDBits/HDBitsParser.cs` | modified | `internal/ports/hdbits/parser.go` |

```diff
diff --git a/src/NzbDrone.Core/Indexers/Definitions/HDBits/HDBitsParser.cs b/src/NzbDrone.Core/Indexers/Definitions/HDBits/HDBitsParser.cs
index 58ccee7..f5dc622 100644
--- a/src/NzbDrone.Core/Indexers/Definitions/HDBits/HDBitsParser.cs
+++ b/src/NzbDrone.Core/Indexers/Definitions/HDBits/HDBitsParser.cs
@@ -1,4 +1,4 @@
 public class HDBitsParser : IParseIndexerResponse
 {
-    // freeleech is 0
+    // freeleech is "yes"
 }
```

## Family base files

### family:gazelle@a69a7ea

Synced to `8b635bb`. Run `prowlarr-sync bump --unit family:gazelle --to a69a7eae0ad82aff9a040ad0283271e4fa61a630` once the diff is ported.

| Commit | Date | Subject |
|---|---|---|
| `a69a7ea` | 2026-01-09 | Fixed: (Gazelle) Page size 100 \| for every site |

| Upstream file | Change | Go files |
|---|---|---|
| `src/NzbDrone.Core/Indexers/Definitions/Gazelle/GazelleBase.cs` | modified | `internal/ports/internal/gazelle/gazelle.go` |

```diff
diff --git a/src/NzbDrone.Core/Indexers/Definitions/Gazelle/GazelleBase.cs b/src/NzbDrone.Core/Indexers/Definitions/Gazelle/GazelleBase.cs
index ab0ac36..13c1160 100644
--- a/src/NzbDrone.Core/Indexers/Definitions/Gazelle/GazelleBase.cs
+++ b/src/NzbDrone.Core/Indexers/Definitions/Gazelle/GazelleBase.cs
@@ -1,4 +1,4 @@
 public abstract class GazelleBase<TSettings> : TorrentIndexerBase<TSettings>
 {
-    // page size 50
+    // page size 100
 }
```

## Obsolete, moved and deleted ports

### moved:port:funfile@f6d2b2a

Upstream moved port:funfile to a Cardigann definition: "Moved to YML for Cardigann". Keep the port until that definition passes the runner tier, then add it to the moved list in internal/ports/all/all.go and delete it.

| Commit | Date | Subject |
|---|---|---|
| `f6d2b2a` | 2026-01-07 | Fixed: (FunFile) Moved to Cardigann |

| Upstream file | Change | Go files |
|---|---|---|
| `src/NzbDrone.Core/Indexers/Definitions/FunFile.cs` | modified | `internal/ports/funfile/funfile.go` |

```diff
diff --git a/src/NzbDrone.Core/Indexers/Definitions/FunFile.cs b/src/NzbDrone.Core/Indexers/Definitions/FunFile.cs
index adc8281..e7196a6 100644
--- a/src/NzbDrone.Core/Indexers/Definitions/FunFile.cs
+++ b/src/NzbDrone.Core/Indexers/Definitions/FunFile.cs
@@ -1,5 +1,6 @@
 namespace NzbDrone.Core.Indexers.Definitions;
 
+[Obsolete("Moved to YML for Cardigann")]
 public class FunFile : TorrentIndexerBase<UserPassTorrentBaseSettings>
 {
     private IndexerCapabilities SetCapabilities()
```

### deleted:port:rarbg@28dd240

Upstream deleted every file port:rarbg maps. Delete the port and add it to the retired list in internal/ports/all/all.go.

| Commit | Date | Subject |
|---|---|---|
| `28dd240` | 2026-01-06 | Remove Rarbg |

| Upstream file | Change | Go files |
|---|---|---|
| `src/NzbDrone.Core/Indexers/Definitions/Rarbg/Rarbg.cs` | deleted | `internal/ports/rarbg/rarbg.go` |
| `src/NzbDrone.Core/Indexers/Definitions/Rarbg/RarbgParser.cs` | deleted | `internal/ports/rarbg/rarbg.go` |

### obsolete:port:zonaq@55e8ea4

Upstream marked port:zonaq obsolete: "Site does not allow automation". Delete the port and add it to the retired list in internal/ports/all/all.go.

| Commit | Date | Subject |
|---|---|---|
| `55e8ea4` | 2026-01-05 | Fixed: (ZonaQ) Obsolete per site policy |

| Upstream file | Change | Go files |
|---|---|---|
| `src/NzbDrone.Core/Indexers/Definitions/ZonaQ.cs` | modified | `internal/ports/zonaq/zonaq.go` |

```diff
diff --git a/src/NzbDrone.Core/Indexers/Definitions/ZonaQ.cs b/src/NzbDrone.Core/Indexers/Definitions/ZonaQ.cs
index a7f04c7..5b04e2e 100644
--- a/src/NzbDrone.Core/Indexers/Definitions/ZonaQ.cs
+++ b/src/NzbDrone.Core/Indexers/Definitions/ZonaQ.cs
@@ -1,5 +1,6 @@
 namespace NzbDrone.Core.Indexers.Definitions;
 
+[Obsolete("Site does not allow automation")]
 public class ZonaQ : TorrentIndexerBase<UserPassTorrentBaseSettings>
 {
     private IndexerCapabilities SetCapabilities()
```

## New sites

### new:DesiGaane@5fd890d

DesiGaane derives from GazelleBase and carries movies, audio. Start its port with `prowlarr-sync new --site DesiGaane`, or list it in skipped.json with why.

- `src/NzbDrone.Core/Indexers/Definitions/DesiGaane.cs`

| Commit | Date | Subject |
|---|---|---|
| `5fd890d` | 2026-01-04 | New: (Indexer) DesiGaane |

## Hidden list

### hidden-list@8fc53d9

The hidden list changed since `8b635bb`. Added: `funfile`. Removed: none.

| Commit | Date | Subject |
|---|---|---|
| `8fc53d9` | 2026-01-08 | Hide the FunFile definition |

## Up to date

- kit at `8b635bb`
- port:alpharatio at `8b635bb`
