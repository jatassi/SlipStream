package quality

import "strings"

type ExclusivityResult struct {
	AreExclusive     bool     `json:"areExclusive"`
	ConflictingAttrs []string `json:"conflictingAttrs,omitempty"`
	OverlappingAttrs []string `json:"overlappingAttrs,omitempty"`
	Reason           string   `json:"reason,omitempty"`
}

func CheckMutualExclusivity(profileA, profileB *Profile) ExclusivityResult {
	result := ExclusivityResult{
		AreExclusive:     false,
		ConflictingAttrs: []string{},
		OverlappingAttrs: []string{},
	}

	conflicts := findConflictingRequiredAttributes(profileA, profileB)
	if len(conflicts) > 0 {
		result.AreExclusive = true
		result.ConflictingAttrs = conflicts
		return result
	}

	if haveNonOverlappingQualities(profileA, profileB) {
		result.AreExclusive = true
		result.Reason = "profiles have different allowed quality tiers"
		return result
	}

	overlaps := findOverlappingAttributes(profileA, profileB)
	result.OverlappingAttrs = overlaps
	if len(overlaps) > 0 {
		result.Reason = "profiles have overlapping requirements and could match the same releases"
	}

	return result
}

func haveNonOverlappingQualities(profileA, profileB *Profile) bool {
	allowedA := getAllowedQualityIDs(profileA)
	allowedB := getAllowedQualityIDs(profileB)

	if len(allowedA) == 0 || len(allowedB) == 0 {
		return false
	}

	for id := range allowedA {
		if !allowedB[id] {
			return true
		}
	}

	for id := range allowedB {
		if !allowedA[id] {
			return true
		}
	}

	return false
}

func getAllowedQualityIDs(profile *Profile) map[int]bool {
	allowed := make(map[int]bool)
	for _, item := range profile.Items {
		if item.Allowed {
			allowed[item.Quality.ID] = true
		}
	}
	return allowed
}

func findConflictingRequiredAttributes(a, b *Profile) []string {
	var conflicts []string

	if HasAttributeConflict(a.HDRSettings, b.HDRSettings) {
		conflicts = append(conflicts, "HDR")
	}
	if HasAttributeConflict(a.VideoCodecSettings, b.VideoCodecSettings) {
		conflicts = append(conflicts, "Video Codec")
	}
	if HasAttributeConflict(a.AudioCodecSettings, b.AudioCodecSettings) {
		conflicts = append(conflicts, "Audio Codec")
	}
	if HasAttributeConflict(a.AudioChannelSettings, b.AudioChannelSettings) {
		conflicts = append(conflicts, "Audio Channels")
	}

	return conflicts
}

func HasAttributeConflict(settingsA, settingsB AttributeSettings) bool {
	requiredA := settingsA.GetRequired()
	requiredB := settingsB.GetRequired()
	notAllowedA := settingsA.GetNotAllowed()
	notAllowedB := settingsB.GetNotAllowed()

	for _, req := range requiredA {
		for _, notAllowed := range notAllowedB {
			if req == notAllowed {
				return true
			}
		}
	}

	for _, req := range requiredB {
		for _, notAllowed := range notAllowedA {
			if req == notAllowed {
				return true
			}
		}
	}

	if len(requiredA) > 0 && len(requiredB) > 0 {
		return !hasOverlap(requiredA, requiredB)
	}

	return false
}

func findOverlappingAttributes(a, b *Profile) []string {
	var overlaps []string

	if hasAttributeOverlap(a.HDRSettings, b.HDRSettings) {
		overlaps = append(overlaps, "HDR")
	}
	if hasAttributeOverlap(a.VideoCodecSettings, b.VideoCodecSettings) {
		overlaps = append(overlaps, "Video Codec")
	}
	if hasAttributeOverlap(a.AudioCodecSettings, b.AudioCodecSettings) {
		overlaps = append(overlaps, "Audio Codec")
	}
	if hasAttributeOverlap(a.AudioChannelSettings, b.AudioChannelSettings) {
		overlaps = append(overlaps, "Audio Channels")
	}

	return overlaps
}

func hasAttributeOverlap(settingsA, settingsB AttributeSettings) bool {
	if !settingsA.HasNonDefaultSettings() || !settingsB.HasNonDefaultSettings() {
		return true
	}

	requiredA := settingsA.GetRequired()
	requiredB := settingsB.GetRequired()

	if len(requiredA) == 0 && len(requiredB) == 0 {
		return true
	}

	if len(requiredA) > 0 && len(requiredB) > 0 {
		return hasOverlap(requiredA, requiredB)
	}

	return true
}

func hasOverlap(a, b []string) bool {
	setA := make(map[string]bool, len(a))
	for _, v := range a {
		setA[v] = true
	}
	for _, v := range b {
		if setA[v] {
			return true
		}
	}
	return false
}

type SlotConfig struct {
	SlotNumber int
	SlotName   string
	Enabled    bool
	Profile    *Profile
}

type AttributeIssue struct {
	Attribute string `json:"attribute"`
	Message   string `json:"message"`
}

type SlotExclusivityError struct {
	SlotA           int              `json:"slotA"`
	SlotB           int              `json:"slotB"`
	SlotAName       string           `json:"slotAName"`
	SlotBName       string           `json:"slotBName"`
	ProfileAName    string           `json:"profileAName"`
	ProfileBName    string           `json:"profileBName"`
	OverlappingAttr string           `json:"overlappingAttr,omitempty"`
	Reason          string           `json:"reason"`
	Issues          []AttributeIssue `json:"issues,omitempty"`
}

