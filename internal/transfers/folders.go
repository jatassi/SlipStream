package transfers

import "github.com/jatassi/slipstream/internal/role"

// BackupFolder is one Backup folder in a Node's intent, as the Admin
// typed it: the Backup names its Node deletes there, and the Backup
// named Check it answers a keyed BLAKE3 of under CheckKey, nil for none.
type BackupFolder struct {
	Folder   string
	Delete   []string
	CheckKey []byte
	Check    string
}

// FolderFacts is one Backup folder as its Node last reported it: the
// Backup names it holds, or why it can't be read, and its answer to the
// last check, nil for none. It is a Worker's word, so a caller trusts
// nothing in it but the names and the sum it compares.
type FolderFacts struct {
	Folder string
	Names  []string
	Error  string
	Check  *CheckAnswer
}

// CheckAnswer is a Node's keyed BLAKE3 of the Backup named Name.
type CheckAnswer struct {
	Name     string
	Key, Sum []byte
	Error    string
}

// FoldersOf is node's Backup folders as it last reported them.
func (t *Transfers) FoldersOf(node role.NodeID) []FolderFacts {
	facts, _ := t.cfg.Facts(node)
	out := make([]FolderFacts, 0, len(facts.GetBackupFolders()))
	for _, f := range facts.GetBackupFolders() {
		ff := FolderFacts{Folder: f.GetFolder(), Names: f.GetNames(), Error: f.GetError()}
		if c := f.GetCheck(); c != nil {
			ff.Check = &CheckAnswer{Name: c.GetName(), Key: c.GetKey(), Sum: c.GetSum(), Error: c.GetError()}
		}
		out = append(out, ff)
	}
	return out
}

// TouchFolders tells nodes that the Backup folders Folders answers for
// them changed.
func (t *Transfers) TouchFolders(nodes ...role.NodeID) { t.cfg.Touch(nodes...) }
