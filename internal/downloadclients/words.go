package downloadclients

import (
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/torrent/clients"
)

// Subject is the client a sentence names: its name, its kind and its
// address.
type Subject struct {
	Name    string
	Kind    clients.Kind
	Address clients.Address
}

func (e External) Subject() Subject { return Subject{Name: e.Name, Kind: e.Kind, Address: e.Address} }

// Says is the sentence for p on client s, whose Node is named node: what
// happened, then what the Admin does about it.
func Says(p clients.Problem, s Subject, node string) string {
	name, product := s.Name, s.Kind.Facts().Name
	switch p.Cause {
	case clients.NoResponse, clients.TimedOut:
		return fmt.Sprintf("%s isn't responding. Check that it's running and that %s can reach %s.", name, node, s.Address.HostPort())
	case clients.TLSFailed:
		return fmt.Sprintf("The secure connection from %s to %s at %s failed. Check that %s uses HTTPS there and that %s trusts its certificate.",
			node, name, s.Address.HostPort(), name, node)
	case clients.ServerError:
		return fmt.Sprintf("%s responded with status %d. Check %s's log for the reason.", name, p.Status, name)
	case clients.WrongCredentials:
		if s.Kind.Facts().SignIn.Username == clients.NotTaken {
			return name + " rejected the password. Check it and test again."
		}
		return name + " rejected the username or password. Check them and test again."
	case clients.WrongAPIKey:
		return name + " rejected the API key. Replace it and test again."
	case clients.Banned:
		return fmt.Sprintf("%s banned %s's address after too many failed sign-ins. Wait an hour or restart %s, then select Check now.", name, node, name)
	case clients.LockedOut:
		return fmt.Sprintf("%s locked out %s after too many failed sign-ins. Restart %s, then select Check now.", name, node, name)
	case clients.VersionTooOld:
		return fmt.Sprintf("%s reached %s %s, which is older than SlipStream supports. Update it to %s or later and test again.",
			node, product, p.Version, s.Kind.Facts().Floor)
	case clients.LabelPluginOff:
		return fmt.Sprintf("%s's Label plugin is off. Turn it on in %s's preferences and test again.", name, name)
	case clients.FolderUnreadable:
		if p.Paired != "" {
			return fmt.Sprintf("%s can't open %s, which is paired with %s. Check the pair, and check that %s can read that folder.", node, p.Folder, p.Paired, node)
		}
		return fmt.Sprintf("%s can't open %s, the folder %s saves to. Under Client folders, pair %s with the folder %s sees.", node, p.Folder, name, p.Folder, node)
	case clients.FolderInRoot:
		return fmt.Sprintf("%s saves to %s, which overlaps a library root on %s. Choose a download folder outside every library root and test again.", name, p.Folder, node)
	case clients.AddressRefused:
		return fmt.Sprintf("%s refused %s's address. Add it to rpc-whitelist in %s's settings, then select Check now.", name, node, name)
	case clients.HostRefused:
		return hostRefused(s, p.Host)
	case clients.DaemonNeeded:
		return name + "'s web UI lists more than one daemon. Choose the daemon SlipStream uses and test again."
	case clients.WrongDaemon:
		return name + "'s web UI is connected to another daemon. Choose the daemon SlipStream uses and test again."
	case clients.NotKept:
		return fmt.Sprintf("%s didn't keep the torrent SlipStream sent it. Check %s's log for the reason.", name, name)
	case clients.MostMissing:
		return fmt.Sprintf("%s no longer lists %d of SlipStream's torrents. Check that the address points at the right %s.", name, p.Missing, product)
	}
	panic("downloadclients: no sentence for cause " + string(p.Cause))
}

func hostRefused(s Subject, host string) string {
	switch s.Kind {
	case clients.Transmission:
		return fmt.Sprintf("%s refused the host name %s. Use its IP address, or add the name to rpc-host-whitelist in %s's settings.", s.Name, host, s.Name)
	case clients.QBittorrent:
		return fmt.Sprintf("%s refused the host name %s. Use its IP address, or add the name to the server domains in %s's Web UI options.", s.Name, host, s.Name)
	case clients.Deluge, clients.RTorrent:
	}
	return fmt.Sprintf("%s refused the host name %s. Use its IP address.", s.Name, host)
}

// Warns is the sentence for a passing Test's warning on client s.
func Warns(w clients.Warning, s Subject) string {
	switch w {
	case clients.RemovesAtLimit:
		return fmt.Sprintf("%s removes torrents that reach a seeding limit. A download could be removed before SlipStream imports it. "+
			"To keep downloads until they're imported, set %s to stop torrents at the limit.", s.Name, s.Name)
	case clients.MovesAfterFinish:
		return s.Name + " moves finished downloads to another folder. Set an Import delay so SlipStream imports after the move."
	case clients.NoMovingState:
		return s.Kind.Facts().Name + " doesn't report when it moves finished downloads. If " + s.Name +
			" moves them, set an Import delay so SlipStream imports after the move."
	}
	panic(fmt.Sprintf("downloadclients: no sentence for warning %d", w))
}

