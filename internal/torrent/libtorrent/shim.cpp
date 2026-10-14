#include "shim.h"

#include <libtorrent/add_torrent_params.hpp>
#include <libtorrent/address.hpp>
#include <libtorrent/alert_types.hpp>
#include <libtorrent/bdecode.hpp>
#include <libtorrent/error_code.hpp>
#include <libtorrent/file_storage.hpp>
#include <libtorrent/session.hpp>
#include <libtorrent/session_params.hpp>
#include <libtorrent/settings_pack.hpp>
#include <libtorrent/torrent_handle.hpp>
#include <libtorrent/torrent_info.hpp>
#include <libtorrent/torrent_status.hpp>
#include <libtorrent/version.hpp>

#include <cstdlib>
#include <cstring>
#include <deque>
#include <memory>
#include <mutex>
#include <stdexcept>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <vector>

// client_data_t only stores typed pointers, so torrent ids ride in it as a
// pointer to this never-defined type.
struct lts_id;

struct lts_session {
	lt::session ses;
	std::mutex mu;
	std::unordered_map<int64_t, lt::torrent_handle> by_id;
	std::unordered_map<lt::torrent_handle, int64_t> by_handle;
	std::unordered_set<int64_t> finishing;
	int64_t next = 1;
	std::vector<lt::alert*> popped;
	std::deque<lts_alert> pending;
	explicit lts_session(lt::session_params p) : ses(std::move(p)) {}
};

