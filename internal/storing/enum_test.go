package storing

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
)

func walkEnum[G ~uint8, W interface {
	~int32
	Descriptor() protoreflect.EnumDescriptor
}](t *testing.T, e enum[G, W]) {
	t.Helper()
	var zero W
	values := zero.Descriptor().Values()
	words := map[string]bool{}
	for i := range values.Len() {
		w := W(values.Get(i).Number())
		g, err := e.parse(w)
		if w == 0 {
			if err == nil {
				t.Errorf("%T UNSPECIFIED parsed as %d", w, g)
			}
			continue
		}
		if err != nil {
			t.Errorf("%T %s: %v", w, values.Get(i).Name(), err)
			continue
		}
		if back := e.of(g).wire; back != w {
			t.Errorf("%T %s maps back to %d", w, values.Get(i).Name(), back)
		}
		word := e.of(g).word
		if word == "" || words[word] {
			t.Errorf("%T %s has word %q, empty or taken", w, values.Get(i).Name(), word)
		}
		words[word] = true
	}
	if len(e) != values.Len()-1 {
		t.Errorf("%T: table has %d values, the wire %d", zero, len(e), values.Len()-1)
	}
	if _, err := e.parse(W(values.Len())); err == nil {
		t.Errorf("%T %d past the end parsed", zero, values.Len())
	}
}

func TestEveryEnumValueMapsBothWays(t *testing.T) {
	walkEnum(t, rootStates)
	walkEnum(t, kinds)
	walkEnum(t, reasons)
	walkEnum(t, purposes)
	walkEnum(t, folderErrorKinds)
	walkEnum(t, missingReasons)
	walkEnum(t, heldFolders)
}

func TestEnumMethodsReadTheTable(t *testing.T) {
	if Unmounted.String() != "unmounted" || Unmounted.Wire() != storagev1.RootState_ROOT_STATE_UNMOUNTED {
		t.Errorf("Unmounted = %s, %s", Unmounted, Unmounted.Wire())
	}
	if s, err := ParseRootState(storagev1.RootState_ROOT_STATE_FOREIGN); s != Foreign || err != nil {
		t.Errorf("ParseRootState(FOREIGN) = %s, %v", s, err)
	}
	if k, err := ParseKind(storagev1.VolumeKind_VOLUME_KIND_NETWORK); k != Network || err != nil {
		t.Errorf("ParseKind(NETWORK) = %s, %v", k, err)
	}
	if ReasonContainsDownloads.String() != "contains_downloads" || SizeDiffers.String() != "size_differs" {
		t.Errorf("words: %s, %s", ReasonContainsDownloads, SizeDiffers)
	}
	if Holding.String() != HoldingDir || Staging.String() != StagingDir {
		t.Errorf("held folders: %s, %s", Holding, Staging)
	}
}
