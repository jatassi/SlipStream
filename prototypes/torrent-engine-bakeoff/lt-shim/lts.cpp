#include "lts.h"

#include <libtorrent/add_torrent_params.hpp>
#include <libtorrent/alert_types.hpp>
#include <libtorrent/session.hpp>
#include <libtorrent/session_params.hpp>
#include <libtorrent/settings_pack.hpp>
#include <libtorrent/torrent_handle.hpp>
#include <libtorrent/torrent_info.hpp>
#include <libtorrent/torrent_status.hpp>

#include <cstring>
#include <mutex>
#include <string>
#include <unordered_map>
#include <vector>

struct lts_id;

struct lts_session {
	lt::session ses;
	std::mutex mu;
	std::unordered_map<lts_torrent, lt::torrent_handle> by_id;
	std::unordered_map<lt::torrent_handle, lts_torrent> by_handle;
	lts_torrent next = 1;
	std::vector<lt::alert*> pending;
	size_t pos = 0;
	explicit lts_session(lt::session_params p) : ses(std::move(p)) {}
};

namespace {

const char* const seedbox[][2] = {
	{"enable_dht", "false"},
	{"enable_lsd", "false"},
	{"enable_upnp", "false"},
	{"enable_natpmp", "false"},
	{"active_downloads", "-1"},
	{"active_seeds", "-1"},
	{"active_limit", "-1"},
	{"active_tracker_limit", "-1"},
	{"dont_count_slow_torrents", "true"},
	{"announce_to_all_tiers", "false"},
	{"announce_to_all_trackers", "false"},
	{"max_peerlist_size", "500"},
	{"max_paused_peerlist_size", "50"},
};

void put(char* dst, int len, const std::string& s) {
	if (!dst || len <= 0) return;
	std::strncpy(dst, s.c_str(), len - 1);
	dst[len - 1] = 0;
}

void apply(lt::settings_pack& p, const char* k, const char* v) {
	int idx = lt::setting_by_name(k);
	if (idx < 0) throw std::runtime_error(std::string("unknown setting ") + k);
	switch (idx & lt::settings_pack::type_mask) {
	case lt::settings_pack::string_type_base: p.set_str(idx, v); break;
	case lt::settings_pack::int_type_base: p.set_int(idx, int(std::stol(v, nullptr, 0))); break;
	case lt::settings_pack::bool_type_base: p.set_bool(idx, !std::strcmp(v, "true") || !std::strcmp(v, "1")); break;
	}
}

int kind(lt::alert* a) {
	switch (a->type()) {
	case lt::add_torrent_alert::alert_type: return LTS_ALERT_ADDED;
	case lt::torrent_finished_alert::alert_type: return LTS_ALERT_FINISHED;
	case lt::torrent_paused_alert::alert_type: return LTS_ALERT_PAUSED;
	case lt::torrent_resumed_alert::alert_type: return LTS_ALERT_RESUMED;
	case lt::torrent_removed_alert::alert_type: return LTS_ALERT_REMOVED;
	case lt::tracker_announce_alert::alert_type: return LTS_ALERT_TRACKER_ANNOUNCE;
	case lt::tracker_reply_alert::alert_type: return LTS_ALERT_TRACKER_REPLY;
	case lt::tracker_error_alert::alert_type: return LTS_ALERT_TRACKER_ERROR;
	}
	return (a->category() & lt::alert_category::error) ? LTS_ALERT_ERROR : LTS_ALERT_OTHER;
}

lt::torrent_handle find(lts_session* s, lts_torrent t) {
	std::lock_guard<std::mutex> l(s->mu);
	auto it = s->by_id.find(t);
	if (it == s->by_id.end()) throw std::runtime_error("unknown torrent");
	return it->second;
}

}

