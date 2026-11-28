package mstest

import (
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/jellyfin"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

const shelves = "/media/Shelves"

type pausing struct {
	w      *World
	served map[string]faketest.Served
	ids    map[string]link.ServerID
}

// newPausing serves Den, which home reaches, and Attic, which seedbox
// reaches, both at hosts of their own with Clip on their Shelves.
func newPausing(t *testing.T) pausing {
	t.Helper()
	p := pausing{w: New(t), served: map[string]faketest.Served{}, ids: map[string]link.ServerID{}}
	for name, id := range map[string]string{"Den": "ddc0b5b7f2824180aae9f2f245dd00ca", "Attic": "a77c0b5b7f2824180aae9f2f245dd00c"} {
		srv := jellyfin.Server{
			Name: name, Version: "12.2.0", ID: id, Keys: []string{name + "-key"},
			Users: []jellyfin.User{{Name: "ada", Password: "ada-password", Admin: true}},
			Root:  "/media", Libraries: []jellyfin.Library{{Name: "Shelves", Type: "homevideos", Folders: []string{shelves}, Scanned: true}},
		}
		p.served[name] = p.w.Jellyfin(t, hostOf(name), srv, shelves+"/Clip.mkv")
	}
	return p
}

func hostOf(name string) string {
	if name == "Den" {
		return "192.168.1.20:8096"
	}
	return "192.168.1.21:8096"
}

func (p pausing) add(t *testing.T, name string, reaching role.NodeID, runsOn *role.NodeID) {
	t.Helper()
	v, err := p.w.Service.Add(t.Context(), mediaservers.Draft{
		Kind: "jellyfin", Name: name, Address: "http://" + hostOf(name), Reaching: reaching, RunsOn: runsOn, APIKey: name + "-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	u, err := storage.ParseUUID(v.ID)
	if err != nil {
		t.Fatal(err)
	}
	p.ids[name] = link.ServerID{UUID: u}
	synctest.Wait()
}

func (p pausing) patch(t *testing.T, name string, patch mediaservers.Patch) {
	t.Helper()
	if _, err := p.w.Service.Update(t.Context(), p.ids[name], patch); err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
}

type streams struct {
	video, audio     link.StreamDecision
	containerChanged bool
}

func (p pausing) play(t *testing.T, name string, s streams) {
	t.Helper()
	p.served[name].Cue(t, "jellyfin-play", map[string]any{
		"path": shelves + "/Clip.mkv", "user": "ada", "player": "Living room", "playerKey": "living-room", "reasons": []string{"ContainerNotSupported"},
		"video": s.video, "audio": s.audio, "subtitles": link.SubtitlesNone, "containerChanged": s.containerChanged,
	})
	synctest.Wait()
}

func off() *bool { return new(bool) }

// TestTranscodingOnIsEachPausingServersOwnVideoTranscode has Den and Attic
// run on home, each reached by its own Node: only a video transcode is
// transcoding, a server with Pause Conversions off isn't listed, and a
// reaching Node that is away leaves only its own server unknown.
func TestTranscodingOnIsEachPausingServersOwnVideoTranscode(t *testing.T) {
	type is struct {
		name       string
		known, now bool
	}
	cases := []struct {
		name string
		act  func(t *testing.T, p pausing)
		want []is
	}{
		{"nothing plays", func(*testing.T, pausing) {}, []is{{"Attic", true, false}, {"Den", true, false}}},
		{"a video transcode", func(t *testing.T, p pausing) {
			p.play(t, "Den", streams{video: link.StreamTranscode, audio: link.StreamDirect, containerChanged: true})
		}, []is{{"Attic", true, false}, {"Den", true, true}}},
		{"an audio only transcode", func(t *testing.T, p pausing) {
			p.play(t, "Den", streams{video: link.StreamDirect, audio: link.StreamTranscode, containerChanged: true})
		}, []is{{"Attic", true, false}, {"Den", true, false}}},
		{"a container change", func(t *testing.T, p pausing) {
			p.play(t, "Den", streams{video: link.StreamDirect, audio: link.StreamDirect, containerChanged: true})
		}, []is{{"Attic", true, false}, {"Den", true, false}}},
		{"Pause Conversions off", func(t *testing.T, p pausing) {
			p.patch(t, "Den", mediaservers.Patch{PausesConversions: off()})
			p.play(t, "Den", streams{video: link.StreamTranscode, audio: link.StreamDirect})
		}, []is{{"Attic", true, false}}},
		{"the reaching Node away", func(t *testing.T, p pausing) {
			p.play(t, "Attic", streams{video: link.StreamTranscode, audio: link.StreamDirect})
			p.w.Offline(p.w.Seedbox)
			synctest.Wait()
		}, []is{{"Attic", false, false}, {"Den", true, false}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				p := newPausing(t)
				p.add(t, "Den", p.w.Home, &p.w.Home)
				p.add(t, "Attic", p.w.Seedbox, &p.w.Home)
				c.act(t, p)
				var got []is
				for _, x := range p.w.TranscodingOn(t, p.w.Home) {
					if x.Server != p.ids[x.Name] {
						t.Errorf("%s is listed as server %s, want %s", x.Name, x.Server, p.ids[x.Name])
					}
					got = append(got, is{x.Name, x.Known, x.Now})
				}
				if !slices.Equal(got, c.want) {
					t.Errorf("transcoding on home = %v, want %v", got, c.want)
				}
				if seedbox := p.w.TranscodingOn(t, p.w.Seedbox); len(seedbox) != 0 {
					t.Errorf("transcoding on seedbox = %v, want none", seedbox)
				}
			})
		})
	}
}

// TestTouchedFollowsEachWriteThatChangesWhatPauses counts Touched over
// writes in turn: a server saved, changed or removed touches Converting's
// holders only when where it runs or its Pause Conversions changes.
func TestTouchedFollowsEachWriteThatChangesWhatPauses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPausing(t)
		rename := "Den downstairs"
		steps := []struct {
			name    string
			act     func(t *testing.T)
			touches int
		}{
			{"Attic saved running on no Node", func(t *testing.T) { p.add(t, "Attic", p.w.Seedbox, nil) }, 0},
			{"Den saved running on home", func(t *testing.T) { p.add(t, "Den", p.w.Home, &p.w.Home) }, 1},
			{"Den renamed", func(t *testing.T) { p.patch(t, "Den", mediaservers.Patch{Name: &rename}) }, 0},
			{"Den's Pause Conversions off", func(t *testing.T) { p.patch(t, "Den", mediaservers.Patch{PausesConversions: off()}) }, 1},
			{"Den's Pause Conversions off again", func(t *testing.T) { p.patch(t, "Den", mediaservers.Patch{PausesConversions: off()}) }, 0},
			{"Attic runs on seedbox", func(t *testing.T) {
				p.patch(t, "Attic", mediaservers.Patch{RunsOn: &mediaservers.Set[role.NodeID]{To: &p.w.Seedbox}})
			}, 1},
			{"Den runs on no Node", func(t *testing.T) { p.patch(t, "Den", mediaservers.Patch{RunsOn: &mediaservers.Set[role.NodeID]{}}) }, 1},
			{"Den removed", func(t *testing.T) { p.remove(t, "Den") }, 0},
			{"Attic removed", func(t *testing.T) { p.remove(t, "Attic") }, 1},
		}
		for _, s := range steps {
			was := p.w.Touched()
			s.act(t)
			if got := p.w.Touched() - was; got != s.touches {
				t.Errorf("%s touched %d times, want %d", s.name, got, s.touches)
			}
		}
		if v, err := p.w.Service.List(t.Context()); err != nil || len(v) != 0 {
			t.Errorf("the servers left are %v, %v; want none", v, err)
		}
	})
}

func (p pausing) remove(t *testing.T, name string) {
	t.Helper()
	if err := p.w.Service.Remove(t.Context(), p.ids[name]); err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
}
