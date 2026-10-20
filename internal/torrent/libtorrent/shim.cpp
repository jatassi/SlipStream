#include "shim.h"

#include <boost/asio/post.hpp>
#include <libtorrent/add_torrent_params.hpp>
#include <libtorrent/address.hpp>
#include <libtorrent/alert_types.hpp>
#include <libtorrent/announce_entry.hpp>
#include <libtorrent/bdecode.hpp>
#include <libtorrent/error_code.hpp>
#include <libtorrent/file_storage.hpp>
#include <libtorrent/ip_filter.hpp>
#include <libtorrent/posix_disk_io.hpp>
#include <libtorrent/read_resume_data.hpp>
#include <libtorrent/session.hpp>
#include <libtorrent/session_params.hpp>
#include <libtorrent/session_stats.hpp>
#include <libtorrent/settings_pack.hpp>
#include <libtorrent/torrent_handle.hpp>
#include <libtorrent/torrent_info.hpp>
#include <libtorrent/torrent_status.hpp>
#include <libtorrent/version.hpp>
#include <libtorrent/write_resume_data.hpp>

#include <algorithm>
#include <chrono>
#include <cstddef>
#include <cstdlib>
#include <cstring>
#include <deque>
#include <filesystem>
#include <map>
#include <memory>
#include <mutex>
#include <optional>
#include <stdexcept>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <vector>

// client_data_t only stores typed pointers, so torrent ids ride in it as a
// pointer to this never-defined type.
struct lts_id;

static_assert(sizeof(lts_error) == 264, "lts_error layout: update TestTheShimStructsMatchGo");
static_assert(sizeof(lts_status) == 368 && offsetof(lts_status, need_save) == 96 && offsetof(lts_status, error) == 100,
	"lts_status layout: update TestTheShimStructsMatchGo");
static_assert(sizeof(lts_add_params) == 80 && offsetof(lts_add_params, seed_mode) == 40 && offsetof(lts_add_params, base_uploaded) == 48,
	"lts_add_params layout: update TestTheShimStructsMatchGo");
static_assert(sizeof(lts_alert) == 320 && offsetof(lts_alert, error) == 16 && offsetof(lts_alert, payload) == 280 && offsetof(lts_alert, at_ms) == 312,
	"lts_alert layout: update TestTheShimStructsMatchGo");
static_assert(sizeof(lts_tracker) == 320 && offsetof(lts_tracker, last_error) == 32 && offsetof(lts_tracker, start_sent) == 312,
	"lts_tracker layout: update TestTheShimStructsMatchGo");
static_assert(sizeof(lts_file) == 24 && offsetof(lts_file, path) == 16, "lts_file layout: update TestTheShimStructsMatchGo");
static_assert(sizeof(lts_limits) == 32 && offsetof(lts_limits, connections) == 16 && offsetof(lts_limits, watch_incoming) == 24,
	"lts_limits layout: update TestTheShimStructsMatchGo");

// parting_torrent is a torrent removed while a tracker had yet to answer its
// started announce. libtorrent announces stopped only to a tracker that
// answered started, so the removal waits for that answer.
struct parting_torrent {
	int64_t id;
	bool delete_files;
	std::string save_path;
};

// deleting_torrent is a torrent whose files libtorrent is deleting: its save
// path goes too once they are gone, if nothing else is left in it.
// libtorrent posts the deletion before the removal, so the deletion waits
// in files_deleted until the removal is out.
struct deleting_torrent {
	int64_t id;
	std::string save_path;
	bool removed = false;
	std::optional<lts_alert> files_deleted;
};

// turn is what a torrent does once no tracker awaits an answer.
enum class turn { pause, resume };

