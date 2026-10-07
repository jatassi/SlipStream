# Conversions run on a bundled jellyfin-ffmpeg and go only to proven Nodes

Every Node runs a pinned jellyfin-ffmpeg build that SlipStream bundles for its OS and architecture and ships in lockstep with each release. jellyfin-ffmpeg is a fork, not upstream FFmpeg. Upstream tone maps Dolby Vision Profile 5 correctly only through libplacebo on a working Vulkan driver. The fork adds DV-aware tone mappers on CUDA, OpenCL and Metal, so more hardware can convert a DV source to SDR. The Controller sends a Conversion only to a Node that has proven its Conversion settings by a short trial run on that source's codec, bit depth and dynamic range. A passed trial is cached until the Node's ffmpeg or driver changes. Build lists and device probes miss filter graphs that fail at runtime, tone mapping included, and Tdarr and Unmanic leave capability to the admin. Decided in [A/V pipeline design](https://github.com/jatassi/SlipStream/issues/18), Q7, Q8, Q9 and Q13. Research in [ffmpeg hardware acceleration and HDR handling](https://github.com/jatassi/SlipStream/issues/17).

## Considered Options

- **Upstream FFmpeg with libplacebo.** Rejected. DV Profile 5 to SDR would need Vulkan on every converting Node.
- **Whatever ffmpeg the system has.** Rejected. Capability would vary by distro, license build and driver, and Nodes would no longer run the same filters in lockstep (ADR 0003).
- **Admin-declared capability per Node, as Tdarr and Unmanic do.** Rejected. The admin would have to know which GPU handles which codec and tone mapper, and a wrong guess fails only after the Conversion starts.

## Consequences

- The package carries a GPL-3.0 binary, run as a separate process. Shipping it carries a source offer, which the license ticket weighs (inferred, not legal advice).
- SlipStream follows the fork's release cadence. An admin can point a Node at another binary, which the UI marks unsupported, and changing it reruns the trial runs.
- A source that no Node can convert waits as "no capable Node" rather than failing.
- The offline suite needs a fake capability source, so simulated Nodes can report GPUs that CI lacks.
