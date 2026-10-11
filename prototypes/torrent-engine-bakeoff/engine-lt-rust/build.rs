fn main() {
    let shim = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../lt-shim");
    println!("cargo:rerun-if-changed={}", shim.join("liblts.a").display());
    println!("cargo:rustc-link-search=native={}", shim.display());
    println!("cargo:rustc-link-lib=static=lts");
    for lib in ["torrent-rasterbar", "ssl", "crypto", "stdc++"] {
        println!("cargo:rustc-link-lib=dylib={lib}");
    }
}
