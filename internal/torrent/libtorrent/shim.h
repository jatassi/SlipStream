#ifndef SLIPSTREAM_LIBTORRENT_SHIM_H
#define SLIPSTREAM_LIBTORRENT_SHIM_H

#ifndef SLIPSTREAM_NATIVE_BUNDLE
#error "native bundle missing: run scripts/native-deps fetch and eval \"$(scripts/native-deps env)\""
#endif

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct lts_session lts_session;

enum {
	LTS_CATEGORY_SHIM = 1,
	LTS_CATEGORY_SYSTEM,
	LTS_CATEGORY_LIBTORRENT,
	LTS_CATEGORY_BDECODE,
	LTS_CATEGORY_OTHER,
	LTS_CATEGORY_EXCEPTION
};

/* Shim codes mirror engine.Code* in internal/torrent/engine. */
enum {
	LTS_SHIM_UNKNOWN_TORRENT = 1
};

enum {
	LTS_DISCOVERY_DHT = 1 << 0,
	LTS_DISCOVERY_LSD = 1 << 1,
	LTS_DISCOVERY_UPNP = 1 << 2,
	LTS_DISCOVERY_NATPMP = 1 << 3
};

enum {
	LTS_STATE_CHECKING_FILES = 1,
	LTS_STATE_DOWNLOADING_METADATA,
	LTS_STATE_DOWNLOADING,
	LTS_STATE_FINISHED,
	LTS_STATE_SEEDING,
	LTS_STATE_CHECKING_RESUME_DATA
};

enum {
	LTS_ALERT_ADDED = 1,
	LTS_ALERT_STATE_CHANGED,
	LTS_ALERT_FINISHED,
	LTS_ALERT_REMOVED,
	LTS_ALERT_TORRENT_FAILED,
	LTS_ALERT_LISTEN_FAILED,
	LTS_ALERT_DROPPED,
	LTS_ALERT_TRACKER_REPLIED,
	LTS_ALERT_TRACKER_FAILED,
	LTS_ALERT_PAUSED,
	LTS_ALERT_RESUMED,
	LTS_ALERT_RESUME_SAVED,
	LTS_ALERT_RESUME_FAILED,
	LTS_ALERT_FILES_DELETED,
	LTS_ALERT_UPDATES
};

/* A category of 0 is no error. */
typedef struct {
	int32_t category;
	int32_t code;
	char text[256];
} lts_error;

typedef struct {
	int64_t torrent;
	int32_t state;
	int32_t paused;
	int64_t done;
	int64_t wanted;
	int64_t down_bps;
	int64_t up_bps;
	int32_t peers;
	int32_t seeds;
	int64_t uploaded;
	int64_t downloaded;
	int64_t active_s;
	int64_t seeding_s;
	/* Unix milliseconds, 0 for never. */
	int64_t last_download_ms;
	int32_t need_save;
	lts_error error;
} lts_status;

typedef struct {
	const char* metainfo;
	size_t metainfo_len;
	/* NULL for none; the info dictionary then comes from metainfo. */
	const char* resume;
	size_t resume_len;
	const char* save_path;
	int32_t seed_mode;
	int64_t base_uploaded;
	int64_t base_downloaded;
	int64_t base_active_s;
	int64_t base_seeding_s;
} lts_add_params;

typedef struct {
	int32_t index;
	int64_t size;
	char* path;
} lts_file;

/* payload is shim-owned until lts_free_alert: the bytes of RESUME_SAVED, or
   payload_len lts_status of UPDATES. */
typedef struct {
	int32_t kind;
	int32_t state;
	int64_t torrent;
	lts_error error;
	void* payload;
	int64_t payload_len;
} lts_alert;

/* Functions that return int return 0 on success and -1 with *err filled. */
/* announce_floor_s is min_announce_interval, the least interval between two
   regular announces to a tracker. */
lts_session* lts_open(const char* address, uint16_t port, uint32_t discovery, int outgoing_utp, int32_t announce_floor_s, lts_error* err);
int lts_listen_port(lts_session* s, uint16_t* port, lts_error* err);
/* Adds the torrent paused and not auto-managed, whatever resume data says. */
int lts_add(lts_session* s, const lts_add_params* p, int64_t* torrent, lts_error* err);
int lts_connect_peer(lts_session* s, int64_t torrent, const char* address, uint16_t port, lts_error* err);
int lts_status_get(lts_session* s, int64_t torrent, lts_status* out, lts_error* err);
/* Pauses gracefully once no tracker awaits its answer to started. */
int lts_pause(lts_session* s, int64_t torrent, lts_error* err);
int lts_resume(lts_session* s, int64_t torrent, lts_error* err);
int lts_save_resume(lts_session* s, int64_t torrent, lts_error* err);
/* Posts UPDATES with the torrents that changed, or with every torrent after
   DROPPED. */
int lts_post_updates(lts_session* s, lts_error* err);
/* Removes once no tracker awaits its answer to started. With delete_files it
   deletes the torrent's files, then its save path if left empty, and posts
   FILES_DELETED after REMOVED. */
int lts_remove(lts_session* s, int64_t torrent, int32_t delete_files, lts_error* err);
/* Bytes done per file, in lts_torrent_files' order. Free *done with lts_free. */
int lts_file_progress(lts_session* s, int64_t torrent, int64_t** done, int32_t* count, lts_error* err);
void lts_free(void* p);
/* Lists the torrent's files but its pad files, with paths relative to the
   save path as libtorrent sanitized them. Free *files with lts_free_files. */
int lts_torrent_files(lts_session* s, int64_t torrent, lts_file** files, int32_t* count, lts_error* err);
void lts_free_files(lts_file* files, int32_t count);
/* Single consumer. Returns the number of alerts copied into out, or -1. */
int lts_wait_alerts(lts_session* s, int timeout_ms, lts_alert* out, int max, lts_error* err);
void lts_free_alert(lts_alert* a);
int lts_close(lts_session* s, lts_error* err);
const char* lts_version(void);

#ifdef __cplusplus
}
#endif

#endif
