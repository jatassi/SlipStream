#ifndef LTS_H
#define LTS_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct lts_session lts_session;
typedef int64_t lts_torrent;

enum {
	LTS_PRESET_DEFAULT = 0,
	LTS_PRESET_SEEDBOX = 1
};

enum {
	LTS_ALERT_OTHER = 0,
	LTS_ALERT_ADDED,
	LTS_ALERT_FINISHED,
	LTS_ALERT_PAUSED,
	LTS_ALERT_RESUMED,
	LTS_ALERT_REMOVED,
	LTS_ALERT_TRACKER_ANNOUNCE,
	LTS_ALERT_TRACKER_REPLY,
	LTS_ALERT_TRACKER_ERROR,
	LTS_ALERT_ERROR
};

typedef struct {
	int32_t type;
	int32_t lt_type;
	lts_torrent torrent;
	char what[48];
	char message[456];
} lts_alert;

typedef struct {
	int32_t state;
	int32_t paused;
	int32_t is_seeding;
	int32_t is_finished;
	int32_t num_peers;
	int32_t num_seeds;
	int32_t upload_rate;
	int32_t download_rate;
	float progress;
	int64_t total_done;
	int64_t total_wanted;
	int64_t all_time_upload;
	int64_t all_time_download;
} lts_status;

lts_session* lts_session_new(int preset, const char* const* keys, const char* const* values, int n, char* err, int errlen);
lts_torrent lts_add(lts_session* s, const char* torrent_file, const char* save_path, int seed_mode, char* err, int errlen);
int lts_pause(lts_session* s, lts_torrent t);
int lts_resume(lts_session* s, lts_torrent t);
int lts_remove(lts_session* s, lts_torrent t);
int lts_status_get(lts_session* s, lts_torrent t, lts_status* out);
int lts_poll_alerts(lts_session* s, lts_alert* out, int max, int timeout_ms);
void lts_close(lts_session* s);

#ifdef __cplusplus
}
#endif

#endif
