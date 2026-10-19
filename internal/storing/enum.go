package storing

import (
	"fmt"

	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
)

type mapping[G ~uint8, W ~int32] struct {
	value G
	wire  W
	word  string
}

// enum is a domain enum's one table: every Go value, its wire value and its
// word, so no second list can drift from it.
type enum[G ~uint8, W ~int32] []mapping[G, W]

func (e enum[G, W]) of(g G) mapping[G, W] {
	for _, m := range e {
		if m.value == g {
			return m
		}
	}
	panic(fmt.Sprintf("storing: %T(%d) is not in its table", g, g))
}

func (e enum[G, W]) parse(w W) (G, error) {
	for _, m := range e {
		if m.wire == w {
			return m.value, nil
		}
	}
	return 0, fmt.Errorf("unknown %T %d", w, w)
}

func (e enum[G, W]) parseWord(word string) (G, error) {
	for _, m := range e {
		if m.word == word {
			return m.value, nil
		}
	}
	var g G
	return 0, fmt.Errorf("unknown %T %q", g, word)
}

// RootState is what a root's Node found at its folder. Only Ready takes
// imports.
type RootState uint8

const (
	Ready RootState = iota + 1
	Missing
	Unmounted
	Foreign
	Unwritable
	Unresponsive
)

var rootStates = enum[RootState, storagev1.RootState]{
	{Ready, storagev1.RootState_ROOT_STATE_READY, "ready"},
	{Missing, storagev1.RootState_ROOT_STATE_MISSING, "missing"},
	{Unmounted, storagev1.RootState_ROOT_STATE_UNMOUNTED, "unmounted"},
	{Foreign, storagev1.RootState_ROOT_STATE_FOREIGN, "foreign"},
	{Unwritable, storagev1.RootState_ROOT_STATE_UNWRITABLE, "unwritable"},
	{Unresponsive, storagev1.RootState_ROOT_STATE_UNRESPONSIVE, "unresponsive"},
}

func (s RootState) String() string                            { return rootStates.of(s).word }
func (s RootState) Wire() storagev1.RootState                 { return rootStates.of(s).wire }
func ParseRootState(w storagev1.RootState) (RootState, error) { return rootStates.parse(w) }

type Kind uint8

const (
	Local Kind = iota + 1
	Removable
	Network
)

var kinds = enum[Kind, storagev1.VolumeKind]{
	{Local, storagev1.VolumeKind_VOLUME_KIND_LOCAL, "local"},
	{Removable, storagev1.VolumeKind_VOLUME_KIND_REMOVABLE, "removable"},
	{Network, storagev1.VolumeKind_VOLUME_KIND_NETWORK, "network"},
}

func (k Kind) String() string                        { return kinds.of(k).word }
func (k Kind) Wire() storagev1.VolumeKind            { return kinds.of(k).wire }
func ParseKind(w storagev1.VolumeKind) (Kind, error) { return kinds.parse(w) }
func ParseKindWord(word string) (Kind, error)        { return kinds.parseWord(word) }

// Reason is why Check refused a folder as a root. A folder equal to
// another root or folder is inside it.
type Reason uint8

const (
	ReasonNotAbsolute Reason = iota + 1
	ReasonNotFound
	ReasonNotAFolder
	ReasonUnwritable
	ReasonInsideRoot
	ReasonContainsRoot
	ReasonInsideData
	ReasonContainsData
	ReasonInsideDownloads
	ReasonContainsDownloads
	ReasonUnresponsive
)

var reasons = enum[Reason, storagev1.Reason]{
	{ReasonNotAbsolute, storagev1.Reason_REASON_NOT_ABSOLUTE, "not_absolute"},
	{ReasonNotFound, storagev1.Reason_REASON_NOT_FOUND, "not_found"},
	{ReasonNotAFolder, storagev1.Reason_REASON_NOT_A_FOLDER, "not_a_folder"},
	{ReasonUnwritable, storagev1.Reason_REASON_UNWRITABLE, "unwritable"},
	{ReasonInsideRoot, storagev1.Reason_REASON_INSIDE_ROOT, "inside_root"},
	{ReasonContainsRoot, storagev1.Reason_REASON_CONTAINS_ROOT, "contains_root"},
	{ReasonInsideData, storagev1.Reason_REASON_INSIDE_DATA, "inside_data"},
	{ReasonContainsData, storagev1.Reason_REASON_CONTAINS_DATA, "contains_data"},
	{ReasonInsideDownloads, storagev1.Reason_REASON_INSIDE_DOWNLOADS, "inside_downloads"},
	{ReasonContainsDownloads, storagev1.Reason_REASON_CONTAINS_DOWNLOADS, "contains_downloads"},
	{ReasonUnresponsive, storagev1.Reason_REASON_UNRESPONSIVE, "unresponsive"},
}

func (r Reason) String() string         { return reasons.of(r).word }
func (r Reason) Wire() storagev1.Reason { return reasons.of(r).wire }

// FolderErrorKind is why a folder op failed. None is folded into another.
type FolderErrorKind uint8

const (
	FolderNotFound FolderErrorKind = iota + 1
	FolderNotAFolder
	FolderDenied
	FolderUnresponsive
	FolderIO
)

var folderErrorKinds = enum[FolderErrorKind, storagev1.FolderErrorKind]{
	{FolderNotFound, storagev1.FolderErrorKind_FOLDER_ERROR_KIND_NOT_FOUND, "not_found"},
	{FolderNotAFolder, storagev1.FolderErrorKind_FOLDER_ERROR_KIND_NOT_A_FOLDER, "not_a_folder"},
	{FolderDenied, storagev1.FolderErrorKind_FOLDER_ERROR_KIND_DENIED, "denied"},
	{FolderUnresponsive, storagev1.FolderErrorKind_FOLDER_ERROR_KIND_UNRESPONSIVE, "unresponsive"},
	{FolderIO, storagev1.FolderErrorKind_FOLDER_ERROR_KIND_IO, "io"},
}

func (k FolderErrorKind) String() string                  { return folderErrorKinds.of(k).word }
func (k FolderErrorKind) Wire() storagev1.FolderErrorKind { return folderErrorKinds.of(k).wire }

type MissingReason uint8

const (
	FileMissing MissingReason = iota + 1
	SizeDiffers
)

var missingReasons = enum[MissingReason, storagev1.MissingReason]{
	{FileMissing, storagev1.MissingReason_MISSING_REASON_MISSING, "missing"},
	{SizeDiffers, storagev1.MissingReason_MISSING_REASON_SIZE_DIFFERS, "size_differs"},
}

func (r MissingReason) String() string                { return missingReasons.of(r).word }
func (r MissingReason) Wire() storagev1.MissingReason { return missingReasons.of(r).wire }

// HeldFolder is one of the folders SlipStream keeps inside a root.
type HeldFolder uint8

const (
	Holding HeldFolder = iota + 1
	Staging
)

var heldFolders = enum[HeldFolder, storagev1.HeldFolder]{
	{Holding, storagev1.HeldFolder_HELD_FOLDER_HOLDING, HoldingDir},
	{Staging, storagev1.HeldFolder_HELD_FOLDER_STAGING, StagingDir},
}

func (f HeldFolder) String() string             { return heldFolders.of(f).word }
func (f HeldFolder) Wire() storagev1.HeldFolder { return heldFolders.of(f).wire }
