package module

// QualityVariables returns the template variables related to quality attributes.
// Shared by all video-type modules.
func QualityVariables() []TemplateVariable {
	return []TemplateVariable{
		{Name: "Quality Title", Description: "Quality with source (e.g., HDTV-720p)", Example: "Bluray-1080p", DataKey: "QualityTitle"},
		{Name: "Quality Full", Description: "Full quality string", Example: "HDTV-720p Proper", DataKey: "QualityFull"},
	}
}

// MediaInfoVariables returns the template variables from media info probing.
func MediaInfoVariables() []TemplateVariable {
	return []TemplateVariable{
		{Name: "MediaInfo VideoCodec", Description: "Video codec", Example: "x264", DataKey: "VideoCodec"},
		{Name: "MediaInfo AudioCodec", Description: "Audio codec", Example: "DTS", DataKey: "AudioCodec"},
		{Name: "MediaInfo AudioChannels", Description: "Audio channels", Example: "5.1", DataKey: "AudioChannels"},
		{Name: "MediaInfo VideoDynamicRange", Description: "Dynamic range", Example: "HDR", DataKey: "VideoDynamicRange"},
	}
}

// MetadataVariables returns the common metadata template variables.
func MetadataVariables() []TemplateVariable {
	return []TemplateVariable{
		{Name: "Release Group", Description: "Release group name", Example: "SPARKS", DataKey: "ReleaseGroup"},
		{Name: "Edition Tags", Description: "Edition info", Example: "Director's Cut", DataKey: "EditionTags"},
	}
}