// Stopped is the sentence of a refused sign-in on a kind with a lockout,
// whose polls stopped.
func Stopped(name string) string {
	return name + " rejected the saved password. SlipStream stopped checking it to avoid a lockout. Change the password, or select Check now once it's fixed."
}

// LocalError is the sentence of a torrent client name stopped with its own
// error text.
func LocalError(name, text string) string {
	return name + ` stopped this torrent with the error "` + text + `". Fix the cause in ` + name + " and the download carries on."
}

// Unreached is the sentence of a Test that couldn't run on node.
func Unreached(node string) string {
	return fmt.Sprintf("SlipStream couldn't reach %s to run the Test. Test again once %s is back.", node, node)
}

// InUse is the sentence of a refusal to turn off, move or delete the
// client node's new Grabs go to.
func InUse(node string) string {
	return node + "'s new Grabs go to this client. Choose another client for them first."
}

// HasTorrents is the sentence of a refusal to delete or move client name
// while it holds count Torrents SlipStream tracks.
func HasTorrents(name string, count int, moving bool) string {
	held := fmt.Sprintf("%s holds %d torrents SlipStream tracks.", name, count)
	if count == 1 {
		held = name + " holds 1 torrent SlipStream tracks."
	}
	if moving {
		return held + " Remove them, or delete the client with Stop tracking, then move it."
	}
	return held + " Stop tracking them to delete it."
}

// Importing is the sentence of a refusal to delete client name while a
// Grab imports from it.
func Importing(name string) string {
	return "A Grab is importing from " + name + ". Delete it once the import finishes."
}

// InClient is the sentence of a refusal to send a torrent to another
// client on node once client already has it, "" for node's built-in
// client.
func InClient(client, node string) string {
	if client == "" {
		client = node + "'s built-in client"
	}
	return client + " already has this torrent. Send it to a client on another Node, or remove it in Activity."
}

// AddressTaken is the sentence of a refusal of an address client name on
// node already uses.
func AddressTaken(name, node string) string {
	return fmt.Sprintf("%s on %s already uses this address.", name, node)
}

// GrabsToRefused is the sentence of a refusal to send node's new Grabs to
// the client e names.
func GrabsToRefused(e *GrabsToError, node string) string {
	if e.Off {
		return e.Name + " is off. Turn it on before new Grabs go to it."
	}
	return fmt.Sprintf("%s is on another Node. Choose a client on %s.", e.Name, node)
}

// Refused is the sentence of a field SlipStream can't use, for kind.
func Refused(r *RefusedError, kind clients.Kind) string {
	name := kind.Facts().Name
	var folder *clients.FolderError
	if errors.As(r.Err, &folder) {
		return folderRefusal(folder, name)
	}
	switch {
	case errors.Is(r.Err, clients.ErrAddressScheme):
		return "Start the address with http:// or https://."
	case errors.Is(r.Err, clients.ErrAddressNotURL):
		return "Write the address as a URL, such as http://localhost:9091."
	case errors.Is(r.Err, clients.ErrAddressHost):
		return "Add the host name or IP address of the computer " + name + " runs on."
	case errors.Is(r.Err, clients.ErrAddressPort):
		return "Use a port between 1 and 65535."
	case errors.Is(r.Err, clients.ErrSignInInAddress):
		return "This address holds a username or password. Put them in the Username and Password fields."
	case errors.Is(r.Err, clients.ErrQueryInAddress):
		return "This address holds a query or fragment. Remove the part from the ? or #."
	case errors.Is(r.Err, clients.ErrLabelEmpty):
		return "Enter a label."
	case errors.Is(r.Err, clients.ErrLabelControl):
		return "Remove the control character from the label."
	case errors.Is(r.Err, clients.ErrLabelComma):
		return name + " splits labels at a comma. Use a label without one."
	case errors.Is(r.Err, clients.ErrLabelWord):
		return name + " takes labels of a-z, 0-9, _ and - only."
	case errors.Is(r.Err, ErrNotTaken):
		return name + " takes no " + fieldWords[r.Field] + "."
	case errors.Is(r.Err, ErrNeeded):
		return "Enter " + name + "'s " + fieldWords[r.Field] + "."
	}
	panic("downloadclients: no sentence for refusal " + r.Error())
}

var fieldWords = map[clients.Field]string{
	clients.FieldUsername: "username", clients.FieldPassword: "password", clients.FieldAPIKey: "API key", clients.FieldDaemon: "daemon",
}

func folderRefusal(e *clients.FolderError, name string) string {
	switch {
	case errors.Is(e.Err, clients.ErrClientFolderNotAbsolute):
		return fmt.Sprintf("In pair %d, write the folder as %s reports it as a full path, such as /downloads.", e.Index+1, name)
	case errors.Is(e.Err, clients.ErrNodeFolderNotAbsolute):
		return fmt.Sprintf("In pair %d, write the folder on the Node as a full path, such as D:\\Torrents.", e.Index+1)
	case errors.Is(e.Err, clients.ErrClientFolderTwice):
		return fmt.Sprintf("Pair %d repeats a folder another pair already names. Pair each of %s's folders once.", e.Index+1, name)
	}
	panic("downloadclients: no sentence for Client folder refusal " + e.Error())
}
