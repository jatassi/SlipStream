# ffmpeg hardware acceleration and HDR handling

Research for [#17](https://github.com/jatassi/SlipStream/issues/17). It feeds the A/V pipeline decision in the map ([#2](https://github.com/jatassi/SlipStream/issues/2), Q14): a Node with the transcoding Role derives Renditions, and the Controller schedules hardware acceleration across Nodes.

This file lays out facts and options. It makes no decision.

Sources were read on 2026-10-04. FFmpeg citations point at `master` head `a35c879` (2026-09-21), written below as `FFmpeg@a35c879`. The newest tagged release in FFmpeg's `Changelog` is 9.0.

## Answer in brief

- ffmpeg drives NVENC/NVDEC (NVIDIA), Quick Sync through libvpl or libmfx (Intel), VA-API (Intel and AMD on Linux), VideoToolbox (macOS), AMF (AMD), D3D11VA/D3D12VA (Windows) and Vulkan Video. All are compiled in at build time. Whether one works is only known at runtime.
- No single query answers "what can this Node encode". The layers are the build (`-hwaccels`, `-encoders`, `-filters`), the device (`-init_hw_device`), the vendor's own caps APIs, and a trial encode. Jellyfin probes the first two and leaves per-codec decode support to an admin checkbox.
- HDR10 to SDR works on every platform. The options are software (`zscale` + `tonemap`), OpenCL, VA-API, QSV VPP, VideoToolbox (`scale_vt`) and Vulkan (`libplacebo`).
- Dolby Vision is the hard case. Upstream, only `libplacebo` applies the DV RPU, so Profile 5 maps to SDR correctly only there. Jellyfin's fork adds DV-aware CUDA, OpenCL and Metal tone mappers.
- No hardware encoder in ffmpeg writes DV RPUs. Only `libx265` and `libsvtav1` (software) do. Tdarr and Unmanic do no tone mapping in their stock code.
- For scheduling, Tdarr runs one Server and many Nodes. Its workers are typed CPU or GPU, and it either shares storage through path translators or ships files with "unmapped" Nodes (a Pro feature above 10 MB). Unmanic links peer installations. It routes tasks by matching library names, and each install keeps its own encoder settings. It reads from shared storage when it can and falls back to an HTTP upload.

## 1. Hardware APIs ffmpeg can drive

### 1.1 Device types and encoders in the source

`libavutil/hwcontext.h` lists these device types: VDPAU, CUDA, VAAPI, DXVA2, QSV, VideoToolbox, D3D11VA, DRM, OpenCL, MediaCodec, Vulkan, D3D12VA, AMF, OHCODEC ([FFmpeg@a35c879 hwcontext.h L28-43](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/libavutil/hwcontext.h#L28-L43)).

The video encoders that `configure` declares for each API ([configure L3584-3790](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/configure#L3584-L3790)):

| API | Platforms | Encoders in FFmpeg | Notes |
|---|---|---|---|
| NVENC | Windows, Linux | `h264_nvenc`, `hevc_nvenc`, `av1_nvenc` | `av1_nvenc` needs `NV_ENC_PIC_PARAMS_AV1` in the SDK headers. |
| Quick Sync (QSV) | Windows, Linux | `h264_qsv`, `hevc_qsv`, `av1_qsv`, `vp9_qsv`, `mpeg2_qsv`, `mjpeg_qsv` | `av1_qsv` depends on `libvpl`. The older `libmfx` path still exists (`HWACCEL_LIBRARY_LIST`, L2184-2190). |
| VA-API | Linux (also libva-win32 since 6.1) | `h264_vaapi`, `hevc_vaapi`, `av1_vaapi`, `vp8_vaapi`, `vp9_vaapi`, `mpeg2_vaapi`, `mjpeg_vaapi` | One API covers Intel (iHD driver) and AMD (Mesa). |
| VideoToolbox | macOS | `h264_videotoolbox`, `hevc_videotoolbox`, `prores_videotoolbox` | No AV1 encoder. |
| AMF | Windows, Linux | `h264_amf`, `hevc_amf`, `av1_amf` | On Linux it needs AMD's proprietary AMF runtime (see 1.3). |
| D3D12VA | Windows | `h264_d3d12va`, `hevc_d3d12va`, `av1_d3d12va` | The HEVC encoder arrived in 7.1 (`Changelog`). |
| Vulkan Video | cross-vendor | `h264_vulkan`, `hevc_vulkan`, `av1_vulkan` | H.264 and H.265 arrived in 7.1, AV1 in 8.0 (`Changelog`). `av1_vulkan` needs `vulkan_1_4`. |
| MediaCodec, V4L2 M2M, RKMPP, OHCodec | Android, ARM SBCs | various | Out of scope for SlipStream Nodes. |

Hardware decode for HEVC goes through `hevc_nvdec`, `hevc_vaapi`, `hevc_videotoolbox`, `hevc_d3d11va`, `hevc_d3d12va`, `hevc_dxva2` and `hevc_vulkan` hwaccels. There are also the standalone `hevc_cuvid`, `hevc_qsv` and `hevc_amf` decoders ([configure L3362-3522, L3585-3695](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/configure#L3362-L3695)).

### 1.2 Licensing constraints on the build

- FFmpeg is LGPL 2.1+ by default. `--enable-gpl` turns it into GPL 2+ ([LICENSE.md](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/LICENSE.md)). `libx264` and `libx265` are on the GPL list ([configure L2015-2027](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/configure#L2015-L2027)).
- `cuda_nvcc` and `cuda_sdk` are on the non-free list ([configure L2179-2182](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/configure#L2179-L2182)). The CUDA filters (`scale_cuda`, `colorspace_cuda`) accept `cuda_llvm` as an alternative, so they build without the non-free flag ([configure L3550-3554](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/configure#L3550-L3554)).
- `nvenc`, `nvdec`, `vaapi`, `videotoolbox`, `amf`, `vulkan`, `d3d11va` and `d3d12va` are autodetected at configure time ([configure L2152-2170](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/configure#L2152-L2170)).

### 1.3 Vendor hardware facts

**NVIDIA.** From the [NVIDIA Video Encode and Decode GPU Support Matrix](https://developer.nvidia.com/video-encode-decode-support-matrix), with the tables parsed from the page HTML:

- Every GeForce row lists "Max # of concurrent sessions" as 12. Most professional rows list "Unrestricted".
- GTX 1080 (Pascal, 6th gen NVENC) encodes HEVC 10-bit but not HEVC B-frames. RTX 2080 and GTX 1660 (Turing, 7th gen) add HEVC B-frames. RTX 3080 (Ampere, 7th gen) is the same. AV1 encode starts with the RTX 40 series (Ada, 8th gen).
- NVDEC: RTX 2080 (Turing, 4th gen) decodes HEVC 4:2:0 at 8, 10 and 12 bit, with no AV1. RTX 3080 (Ampere, 5th gen) adds AV1 8 and 10 bit. RTX 50 (Blackwell, 6th gen) adds H.264 and HEVC 4:2:2.
- Most consumer cards have one NVENC engine. The RTX 4080, 5080 and some others have two.
- Jellyfin requires driver 522.25 on Windows and 520.56.06 on Linux, and says Maxwell is the oldest supported architecture ([Jellyfin NVIDIA](https://jellyfin.org/docs/general/post-install/transcoding/hardware-acceleration/nvidia)).

**Intel.** From the [intel/media-driver README](https://github.com/intel/media-driver/blob/master/README.md):

- HEVC 10-bit low-power (VDEnc) encode is listed from ICL onward. KBLx has only shader-assisted "Es" encode. AV1 encode starts with DG2 (Arc Alchemist).
- "HDR10 TM" (tone mapping) is listed for ICL, TGLx, DG1, DG2, MTLx, LNL, BMG, PTL and NVL in the Full-Feature build only. It is absent from the Free-Kernel build and from EHL/JSL.
- Debian and Ubuntu ship the Full-Feature build as `intel-media-va-driver-non-free`.
- Low-power encode bitrate control needs HuC firmware. The kernel loads it by default only from ADL onward.
- From MTL, low-power and shader encode merge into `VAEntrypointEncSlice`.
- The [libvpl README](https://github.com/intel/libvpl) says the VPL GPU runtime covers Tiger Lake and newer. The legacy Media SDK runtime covers back to Broadwell.
- Jellyfin says "there is no concurrent encoding sessions limit on Intel iGPU and ARC dGPU" ([Jellyfin Intel](https://jellyfin.org/docs/general/post-install/transcoding/hardware-acceleration/intel)).

**AMD.**

- Windows uses AMF. Linux uses VA-API through Mesa radeonsi ([Jellyfin AMD](https://jellyfin.org/docs/general/post-install/transcoding/hardware-acceleration/amd)).
- Jellyfin says AMD H.264 quality "has been unsatisfactory" for lack of B-frames, and that AV1 on RX 7000 (VCN 4.0) "seems to be the savior of AMD encoding quality". HEVC 10-bit encode needs RX 5000 or a Ryzen 4000 APU. AV1 encode needs RX 7000 or Ryzen 7000/8000.
- AMF on Linux needs AMD's runtime. From the 25.20 Linux driver on, "the AMF runtime is released separately" ([AMF README](https://github.com/GPUOpen-LibrariesAndSDKs/AMF/blob/master/README.md)).

**Apple.**

- Jellyfin supports macOS 12 and later. It lists HEVC decode and encode on Macs from 2017 (except the 13-inch MacBook Air 2017), and AV1 decode from M3 ([Jellyfin Apple](https://jellyfin.org/docs/general/post-install/transcoding/hardware-acceleration/apple)).
- In `hevc_videotoolbox`, constant-quality mode (`-q:v`) works only on Apple Silicon. The source comment reads "constant quality only on Macs with Apple Silicon" ([videotoolboxenc.c L1126-1129](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/libavcodec/videotoolboxenc.c#L1126-L1129)).
- `allow_sw` defaults to 0, so the encoder fails rather than falling back to software in silence (L2781-2784).

## 2. Detecting capabilities

### 2.1 Layers of detection

| Layer | What it answers | How |
|---|---|---|
| Build | Was this API, encoder or filter compiled in? | `ffmpeg -hwaccels`, `-encoders`, `-decoders`, `-filters`, `-init_hw_device list`, `-h encoder=X`, `-h filter=X`. The docs warn that for `-hwaccels`, "Actual runtime availability depends on the hardware and its suitable driver being installed" ([ffmpeg.texi L2100-2170](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/doc/ffmpeg.texi#L2100-L2170)). |
| Device | Does the driver open a device, and whose is it? | `ffmpeg -v verbose -init_hw_device vaapi=va:/dev/dri/renderD128`, or `cuda:0`, `qsv:hw,child_device=...`, `vulkan:N`, `d3d11va:,vendor_id=0x8086` ([ffmpeg.texi L1888-2096](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/doc/ffmpeg.texi#L1888-L2096)). |
| Codec and profile | Can this device decode or encode HEVC Main10, AV1 and so on? | Vendor APIs. On VA-API, `vaQueryConfigProfiles` and `vaQueryConfigEntrypoints` (`VAEntrypointVLD`=decode, `VAEntrypointEncSlice`/`EncSliceLP`=encode, `VAEntrypointVideoProc`=VPP) ([libva va.h L554-570, L1545-1572](https://github.com/intel/libva/blob/master/va/va.h)), and `vainfo` prints them. On NVENC, `NvEncGetEncodeCaps` with caps such as `NV_ENC_CAPS_SUPPORT_10BIT_ENCODE` and `NV_ENC_CAPS_NUM_ENCODER_ENGINES` ([nvEncodeAPI.h](https://github.com/FFmpeg/nv-codec-headers/blob/master/include/ffnvcodec/nvEncodeAPI.h)). On VideoToolbox, `VTCopyVideoEncoderList` with `kVTVideoEncoderList_IsHardwareAccelerated`, plus `VTIsHardwareDecodeSupported` ([Apple](https://developer.apple.com/documentation/videotoolbox/vtcopyvideoencoderlist(_:_:))). |
| Trial | Does the whole pipeline run end to end? | A short encode from `lavfi`. Tdarr's docs verify a Node this way, for example `ffmpeg -hwaccel qsv -f lavfi -i color=c=black:s=256x256:d=1:r=30 -c:v:0 hevc_qsv -f null /dev/null` ([Tdarr hardware transcoding](https://docs.tdarr.io/docs/installation/docker/hardware-transcoding)). |

### 2.2 What Jellyfin does (prior art)

`EncoderValidator.cs` ([jellyfin@bc753b1](https://github.com/jellyfin/jellyfin/blob/bc753b18ec5923ae1f3026cfe4ad51534521ab2a/MediaBrowser.MediaEncoding/Encoder/EncoderValidator.cs)) does the following:

- It parses `ffmpeg -version`, `-hwaccels`, `-decoders`, `-encoders` and `-filters` against required lists. The required filters include `tonemap_cuda`, `tonemap_opencl`, `tonemap_vaapi`, `tonemap_videotoolbox`, `tonemapx` and `libplacebo`.
- It checks filter options by grepping `ffmpeg -h filter=<name>`. One example is whether `tonemap_opencl` offers `bt2390`.
- It checks bitstream-filter options through `-h bsf=`, for example `dovi_rpu` `strip` and `hevc_metadata` `remove_dovi`.
- It identifies a VA-API render node by running `-v verbose -init_hw_device vaapi=va:<node>` and matching the driver name in stderr.
- It checks Vulkan by deriving `vulkan=vk@dr` from a DRM device and looking for required extension names.
- It sets 4.4 as the minimum ffmpeg version.

Per-codec hardware decode support is not auto-detected. The setup guides tell the admin to "uncheck the unsupported codecs" ([Jellyfin Intel](https://jellyfin.org/docs/general/post-install/transcoding/hardware-acceleration/intel)).

### 2.3 Detection pitfalls in the sources

- QSV's `-hwaccel qsv` "does not enable accelerated decoding" by itself. It enables zero-copy transcoding only when "both the decoder and the encoder must support QSV acceleration and no filters must be used" ([ffmpeg.texi L2135-2143](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/doc/ffmpeg.texi#L2135-L2143)).
- A mid-stream colourspace change makes ffmpeg rebuild the filter graph. With frames on the GPU this fails with `-38 (Function not implemented)` on QSV, NVENC and VAAPI. Unmanic's workaround is a "Safe decode mode" that round-trips frames through the CPU ([Unmanic guide](https://docs.unmanic.app/docs/guides/handling_colorspace_changes_hw_decode)).

## 3. HDR to SDR tone mapping

### 3.1 Upstream filters

| Filter | Runs on | Inputs | Algorithms and controls | Dolby Vision |
|---|---|---|---|---|
| `zscale` + `tonemap` | CPU (`tonemap` needs libzimg for `zscale`) | Linear-light float. The doc example is `zscale=transfer=linear,tonemap=clip,zscale=transfer=bt709,format=yuv420p` ([filters.texi L24616-24721](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/doc/filters.texi#L24616-L24721)). | `none, clip, linear, gamma, reinhard, hable, mobius`, plus `param`, `desat`, `peak`. | No. `vf_tonemap.c` has no DOVI reference. |
| `tonemap_opencl` | OpenCL GPU | "HDR(PQ/HLG)" | Same operators. Adds `threshold` scene detection. Output `p010` or `nv12` ([L28849-28944](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/doc/filters.texi#L28849-L28944)). | No. |
| `tonemap_vaapi` | VA-API VPP | "It currently only accepts HDR10 as input." Needs `VAProcFilterParameterBufferHDRToneMapping` ([L29190-29252](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/doc/filters.texi#L29190-L29252); [configure L4265](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/configure#L4265)). | Fixed-function. Can also do HDR-to-HDR through `display`. | No. |
| `vpp_qsv` `tonemap=1` | QSV VPP | Acts only "if the input has HDR metadata". It reads mastering display and CLL side data ([vf_vpp_qsv.c L419-476, L930](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/libavfilter/vf_vpp_qsv.c#L930)). | Fixed-function. | No. |
| `scale_vt` | VideoToolbox (`VTPixelTransferSession`) | The doc example is "Perform HDR to SDR conversion" with `color_matrix/primaries/transfer=bt709` ([L29506-29540](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/doc/filters.texi#L29506-L29540)). | No operator choice. | No. |
| `libplacebo` | Vulkan (CPU possible through Mesa llvmpipe) | Any tagged HDR. `peak_detect` is on by default. | `auto, clip, st2094-40, st2094-10, bt.2390, bt.2446a, spline, reinhard, mobius, hable, gamma, linear`, plus `gamut_mode` (default `perceptual`) and `inverse_tonemapping` ([L17510-17660](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/doc/filters.texi#L17510-L17660)). | Yes. `apply_dolbyvision` is on by default: "Apply Dolby Vision RPU metadata if present in source frames, and strip it from the output" (L17292-17296). |

`vpp_amf` lists `in_trc`/`out_trc` and colour profile options. The 9.0 `Changelog` says "Extend AMF Color Converter (vf_vpp_amf) HDR capabilities". Whether it tone maps PQ to SDR was not established from its doc text. That is a gap.

A grep of `FFmpeg@a35c879` finds `dovi` in `vf_libplacebo.c` only, among `vf_tonemap.c`, `vf_tonemap_opencl.c`, `vf_scale_vt.c`, `vf_vpp_qsv.c` and `vf_libplacebo.c`. Upstream has no `tonemap_cuda` filter. `libavfilter` contains only `vf_tonemap.c`, `vf_tonemap_opencl.c` and `vf_tonemap_vaapi.c` by that name.

### 3.2 Dolby Vision mechanics

- **Profiles and base layers.**
  - FFmpeg's DV encoder setup maps Profile 5 to "HEVC with proprietary IPTPQc2", with compatibility ID 0.
  - Profile 8 maps to compatibility ID 1 when the base layer is BT.2020+PQ (HDR10), 4 when it is BT.2020+HLG, and 2 when it is BT.709 SDR ([dovi_rpuenc.c L129-153](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/libavcodec/dovi_rpuenc.c#L129-L153)).
  - Dolby's TS spec says a zero BL compatibility ID decodes "to output a non-SDR and non-HDR signal that is not compliant with any standard" ([Dolby Vision in MPEG-2 TS v1.2, §1.1.1.4](https://professional.dolby.com/siteassets/pdfs/dolby-vision-bitstreams-in-mpeg-2-transport-stream-multiplex-v1.2.pdf)).
  - Dolby's support article says Profile 8.1 is cross-compatible with HDR10 and 8.4 with HLG ([Dolby, "What are Dolby Vision profiles and levels?"](https://professionalsupport.dolby.com/s/article/What-is-Dolby-Vision-Profile?language=en_US)). I read this from the search-result snippet because the page failed TLS verification on fetch.
- **Consequence (inferred).** A tone mapper that ignores the RPU produces correct-looking SDR from P8.1 and P8.4, because it maps the HDR10 or HLG base layer. On P5 it produces wrong colours, because the base layer is IPTPQc2. This matches Jellyfin listing "Dolby Vision P5" support as a feature of only some tone-mapping methods.
- **Decoding.** The HEVC decoder parses the RPU NAL and attaches `AV_FRAME_DATA_DOVI_METADATA` side data to frames ([hevcdec.c L3329-3336, L4002](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/libavcodec/hevc/hevcdec.c#L3329-L3336)). I infer that the `nvdec`/`vaapi`/`d3d11va` hwaccels keep this side data because they share that parser. The standalone `hevc_cuvid` and `hevc_qsv` decoders do not use it. Jellyfin requires its "enhanced NVDEC" toggle (NVDEC instead of CUVID) and the "native DXVA or VA-API" toggle (instead of QSV decoders) for DV ([Jellyfin NVIDIA](https://jellyfin.org/docs/general/post-install/transcoding/hardware-acceleration/nvidia), [Jellyfin Intel](https://jellyfin.org/docs/general/post-install/transcoding/hardware-acceleration/intel)).
- **Profile 7 (UHD Blu-ray dual layer).** The `dovi_split` bitstream filter (9.0) splits the base layer, the enhancement layer and the RPU. Its default mode `bl` emits "a plain HEVC stream with no Dolby Vision markers" ([bitstream_filters.texi L127-152](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/doc/bitstream_filters.texi#L127-L152)). `dovi_rpu` with `strip` removes DV metadata ([L104-125](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/doc/bitstream_filters.texi#L104-L125)).
- **Encoding DV.** `libx265` and `libsvtav1` expose `dolbyvision` ("Enable Dolby Vision RPU coding", auto by default) ([libx265.c L1055](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/libavcodec/libx265.c#L1055), [libsvtav1.c L832](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/libavcodec/libsvtav1.c#L832)). Profiles 4 and 7 fail with "Coding of Dolby Vision enhancement layers is currently unsupported" ([dovi_rpuenc.c L120-128](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/libavcodec/dovi_rpuenc.c#L120-L128)). `nvenc.c` and `videotoolboxenc.c` have zero `dovi` references, so hardware encodes drop DV.

### 3.3 Keeping HDR10 when a Rendition stays HDR

These encoders read `AV_FRAME_DATA_MASTERING_DISPLAY_METADATA` and write it out: `libx265.c`, `libsvtav1.c`, `nvenc.c` (HEVC and AV1), `qsvenc_hevc.c` and `vaapi_encode_h265.c`. A grep of `FFmpeg@a35c879` (for example [qsvenc_hevc.c L175](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/libavcodec/qsvenc_hevc.c#L175), [vaapi_encode_h265.c L535](https://github.com/FFmpeg/FFmpeg/blob/a35c8799920a93b99781eab6e70a5795ee7d182b/libavcodec/vaapi_encode_h265.c#L535)) found no such reference in `amfenc*.c` or `videotoolboxenc.c`. Whether those two keep static HDR10 metadata by another route was not established.

### 3.4 Jellyfin's fork (prior art, not upstream)

jellyfin-ffmpeg (latest tag `v8.1.3-1`, 2026-09-27) carries patches that add the following:

- `tonemap_cuda` with `apply_dovi` (default on) and a `bt2390` default ([0004](https://github.com/jellyfin/jellyfin-ffmpeg/blob/jellyfin/debian/patches/0004-add-cuda-tonemap-impl.patch)).
- A BT.2390 EETF and DV support in `tonemap_opencl` ([0007](https://github.com/jellyfin/jellyfin-ffmpeg/blob/jellyfin/debian/patches/0007-add-bt2390-eetf-and-code-refactor-to-opencl-tonemap.patch)).
- `tonemap_videotoolbox` on Metal (0046), a SIMD `tonemapx` CPU filter (0053) and a D3D11 tonemap (0094).
- Relaxing `vpp_qsv` tone mapping to work without HDR10 metadata (0070).
- HDR metadata for `hevc_nvenc` (0032).

Jellyfin's per-vendor picks are listed below.

| Vendor | Method | DV P5 |
|---|---|---|
| NVIDIA | CUDA only | Yes |
| Intel | OpenCL, or QSV VPP (Linux only) | OpenCL yes, VPP no |
| AMD | OpenCL, or Vulkan/libplacebo (Linux only) | Yes for both |
| Apple | Metal, or VideoToolbox native | Metal yes, VT native no |

Jellyfin added DV P5 and P8 to SDR in 10.8 ([Jellyfin hardware acceleration](https://jellyfin.org/docs/general/post-install/transcoding/hardware-acceleration/)).

## 4. How Tdarr and Unmanic schedule across machines

### 4.1 Tdarr

- **Licence.** Tdarr is a proprietary EULA with Free, Personal and Business tiers, and it bars reverse engineering ([LICENSE.md](https://github.com/HaveAGitGat/Tdarr/blob/master/LICENSE.md)). Community plugins are GPL-3.0 ([Tdarr_Plugins](https://github.com/HaveAGitGat/Tdarr_Plugins)).
- **Topology.** The Server "Does not do any encoding". Nodes connect to it, on the same machine or on others ([Getting started](https://docs.tdarr.io/docs/installation/getting-started)). Nodes poll the Server every `pollInterval` (default 2000 ms) and carry a `priority` (-1 none, 0 highest) ([Variables](https://docs.tdarr.io/docs/installation/variables)).
- **Workers.** Each Node has four worker types: Transcode CPU, Transcode GPU, Health Check CPU and Health Check GPU.
  - A transcode worker classifies a job by scanning its FFmpeg/HandBrake arguments "for GPU terms such as `nvenc`, `cuda`, `vaapi`". It does not probe the hardware ([Workers](https://docs.tdarr.io/docs/nodes/workers)).
  - "Allow GPU workers to do CPU tasks" is a per-Node toggle. Counts come from the UI or the `transcodegpuWorkers` and `transcodecpuWorkers` environment variables.
  - The FAQ suggests about 3 workers. On a mixed fleet, the advice is to launch GPU workers only on the GPU Node ([FAQ](https://docs.tdarr.io/docs/faq)).
- **Routing.** Node tags plus the "Worker Type" and "Requeue" flow plugins pin flow steps to particular Nodes ([Tags](https://docs.tdarr.io/docs/nodes/tags)). Libraries carry processing schedules. Workers idle outside them ([Workers](https://docs.tdarr.io/docs/nodes/workers)).
- **Files.**
  - "Mapped" Nodes must share the filesystem or use `pathTranslators`. The FAQ explains this choice as network efficiency.
  - "Unmapped" Nodes (2.27.01) download and upload working files themselves. Without Pro they are capped at 10 MB for video. They cannot run filesystem steps such as "Replace Original File", so a flow hands back to a mapped Node ([Nodes](https://docs.tdarr.io/docs/nodes/nodes)).
  - The transcode cache is required. Output is copied back over the original ([Transcode cache](https://docs.tdarr.io/docs/library-setup/transcode-cache)).
- **HDR.** A GitHub code search of the `HaveAGitGat` org for `tonemap` returned 0 hits (search index, default branches only).

### 4.2 Unmanic

- **Licence.** Unmanic is GPL-3.0 Python ([Unmanic/unmanic](https://github.com/Unmanic/unmanic)). Supporter status unlocks "Unlimited linked installations" and "Unlimited libraries" ([Supporter features](https://docs.unmanic.app/docs/using_unmanic/supporter_features)). The code compares the link count against `session.link_count` ([installation_link.py L858-892](https://github.com/Unmanic/unmanic/blob/1c324b8fc3974ffce3d7cc945adb938fe7182910/unmanic/libs/installation_link.py#L858-L892)).
- **Topology.** Installations are peers. A link is configured per direction ("Send tasks", "Receive tasks"), so there is no fixed server ([Remote configuration](https://docs.unmanic.app/docs/configuration/linking/link_remote_configuration)).
- **Local scheduling.**
  - Worker groups have 0 to 12 workers each. Tags route work. A tagged group takes tasks only from libraries sharing a tag. An untagged group takes only untagged libraries ([Workers settings](https://docs.unmanic.app/docs/configuration/workers_settings); [taskqueue.py L86-93](https://github.com/Unmanic/unmanic/blob/1c324b8fc3974ffce3d7cc945adb938fe7182910/unmanic/libs/taskqueue.py#L86-L93)).
  - Each group has a timed event schedule (set count, pause, resume).
- **Cross-machine routing.**
  - "Linked task routing is based on library names and link configuration, not on worker capability tags across installations." The remote then applies "its own library settings, tags, and plugins" ([Link overview](https://docs.unmanic.app/docs/configuration/linking/link_overview)).
  - The documented mixed-GPU pattern (VideoToolbox Mac, NVENC server, QSV NUC) gives the same library name a different encoder configuration on each install. Hardware capability is configured by hand, per install.
  - A remote is eligible when it is available, has at least one worker, and is under its pending-task preload target (default 2) ([installation_link.py L759-856](https://github.com/Unmanic/unmanic/blob/1c324b8fc3974ffce3d7cc945adb938fe7182910/unmanic/libs/installation_link.py#L759-L856)).
  - "Distributed worker count" shares one worker target across linked installs.
- **Files.**
  - Unmanic first tries to create the remote task by relative path under the remote library path, which means shared storage. If "path does not exist", it falls back to an HTTP upload, with optional checksum validation.
  - Uploads are limited by network transfer locks: one per remote by default, capped at 5. Files under 100 MB transfer in parallel (installation_link.py L136-162 and `__send_task_to_remote_worker_and_monitor`).
  - Results come back by a move on shared storage when it is writable, or by HTTP download ([Link overview](https://docs.unmanic.app/docs/configuration/linking/link_overview)).
- **HDR.** The Video Transcoder plugin preserves HDR10 colour tags when the source is HDR ([plugin.video_transcoder lib/encoders/base.py](https://github.com/Unmanic/plugin.video_transcoder/blob/master/lib/encoders/base.py)). A GitHub code search of the `Unmanic` org for `tonemap` returned no hits.

## 5. Options for the decision (no pick made)

**A. Which ffmpeg a Node runs.**

1. Upstream FFmpeg (static or distro builds). DV-correct tone mapping exists only through `libplacebo`, which needs a working Vulkan driver.
2. jellyfin-ffmpeg. It adds DV-aware CUDA, OpenCL and Metal tone mappers and `tonemapx`. The cost is a fork tied to Jellyfin's release cadence, and patches that are not upstream.
3. Use whatever is on the system. This is the least packaging work. Capability then varies by distro, with GPL vs LGPL, Full-Feature vs Free-Kernel Intel driver, and libplacebo present or absent.

**B. Tone-mapping engine.**

1. `libplacebo` everywhere. It is one code path, applies DV RPU and has the most operators. It needs Vulkan on every transcoding Node. The CPU fallback through llvmpipe is documented but slow by design (inferred).
2. Per-vendor filters (`tonemap_opencl`, `tonemap_vaapi`, `vpp_qsv`, `scale_vt`). These are fast on their own hardware, but no upstream one handles DV P5.
3. Software `zscale` + `tonemap` as the universal fallback. It needs no GPU, does not handle DV, and is the slowest (inferred).

**C. How a Node reports capability to the Controller.**

1. Build-and-device probe as Jellyfin does it, plus admin overrides per codec.
2. Vendor caps APIs (`vainfo`-style VA-API queries, `NvEncGetEncodeCaps`, `VTCopyVideoEncoderList`) for a codec and profile matrix.
3. A trial encode per Rendition recipe. This is the only layer that proves the whole filter graph, including tone mapping. It costs seconds per probe (inferred).

These layers stack. The map's offline mode (Q17) would need a fake capability source for CI. That is inferred, and no source covers it.

**D. Cross-Node scheduling shape.**

1. Tdarr-like: a central scheduler with typed workers per Node, pulled by Node polling. Capability is inferred from the command, not probed.
2. Unmanic-like: peer routing by a shared name, with each Node owning its own encoder recipe.
3. A capability-matched scheduler. The Controller matches a Rendition recipe's needs (codec, bit depth, tone mapper, DV) against each Node's probed capability record. Neither tool does this. Both leave capability to admin configuration.

The map already fixes file movement as a one-way push to the importing Node (Q11). Both tools prefer shared storage and fall back to HTTP transfer, and Tdarr gates that fallback behind Pro.

**E. DV Renditions.**

- A derived SDR Rendition from a P5 source needs a DV-aware tone mapper (`libplacebo`, or a fork's filter).
- A derived HDR Rendition from DV keeps DV only through software `libx265` or `libsvtav1`. Hardware encoders emit HDR10 at best, using the P8 base layer, and drop DV.
- P7 sources need `dovi_split` (or just the HEVC base layer) before any re-encode.

## Gaps

- Dolby's Profiles and Levels spec was not read directly. The profile-to-compatibility facts come from FFmpeg source, Dolby's TS spec and a Dolby support snippet.
- Whether `vpp_amf` (9.0) tone maps PQ to SDR, and whether AMF and VideoToolbox encoders keep HDR10 metadata by a route other than frame side data, is unverified.
- No ffmpeg binary was available on the research machine, so nothing above was run. Every claim comes from docs or source.