namespace {

using sp = lt::settings_pack;

struct shim_error : std::runtime_error {
	int32_t code;
	shim_error(int32_t c, char const* text) : std::runtime_error(text), code(c) {}
};

struct int_setting {
	int name;
	int value;
};

struct bool_setting {
	int name;
	bool value;
};

// The production preset, applied on top of lt::high_performance_seed(). A
// tracker on loopback must then announce on a path ending in /announce, and
// an HTTPS tracker must present a certificate OpenSSL's default verify
// paths trust. No caller reaches peer_fingerprint: libtorrent's -LT peer id
// always holds.
int_setting const preset_ints[] = {
	{sp::active_downloads, -1},
	{sp::active_seeds, -1},
	{sp::active_limit, -1},
	{sp::active_tracker_limit, -1},
	{sp::max_peerlist_size, 500},
	{sp::max_paused_peerlist_size, 50},
};

bool_setting const preset_bools[] = {
	{sp::dont_count_slow_torrents, true},
	{sp::announce_to_all_tiers, false},
	{sp::announce_to_all_trackers, false},
	{sp::validate_https_trackers, true},
	{sp::ssrf_mitigation, true},
};

void fill(lts_error* err, int32_t category, int32_t code, std::string const& text) {
	err->category = category;
	err->code = code;
	std::strncpy(err->text, text.c_str(), sizeof(err->text) - 1);
	err->text[sizeof(err->text) - 1] = 0;
}

void fill(lts_error* err, lt::error_code const& ec) {
	auto const& cat = ec.category();
	int32_t category = LTS_CATEGORY_OTHER;
	if (cat == lt::libtorrent_category()) category = LTS_CATEGORY_LIBTORRENT;
	else if (cat == lt::bdecode_category()) category = LTS_CATEGORY_BDECODE;
	else if (cat == lt::system_category() || cat == lt::generic_category()) category = LTS_CATEGORY_SYSTEM;
	fill(err, category, ec.value(), ec.message());
}

void check(lt::error_code const& ec) {
	if (ec) throw lt::system_error(ec);
}

template <class F>
int guard(lts_error* err, F&& f) {
	try {
		f();
		return 0;
	} catch (lt::system_error const& e) {
		fill(err, e.code());
	} catch (shim_error const& e) {
		fill(err, LTS_CATEGORY_SHIM, e.code, e.what());
	} catch (std::exception const& e) {
		fill(err, LTS_CATEGORY_EXCEPTION, 0, e.what());
	} catch (...) {
		fill(err, LTS_CATEGORY_EXCEPTION, 0, "unknown exception");
	}
	return -1;
}

lt::torrent_handle find(lts_session* s, int64_t torrent) {
	std::lock_guard<std::mutex> l(s->mu);
	auto it = s->by_id.find(torrent);
	if (it == s->by_id.end()) throw shim_error(LTS_SHIM_UNKNOWN_TORRENT, "unknown torrent");
	return it->second;
}

int32_t state(lt::torrent_status::state_t st) {
	switch (st) {
	case lt::torrent_status::checking_files: return LTS_STATE_CHECKING_FILES;
	case lt::torrent_status::downloading_metadata: return LTS_STATE_DOWNLOADING_METADATA;
	case lt::torrent_status::downloading: return LTS_STATE_DOWNLOADING;
	case lt::torrent_status::finished: return LTS_STATE_FINISHED;
	case lt::torrent_status::seeding: return LTS_STATE_SEEDING;
	case lt::torrent_status::checking_resume_data: return LTS_STATE_CHECKING_RESUME_DATA;
	}
	throw std::logic_error("unknown torrent state");
}

int64_t id_of(lt::client_data_t const& userdata) {
	return int64_t(reinterpret_cast<intptr_t>(userdata.get<lts_id>()));
}

// Called with s->mu held. Alerts the engine does not model are skipped.
void convert(lts_session* s, lt::alert* a) {
	lts_alert o{};
	auto torrent = [&](lt::torrent_alert const* ta) {
		auto it = s->by_handle.find(ta->handle);
		return it == s->by_handle.end() ? 0 : it->second;
	};
	switch (a->type()) {
	case lt::add_torrent_alert::alert_type: {
		auto* at = static_cast<lt::add_torrent_alert*>(a);
		if (at->error) return;
		o.kind = LTS_ALERT_ADDED;
		o.torrent = id_of(at->params.userdata);
		break;
	}
	case lt::state_changed_alert::alert_type: {
		auto* sc = static_cast<lt::state_changed_alert*>(a);
		o.kind = LTS_ALERT_STATE_CHANGED;
		o.torrent = torrent(sc);
		o.state = state(sc->state);
		break;
	}
	// libtorrent posts torrent_finished once the last piece passes its hash
	// check, which can be before that piece's blocks are written to the file.
	// Finishing also releases the torrent's files behind a fence on every
	// queued write, and that release posts cache_flushed. Finished waits for it
	// so a reader of the file sees the whole payload.
	case lt::torrent_finished_alert::alert_type: {
		int64_t id = torrent(static_cast<lt::torrent_alert*>(a));
		if (id != 0) s->finishing.insert(id);
		return;
	}
	case lt::cache_flushed_alert::alert_type: {
		int64_t id = torrent(static_cast<lt::torrent_alert*>(a));
		if (s->finishing.erase(id) == 0) return;
		o.kind = LTS_ALERT_FINISHED;
		o.torrent = id;
		break;
	}
	case lt::torrent_removed_alert::alert_type:
		o.kind = LTS_ALERT_REMOVED;
		o.torrent = id_of(static_cast<lt::torrent_removed_alert*>(a)->userdata);
		s->finishing.erase(o.torrent);
		break;
	case lt::torrent_error_alert::alert_type: {
		auto* te = static_cast<lt::torrent_error_alert*>(a);
		o.kind = LTS_ALERT_TORRENT_FAILED;
		o.torrent = torrent(te);
		fill(&o.error, te->error);
		break;
	}
	case lt::tracker_reply_alert::alert_type:
		o.kind = LTS_ALERT_TRACKER_REPLIED;
		o.torrent = torrent(static_cast<lt::torrent_alert*>(a));
		break;
	case lt::tracker_error_alert::alert_type: {
		auto* te = static_cast<lt::tracker_error_alert*>(a);
		o.kind = LTS_ALERT_TRACKER_FAILED;
		o.torrent = torrent(te);
		fill(&o.error, te->error);
		break;
	}
	case lt::listen_failed_alert::alert_type:
		o.kind = LTS_ALERT_LISTEN_FAILED;
		fill(&o.error, static_cast<lt::listen_failed_alert*>(a)->error);
		s->pending.push_back(o);
		return;
	case lt::alerts_dropped_alert::alert_type:
		o.kind = LTS_ALERT_DROPPED;
		s->pending.push_back(o);
		return;
	default:
		return;
	}
	if (o.torrent != 0) s->pending.push_back(o);
}

std::string listen_interface(lt::address const& addr, uint16_t port) {
	std::string host = addr.to_string();
	if (addr.is_v6()) host = "[" + host + "]";
	return host + ":" + std::to_string(port);
}

}

