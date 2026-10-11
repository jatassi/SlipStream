mod lt;

use serde::Deserialize;
use serde_json::{json, Value};
use std::collections::HashMap;
use std::fs::OpenOptions;
use std::io::Write;
use std::path::Path;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{mpsc, Arc, Mutex};
use std::thread;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

#[derive(Deserialize)]
struct Entry {
    name: String,
    torrent: String,
    data: String,
}

struct Status(Mutex<Option<std::fs::File>>);

impl Status {
    fn emit(&self, event: &str, extra: Value) {
        let t = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs_f64();
        let mut m = json!({"t": t, "event": event});
        if let Value::Object(x) = extra {
            m.as_object_mut().unwrap().extend(x);
        }
        let line = m.to_string();
        eprintln!("{line}");
        if let Some(f) = self.0.lock().unwrap().as_mut() {
            writeln!(f, "{line}").unwrap();
        }
    }

    fn fatal(&self, err: String) -> ! {
        self.emit("error", json!({"error": err}));
        std::process::exit(1)
    }
}

fn main() {
    let mut args: HashMap<String, String> = HashMap::new();
    let argv: Vec<String> = std::env::args().skip(1).collect();
    let mut i = 0;
    while i < argv.len() {
        let k = argv[i].trim_start_matches('-').to_string();
        if k == "v" {
            args.insert(k, "1".into());
            i += 1;
        } else {
            args.insert(k, argv.get(i + 1).cloned().unwrap_or_default());
            i += 2;
        }
    }
    let arg = |k: &str, d: &str| args.get(k).cloned().unwrap_or_else(|| d.to_string());
    let mode = arg("mode", "seed");
    let verbose = args.contains_key("v");

    let status = Arc::new(Status(Mutex::new(args.get("status").map(|p| {
        OpenOptions::new().create(true).append(true).open(p).unwrap()
    }))));

    let fixtures = arg("fixtures", "");
    let raw = std::fs::read(Path::new(&fixtures).join("manifest.json")).unwrap_or_else(|e| status.fatal(e.to_string()));
    let man: Vec<Entry> = serde_json::from_slice(&raw).unwrap_or_else(|e| status.fatal(e.to_string()));

    let mut kv = vec![
        ("listen_interfaces", format!("127.0.0.1:{}", arg("listen-port", "6881"))),
        ("ssrf_mitigation", "false".to_string()),
    ];
    if let Some(p) = args.get("peer-id-prefix") {
        kv.push(("peer_fingerprint", p.clone()));
    }
    let s = Arc::new(lt::Session::new(&kv).unwrap_or_else(|e| status.fatal(e)));

    let stop = Arc::new(AtomicBool::new(false));
    let (tx, rx) = mpsc::channel::<lt::Alert>();
    let forward = mode == "lifecycle";
    let poller = {
        let (s, stop, status) = (s.clone(), stop.clone(), status.clone());
        thread::spawn(move || {
            let mut buf = vec![unsafe { std::mem::zeroed::<lt::LtsAlert>() }; 64];
            while !stop.load(Ordering::Relaxed) {
                for a in s.poll(&mut buf, 200) {
                    if a.kind == lt::ALERT_ERROR || a.kind == lt::ALERT_TRACKER_ERROR {
                        status.emit("error", json!({"alert": a.what, "torrent": a.torrent, "message": a.message}));
                    } else if verbose {
                        eprintln!("alert {} t={} {}", a.what, a.torrent, a.message);
                    }
                    if forward && (lt::ALERT_ADDED..=lt::ALERT_REMOVED).contains(&a.kind) {
                        let _ = tx.send(a);
                    }
                }
            }
        })
    };
    let shutdown = move |s: Arc<lt::Session>| {
        stop.store(true, Ordering::Relaxed);
        poller.join().unwrap();
        Arc::into_inner(s).unwrap().close();
    };

    match mode.as_str() {
        "seed" => seed(s, &man, &status, shutdown),
        "lifecycle" => {
            let idx: usize = arg("index", "0").parse().unwrap();
            lifecycle(s, &man[idx], &arg("data-dir", ""), rx, &status, shutdown)
        }
        m => status.fatal(format!("unknown mode {m}")),
    }
}

fn seed(s: Arc<lt::Session>, man: &[Entry], status: &Status, shutdown: impl FnOnce(Arc<lt::Session>)) {
    let mut signals = signal_hook::iterator::Signals::new([signal_hook::consts::SIGTERM, signal_hook::consts::SIGINT]).unwrap();
    let start = Instant::now();
    let mut ts = Vec::with_capacity(man.len());
    for e in man {
        let dir = Path::new(&e.data).parent().unwrap().to_str().unwrap();
        match s.add(&e.torrent, dir, true) {
            Ok(t) => ts.push(t),
            Err(err) => status.emit("error", json!({"torrent": e.name, "error": err})),
        }
    }
    for &t in &ts {
        while matches!(s.status(t), Ok(st) if st.is_seeding == 0) {
            thread::sleep(Duration::from_millis(5));
        }
    }
    status.emit("all_added", json!({"elapsed": start.elapsed().as_secs_f64(), "count": ts.len()}));
    signals.forever().next();
    let stop_at = Instant::now();
    shutdown(s);
    status.emit("shutdown_done", json!({"elapsed": stop_at.elapsed().as_secs_f64()}));
}

fn lifecycle(s: Arc<lt::Session>, e: &Entry, data_dir: &str, rx: mpsc::Receiver<lt::Alert>, status: &Status, shutdown: impl FnOnce(Arc<lt::Session>)) {
    let start = Instant::now();
    let step = |name: &str, extra: Value| {
        let mut m = json!({"step": name, "elapsed": start.elapsed().as_secs_f64()});
        if let Value::Object(x) = extra {
            m.as_object_mut().unwrap().extend(x);
        }
        status.emit("step", m);
    };
    let wait_for = |t: i64, kind: i32| {
        while let Ok(a) = rx.recv() {
            if a.torrent == t && a.kind == kind {
                return;
            }
        }
    };
    let t = s.add(&e.torrent, data_dir, false).unwrap_or_else(|err| status.fatal(err));
    step("added", json!({}));
    wait_for(t, lt::ALERT_FINISHED);
    let done = s.status(t).map(|st| st.total_done).unwrap_or(-1);
    step("completed", json!({"total_done": done}));
    thread::sleep(Duration::from_secs(45));
    s.pause(t).unwrap_or_else(|err| status.fatal(err));
    wait_for(t, lt::ALERT_PAUSED);
    step("paused", json!({}));
    thread::sleep(Duration::from_secs(45));
    s.resume(t).unwrap_or_else(|err| status.fatal(err));
    wait_for(t, lt::ALERT_RESUMED);
    step("resumed", json!({}));
    thread::sleep(Duration::from_secs(45));
    s.remove(t).unwrap_or_else(|err| status.fatal(err));
    wait_for(t, lt::ALERT_REMOVED);
    step("removed", json!({}));
    thread::sleep(Duration::from_secs(10));
    shutdown(s);
    step("shutdown", json!({}));
}
