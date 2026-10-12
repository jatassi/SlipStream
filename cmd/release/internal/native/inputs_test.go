package native

import (
	"slices"
	"testing"
)

func TestRepoInputsListEveryTarget(t *testing.T) {
	in, err := LoadInputs("../../../../native/inputs.json")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tg := range in.Targets {
		got = append(got, tg.Name+":"+string(tg.Gate))
	}
	want := []string{"linux-amd64:pr", "linux-arm64:native", "windows-amd64:native", "darwin-arm64:native", "darwin-amd64:native"}
	if !slices.Equal(got, want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}
}

func TestParseInputsRejectsUnknownGate(t *testing.T) {
	_, err := ParseInputs([]byte(`{"targets": [{"name": "linux-amd64", "gate": "nightly"}], "components": []}`))
	if err == nil {
		t.Fatal("parsed a target with gate nightly")
	}
}

func TestParseInputsRejectsUnknownLayerKind(t *testing.T) {
	_, err := ParseInputs([]byte(`{"targets": [{"name": "linux-amd64", "gate": "pr", "layers": ["libs", "ffmpeg"]}], "components": []}`))
	want := `native inputs: target linux-amd64: unknown layer kind "ffmpeg"`
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %s", err, want)
	}
}