struct lts_session {
	lt::session ses;
	// popped is the alert pump's alone.
	std::vector<lt::alert*> popped;
	// mu guards everything below.
	std::mutex mu;
	std::unordered_map<int64_t, lt::torrent_handle> by_id;
	std::unordered_map<lt::torrent_handle, int64_t> by_handle;
	std::unordered_set<int64_t> finishing;
	std::unordered_map<lt::torrent_handle, parting_torrent> parting;
	// turning holds each torrent's pause or resume that waits for its
	// trackers to answer, as awaits_answer says.
	std::unordered_map<lt::torrent_handle, turn> turning;
	std::map<lt::info_hash_t, deleting_torrent> deleting;
	// rechecking holds each torrent whose Recheck has yet to pause it.
	// libtorrent pauses it before it posts torrent_checked, so its pause
	// stands for both.
	std::unordered_set<lt::torrent_handle> rechecking;
	// resync is set once alerts were dropped: the next post_updates sends
	// every torrent, since the dropped ones may have carried changes.
	bool resync = false;
	int64_t next = 1;
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
	if (ec == boost::system::errc::no_space_on_device) {
		fill(err, LTS_CATEGORY_SHIM, LTS_SHIM_DISK_FULL, ec.message());
		return;
	}
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

// alert_mask is the alerts the shim converts. peer brings
// incoming_connection, among many the shim skips, so it is on only while the
// caller watches incoming peers.
int alert_mask(bool watch_incoming) {
	lt::alert_category_t mask = lt::alert_category::error | lt::alert_category::status | lt::alert_category::storage | lt::alert_category::tracker;
	if (watch_incoming) mask |= lt::alert_category::peer;
	return int(static_cast<std::uint32_t>(mask));
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

// awaits_answer is whether a tracker has yet to answer an announce of h.
// libtorrent sends no stopped to a tracker it awaits, and none later. It
// skips started too, and sends it only an interval after the answer. So a
// torrent pauses, resumes, leaves or closes only once none awaits an
// answer.
bool awaits_answer(lt::torrent_handle const& h) {
	for (lt::announce_entry const& ae : h.trackers()) {
		for (lt::announce_endpoint const& ep : ae.endpoints) {
			for (lt::announce_infohash const& ih : ep.info_hashes) {
				if (ih.updating) return true;
			}
		}
	}
	return false;
}

void remove_now(lts_session* s, lt::torrent_handle const& h, parting_torrent const& p) {
	lt::remove_flags_t flags{};
	if (p.delete_files) {
		flags = lt::session::delete_files;
		s->deleting[h.info_hashes()] = deleting_torrent{p.id, p.save_path, false, std::nullopt};
	}
	s->ses.remove_torrent(h, flags);
}

// settle carries out the pause, resume or removal of h that waited for its
// trackers to answer, once none awaits an answer. Called with s->mu held.
void settle(lts_session* s, lt::torrent_handle const& h) {
	auto p = s->parting.find(h);
	auto t = s->turning.find(h);
	if ((p == s->parting.end() && t == s->turning.end()) || awaits_answer(h)) return;
	if (t != s->turning.end()) {
		turn const to = t->second;
		s->turning.erase(t);
		if (to == turn::pause) {
			h.pause(lt::torrent_handle::graceful_pause);
		} else {
			h.unset_flags(lt::torrent_flags::upload_mode);
			h.resume();
		}
	}
	if (p != s->parting.end()) {
		parting_torrent const gone = p->second;
		s->parting.erase(p);
		remove_now(s, h, gone);
	}
}

// turn_to pauses or resumes h now, or once no tracker awaits an answer.
// Called with s->mu held.
void turn_to(lts_session* s, lt::torrent_handle const& h, turn to) {
	s->turning[h] = to;
	settle(s, h);
}

int64_t unix_ms(lt::time_point t) {
	if (t.time_since_epoch().count() == 0) return 0;
	auto const wall = std::chrono::system_clock::now() + std::chrono::duration_cast<std::chrono::system_clock::duration>(t - lt::clock_type::now());
	return std::chrono::duration_cast<std::chrono::milliseconds>(wall.time_since_epoch()).count();
}

int64_t unix_ms(lt::time_point32 t) {
	return t == (lt::time_point32::min)() ? 0 : unix_ms(lt::time_point(t));
}

void fill(lts_status* out, int64_t id, lt::torrent_status const& st) {
	*out = lts_status{};
	out->torrent = id;
	out->state = state(st.state);
	out->paused = bool(st.flags & lt::torrent_flags::paused);
	out->done = st.total_done;
	out->wanted = st.total_wanted;
	out->down_bps = st.download_payload_rate;
	out->up_bps = st.upload_payload_rate;
	out->peers = st.num_peers;
	out->seeds = st.num_seeds;
	out->uploaded = st.all_time_upload;
	out->downloaded = st.all_time_download;
	out->active_s = st.active_duration.count();
	out->seeding_s = st.seeding_duration.count();
	out->last_download_ms = unix_ms(st.last_download);
	out->need_save = st.need_save_resume;
	if (st.errc) fill(&out->error, st.errc);
}

// updates is an UPDATES alert of the statuses of the torrents the caller
// holds. Called with s->mu held.
lts_alert updates(lts_session* s, std::vector<lt::torrent_status> const& sts) {
	lts_alert o{};
	o.kind = LTS_ALERT_UPDATES;
	auto* out = static_cast<lts_status*>(std::calloc(sts.size() + 1, sizeof(lts_status)));
	if (out == nullptr) throw std::bad_alloc();
	int64_t n = 0;
	for (lt::torrent_status const& st : sts) {
		auto it = s->by_handle.find(st.handle);
		if (it == s->by_handle.end()) continue;
		fill(&out[n++], it->second, st);
	}
	o.payload = out;
	o.payload_len = n;
	return o;
}

// remove_save_path removes the save path a deletion emptied, and leaves it
// when anything else is in it.
void remove_save_path(std::string const& save_path, lts_error* err) {
	std::error_code ec;
	std::filesystem::path const dir = std::filesystem::u8path(save_path);
	if (std::filesystem::exists(dir, ec) && std::filesystem::is_empty(dir, ec)) std::filesystem::remove(dir, ec);
	if (ec) fill(err, LTS_CATEGORY_SYSTEM, ec.value(), ec.message());
}

// tracker_index is url's index in h's tracker list.
int32_t tracker_index(lt::torrent_handle const& h, std::string const& url) {
	std::vector<lt::announce_entry> const trackers = h.trackers();
	for (std::size_t i = 0; i < trackers.size(); ++i) {
		if (trackers[i].url == url) return int32_t(i);
	}
	throw std::logic_error("an alert's tracker is not in its torrent's list");
}

int32_t announce_event(lt::event_t e) {
	switch (e) {
	case lt::event_t::completed: return LTS_EVENT_COMPLETED;
	case lt::event_t::started: return LTS_EVENT_STARTED;
	case lt::event_t::stopped: return LTS_EVENT_STOPPED;
	// paused is the regular announce of a torrent finished but not seeding.
	case lt::event_t::none:
	case lt::event_t::paused: return LTS_EVENT_NONE;
	}
	throw std::logic_error("unknown announce event");
}

// set_text makes text the alert's payload.
void set_text(lts_alert* o, char const* text) {
	std::size_t const n = std::strlen(text);
	if (n == 0) return;
	o->payload = std::malloc(n);
	if (o->payload == nullptr) throw std::bad_alloc();
	std::memcpy(o->payload, text, n);
	o->payload_len = int64_t(n);
}

// tracker_list lays the lts_tracker array and the text it points to out in
// one block, so lts_free_alert frees it whole.
void tracker_list(lts_alert* o, std::vector<lt::announce_entry> const& trackers) {
	auto const first = [](lt::announce_entry const& ae) -> lt::announce_infohash const* {
		return ae.endpoints.empty() ? nullptr : &ae.endpoints.front().info_hashes[lt::protocol_version::V1];
	};
	std::size_t bytes = trackers.size() * sizeof(lts_tracker);
	for (lt::announce_entry const& ae : trackers) {
		lt::announce_infohash const* ih = first(ae);
		bytes += ae.url.size() + 1 + (ih ? ih->message.size() : 0) + 1;
	}
	auto* block = static_cast<char*>(std::calloc(1, bytes + 1));
	if (block == nullptr) throw std::bad_alloc();
	auto* out = reinterpret_cast<lts_tracker*>(block);
	char* text = block + trackers.size() * sizeof(lts_tracker);
	auto const put = [&](std::string const& str) {
		char* at = text;
		std::memcpy(at, str.c_str(), str.size() + 1);
		text += str.size() + 1;
		return at;
	};
	for (std::size_t i = 0; i < trackers.size(); ++i) {
		lt::announce_entry const& ae = trackers[i];
		lt::announce_infohash const* ih = first(ae);
		out[i].url = put(ae.url);
		out[i].message = put(ih ? ih->message : std::string());
		out[i].tier = ae.tier;
		out[i].endpoints = int32_t(ae.endpoints.size());
		if (ih == nullptr) continue;
		out[i].next_announce_ms = unix_ms(ih->next_announce);
		out[i].min_announce_ms = unix_ms(ih->min_announce);
		if (ih->last_error) fill(&out[i].last_error, ih->last_error);
		out[i].fails = ih->fails;
		out[i].updating = ih->updating;
		out[i].start_sent = ih->start_sent;
	}
	o->payload = block;
	o->payload_len = int64_t(trackers.size());
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
	case lt::torrent_removed_alert::alert_type: {
		auto* ra = static_cast<lt::torrent_removed_alert*>(a);
		o.kind = LTS_ALERT_REMOVED;
		o.torrent = id_of(ra->userdata);
		s->finishing.erase(o.torrent);
		s->pending.push_back(o);
		auto it = s->deleting.find(ra->info_hashes);
		if (it == s->deleting.end()) return;
		if (!it->second.files_deleted) {
			it->second.removed = true;
			return;
		}
		s->pending.push_back(*it->second.files_deleted);
		s->deleting.erase(it);
		return;
	}
	case lt::torrent_error_alert::alert_type: {
		auto* te = static_cast<lt::torrent_error_alert*>(a);
		s->rechecking.erase(te->handle);
		o.kind = LTS_ALERT_TORRENT_FAILED;
		o.torrent = torrent(te);
		fill(&o.error, te->error);
		break;
	}
	case lt::tracker_announce_alert::alert_type: {
		auto* ta = static_cast<lt::tracker_announce_alert*>(a);
		o.kind = LTS_ALERT_TRACKER_ANNOUNCED;
		o.torrent = torrent(ta);
		if (o.torrent == 0) return;
		o.tracker = tracker_index(ta->handle, ta->tracker_url());
		o.event = announce_event(ta->event);
		o.at_ms = unix_ms(ta->timestamp());
		break;
	}
	case lt::tracker_reply_alert::alert_type: {
		auto* tr = static_cast<lt::tracker_reply_alert*>(a);
		settle(s, tr->handle);
		o.kind = LTS_ALERT_TRACKER_REPLIED;
		o.torrent = torrent(tr);
		if (o.torrent == 0) return;
		o.tracker = tracker_index(tr->handle, tr->tracker_url());
		o.at_ms = unix_ms(tr->timestamp());
		break;
	}
	case lt::tracker_warning_alert::alert_type: {
		auto* tw = static_cast<lt::tracker_warning_alert*>(a);
		o.kind = LTS_ALERT_TRACKER_WARNED;
		o.torrent = torrent(tw);
		if (o.torrent == 0) return;
		o.tracker = tracker_index(tw->handle, tw->tracker_url());
		set_text(&o, tw->warning_message());
		break;
	}
	case lt::tracker_error_alert::alert_type: {
		auto* te = static_cast<lt::tracker_error_alert*>(a);
		settle(s, te->handle);
		o.kind = LTS_ALERT_TRACKER_FAILED;
		o.torrent = torrent(te);
		if (o.torrent == 0) return;
		o.tracker = tracker_index(te->handle, te->tracker_url());
		o.times = te->times_in_row;
		o.at_ms = unix_ms(te->timestamp());
		fill(&o.error, te->error);
		set_text(&o, te->failure_reason());
		break;
	}
	case lt::tracker_list_alert::alert_type: {
		auto* tl = static_cast<lt::tracker_list_alert*>(a);
		o.kind = LTS_ALERT_TRACKER_LIST;
		o.torrent = torrent(tl);
		if (o.torrent == 0) return;
		tracker_list(&o, tl->trackers);
		break;
	}
	case lt::listen_failed_alert::alert_type:
		o.kind = LTS_ALERT_LISTEN_FAILED;
		fill(&o.error, static_cast<lt::listen_failed_alert*>(a)->error);
		s->pending.push_back(o);
		return;
	case lt::alerts_dropped_alert::alert_type:
		o.kind = LTS_ALERT_DROPPED;
		s->resync = true;
		s->pending.push_back(o);
		return;
	case lt::torrent_paused_alert::alert_type: {
		auto* tp = static_cast<lt::torrent_paused_alert*>(a);
		o.kind = LTS_ALERT_PAUSED;
		o.torrent = torrent(tp);
		// libtorrent dials a peer it dropped again only a minute later, so
		// the peers a pause dropped would be out of reach of a resume that
		// soon. A paused torrent forgets them: its resume's started announce
		// lists them again.
		if (o.torrent != 0 && (tp->handle.flags() & lt::torrent_flags::paused)) tp->handle.clear_peers();
		if (o.torrent != 0 && s->rechecking.erase(tp->handle) != 0) {
			lts_alert checked{};
			checked.kind = LTS_ALERT_CHECKED;
			checked.torrent = o.torrent;
			s->pending.push_back(checked);
		}
		break;
	}
	case lt::file_error_alert::alert_type: {
		auto* fe = static_cast<lt::file_error_alert*>(a);
		o.kind = LTS_ALERT_FILE_ERROR;
		o.torrent = torrent(fe);
		fill(&o.error, fe->error);
		// libtorrent only stops downloading a torrent whose write found the
		// disk full, and never starts it again unless it is auto-managed.
		if (o.torrent != 0 && o.error.category == LTS_CATEGORY_SHIM && o.error.code == LTS_SHIM_DISK_FULL) turn_to(s, fe->handle, turn::pause);
		break;
	}
	case lt::hash_failed_alert::alert_type:
		o.kind = LTS_ALERT_HASH_FAILED;
		o.torrent = torrent(static_cast<lt::torrent_alert*>(a));
		break;
	case lt::incoming_connection_alert::alert_type:
		o.kind = LTS_ALERT_INCOMING_PEER;
		set_text(&o, static_cast<lt::incoming_connection_alert*>(a)->endpoint.address().to_string().c_str());
		s->pending.push_back(o);
		return;
	case lt::session_stats_alert::alert_type: {
		static int const sent = lt::find_metric_idx("net.sent_payload_bytes");
		static int const recv = lt::find_metric_idx("net.recv_payload_bytes");
		auto* ss = static_cast<lt::session_stats_alert*>(a);
		lt::span<std::int64_t const> const counters = ss->counters();
		auto* payload = static_cast<int64_t*>(std::malloc(2 * sizeof(int64_t)));
		if (payload == nullptr) throw std::bad_alloc();
		payload[0] = counters[sent];
		payload[1] = counters[recv];
		o.kind = LTS_ALERT_STATS;
		o.payload = payload;
		o.payload_len = 2;
		o.at_ms = unix_ms(ss->timestamp());
		s->pending.push_back(o);
		return;
	}
	case lt::torrent_resumed_alert::alert_type:
		o.kind = LTS_ALERT_RESUMED;
		o.torrent = torrent(static_cast<lt::torrent_alert*>(a));
		break;
	case lt::save_resume_data_alert::alert_type: {
		auto* sr = static_cast<lt::save_resume_data_alert*>(a);
		o.kind = LTS_ALERT_RESUME_SAVED;
		o.torrent = torrent(sr);
		if (o.torrent == 0) return;
		std::vector<char> const buf = lt::write_resume_data_buf(sr->params);
		o.payload = std::malloc(buf.size());
		if (o.payload == nullptr) throw std::bad_alloc();
		std::memcpy(o.payload, buf.data(), buf.size());
		o.payload_len = int64_t(buf.size());
		break;
	}
	case lt::save_resume_data_failed_alert::alert_type: {
		auto* sf = static_cast<lt::save_resume_data_failed_alert*>(a);
		o.kind = LTS_ALERT_RESUME_FAILED;
		o.torrent = torrent(sf);
		fill(&o.error, sf->error);
		break;
	}
	case lt::torrent_deleted_alert::alert_type:
	case lt::torrent_delete_failed_alert::alert_type: {
		lt::info_hash_t ih;
		lt::error_code failed;
		if (a->type() == lt::torrent_deleted_alert::alert_type) {
			ih = static_cast<lt::torrent_deleted_alert*>(a)->info_hashes;
		} else {
			auto* df = static_cast<lt::torrent_delete_failed_alert*>(a);
			ih = df->info_hashes;
			failed = df->error;
		}
		auto it = s->deleting.find(ih);
		if (it == s->deleting.end()) return;
		o.kind = LTS_ALERT_FILES_DELETED;
		o.torrent = it->second.id;
		if (failed) fill(&o.error, failed);
		else remove_save_path(it->second.save_path, &o.error);
		if (!it->second.removed) {
			it->second.files_deleted = o;
			return;
		}
		s->deleting.erase(it);
		break;
	}
	case lt::state_update_alert::alert_type:
		s->pending.push_back(updates(s, static_cast<lt::state_update_alert*>(a)->status));
		return;
	default:
		return;
	}
	if (o.torrent != 0) s->pending.push_back(o);
}

// every_peer_global puts every peer in the global peer class, whose rate
// limits the caller sets. libtorrent's default exempts peers on local
// networks and loopback from them.
lt::ip_filter every_peer_global() {
	std::uint32_t const global = 1u << static_cast<std::uint32_t>(lt::session::global_peer_class_id);
	lt::ip_filter f;
	f.add_rule(lt::make_address("0.0.0.0"), lt::make_address("255.255.255.255"), global);
	f.add_rule(lt::make_address("::"), lt::make_address("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"), global);
	return f;
}

std::string listen_interface(char const* address, uint16_t port) {
	if (*address == '\0') return {};
	lt::address const addr = lt::make_address(address);
	std::string host = addr.to_string();
	if (addr.is_v6()) host = "[" + host + "]";
	return host + ":" + std::to_string(port);
}

}