func ValidateSlotExclusivity(slots []SlotConfig) ([]SlotExclusivityError, bool) {
	var errors []SlotExclusivityError

	for i := 0; i < len(slots); i++ {
		if !isSlotEnabledWithProfile(slots[i]) {
			continue
		}

		for j := i + 1; j < len(slots); j++ {
			if !isSlotEnabledWithProfile(slots[j]) {
				continue
			}

			if err := checkSlotPairExclusivity(slots[i], slots[j]); err != nil {
				errors = append(errors, *err)
			}
		}
	}

	return errors, len(errors) == 0
}

func isSlotEnabledWithProfile(slot SlotConfig) bool {
	return slot.Enabled && slot.Profile != nil
}

func checkSlotPairExclusivity(slotA, slotB SlotConfig) *SlotExclusivityError {
	result := CheckMutualExclusivity(slotA.Profile, slotB.Profile)
	if result.AreExclusive {
		return nil
	}

	reason, issues := buildExclusivityErrorDetails(slotA, slotB, result)
	err := &SlotExclusivityError{
		SlotA:        slotA.SlotNumber,
		SlotB:        slotB.SlotNumber,
		SlotAName:    slotA.SlotName,
		SlotBName:    slotB.SlotName,
		ProfileAName: slotA.Profile.Name,
		ProfileBName: slotB.Profile.Name,
		Reason:       reason,
		Issues:       issues,
	}

	if len(result.OverlappingAttrs) > 0 {
		err.OverlappingAttr = result.OverlappingAttrs[0]
	}

	return err
}

func buildExclusivityErrorDetails(slotA, slotB SlotConfig, result ExclusivityResult) (string, []AttributeIssue) {
	if len(result.OverlappingAttrs) == 0 {
		return "profiles have no differentiating required attributes", nil
	}

	var issues []AttributeIssue
	for _, attr := range result.OverlappingAttrs {
		message := buildAttributeOverlapDetail(attr, slotA.Profile, slotB.Profile)
		if message != "" {
			issues = append(issues, AttributeIssue{
				Attribute: attr,
				Message:   message,
			})
		}
	}

	if len(issues) == 0 {
		return "profiles could match the same releases", nil
	}

	var details []string
	for _, issue := range issues {
		details = append(details, issue.Message)
	}
	return strings.Join(details, "; "), issues
}

func buildAttributeOverlapDetail(attr string, profileA, profileB *Profile) string {
	settingsA, settingsB := getAttributeSettings(attr, profileA, profileB)
	if settingsA == nil || settingsB == nil {
		return ""
	}

	reqA := settingsA.GetRequired()
	reqB := settingsB.GetRequired()

	if len(reqA) == 0 && len(reqB) == 0 {
		return "Neither profile has required values (both accept any)"
	}

	if len(reqA) > 0 && len(reqB) > 0 {
		return buildBothRequiredMessage(reqA, reqB)
	}

	return buildOneRequiredMessage(reqA, reqB, profileA.Name, profileB.Name)
}

func getAttributeSettings(attr string, profileA, profileB *Profile) (settingsA, settingsB *AttributeSettings) {
	switch attr {
	case "HDR":
		return &profileA.HDRSettings, &profileB.HDRSettings
	case "Video Codec":
		return &profileA.VideoCodecSettings, &profileB.VideoCodecSettings
	case "Audio Codec":
		return &profileA.AudioCodecSettings, &profileB.AudioCodecSettings
	case "Audio Channels":
		return &profileA.AudioChannelSettings, &profileB.AudioChannelSettings
	default:
		return nil, nil
	}
}

func buildBothRequiredMessage(reqA, reqB []string) string {
	overlapping := findOverlappingValues(reqA, reqB)
	if len(overlapping) > 0 {
		return "Both profiles require " + strings.Join(overlapping, ", ")
	}
	return ""
}

func buildOneRequiredMessage(reqA, reqB []string, nameA, nameB string) string {
	if len(reqA) > 0 {
		return nameA + " requires " + strings.Join(reqA, ", ") + "; " + nameB + " accepts any"
	}
	if len(reqB) > 0 {
		return nameB + " requires " + strings.Join(reqB, ", ") + "; " + nameA + " accepts any"
	}
	return ""
}

func findOverlappingValues(a, b []string) []string {
	var overlapping []string
	setB := make(map[string]bool, len(b))
	for _, v := range b {
		setB[v] = true
	}
	for _, v := range a {
		if setB[v] {
			overlapping = append(overlapping, v)
		}
	}
	return overlapping
}

func GetProfileExclusivityHints(profileA, profileB *Profile) []string {
	var hints []string

	result := CheckMutualExclusivity(profileA, profileB)
	if result.AreExclusive {
		return hints
	}

	if containsString(result.OverlappingAttrs, "HDR") {
		hints = append(hints, "Set one profile to require HDR formats (e.g., DV, HDR10) and the other to require SDR")
	}
	if containsString(result.OverlappingAttrs, "Video Codec") {
		hints = append(hints, "Set one profile to require different video codecs (e.g., one requires x265, other requires x264)")
	}
	if containsString(result.OverlappingAttrs, "Audio Codec") {
		hints = append(hints, "Set one profile to require lossless audio (e.g., TrueHD) and other to require lossy (e.g., DD)")
	}
	if containsString(result.OverlappingAttrs, "Audio Channels") {
		hints = append(hints, "Set one profile to require surround (7.1, 5.1) and other to require stereo (2.0)")
	}

	if len(hints) == 0 {
		hints = append(hints, "Ensure at least one attribute has conflicting required values between profiles")
	}

	return hints
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
