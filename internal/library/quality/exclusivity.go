package quality

import "strings"

// Mutual Exclusivity checking for quality profiles
// Implements Requirements 3.1.1-3.1.4 from the Multiple Quality Versions spec

// ExclusivityResult contains details about a profile exclusivity check
type ExclusivityResult struct {
	AreExclusive       bool     `json:"areExclusive"`
	ConflictingAttrs   []string `json:"conflictingAttrs,omitempty"`
	OverlappingAttrs   []string `json:"overlappingAttrs,omitempty"`
	Reason             string   `json:"reason,omitempty"`
}

// CheckMutualExclusivity determines if two profiles are mutually exclusive.
// Req 3.1.1: Two profiles are mutually exclusive if their required attributes conflict.
// Req 3.1.3: Preferred attributes do not affect exclusivity calculation.
// Also checks quality tier differences - profiles with non-overlapping allowed qualities are exclusive.
func CheckMutualExclusivity(profileA, profileB *Profile) ExclusivityResult {
	result := ExclusivityResult{
		AreExclusive:     false,
		ConflictingAttrs: []string{},
		OverlappingAttrs: []string{},
	}

	// Check 1: Conflicting attribute requirements (required vs notAllowed)
	conflicts := findConflictingRequiredAttributes(profileA, profileB)
	if len(conflicts) > 0 {
		result.AreExclusive = true
		result.ConflictingAttrs = conflicts
		return result
	}

	// Check 2: Non-overlapping allowed quality tiers
	// If profiles have different allowed qualities, they're exclusive for those quality levels
	if haveNonOverlappingQualities(profileA, profileB) {
		result.AreExclusive = true
		result.Reason = "profiles have different allowed quality tiers"
		return result
	}

	// Check 3: Attribute overlaps (both could match same releases)
	overlaps := findOverlappingAttributes(profileA, profileB)
	result.OverlappingAttrs = overlaps
	if len(overlaps) > 0 {
		result.Reason = "profiles have overlapping requirements and could match the same releases"
	}

	return result
}

// haveNonOverlappingQualities checks if profiles have any allowed qualities that don't overlap.
// Returns true if at least one profile allows a quality the other doesn't.
func haveNonOverlappingQualities(profileA, profileB *Profile) bool {
	allowedA := getAllowedQualityIDs(profileA)
	allowedB := getAllowedQualityIDs(profileB)

	// If either has no allowed qualities, they can't be differentiated by quality
	if len(allowedA) == 0 || len(allowedB) == 0 {
		return false
	}

	// Check if A has any quality not in B
	for id := range allowedA {
		if !allowedB[id] {
			return true
		}
	}

	// Check if B has any quality not in A
	for id := range allowedB {
		if !allowedA[id] {
			return true
		}
	}

	return false
}

// getAllowedQualityIDs returns a set of quality IDs that are allowed in the profile.
func getAllowedQualityIDs(profile *Profile) map[int]bool {
	allowed := make(map[int]bool)
	for _, item := range profile.Items {
		if item.Allowed {
			allowed[item.Quality.ID] = true
		}
	}
	return allowed
}

// findConflictingRequiredAttributes checks if profiles have opposing required/notAllowed values.
// Req 3.1.2: Conflict means Profile A requires attribute X, Profile B has X as notAllowed (or vice versa).
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

// hasAttributeConflict returns true if two attribute settings have conflicting per-item modes.
// Conflict occurs when:
// - Profile A requires X and Profile B has X as notAllowed (or vice versa)
// - Both profiles have required values with no overlap
// HasAttributeConflict checks if two attribute settings have conflicting requirements.
// Returns true if one profile requires a value that the other has as notAllowed,
// or if both have required values with no overlap.
func HasAttributeConflict(settingsA, settingsB AttributeSettings) bool {
	// Check for required vs notAllowed conflicts
	requiredA := settingsA.GetRequired()
	requiredB := settingsB.GetRequired()
	notAllowedA := settingsA.GetNotAllowed()
	notAllowedB := settingsB.GetNotAllowed()

	// Profile A requires X, Profile B has X as notAllowed
	for _, req := range requiredA {
		for _, notAllowed := range notAllowedB {
			if req == notAllowed {
				return true
			}
		}
	}

	// Profile B requires X, Profile A has X as notAllowed
	for _, req := range requiredB {
		for _, notAllowed := range notAllowedA {
			if req == notAllowed {
				return true
			}
		}
	}

	// Both have required values with no overlap
	if len(requiredA) > 0 && len(requiredB) > 0 {
		return !hasOverlap(requiredA, requiredB)
	}

	return false
}

