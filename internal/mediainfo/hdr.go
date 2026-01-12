package mediainfo

import "strings"

// HDRInfo contains parsed HDR information.
type HDRInfo struct {
	BitDepth       int
	ColorPrimaries string
	TransferFunc   string
	MatrixCoeffs   string
	HDRFormat      string
	HasDolbyVision bool
	HasHDR10       bool
	HasHDR10Plus   bool
	HasHLG         bool
}

// DetectHDRType determines the HDR type from video stream properties.
func DetectHDRType(info HDRInfo) (dynamicRange, dynamicRangeType string) {
	// Detect specific HDR formats first
	hasDV := info.HasDolbyVision || containsDolbyVision(info.HDRFormat)
	hasHDR10 := info.HasHDR10 || containsHDR10(info.TransferFunc, info.ColorPrimaries)
	hasHDR10Plus := info.HasHDR10Plus || containsHDR10Plus(info.HDRFormat)
	hasHLG := info.HasHLG || containsHLG(info.TransferFunc)

	// Determine combined type
	switch {
	case hasDV && hasHDR10:
		return "HDR", string(HDRTypeDVHDR10)
	case hasDV:
		return "HDR", string(HDRTypeDolbyVision)
	case hasHDR10Plus:
		return "HDR", string(HDRTypeHDR10Plus)
	case hasHDR10:
		return "HDR", string(HDRTypeHDR10)
	case hasHLG:
		return "HDR", string(HDRTypeHLG)
	case info.BitDepth >= 10 && isHDRColorSpace(info.ColorPrimaries, info.TransferFunc):
		return "HDR", string(HDRTypeGenericHDR)
	default:
		return "", ""
	}
}

// containsDolbyVision checks if the HDR format indicates Dolby Vision.
func containsDolbyVision(format string) bool {
	lower := strings.ToLower(format)
	return strings.Contains(lower, "dolby vision") ||
		strings.Contains(lower, "dv") ||
		strings.Contains(lower, "dvhe") ||
		strings.Contains(lower, "dvh1")
}

// containsHDR10 checks if transfer/color properties indicate HDR10.
func containsHDR10(transfer, primaries string) bool {
	lower := strings.ToLower(transfer + " " + primaries)
	// HDR10 uses PQ (SMPTE 2084) transfer + BT.2020 color space
	hasPQ := strings.Contains(lower, "smpte st 2084") ||
		strings.Contains(lower, "pq") ||
		strings.Contains(lower, "st2084")
	hasBT2020 := strings.Contains(lower, "bt.2020") ||
		strings.Contains(lower, "bt2020") ||
		strings.Contains(lower, "rec.2020")
	return hasPQ && hasBT2020
}

// containsHDR10Plus checks if the HDR format indicates HDR10+.
func containsHDR10Plus(format string) bool {
	lower := strings.ToLower(format)
	return strings.Contains(lower, "hdr10+") ||
		strings.Contains(lower, "hdr10plus") ||
		strings.Contains(lower, "smpte st 2094")
}

// containsHLG checks if the transfer function indicates HLG.
func containsHLG(transfer string) bool {
	lower := strings.ToLower(transfer)
	return strings.Contains(lower, "hlg") ||
		strings.Contains(lower, "hybrid log-gamma") ||
		strings.Contains(lower, "arib std-b67")
}

// isHDRColorSpace checks if the color properties indicate an HDR color space.
func isHDRColorSpace(primaries, transfer string) bool {
	lower := strings.ToLower(primaries + " " + transfer)
	return strings.Contains(lower, "bt.2020") ||
		strings.Contains(lower, "bt2020") ||
		strings.Contains(lower, "rec.2020")
}

// FormatHDRSimple returns a simple HDR indicator.
func FormatHDRSimple(dynamicRange string) string {
	if dynamicRange != "" {
		return "HDR"
	}
	return ""
}
