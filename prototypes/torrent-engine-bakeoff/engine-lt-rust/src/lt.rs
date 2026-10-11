use std::ffi::{c_char, c_int, CStr, CString};

#[repr(C)]
pub struct LtsSession {
    _p: [u8; 0],
}

pub const PRESET_SEEDBOX: c_int = 1;
pub const ALERT_ADDED: i32 = 1;
pub const ALERT_FINISHED: i32 = 2;
pub const ALERT_PAUSED: i32 = 3;
pub const ALERT_RESUMED: i32 = 4;
pub const ALERT_REMOVED: i32 = 5;
pub const ALERT_TRACKER_ERROR: i32 = 8;
pub const ALERT_ERROR: i32 = 9;

#[repr(C)]
#[derive(Clone, Copy)]
pub struct LtsAlert {
    pub kind: i32,
    pub lt_type: i32,
    pub torrent: i64,
    pub what: [c_char; 48],
    pub message: [c_char; 456],
}

#[repr(C)]
#[derive(Default, Debug)]
pub struct LtsStatus {
    pub state: i32,
    pub paused: i32,
    pub is_seeding: i32,
    pub is_finished: i32,
    pub num_peers: i32,
    pub num_seeds: i32,
    pub upload_rate: i32,
    pub download_rate: i32,
    pub progress: f32,
    pub total_done: i64,
    pub total_wanted: i64,
    pub all_time_upload: i64,
    pub all_time_download: i64,
}

extern "C" {
    fn lts_session_new(preset: c_int, keys: *const *const c_char, values: *const *const c_char, n: c_int, err: *mut c_char, errlen: c_int) -> *mut LtsSession;
    fn lts_add(s: *mut LtsSession, torrent_file: *const c_char, save_path: *const c_char, seed_mode: c_int, err: *mut c_char, errlen: c_int) -> i64;
    fn lts_pause(s: *mut LtsSession, t: i64) -> c_int;
    fn lts_resume(s: *mut LtsSession, t: i64) -> c_int;
    fn lts_remove(s: *mut LtsSession, t: i64) -> c_int;
    fn lts_status_get(s: *mut LtsSession, t: i64, out: *mut LtsStatus) -> c_int;
    fn lts_poll_alerts(s: *mut LtsSession, out: *mut LtsAlert, max: c_int, timeout_ms: c_int) -> c_int;
    fn lts_close(s: *mut LtsSession);
}

pub struct Session(*mut LtsSession);

unsafe impl Send for Session {}
unsafe impl Sync for Session {}

pub struct Alert {
    pub kind: i32,
    pub torrent: i64,
    pub what: String,
    pub message: String,
}

fn cstr(buf: &[c_char]) -> String {
    unsafe { CStr::from_ptr(buf.as_ptr()) }.to_string_lossy().into_owned()
}

fn rc(r: c_int, op: &str) -> Result<(), String> {
    if r == 0 { Ok(()) } else { Err(format!("{op} failed")) }
}

impl Session {
    pub fn new(kv: &[(&str, String)]) -> Result<Session, String> {
        let keys: Vec<CString> = kv.iter().map(|(k, _)| CString::new(*k).unwrap()).collect();
        let vals: Vec<CString> = kv.iter().map(|(_, v)| CString::new(v.as_str()).unwrap()).collect();
        let kp: Vec<*const c_char> = keys.iter().map(|c| c.as_ptr()).collect();
        let vp: Vec<*const c_char> = vals.iter().map(|c| c.as_ptr()).collect();
        let mut err = [0 as c_char; 256];
        let p = unsafe { lts_session_new(PRESET_SEEDBOX, kp.as_ptr(), vp.as_ptr(), kv.len() as c_int, err.as_mut_ptr(), err.len() as c_int) };
        if p.is_null() { Err(cstr(&err)) } else { Ok(Session(p)) }
    }

    pub fn add(&self, torrent_file: &str, save_path: &str, seed_mode: bool) -> Result<i64, String> {
        let tf = CString::new(torrent_file).unwrap();
        let sp = CString::new(save_path).unwrap();
        let mut err = [0 as c_char; 256];
        let t = unsafe { lts_add(self.0, tf.as_ptr(), sp.as_ptr(), seed_mode as c_int, err.as_mut_ptr(), err.len() as c_int) };
        if t < 0 { Err(cstr(&err)) } else { Ok(t) }
    }

    pub fn pause(&self, t: i64) -> Result<(), String> { rc(unsafe { lts_pause(self.0, t) }, "pause") }
    pub fn resume(&self, t: i64) -> Result<(), String> { rc(unsafe { lts_resume(self.0, t) }, "resume") }
    pub fn remove(&self, t: i64) -> Result<(), String> { rc(unsafe { lts_remove(self.0, t) }, "remove") }

    pub fn status(&self, t: i64) -> Result<LtsStatus, String> {
        let mut st = LtsStatus::default();
        rc(unsafe { lts_status_get(self.0, t, &mut st) }, "status").map(|_| st)
    }

    pub fn poll(&self, buf: &mut [LtsAlert], timeout_ms: i32) -> Vec<Alert> {
        let n = unsafe { lts_poll_alerts(self.0, buf.as_mut_ptr(), buf.len() as c_int, timeout_ms) };
        buf[..n.max(0) as usize]
            .iter()
            .map(|a| Alert { kind: a.kind, torrent: a.torrent, what: cstr(&a.what), message: cstr(&a.message) })
            .collect()
    }

    pub fn close(self) {
        unsafe { lts_close(self.0) }
    }
}