extern "C" {

lts_session* lts_open(char const* address, uint16_t port, uint32_t discovery, int32_t announce_floor_s, int32_t peers_share_address, lts_error* err) {
	lts_session* s = nullptr;
	guard(err, [&] {
		sp p = lt::high_performance_seed();
		for (auto const& kv : preset_ints) p.set_int(kv.name, kv.value);
		for (auto const& kv : preset_bools) p.set_bool(kv.name, kv.value);
		p.set_bool(sp::enable_dht, discovery & LTS_DISCOVERY_DHT);
		p.set_bool(sp::enable_lsd, discovery & LTS_DISCOVERY_LSD);
		p.set_bool(sp::enable_upnp, discovery & LTS_DISCOVERY_UPNP);
		p.set_bool(sp::enable_natpmp, discovery & LTS_DISCOVERY_NATPMP);
		p.set_int(sp::min_announce_interval, announce_floor_s);
		p.set_bool(sp::allow_multiple_connections_per_ip, peers_share_address != 0);
		p.set_str(sp::listen_interfaces, listen_interface(address, port));
		p.set_int(sp::max_retry_port_bind, 0);
		p.set_bool(sp::listen_system_port_fallback, false);
		p.set_int(sp::alert_mask, alert_mask(false));
		lt::session_params params(std::move(p));
#ifdef _WIN32
		// mmap_disk_io flushes each mapped file as it closes it on Windows, so
		// a session took seconds to close, and it wrote a sixth as fast.
		params.disk_io_constructor = lt::posix_disk_io_constructor;
#endif
		auto opened = std::make_unique<lts_session>(std::move(params));
		opened->ses.set_peer_class_filter(every_peer_global());
		s = opened.release();
	});
	return s;
}

int lts_configure(lts_session* s, lts_limits const* l, lts_error* err) {
	return guard(err, [&] {
		sp p;
		p.set_int(sp::download_rate_limit, int(l->download_limit));
		p.set_int(sp::upload_rate_limit, int(l->upload_limit));
		p.set_int(sp::connections_limit, l->connections);
		p.set_bool(sp::enable_outgoing_utp, l->utp != 0);
		p.set_bool(sp::enable_incoming_utp, l->utp != 0);
		p.set_int(sp::alert_mask, alert_mask(l->watch_incoming != 0));
		s->ses.apply_settings(std::move(p));
	});
}

int lts_identity(lts_session* s, char* fingerprint, size_t fingerprint_len, char* user_agent, size_t user_agent_len, lts_error* err) {
	return guard(err, [&] {
		sp const p = s->ses.get_settings();
		auto const copy = [](std::string const& from, char* to, size_t n) {
			if (from.size() >= n) throw std::length_error("identity longer than its buffer");
			std::memcpy(to, from.c_str(), from.size() + 1);
		};
		copy(p.get_str(sp::peer_fingerprint), fingerprint, fingerprint_len);
		copy(p.get_str(sp::user_agent), user_agent, user_agent_len);
	});
}

int lts_listen_on(lts_session* s, char const* address, uint16_t port, lts_error* err) {
	return guard(err, [&] {
		sp p;
		p.set_str(sp::listen_interfaces, listen_interface(address, port));
		s->ses.apply_settings(std::move(p));
	});
}

int lts_listen_port(lts_session* s, uint16_t* port, lts_error* err) {
	return guard(err, [&] { *port = s->ses.listen_port(); });
}

int lts_add(lts_session* s, lts_add_params const* p, int64_t* torrent, lts_error* err) {
	return guard(err, [&] {
		lt::error_code ec;
		lt::add_torrent_params atp;
		if (p->resume != nullptr) {
			atp = lt::read_resume_data(lt::span<char const>(p->resume, std::ptrdiff_t(p->resume_len)), ec);
			check(ec);
		} else {
			atp.ti = std::make_shared<lt::torrent_info>(lt::span<char const>(p->metainfo, std::ptrdiff_t(p->metainfo_len)), ec, lt::from_span);
			check(ec);
		}
		atp.save_path = p->save_path;
		atp.flags &= ~lt::torrent_flags::auto_managed;
		atp.flags |= lt::torrent_flags::paused;
		if (p->seed_mode) atp.flags |= lt::torrent_flags::seed_mode;
		atp.total_uploaded = std::max(atp.total_uploaded, p->base_uploaded);
		atp.total_downloaded = std::max(atp.total_downloaded, p->base_downloaded);
		atp.active_time = std::max(atp.active_time, int(p->base_active_s));
		atp.seeding_time = std::max(atp.seeding_time, int(p->base_seeding_s));
		std::lock_guard<std::mutex> l(s->mu);
		int64_t id = s->next++;
		atp.userdata = lt::client_data_t(reinterpret_cast<lts_id*>(intptr_t(id)));
		lt::torrent_handle h = s->ses.add_torrent(std::move(atp), ec);
		check(ec);
		// Adding a parting torrent again returns its handle: it stays, under
		// the id it had, and pauses as any add does.
		if (auto pt = s->parting.find(h); pt != s->parting.end()) {
			id = pt->second.id;
			s->parting.erase(pt);
			turn_to(s, h, turn::pause);
		}
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

int lts_pause(lts_session* s, int64_t torrent, lts_error* err) {
	return guard(err, [&] {
		lt::torrent_handle h = find(s, torrent);
		std::lock_guard<std::mutex> l(s->mu);
		turn_to(s, h, turn::pause);
	});
}

int lts_resume(lts_session* s, int64_t torrent, lts_error* err) {
	return guard(err, [&] {
		lt::torrent_handle h = find(s, torrent);
		std::lock_guard<std::mutex> l(s->mu);
		turn_to(s, h, turn::resume);
	});
}

int lts_recheck(lts_session* s, int64_t torrent, lts_error* err) {
	return guard(err, [&] {
		lt::torrent_handle h = find(s, torrent);
		std::lock_guard<std::mutex> l(s->mu);
		s->rechecking.insert(h);
		// A check that finds no files ends at once, paused or not. Were it to
		// end before resume ran, it would spend stop_when_ready on a paused
		// torrent with no pause to post, and resume would then run it. Run on
		// the network thread, these calls finish before the check can.
		boost::asio::post(s->ses.get_context(), [h] {
			if (!h.is_valid()) return;
			h.force_recheck();
			h.set_flags(lt::torrent_flags::stop_when_ready);
			h.unset_flags(lt::torrent_flags::upload_mode);
			h.resume();
		});
	});
}

int lts_reannounce(lts_session* s, int64_t torrent, int32_t tracker, int32_t delay_s, int32_t ignore_min_interval, lts_error* err) {
	return guard(err, [&] {
		lt::reannounce_flags_t const flags = ignore_min_interval != 0 ? lt::torrent_handle::ignore_min_interval : lt::reannounce_flags_t{};
		find(s, torrent).force_reannounce(delay_s, tracker, flags);
	});
}

int lts_post_trackers(lts_session* s, int64_t torrent, lts_error* err) {
	return guard(err, [&] { find(s, torrent).post_trackers(); });
}

int lts_save_resume(lts_session* s, int64_t torrent, lts_error* err) {
	return guard(err, [&] {
		find(s, torrent).save_resume_data(lt::torrent_handle::flush_disk_cache | lt::torrent_handle::save_info_dict);
	});
}

int lts_post_updates(lts_session* s, lts_error* err) {
	return guard(err, [&] {
		std::unique_lock<std::mutex> l(s->mu);
		if (!s->resync) {
			l.unlock();
			s->ses.post_torrent_updates();
			return;
		}
		s->resync = false;
		s->pending.push_back(updates(s, s->ses.get_torrent_status([](lt::torrent_status const&) { return true; })));
	});
}

int lts_post_stats(lts_session* s, lts_error* err) {
	return guard(err, [&] { s->ses.post_session_stats(); });
}

int lts_remove(lts_session* s, int64_t torrent, int32_t delete_files, lts_error* err) {
	return guard(err, [&] {
		lt::torrent_handle h = find(s, torrent);
		std::string save_path;
		if (delete_files) save_path = h.status(lt::torrent_handle::query_save_path).save_path;
		std::lock_guard<std::mutex> l(s->mu);
		s->by_handle.erase(h);
		s->by_id.erase(torrent);
		s->turning.erase(h);
		s->rechecking.erase(h);
		s->parting[h] = parting_torrent{torrent, delete_files != 0, save_path};
		settle(s, h);
	});
}

int lts_file_progress(lts_session* s, int64_t torrent, int64_t** done, int32_t* count, lts_error* err) {
	return guard(err, [&] {
		lt::torrent_handle h = find(s, torrent);
		std::shared_ptr<lt::torrent_info const> ti = h.torrent_file();
		lt::file_storage const& fs = ti->files();
		std::vector<std::int64_t> progress = h.file_progress();
		auto* out = static_cast<int64_t*>(std::calloc(std::size_t(fs.num_files()) + 1, sizeof(int64_t)));
		if (out == nullptr) throw std::bad_alloc();
		int32_t n = 0;
		for (lt::file_index_t i : fs.file_range()) {
			if (!fs.pad_file_at(i)) out[n++] = progress[std::size_t(static_cast<int>(i))];
		}
		*done = out;
		*count = n;
	});
}

void lts_free(void* p) {
	std::free(p);
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
		bool empty;
		{
			std::lock_guard<std::mutex> l(s->mu);
			empty = s->pending.empty();
		}
		if (empty && s->ses.wait_for_alert(lt::milliseconds(timeout_ms))) {
			s->ses.pop_alerts(&s->popped);
			std::lock_guard<std::mutex> l(s->mu);
			for (lt::alert* a : s->popped) convert(s, a);
		}
		std::lock_guard<std::mutex> l(s->mu);
		for (; n < max && !s->pending.empty(); ++n) {
			out[n] = s->pending.front();
			s->pending.pop_front();
		}
	});
	if (rc < 0) {
		// The batch being converted is lost, which the caller reports as a drop.
		std::lock_guard<std::mutex> l(s->mu);
		s->resync = true;
		return -1;
	}
	return n;
}

void lts_free_alert(lts_alert* a) {
	std::free(a->payload);
	a->payload = nullptr;
}

int lts_close(lts_session* s, lts_error* err) {
	int rc = guard(err, [&] {
		// Each torrent's next regular announce moves an announce floor away
		// first, so none starts between settle seeing no tracker awaits an
		// answer and the pause it posts.
		int const floor = s->ses.get_settings().get_int(sp::min_announce_interval);
		{
			std::lock_guard<std::mutex> l(s->mu);
			for (lt::torrent_handle const& h : s->ses.get_torrents()) {
				h.force_reannounce(floor);
				if (s->parting.count(h) == 0) turn_to(s, h, turn::pause);
			}
		}
		// An announce ends within tracker_completion_timeout, and its answer
		// wakes the wait sooner.
		auto const answer = lt::seconds(s->ses.get_settings().get_int(sp::tracker_completion_timeout));
		for (;;) {
			{
				std::lock_guard<std::mutex> l(s->mu);
				std::vector<lt::torrent_handle> waiting;
				for (auto const& t : s->turning) waiting.push_back(t.first);
				for (auto const& p : s->parting) waiting.push_back(p.first);
				for (lt::torrent_handle const& h : waiting) settle(s, h);
				if (s->turning.empty() && s->parting.empty()) break;
			}
			s->ses.wait_for_alert(answer);
			s->ses.pop_alerts(&s->popped);
		}
		lt::session_proxy proxy = s->ses.abort();
	});
	for (lts_alert& a : s->pending) lts_free_alert(&a);
	delete s;
	return rc;
}

char const* lts_version(void) {
	return lt::version();
}

}
