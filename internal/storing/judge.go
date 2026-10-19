package storing

import "github.com/jatassi/slipstream/internal/storage"

type reserved struct {
	path             string
	root             storage.UUID
	inside, contains Reason
}

func apartFrom(roots []Other) []reserved {
	all := make([]reserved, 0, len(roots))
	for _, o := range roots {
		all = append(all, reserved{o.Path, o.ID, ReasonInsideRoot, ReasonContainsRoot})
	}
	return all
}

// forRoot is the Node's own folders a root may not overlap: its data
// folder and its downloads folders.
func forRoot(data string, downloads []string) []reserved {
	all := []reserved{{path: data, inside: ReasonInsideData, contains: ReasonContainsData}}
	for _, d := range downloads {
		all = append(all, reserved{path: d, inside: ReasonInsideDownloads, contains: ReasonContainsDownloads})
	}
	return all
}

// judge refuses a clean folder that is inside, equal to or contains a
// reserved folder, in order. It walks each one's ancestors with same, the
// file-identity comparison (os.SameFile on the Node), so case, separators
// and links can't fool it. A folder same can't read is never the same as
// another: a root that can't be read can't hold folder now.
func judge(style PathStyle, folder string, all []reserved, same func(a, b string) bool) *Refusal {
	for _, r := range all {
		if within(style, folder, r.path, same) {
			return &Refusal{Reason: r.inside, Root: r.root}
		}
		if within(style, r.path, folder, same) {
			return &Refusal{Reason: r.contains, Root: r.root}
		}
	}
	return nil
}

func within(style PathStyle, p, dir string, same func(a, b string) bool) bool {
	for at, ok := p, true; ok; at, ok = style.Parent(at) {
		if same(at, dir) {
			return true
		}
	}
	return false
}
