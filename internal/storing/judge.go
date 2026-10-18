package storing

import "github.com/jatassi/slipstream/internal/storage"

// reserved is a folder no root may overlap, and the reasons it refuses.
type reserved struct {
	path             string
	root             storage.UUID
	inside, contains Reason
}

// judge refuses a clean folder that is inside, equal to or contains another
// root, the data dir or the downloads dir. It walks each one's ancestors
// with same, the file-identity comparison (os.SameFile on the Node), so
// case, separators and links can't fool it. A folder same can't read is
// never the same as another: a root that can't be read can't hold folder
// now.
func judge(style PathStyle, folder string, others []Other, data, downloads string, same func(a, b string) bool) *Refusal {
	var all []reserved
	for _, o := range others {
		all = append(all, reserved{o.Path, o.ID, ReasonInsideRoot, ReasonContainsRoot})
	}
	for _, r := range []reserved{
		{path: data, inside: ReasonInsideData, contains: ReasonContainsData},
		{path: downloads, inside: ReasonInsideDownloads, contains: ReasonContainsDownloads},
	} {
		if r.path != "" {
			all = append(all, r)
		}
	}
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

// within reports whether p or one of its ancestors is the same folder as
// dir.
func within(style PathStyle, p, dir string, same func(a, b string) bool) bool {
	for at, ok := p, true; ok; at, ok = style.Parent(at) {
		if same(at, dir) {
			return true
		}
	}
	return false
}