extern "C" {

lts_session* lts_session_new(int preset, const char* const* keys, const char* const* values, int n, char* err, int errlen) {
	try {
		lt::settings_pack p;
		if (preset == LTS_PRESET_SEEDBOX) {
			p = lt::high_performance_seed();
			for (auto& kv : seedbox) apply(p, kv[0], kv[1]);
		}
		p.set_int(lt::settings_pack::alert_mask, int(static_cast<std::uint32_t>(
			lt::alert_category::error | lt::alert_category::status | lt::alert_category::tracker | lt::alert_category::storage)));
		for (int i = 0; i < n; ++i) apply(p, keys[i], values[i]);
		return new lts_session(lt::session_params(std::move(p)));
	} catch (std::exception const& e) {
		put(err, errlen, e.what());
	} catch (...) {
		put(err, errlen, "unknown exception");
	}
	return nullptr;
}

lts_torrent lts_add(lts_session* s, const char* torrent_file, const char* save_path, int seed_mode, char* err, int errlen) {
	try {
		lt::add_torrent_params atp;
		atp.ti = std::make_shared<lt::torrent_info>(std::string(torrent_file));
		atp.save_path = save_path;
		atp.flags &= ~(lt::torrent_flags::auto_managed | lt::torrent_flags::paused);
		if (seed_mode) atp.flags |= lt::torrent_flags::seed_mode;
		std::lock_guard<std::mutex> l(s->mu);
		lts_torrent id = s->next++;
		atp.userdata = lt::client_data_t(reinterpret_cast<lts_id*>(id));
		lt::torrent_handle h = s->ses.add_torrent(std::move(atp));
		s->by_id[id] = h;
		s->by_handle[h] = id;
		return id;
	} catch (std::exception const& e) {
		put(err, errlen, e.what());
	} catch (...) {
		put(err, errlen, "unknown exception");
	}
	return -1;
}

int lts_pause(lts_session* s, lts_torrent t) {
	try { find(s, t).pause(); return 0; } catch (...) { return -1; }
}

int lts_resume(lts_session* s, lts_torrent t) {
	try { find(s, t).resume(); return 0; } catch (...) { return -1; }
}

int lts_remove(lts_session* s, lts_torrent t) {
	try {
		lt::torrent_handle h = find(s, t);
		s->ses.remove_torrent(h);
		std::lock_guard<std::mutex> l(s->mu);
		s->by_handle.erase(h);
		s->by_id.erase(t);
		return 0;
	} catch (...) {
		return -1;
	}
}

int lts_status_get(lts_session* s, lts_torrent t, lts_status* out) {
	try {
		lt::torrent_status st = find(s, t).status();
		out->state = int32_t(st.state);
		out->paused = bool(st.flags & lt::torrent_flags::paused);
		out->is_seeding = st.is_seeding;
		out->is_finished = st.is_finished;
		out->num_peers = st.num_peers;
		out->num_seeds = st.num_seeds;
		out->upload_rate = st.upload_rate;
		out->download_rate = st.download_rate;
		out->progress = st.progress;
		out->total_done = st.total_done;
		out->total_wanted = st.total_wanted;
		out->all_time_upload = st.all_time_upload;
		out->all_time_download = st.all_time_download;
		return 0;
	} catch (...) {
		return -1;
	}
}

int lts_poll_alerts(lts_session* s, lts_alert* out, int max, int timeout_ms) {
	try {
		if (s->pos >= s->pending.size()) {
			s->pending.clear();
			s->pos = 0;
			if (!s->ses.wait_for_alert(lt::milliseconds(timeout_ms))) return 0;
			s->ses.pop_alerts(&s->pending);
		}
		std::lock_guard<std::mutex> l(s->mu);
		int n = 0;
		for (; n < max && s->pos < s->pending.size(); ++n) {
			lt::alert* a = s->pending[s->pos++];
			lts_alert& o = out[n];
			o.type = kind(a);
			o.lt_type = a->type();
			o.torrent = 0;
			if (auto* r = lt::alert_cast<lt::torrent_removed_alert>(a)) {
				o.torrent = lts_torrent(reinterpret_cast<intptr_t>(r->userdata.get<lts_id>()));
			} else if (auto* ta = dynamic_cast<lt::torrent_alert*>(a)) {
				auto it = s->by_handle.find(ta->handle);
				if (it != s->by_handle.end()) o.torrent = it->second;
			}
			put(o.what, sizeof(o.what), a->what());
			put(o.message, sizeof(o.message), a->message());
		}
		return n;
	} catch (...) {
		return -1;
	}
}

void lts_close(lts_session* s) {
	try {
		{ lt::session_proxy p = s->ses.abort(); }
		delete s;
	} catch (...) {
	}
}

}