// findOverlappingAttributes identifies which attributes could match the same releases.
// This is used to warn users about profiles that aren't mutually exclusive.
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

// hasAttributeOverlap returns true if two settings could potentially match the same release.
// With per-item modes, overlap occurs when there's no conflict between required/notAllowed.
func hasAttributeOverlap(settingsA, settingsB AttributeSettings) bool {
	// If neither has any non-default settings, they overlap (both accept anything)
	if !settingsA.HasNonDefaultSettings() || !settingsB.HasNonDefaultSettings() {
		return true
	}

	requiredA := settingsA.GetRequired()
	requiredB := settingsB.GetRequired()

	// If neither has required values, they can overlap (preferred/any modes)
	if len(requiredA) == 0 && len(requiredB) == 0 {
		return true
	}

	// If both have required values, check for overlap
	if len(requiredA) > 0 && len(requiredB) > 0 {
		return hasOverlap(requiredA, requiredB)
	}

	// One has required, the other doesn't - they can overlap
	return true
}

// hasOverlap checks if two string slices share any common elements
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

// SlotConfig represents a slot's profile assignment for exclusivity validation
type SlotConfig struct {
	SlotNumber int
	SlotName   string
	Enabled    bool
	Profile    *Profile
}

// AttributeIssue describes a specific attribute overlap between two profiles
type AttributeIssue struct {
	Attribute string `json:"attribute"` // "HDR", "Video Codec", etc.
	Message   string `json:"message"`   // Detailed explanation
}

// SlotExclusivityError contains details about slot exclusivity validation failures
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

// ValidateSlotExclusivity checks that all enabled slots have mutually exclusive profiles.
// Req 3.1.4: System prevents saving slot configuration if assigned profiles overlap.
func ValidateSlotExclusivity(slots []SlotConfig) ([]SlotExclusivityError, bool) {
	var errors []SlotExclusivityError

	for i := 0; i < len(slots); i++ {
		slotA := slots[i]
		if !slotA.Enabled || slotA.Profile == nil {
			continue
		}

		for j := i + 1; j < len(slots); j++ {
			slotB := slots[j]
			if !slotB.Enabled || slotB.Profile == nil {
				continue
			}

			result := CheckMutualExclusivity(slotA.Profile, slotB.Profile)
			if !result.AreExclusive {
				reason, issues := buildExclusivityErrorDetails(slotA, slotB, result)
				err := SlotExclusivityError{
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
				errors = append(errors, err)
			}
		}
	}

	return errors, len(errors) == 0
}

func buildExclusivityErrorDetails(slotA, slotB SlotConfig, result ExclusivityResult) (string, []AttributeIssue) {
	if len(result.OverlappingAttrs) == 0 {
		return "profiles have no differentiating required attributes", nil
	}

	// Build detailed explanations for each overlapping attribute
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

	// Build a summary reason from the issues
	var details []string
	for _, issue := range issues {
		details = append(details, issue.Message)
	}
	return strings.Join(details, "; "), issues
}

func buildAttributeOverlapDetail(attr string, profileA, profileB *Profile) string {
	var settingsA, settingsB AttributeSettings
	switch attr {
	case "HDR":
		settingsA, settingsB = profileA.HDRSettings, profileB.HDRSettings
	case "Video Codec":
		settingsA, settingsB = profileA.VideoCodecSettings, profileB.VideoCodecSettings
	case "Audio Codec":
		settingsA, settingsB = profileA.AudioCodecSettings, profileB.AudioCodecSettings
	case "Audio Channels":
		settingsA, settingsB = profileA.AudioChannelSettings, profileB.AudioChannelSettings
	default:
		return ""
	}

	reqA := settingsA.GetRequired()
	reqB := settingsB.GetRequired()

	// Both have no requirements - accepts anything
	if len(reqA) == 0 && len(reqB) == 0 {
		return "Neither profile has required values (both accept any)"
	}

	// Both have requirements that overlap
	if len(reqA) > 0 && len(reqB) > 0 {
		overlapping := findOverlappingValues(reqA, reqB)
		if len(overlapping) > 0 {
			return "Both profiles require " + strings.Join(overlapping, ", ")
		}
	}

	// One has requirements, the other doesn't
	if len(reqA) > 0 && len(reqB) == 0 {
		return profileA.Name + " requires " + strings.Join(reqA, ", ") + "; " + profileB.Name + " accepts any"
	}
	if len(reqB) > 0 && len(reqA) == 0 {
		return profileB.Name + " requires " + strings.Join(reqB, ", ") + "; " + profileA.Name + " accepts any"
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

// GetProfileExclusivityHints returns suggestions for making profiles mutually exclusive
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
