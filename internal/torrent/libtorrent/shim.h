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
	LTS_ALERT_DROPPED
};

typedef struct {
	int32_t category;
	int32_t code;
	char text[256];
} lts_error;

typedef struct {
	int32_t state;
	int32_t peers;
	int64_t done;
	int64_t wanted;
} lts_status;

typedef struct {
	int32_t kind;
	int32_t state;
	int64_t torrent;
	lts_error error;
} lts_alert;

/* Functions that return int return 0 on success and -1 with *err filled. */
/* Port 0 picks a port both TCP and UDP can bind on address. */
lts_session* lts_open(const char* address, uint16_t port, uint32_t discovery, lts_error* err);
int lts_listen_port(lts_session* s, uint16_t* port, lts_error* err);
int lts_add(lts_session* s, const char* metainfo, size_t len, const char* save_path, int seed_mode, int64_t* torrent, lts_error* err);
int lts_connect_peer(lts_session* s, int64_t torrent, const char* address, uint16_t port, lts_error* err);
int lts_status_get(lts_session* s, int64_t torrent, lts_status* out, lts_error* err);
int lts_remove(lts_session* s, int64_t torrent, lts_error* err);
/* Single consumer. Returns the number of alerts copied into out, or -1. */
int lts_wait_alerts(lts_session* s, int timeout_ms, lts_alert* out, int max, lts_error* err);
int lts_close(lts_session* s, lts_error* err);
const char* lts_version(void);

#ifdef __cplusplus
}
#endif

#endif
