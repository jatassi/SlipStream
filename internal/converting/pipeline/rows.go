package pipeline

import (
	"fmt"
	"strconv"

	"github.com/jatassi/slipstream/internal/encode"
)

// Rows are every pipeline in proof order: full hardware, then a hardware
// encode with OpenCL tone mapping, then the processor.
var Rows = []Row{
	qsvWindows, qsvLinux, vaapi, cuda, videoToolbox,
	openCL(qsvWindows, "qsv-windows-opencl"), openCL(vaapi, "vaapi-opencl"), cudaOpenCL,
	processor,
}

// qsvWindows is the recorded run's chain: D3D11VA decode, OpenCL tone
// mapping mapped back to Quick Sync, and a Quick Sync encode. Deriving
// back through D3D11VA fails, so it maps straight to Quick Sync.
var qsvWindows = Row{
	Name:     "qsv-windows",
	OS:       []string{"windows"},
	Pipeline: encode.Pipeline{Path: encode.Hardware, API: encode.QSV},
	Init: func(string) []string {
		return []string{
			"-init_hw_device", "d3d11va=dx11:,vendor=0x8086", "-init_hw_device", "qsv=qs@dx11",
			"-init_hw_device", "opencl=ocl@dx11", "-filter_hw_device", "ocl",
		}
	},
	Decode:  hwDecode("-hwaccel", "d3d11va", "-hwaccel_output_format", "d3d11"),
	Filter:  qsvFilter,
	Encoder: map[encode.Codec]string{encode.H264: "h264_qsv", encode.HEVC: "hevc_qsv", encode.AV1: "av1_qsv"},
	Rate: func(_ encode.Codec, kbps int) []string {
		return append([]string{"-preset", "veryfast"}, bitrate(kbps)...)
	},
}

// qsvLinux is Quick Sync over a VA-API render node; Device is its path.
var qsvLinux = Row{
	Name:     "qsv-linux",
	OS:       []string{"linux"},
	Pipeline: encode.Pipeline{Path: encode.Hardware, API: encode.QSV},
	Init: func(device string) []string {
		return []string{
			"-init_hw_device", "vaapi=va:" + device, "-init_hw_device", "qsv=qs@va",
			"-init_hw_device", "opencl=ocl@va", "-filter_hw_device", "ocl",
		}
	},
	Decode:  hwDecode("-hwaccel", "vaapi", "-hwaccel_output_format", "vaapi", "-hwaccel_device", "va"),
	Filter:  qsvFilter,
	Encoder: qsvWindows.Encoder,
	Rate:    qsvWindows.Rate,
}

func qsvFilter(n Need) string {
	switch {
	case !n.OnDevice:
		return openCLChain(n)
	case n.ToneMap:
		return fmt.Sprintf("hwmap=derive_device=opencl,%s,hwmap=derive_device=qsv:reverse=1,scale_qsv=w=%d:h=%d",
			ToneMap(n, "tonemap_opencl", deviceFormat(n)), n.W, n.H)
	}
	return fmt.Sprintf("hwmap=derive_device=qsv,format=qsv,scale_qsv=w=%d:h=%d:format=%s", n.W, n.H, deviceFormat(n))
}

// vaapi is VA-API on a render node; Device is its path.
var vaapi = Row{
	Name:     "vaapi",
	OS:       []string{"linux"},
	Pipeline: encode.Pipeline{Path: encode.Hardware, API: encode.VAAPI},
	Init: func(device string) []string {
		return []string{"-init_hw_device", "vaapi=va:" + device, "-init_hw_device", "opencl=ocl@va", "-filter_hw_device", "ocl"}
	},
	Decode: hwDecode("-hwaccel", "vaapi", "-hwaccel_output_format", "vaapi", "-hwaccel_device", "va"),
	Filter: func(n Need) string {
		switch {
		case !n.OnDevice:
			return openCLChain(n) + ",hwupload=derive_device=vaapi"
		case n.ToneMap:
			return fmt.Sprintf("hwmap=derive_device=opencl,%s,hwmap=derive_device=vaapi:reverse=1,scale_vaapi=w=%d:h=%d",
				ToneMap(n, "tonemap_opencl", deviceFormat(n)), n.W, n.H)
		}
		return fmt.Sprintf("scale_vaapi=w=%d:h=%d:format=%s", n.W, n.H, deviceFormat(n))
	},
	Encoder: map[encode.Codec]string{encode.H264: "h264_vaapi", encode.HEVC: "hevc_vaapi", encode.AV1: "av1_vaapi"},
	Rate:    func(_ encode.Codec, kbps int) []string { return append([]string{"-rc_mode", "VBR"}, bitrate(kbps)...) },
}

var nvenc = map[encode.Codec]string{encode.H264: "h264_nvenc", encode.HEVC: "hevc_nvenc", encode.AV1: "av1_nvenc"}

func nvencRate(_ encode.Codec, kbps int) []string {
	return append([]string{"-preset", "p4", "-rc", "vbr"}, bitrate(kbps)...)
}

