package downloadclients_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/link"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	tc "github.com/jatassi/slipstream/internal/torrent/clients"
)

func qbittorrent(t *testing.T, node role.NodeID) dc.Imported {
	t.Helper()
	a, err := tc.ParseAddress("http://192.168.1.20:8080", tc.QBittorrent)
	if err != nil {
		t.Fatal(err)
	}
	l, err := tc.ParseLabel("tv-sonarr", tc.QBittorrent)
	if err != nil {
		t.Fatal(err)
	}
	return dc.Imported{Node: node, Kind: tc.QBittorrent, Name: "qBittorrent", Address: a, Secrets: tc.Secrets{APIKey: "letmein"}, Label: l}
}

func TestTheIntentCarriesTheNodesClientsWithoutSecrets(t *testing.T) {
	lib, sealer := open(t), secrettest.Sealer(t)
	media, seedbox := addNode(t, lib, "Media server"), addNode(t, lib, "Seedbox")
	trans := mustImport(t, lib, sealer, transmission(t, media, "Transmission", "http://192.168.1.20:9091"))
	exec(t, lib, `UPDATE downloadclients_client SET folder = '/downloads', client_folders = '[{"Client":"/downloads","Node":"D:\\Torrents"}]',
		check_every_seconds = 30 WHERE id = ?`, trans)
	qbit := mustImport(t, lib, sealer, qbittorrent(t, media))
	exec(t, lib, "UPDATE downloadclients_client SET is_on = 0, settings_version = 3 WHERE id = ?", qbit)
	gone := mustImport(t, lib, sealer, transmission(t, media, "Old Transmission", "http://192.168.1.30:9091"))
	exec(t, lib, "UPDATE downloadclients_client SET deleted_at = ?, secrets_sealed = NULL WHERE id = ?", storage.Now(), gone)
	mustImport(t, lib, sealer, transmission(t, seedbox, "Seedbox Transmission", "http://10.0.0.5:9091"))

	m, raw := dotTorrent(t, "Orchard")
	inTrans, _ := mustAttach(t, lib, dc.Want{Target: dc.Target{Node: media, Client: &trans}, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
	m, raw = dotTorrent(t, "Harbor")
	builtIn, _ := mustAttach(t, lib, dc.Want{Target: dc.Target{Node: media}, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
	m, raw = dotTorrent(t, "Attic")
	removing, _ := mustAttach(t, lib, dc.Want{Target: dc.Target{Node: media, Client: &trans}, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
	removeTorrent(t, lib, removing, dc.ByAdmin)

	s := &mediaServer{lib: lib, node: media}
	in := s.intent(t)
	want := []*dlv1.Client{
		{
			Id: trans.Bytes(), Kind: dlv1.Kind_KIND_TRANSMISSION, Name: "Transmission", Address: "http://192.168.1.20:9091/transmission/rpc",
			Username: "jackson", Label: "tv-sonarr", Folder: "/downloads",
			ClientFolders:     []*dlv1.ClientFolder{{Client: "/downloads", Node: `D:\Torrents`}},
			CheckEverySeconds: 30, ImportDelaySeconds: 300, On: true, SettingsVersion: 1,
		},
		{
			Id: qbit.Bytes(), Kind: dlv1.Kind_KIND_QBITTORRENT, Name: "qBittorrent", Address: "http://192.168.1.20:8080", Label: "tv-sonarr",
			CheckEverySeconds: int64(dc.CheckEvery.Default.Seconds()), SettingsVersion: 3,
		},
	}
	if bytes.Compare(trans.Bytes(), qbit.Bytes()) > 0 {
		want[0], want[1] = want[1], want[0]
	}
	if len(in.GetClients()) != len(want) {
		t.Fatalf("the intent carries %d clients, want Media server's two live ones: %v", len(in.GetClients()), in.GetClients())
	}
	for i := range want {
		if !proto.Equal(in.GetClients()[i], want[i]) {
			t.Errorf("client %d in the intent is %v, want %v", i, in.GetClients()[i], want[i])
		}
	}

	b, err := proto.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"hunter2", "letmein"} {
		if bytes.Contains(b, []byte(secret)) {
			t.Errorf("the intent's bytes hold the secret %q", secret)
		}
	}

	clientOf := map[dc.ID][]byte{}
	for _, tr := range in.GetTorrents() {
		clientOf[idFrom(tr.GetId())] = tr.GetClientId()
	}
	for _, d := range in.GetDiscards() {
		clientOf[idFrom(d.GetId())] = d.GetClientId()
	}
	for id, want := range map[dc.ID][]byte{inTrans: trans.Bytes(), builtIn: nil, removing: trans.Bytes()} {
		if got, ok := clientOf[id]; !ok || !bytes.Equal(got, want) {
			t.Errorf("the intent names client %x for torrent %s (listed %t), want %x", got, id, ok, want)
		}
	}

	again := s.intent(t)
	if !proto.Equal(in, again) {
		t.Error("the same library gave two different intents")
	}
}

func TestClientSecretsAnswerOnlyTheClientsOwnNode(t *testing.T) {
	lib, sealer := open(t), secrettest.Sealer(t)
	media, seedbox := addNode(t, lib, "Media server"), addNode(t, lib, "Seedbox")
	trans := mustImport(t, lib, sealer, transmission(t, media, "Transmission", "http://192.168.1.20:9091"))
	qbit := mustImport(t, lib, sealer, qbittorrent(t, media))

	key := link.Key{7}
	nodes := callers{key: media}
	d := rpc.NewDispatch(nil, key, key, func(role.Name) bool { return true })
	downloadingv1connect.DownloadingControlService.Mount(rpc.NewCluster(d, noRoster{}), dc.ControlService{Library: lib, Sealer: sealer, Nodes: nodes, Types: types{}})
	svc := downloadingv1connect.DownloadingControlService.Call(d)
	fetch := func(id dc.ClientID) (*dlv1.ClientSecretsResponse, error) {
		return svc.ClientSecrets(t.Context(), &dlv1.ClientSecretsRequest{Id: id.Bytes()})
	}

	for id, want := range map[dc.ClientID]*dlv1.ClientSecretsResponse{
		trans: {Secrets: &dlv1.Secrets{Password: "hunter2"}, SettingsVersion: 1},
		qbit:  {Secrets: &dlv1.Secrets{ApiKey: "letmein"}, SettingsVersion: 1},
	} {
		if res, err := fetch(id); err != nil || !proto.Equal(res, want) {
			t.Errorf("Media server fetched %v, %v for client %s; want %v", res, err, id, want)
		}
	}

	exec(t, lib, "UPDATE downloadclients_client SET settings_version = 2 WHERE id = ?", trans)
	if res, err := fetch(trans); err != nil || res.GetSettingsVersion() != 2 {
		t.Errorf("after an edit Media server fetched %v, %v, want the secrets at settings version 2", res, err)
	}

	nodes[key] = seedbox
	if _, err := fetch(trans); rpc.CodeOf(err) != rpc.CodeNotFound {
		t.Errorf("another Node's fetch = %v, want NotFound", err)
	}
	delete(nodes, key)
	if _, err := fetch(trans); rpc.CodeOf(err) != rpc.CodeNotFound {
		t.Errorf("an unknown caller's fetch = %v, want NotFound", err)
	}
	nodes[key] = media
	exec(t, lib, "UPDATE downloadclients_client SET deleted_at = ?, secrets_sealed = NULL WHERE id = ?", storage.Now(), trans)
	if _, err := fetch(trans); rpc.CodeOf(err) != rpc.CodeNotFound {
		t.Errorf("a fetch for a deleted client = %v, want NotFound", err)
	}
}

func TestEveryKindReachesTheIntent(t *testing.T) {
	lib, sealer := open(t), secrettest.Sealer(t)
	media := addNode(t, lib, "Media server")
	for i, k := range tc.Kinds {
		a, err := tc.ParseAddress(fmt.Sprintf("http://192.168.1.%d:9000", 20+i), k)
		if err != nil {
			t.Fatal(err)
		}
		l, err := tc.ParseLabel("tv", k)
		if err != nil {
			t.Fatal(err)
		}
		mustImport(t, lib, sealer, dc.Imported{Node: media, Kind: k, Name: k.Facts().Name, Address: a, Label: l})
	}
	s := &mediaServer{lib: lib, node: media}
	got := map[string]bool{}
	for _, c := range s.intent(t).GetClients() {
		got[c.GetKind().String()] = true
	}
	for _, k := range tc.Kinds {
		if want := "KIND_" + strings.ToUpper(string(k)); !got[want] {
			t.Errorf("the intent carries no client of kind %s, want %s", k, want)
		}
	}
}