extern "C" {

lts_session* lts_open(char const* address, uint16_t port, uint32_t discovery, int outgoing_utp, lts_error* err) {
	lts_session* s = nullptr;
	guard(err, [&] {
		lt::address addr = lt::make_address(address);
		sp p = lt::high_performance_seed();
		for (auto const& kv : preset_ints) p.set_int(kv.name, kv.value);
		for (auto const& kv : preset_bools) p.set_bool(kv.name, kv.value);
		p.set_bool(sp::enable_dht, discovery & LTS_DISCOVERY_DHT);
		p.set_bool(sp::enable_lsd, discovery & LTS_DISCOVERY_LSD);
		p.set_bool(sp::enable_upnp, discovery & LTS_DISCOVERY_UPNP);
		p.set_bool(sp::enable_natpmp, discovery & LTS_DISCOVERY_NATPMP);
		p.set_bool(sp::enable_outgoing_utp, outgoing_utp != 0);
		p.set_str(sp::listen_interfaces, listen_interface(addr, port));
		p.set_int(sp::alert_mask, int(static_cast<std::uint32_t>(lt::alert_category::error | lt::alert_category::status | lt::alert_category::storage | lt::alert_category::tracker)));
		s = new lts_session(lt::session_params(std::move(p)));
	});
	return s;
}

int lts_listen_port(lts_session* s, uint16_t* port, lts_error* err) {
	return guard(err, [&] { *port = s->ses.listen_port(); });
}

int lts_add(lts_session* s, char const* metainfo, size_t len, char const* save_path, int seed_mode, int64_t* torrent, lts_error* err) {
	return guard(err, [&] {
		lt::error_code ec;
		auto ti = std::make_shared<lt::torrent_info>(lt::span<char const>(metainfo, std::ptrdiff_t(len)), ec, lt::from_span);
		check(ec);
		lt::add_torrent_params atp;
		atp.ti = std::move(ti);
		atp.save_path = save_path;
		atp.flags &= ~(lt::torrent_flags::auto_managed | lt::torrent_flags::paused);
		if (seed_mode) atp.flags |= lt::torrent_flags::seed_mode;
		std::lock_guard<std::mutex> l(s->mu);
		int64_t id = s->next++;
		atp.userdata = lt::client_data_t(reinterpret_cast<lts_id*>(intptr_t(id)));
		lt::torrent_handle h = s->ses.add_torrent(std::move(atp), ec);
		check(ec);
		s->by_id[id] = h;
		s->by_handle[h] = id;
		*torrent = id;
	});
}

int lts_connect_peer(lts_session* s, int64_t torrent, char const* address, uint16_t port, lts_error* err) {
	return guard(err, [&] {
		lt::error_code ec;
		lt::address addr = lt::make_address(address, ec);
		check(ec);
		find(s, torrent).connect_peer(lt::tcp::endpoint(addr, port));
	});
}

int lts_status_get(lts_session* s, int64_t torrent, lts_status* out, lts_error* err) {
	return guard(err, [&] {
		lt::torrent_status st = find(s, torrent).status();
		out->state = state(st.state);
		out->peers = st.num_peers;
		out->done = st.total_done;
		out->wanted = st.total_wanted;
	});
}

int lts_remove(lts_session* s, int64_t torrent, lts_error* err) {
	return guard(err, [&] {
		lt::torrent_handle h = find(s, torrent);
		s->ses.remove_torrent(h);
		std::lock_guard<std::mutex> l(s->mu);
		s->by_handle.erase(h);
		s->by_id.erase(torrent);
	});
}

int lts_torrent_files(lts_session* s, int64_t torrent, lts_file** files, int32_t* count, lts_error* err) {
	return guard(err, [&] {
		std::shared_ptr<lt::torrent_info const> ti = find(s, torrent).torrent_file();
		lt::file_storage const& fs = ti->files();
		std::vector<lt::file_index_t> kept;
		std::vector<std::string> paths;
		for (lt::file_index_t i : fs.file_range()) {
			if (fs.pad_file_at(i)) continue;
			kept.push_back(i);
			paths.push_back(fs.file_path(i));
		}
		auto* out = static_cast<lts_file*>(std::calloc(kept.size() + 1, sizeof(lts_file)));
		if (out == nullptr) throw std::bad_alloc();
		for (std::size_t n = 0; n < kept.size(); ++n) {
			out[n].index = int32_t(static_cast<int>(kept[n]));
			out[n].size = fs.file_size(kept[n]);
			out[n].path = static_cast<char*>(std::malloc(paths[n].size() + 1));
			if (out[n].path == nullptr) {
				lts_free_files(out, int32_t(n));
				throw std::bad_alloc();
			}
			std::memcpy(out[n].path, paths[n].c_str(), paths[n].size() + 1);
		}
		*files = out;
		*count = int32_t(kept.size());
	});
}

void lts_free_files(lts_file* files, int32_t count) {
	for (int32_t i = 0; i < count; ++i) std::free(files[i].path);
	std::free(files);
}

int lts_wait_alerts(lts_session* s, int timeout_ms, lts_alert* out, int max, lts_error* err) {
	int n = 0;
	int rc = guard(err, [&] {
		if (s->pending.empty() && s->ses.wait_for_alert(lt::milliseconds(timeout_ms))) {
			s->ses.pop_alerts(&s->popped);
			std::lock_guard<std::mutex> l(s->mu);
			for (lt::alert* a : s->popped) convert(s, a);
		}
		for (; n < max && !s->pending.empty(); ++n) {
			out[n] = s->pending.front();
			s->pending.pop_front();
		}
	});
	return rc < 0 ? -1 : n;
}

int lts_close(lts_session* s, lts_error* err) {
	int rc = guard(err, [&] { lt::session_proxy proxy = s->ses.abort(); });
	delete s;
	return rc;
}

char const* lts_version(void) {
	return lt::version();
}

}