// cuda is NVIDIA's decoder, CUDA tone mapping and NVENC; Device is the
// GPU's index.
var cuda = Row{
	Name:     "cuda",
	OS:       []string{"windows", "linux"},
	Pipeline: encode.Pipeline{Path: encode.Hardware, API: encode.CUDA},
	Init: func(device string) []string {
		return []string{"-init_hw_device", "cuda=cu:" + device, "-filter_hw_device", "cu"}
	},
	Decode: hwDecode("-hwaccel", "cuda", "-hwaccel_output_format", "cuda", "-hwaccel_device", "cu"),
	Filter: func(n Need) string {
		switch {
		case n.OnDevice && n.ToneMap:
			return fmt.Sprintf("scale_cuda=w=%d:h=%d,%s", n.W, n.H, ToneMap(n, "tonemap_cuda", deviceFormat(n)))
		case n.OnDevice:
			return fmt.Sprintf("scale_cuda=w=%d:h=%d:format=%s", n.W, n.H, deviceFormat(n))
		case n.ToneMap:
			return fmt.Sprintf("scale=w=%d:h=%d,format=p010,hwupload,%s", n.W, n.H, ToneMap(n, "tonemap_cuda", deviceFormat(n)))
		}
		return fmt.Sprintf("scale=w=%d:h=%d,format=%s", n.W, n.H, deviceFormat(n))
	},
	Encoder: nvenc,
	Rate:    nvencRate,
}

var videoToolbox = Row{
	Name:     "videotoolbox",
	OS:       []string{"darwin"},
	Pipeline: encode.Pipeline{Path: encode.Hardware, API: encode.VideoToolbox},
	Init: func(string) []string {
		return []string{"-init_hw_device", "videotoolbox=vt", "-filter_hw_device", "vt"}
	},
	Decode: hwDecode("-hwaccel", "videotoolbox", "-hwaccel_output_format", "videotoolbox_vld"),
	Filter: func(n Need) string {
		switch {
		case n.OnDevice && n.ToneMap:
			return fmt.Sprintf("scale_vt=w=%d:h=%d,%s", n.W, n.H, ToneMap(n, "tonemap_videotoolbox", deviceFormat(n)))
		case n.OnDevice:
			return fmt.Sprintf("scale_vt=w=%d:h=%d", n.W, n.H)
		case n.ToneMap:
			return fmt.Sprintf("scale=w=%d:h=%d,format=p010,hwupload,%s", n.W, n.H, ToneMap(n, "tonemap_videotoolbox", deviceFormat(n)))
		}
		return fmt.Sprintf("scale=w=%d:h=%d,format=%s", n.W, n.H, deviceFormat(n))
	},
	Encoder: map[encode.Codec]string{encode.H264: "h264_videotoolbox", encode.HEVC: "hevc_videotoolbox"},
	Rate:    func(_ encode.Codec, kbps int) []string { return bitrate(kbps) },
}

// openCL is r decoding on the processor and tone mapping with OpenCL, for
// a device whose own decoder or tone mapper fails its test run.
func openCL(r Row, name string) Row {
	r.Name = name
	r.Pipeline.Path = encode.HardwareOpenCL
	r.Decode = nil
	return r
}

// cudaOpenCL is NVENC with OpenCL tone mapping; NVENC takes the frames
// from memory.
var cudaOpenCL = Row{
	Name:     "cuda-opencl",
	OS:       []string{"windows", "linux"},
	Pipeline: encode.Pipeline{Path: encode.HardwareOpenCL, API: encode.CUDA},
	Init: func(string) []string {
		return []string{"-init_hw_device", "opencl=ocl", "-filter_hw_device", "ocl"}
	},
	Filter:  openCLChain,
	Encoder: nvenc,
	Rate:    nvencRate,
}

// processor tone maps with jellyfin-ffmpeg's tonemapx, which applies Dolby
// Vision metadata, before it scales: scale can't read a Profile 5 picture.
var processor = Row{
	Name:     "processor",
	OS:       []string{"windows", "linux", "darwin"},
	Pipeline: encode.Pipeline{Path: encode.Processor},
	Init:     func(string) []string { return nil },
	Filter: func(n Need) string {
		format := "yuv420p"
		if n.Keep10 {
			format = "yuv420p10le"
		}
		if n.ToneMap {
			return fmt.Sprintf("%s,scale=w=%d:h=%d", ToneMap(n, "tonemapx", format), n.W, n.H)
		}
		return fmt.Sprintf("scale=w=%d:h=%d,format=%s", n.W, n.H, format)
	},
	Encoder: map[encode.Codec]string{encode.H264: "libx264", encode.HEVC: "libx265", encode.AV1: "libsvtav1"},
	Rate: func(c encode.Codec, kbps int) []string {
		if c == encode.AV1 {
			return []string{"-preset", "8", "-b:v", kilo(kbps)}
		}
		return append([]string{"-preset", "veryfast"}, bitrate(kbps)...)
	},
	DoVi: true,
}

// openCLChain scales and tone maps frames decoded on the processor, on the
// OpenCL filter device, and hands them back in memory.
func openCLChain(n Need) string {
	if n.ToneMap {
		return fmt.Sprintf("scale=w=%d:h=%d,format=p010,hwupload,%s,hwdownload,format=%s",
			n.W, n.H, ToneMap(n, "tonemap_opencl", deviceFormat(n)), deviceFormat(n))
	}
	return fmt.Sprintf("scale=w=%d:h=%d,format=%s", n.W, n.H, deviceFormat(n))
}

func hwDecode(args ...string) func(encode.Class) []string {
	return func(c encode.Class) []string {
		if c == encode.ClassDecodeSDR || c == encode.ClassDecodeHDR10 {
			return nil
		}
		return args
	}
}

func deviceFormat(n Need) string {
	if n.Keep10 {
		return "p010"
	}
	return "nv12"
}

// bitrate is the target with a cap of one and a half times it, over a
// buffer of two seconds' target.
func bitrate(kbps int) []string {
	return []string{"-b:v", kilo(kbps), "-maxrate", kilo(kbps * 3 / 2), "-bufsize", kilo(kbps * 2)}
}

func kilo(kbps int) string { return strconv.Itoa(kbps) + "k" }
