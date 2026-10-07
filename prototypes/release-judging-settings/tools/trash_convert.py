#!/usr/bin/env python3
"""Convert TRaSH-Guides JSON (custom formats, groups, profiles, sizes) into a SlipStream Knowledge Pack.

Rerunnable: python3 convert.py [--guides PATH]. Stdlib only, plus `node` on PATH for regex-compat.json.
Every claim is classified by the CLASSIFY table below; a CF missing from it fails the run.
"""
import argparse
import collections
import json
import os
import re
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_GUIDES = os.path.join(os.path.dirname(HERE), "trash", "Guides")
APPS = ("radarr", "sonarr")
SOURCE_DATE = "2026-09-28"

# ---------------------------------------------------------------- enum decoding
# Values from Radarr/Sonarr source enums; decode() output is cross-checked against TRaSH spec names.
LANGUAGE = {-2: "Original", -1: "Any", 0: "Unknown", 1: "English", 2: "French", 3: "Spanish", 4: "German",
            5: "Italian", 6: "Danish", 7: "Dutch", 8: "Japanese", 9: "Icelandic", 10: "Chinese", 11: "Russian",
            12: "Polish", 13: "Vietnamese", 14: "Swedish", 15: "Norwegian", 16: "Finnish", 17: "Turkish",
            18: "Portuguese", 19: "Flemish", 20: "Greek", 21: "Korean", 22: "Hungarian", 23: "Hebrew",
            24: "Lithuanian", 25: "Czech"}
ENUMS = {
    "radarr": {
        "SourceSpecification": {0: "UNKNOWN", 1: "CAM", 2: "TELESYNC", 3: "TELECINE", 4: "WORKPRINT", 5: "DVD",
                                6: "TV", 7: "WEBDL", 8: "WEBRIP", 9: "BLURAY"},
        "QualityModifierSpecification": {0: "NONE", 1: "REGIONAL", 2: "SCREENER", 3: "RAWHD", 4: "BRDISK",
                                         5: "REMUX"},
        "IndexerFlagSpecification": {1: "G_Freeleech", 2: "G_Halfleech", 4: "G_DoubleUpload", 8: "PTP_Golden",
                                     16: "PTP_Approved", 32: "G_Internal", 128: "G_Scene", 2048: "Nuked"},
        "LanguageSpecification": LANGUAGE,
    },
    "sonarr": {
        "SourceSpecification": {0: "Unknown", 1: "Television", 2: "TelevisionRaw", 3: "Web", 4: "WebRip", 5: "DVD",
                                6: "Bluray", 7: "BlurayRaw"},
        "IndexerFlagSpecification": {1: "G_Freeleech", 2: "G_Halfleech", 4: "G_DoubleUpload", 8: "G_Internal",
                                     16: "G_Scene", 128: "Nuked"},
        "ReleaseTypeSpecification": {0: "Unknown", 1: "SingleEpisode", 2: "MultiEpisode", 3: "SeasonPack"},
        "LanguageSpecification": LANGUAGE,
    },
}
# Spec-name words that legitimately name a decoded value (after stripping "Not"/"Language").
ENUM_ALIASES = {
    "WEBDL": {"webdl"}, "WEBRIP": {"webrip"}, "BLURAY": {"bluray"}, "DVD": {"dvd"}, "REMUX": {"remux"},
    "Web": {"webdl"}, "WebRip": {"webrip"}, "Bluray": {"bluray"}, "BlurayRaw": {"remux", "blurayremux"},
    "G_Freeleech": {"freeleech"}, "G_Internal": {"internal"},
    "SingleEpisode": {"singleepisode"}, "MultiEpisode": {"multiepisode"}, "SeasonPack": {"seasonpacks"},
    "Original": {"original", "wronglanguage"},
}


def decode(app, impl, value):
    if impl == "ResolutionSpecification":
        return f"{value}p"
    table = ENUMS[app].get(impl)
    return None if table is None else table.get(value, f"?{value}")


def name_matches(decoded, spec_name):
    key = re.sub(r"[^a-z0-9]", "", re.sub(r"(?i)\b(not|language)\b", "", spec_name).lower())
    if decoded.endswith("p") and decoded[:-1].isdigit():
        return key in {decoded.lower(), "4kresolution"}
    return key in ENUM_ALIASES.get(decoded, set()) | {decoded.lower().replace("_", "")}


# ---------------------------------------------------------------- classification table
def term(*pairs, conf="high", why=None):
    sets = [{"key": k, "value": v} for k, v in zip(pairs[::2], pairs[1::2])]
    return {"kind": "term", "sets": sets, "confidence": conf, "why": why}


def hazard(*pairs, conf="high", why=None):
    sets = [{"key": k, "value": v} for k, v in zip(pairs[::2], pairs[1::2])]
    return {"kind": "hazard", "sets": sets, "confidence": conf, "why": why}


def rep(conf="high", why=None):
    return {"kind": "reputation", "confidence": conf, "why": why}


def untranslated(reason):
    return {"kind": None, "reason": reason}


SVC = "service"
LANG_PREF = ("language preference expressed as a reverse-scored LanguageSpecification filter over the parsed "
             "language; it states what the user wants, not a fact about the release, so it belongs in the "
             "user's priority list")
RES_PREF = ("pure resolution boost; resolution is already a parsed Quality dimension that the user's priority "
            "list orders, so a claim would duplicate it")
BOOST = "score lever over facts already claimed by other terms (service/source/resolution); carries no new fact"

CLASSIFY = {
    # audio codecs
    "aac": term("audio", "aac"), "atmos-undefined": term("audio", "atmos_undefined"), "dd": term("audio", "dd"),
    "ddplus": term("audio", "ddp"), "ddplus-atmos": term("audio", "ddp_atmos"), "dts": term("audio", "dts"),
    "dts-es": term("audio", "dts_es"), "dts-hd-hra": term("audio", "dtshd_hra"),
    "dts-hd-ma": term("audio", "dtshd_ma"), "dts-x": term("audio", "dtsx"), "flac": term("audio", "flac"),
    "mp3": term("audio", "mp3"), "opus": term("audio", "opus"), "pcm": term("audio", "pcm"),
    "truehd": term("audio", "truehd"),
    "truehd-atmos": term("audio", "truehd_atmos", conf="medium",
                         why="ATMOS spec also fires on group names (CtrlHD, W4NK3R, HQMUX, DON, 3L) as a proxy "
                             "for untagged Atmos"),
    # audio channels
    "10-mono": term("channels", "1.0"), "20-stereo": term("channels", "2.0"), "30-sound": term("channels", "3.0"),
    "40-sound": term("channels", "4.0"), "51-surround": term("channels", "5.1"),
    "61-surround": term("channels", "6.1"), "71-surround": term("channels", "7.1"),
    # HDR
    "hdr": term("hdr", "hdr_any", conf="high",
                why="TRaSH folds DV-with-fallback, HDR, HDR10, HDR10+, HLG, PQ and known-HDR groups into one "
                    "union CF; the specific format is not distinguishable from the match alone"),
    "dv-boost": term("hdr", "dv"), "hdr10plus-boost": term("hdr", "hdr10p"),
    "dv-disk": term("hdr", "dv", "flag", "dv_disk", conf="medium",
                    why="disc-sourced DV is inferred from one group (FraMeSToR) plus a DV tag, excluding hybrids"),
    "dv-wo-hdr-fallback": hazard("hdr", "dv_no_fallback"),
    "sdr": hazard("hdr", "sdr"), "sdr-no-webdl": hazard("hdr", "sdr"),
    "hlg": hazard("hdr", "hlg", conf="medium",
                  why="standalone CF in no group or profile; TRaSH's description scopes the -10000 to devices "
                      "that cannot play HLG"),
    "generated-dynamic-hdr": hazard("flag", "generated_dynamic_hdr"),
    # editions (radarr movie versions, sonarr series versions)
    "imax": term("edition", "imax"), "imax-enhanced": term("edition", "imax_enhanced"),
    "criterion-collection": term("edition", "criterion", conf="medium",
                                 why="also matches a bare 'CC' token on Bluray/DVD sources"),
    "masters-of-cinema": term("edition", "masters_of_cinema", conf="medium",
                              why="also matches a bare 'MoC' token"),
    "vinegar-syndrome": term("edition", "vinegar_syndrome"),
    "remaster": term("edition", "remaster"), "4k-remaster": term("edition", "remaster_4k"),
    "theatrical-cut": term("edition", "theatrical"), "open-matte": term("edition", "open_matte"),
    "special-edition": term("edition", "special", conf="medium",
                            why="catch-all regex (extended|uncut|directors|special|unrated|uncensored|cut|version"
                                "|edition); TRaSH has no separate extended/directors/unrated CFs"),
    "hybrid": term("flag", "hybrid"),
    # flags
    "internal": term("flag", "internal"),
    "p2p-internal": term("flag", "indexer_internal", conf="certain"),
    "freeleech": term("flag", "freeleech", conf="certain"),
    "hfr": term("flag", "hfr"),
    "uncensored": term("flag", "uncensored", conf="medium",
                       why="regex also matches Uncut/Unrated/AT-X, which are edition-like rather than anime "
                           "censorship facts"),
    # codecs
    "x264": term("codec", "x264"), "x265": term("codec", "x265"), "vc-1": term("codec", "vc1"),
    "vp9": term("codec", "vp9"), "mpeg2": term("codec", "mpeg2"), "10bit": term("bit_depth", "10"),
    "av1": hazard("codec", "av1"), "x266": hazard("codec", "x266"),
    "x265-hd": hazard("codec", "x265", "flag", "x265_hd"),
    "x265-no-hdrdv": hazard("codec", "x265", "flag", "x265_no_hdr"),
    # revisions
    "repack-proper": term("revision", "repack"), "repack2": term("revision", "repack2"),
    "repack3": term("revision", "repack3"),
    "v1": term("revision", "v1"), "v2": term("revision", "v2"), "v3": term("revision", "v3"),
    "v4": term("revision", "v4"), "v0": hazard("revision", "v0"),
    # sonarr release type
    "season-pack": term("release_type", "season_pack", conf="certain"),
    "multi-episode": term("release_type", "multi_episode", conf="certain"),
    "single-episode": term("release_type", "single_episode", conf="certain"),
    # languages
    "multi": term("language", "multi"),
    "anime-dual-audio": term("language", "dual_audio"),
    "dual-audio-asian": term("language", "dual_audio"),
    "german": term("language", "de_dub", conf="medium",
                   why="defined negatively: German parsed, original not parsed, and no DL/ML/Subbed tag"),
    "german-dl": term("language", "de_dl"),
    "german-dl-undefined": term("language", "de_dl_undefined", conf="medium",
                                why="German parsed with a DL/ML title tag but the original language was not "
                                    "parsed, so dual-language is inferred from the tag"),
    "german-subbed": term("language", "de_subbed"),
    "language-original-plus-french": term("language", "original_plus_fr"),
    "french-vf2": term("language", "fr_vf2"), "french-vfb": term("language", "fr_vfb"),
    "french-vff": term("language", "fr_vff"), "french-vfi": term("language", "fr_vfi"),
    "french-vfq": term("language", "fr_vfq"), "french-vof": term("language", "fr_vof"),
    "french-voq": term("language", "fr_voq"), "french-vq": term("language", "fr_vq"),
    "french-vostfr": term("language", "fr_vostfr"),
    "fansub": term("language", "fr_fansub"),
    # unwanted
    "3d": hazard("flag", "3d"), "bad-dual-groups": hazard("flag", "bad_dual_group"),
    "black-and-white-editions": hazard("flag", "black_and_white"), "bw": hazard("flag", "black_and_white"),
    "br-disk": hazard("flag", "br_disk"), "br-disk-btn": hazard("flag", "br_disk"),
    "extras": hazard("flag", "extras"), "lq": hazard("flag", "lq"), "lq-release-title": hazard("flag", "lq"),
    "line-mic-dubbed": hazard("flag", "line_mic_dubbed"), "no-rlsgroup": hazard("flag", "no_rls_group"),
    "obfuscated": hazard("flag", "obfuscated"), "retags": hazard("flag", "retags"),
    "scene": hazard("flag", "scene"), "sing-along-versions": hazard("flag", "sing_along"),
    "upscaled": hazard("flag", "upscaled"),
    "with-ad": hazard("flag", "with_ad"), "with-asl": hazard("flag", "with_asl"),
    "with-basl": hazard("flag", "with_basl"), "with-bsl": hazard("flag", "with_bsl"),
    "dubs-only": hazard("flag", "dubs_only"), "fastsub": hazard("flag", "fastsub"),
    "anime-raws": hazard("flag", "anime_raw"), "anime-lq-groups": hazard("flag", "lq"),
    "asian-lq": hazard("flag", "lq"), "french-lq": hazard("flag", "lq"), "german-lq": hazard("flag", "lq"),
    "german-lq-release-title": hazard("flag", "lq"), "german-microsized": hazard("flag", "microsized"),
    # untranslated
    "720p": untranslated(RES_PREF), "1080p": untranslated(RES_PREF), "2160p": untranslated(RES_PREF),
    "webdl-boost": untranslated(BOOST), "hd-streaming-boost": untranslated(BOOST),
    "uhd-streaming-boost": untranslated(BOOST),
    "german-1080p-booster": untranslated(BOOST + " (German dual-language at a resolution)"),
    "german-2160p-booster": untranslated(BOOST + " (German dual-language at a resolution)"),
    "language-not-english": untranslated(LANG_PREF), "language-not-original": untranslated(LANG_PREF),
    "language-not-french": untranslated(LANG_PREF), "wrong-language": untranslated(LANG_PREF),
    "not-german-or-english": untranslated(LANG_PREF),
    "not-german-japanese-or-english": untranslated(LANG_PREF),
    "not-german-japanese-korean-chinese-or-english": untranslated(LANG_PREF),
}

SERVICE_SLUGS = {
    "amzn", "atv", "atvp", "aubc", "bcore", "cbc", "cnlp", "cpng", "crav", "crit", "dsnp", "fah", "fod", "friday",
    "funi", "hami", "hbo", "hmax", "htsr", "hulu", "ip", "iqiy", "it", "itvx", "kcw", "kktv", "linetv", "ma", "max",
    "my5", "mytvsuper", "nf", "now", "ovid", "pathe", "pcok", "play", "pmtp", "roku", "stan", "strp", "tver", "tving",
    "u-next", "vdl", "viki", "viu", "vrv", "wavve", "wetv", "youku", "4od", "abema", "all4", "bglobal", "bilibili",
    "cc", "cr", "dcu", "dmm-tv", "dscp", "french-adn", "french-auvio", "french-mycanal", "french-salto",
    "french-wkn", "hidive", "nlz", "qibi", "red", "sho", "syfy",
}
SERVICE_CANON = {"Hulu": "HULU"}  # radarr spells it Hulu, sonarr HULU
SERVICE_CONF = {
    "ma": ("medium", "bare 'MA' token; lookbehind excludes DTS-HD MA and lookahead requires WEB-DL/Rip"),
    "bcore": ("medium", "second alternative matches any 'CORE' token after a resolution"),
    "cc": ("medium", "bare 'CC' token, also the Criterion abbreviation"),
    "red": ("medium", "bare 'RED' token before WEB, or [RED] rename"),
    "now": ("medium", "bare 'NOW' token"), "play": ("medium", "bare 'Play' token before WEB"),
    "it": ("medium", "bare 'iT' token before WEB"), "max": ("medium", "bare 'MAX' token before WEB"),
}

REPUTATION_NAME = re.compile(r"\bTier\s*0?(\d+)\b|\bScene\b|FanSub\b", re.I)
SINGLE_GROUP_REPUTATIONS = {"bhdstudio", "hallowed", "framestor", "thefarm", "126811", "flux", "sic", "mainframe",
                            "uhd-bluray-rlsgrp-sqp-4-ma"}
REPUTATION_NOTES = {
    "126811": "TRaSH's LQ CF also lists 126811; this CF re-admits only its indexer-flagged Internal releases. "
              "Under scores that works by summation (+10002 vs -10000 in sqp-4-ma-hybrid); a scoreless pack "
              "needs an explicit precedence rule for it.",
}


def classify(app, slug, cf):
    if slug in CLASSIFY:
        return dict(CLASSIFY[slug])
    if slug in SERVICE_SLUGS:
        conf, why = SERVICE_CONF.get(slug, ("high", None))
        return term(SVC, SERVICE_CANON.get(cf["name"], cf["name"]), conf=conf, why=why)
    if REPUTATION_NAME.search(cf["name"]) and slug not in ("scene",):
        return rep()
    if slug in SINGLE_GROUP_REPUTATIONS:
        return rep(conf="medium", why="single named group boosted by specific profiles (mostly SQP); a "
                                      "preference for one group rather than a curated tier")
    raise SystemExit(f"unclassified CF {app}/{slug} ({cf['name']}); add it to CLASSIFY")


def prior_of(slug, name):
    m = re.search(r"\bTier\s*0?(\d+)\b", name, re.I)
    if m:
        return f"tier{int(m.group(1))}"
    if re.search(r"\bScene\b", name, re.I):
        return "scene"
    if re.search(r"FanSub\b", name, re.I):
        return "fansub"
    return "preferred"


# ---------------------------------------------------------------- loading
def load_json(path):
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def git_commit(guides):
    try:
        return subprocess.check_output(["git", "-C", guides, "rev-parse", "HEAD"], text=True).strip()
    except (OSError, subprocess.CalledProcessError):
        return None


class AppData:
    def __init__(self, guides, app):
        self.app = app
        base = os.path.join(guides, "docs", "json", app)
        self.rel = lambda *p: "/".join(("docs", "json", app) + p)
        self.cfs = {}
        for fn in sorted(os.listdir(os.path.join(base, "cf"))):
            cf = load_json(os.path.join(base, "cf", fn))
            cf["_slug"], cf["_file"] = fn[:-5], self.rel("cf", fn)
            self.cfs[cf["trash_id"]] = cf
        self.groups = []
        for fn in sorted(os.listdir(os.path.join(base, "cf-groups"))):
            g = load_json(os.path.join(base, "cf-groups", fn))
            g["_slug"] = fn[:-5]
            self.groups.append(g)
        self.profiles = {}
        for fn in sorted(os.listdir(os.path.join(base, "quality-profiles"))):
            p = load_json(os.path.join(base, "quality-profiles", fn))
            p["_slug"], p["_file"] = fn[:-5], self.rel("quality-profiles", fn)
            self.profiles[p["trash_id"]] = p
        self.profile_groups = load_json(os.path.join(base, "quality-profile-groups", "groups.json"))
        self.profile_group_of = {tid: g["name"] for g in self.profile_groups for tid in g["profiles"].values()}
        self.conflicts = load_json(os.path.join(base, "conflicts.json"))
        self.sizes = [(self.rel("quality-size", fn), load_json(os.path.join(base, "quality-size", fn)))
                      for fn in sorted(os.listdir(os.path.join(base, "quality-size")))]
        self.inclusions = self._inclusions()

    def _inclusions(self):
        """profile trash_id -> {cf trash_id -> [inclusion path dicts]}"""
        inc = collections.defaultdict(lambda: collections.defaultdict(list))
        for pid, p in self.profiles.items():
            for cf_id in p["formatItems"].values():
                inc[pid][cf_id].append({"via": "formatItems", "enabled_by_default": True})
        for g in self.groups:
            group_default = g.get("default") == "true"
            for pid in g["quality_profiles"]["include"].values():
                for c in g["custom_formats"]:
                    inc[pid][c["trash_id"]].append({
                        "via": f"cf-group:{g['_slug']}",
                        "group_default": group_default, "cf_required": c["required"],
                        "cf_default": bool(c.get("default")),
                        "enabled_by_default": group_default and (c["required"] or bool(c.get("default"))),
                    })
        return inc

    def score(self, cf, profile):
        ts = cf.get("trash_scores", {})
        sset = profile.get("trash_score_set")
        if sset and sset in ts:
            return ts[sset], f"trash_scores.{sset}"
        if "default" in ts:
            return ts["default"], "trash_scores.default"
        return None, "unscored"

    def english_profiles(self):
        return [pid for pid in self.profiles if self.profile_group_of.get(pid) in ("Standard", "SQP")]


# ---------------------------------------------------------------- claim building
def build_when(app, cf, decode_issues):
    out = []
    for s in cf["specifications"]:
        w = {"name": s["name"], "implementation": s["implementation"], "negate": s["negate"],
             "required": s["required"], "value": s["fields"].get("value")}
        extra = {k: v for k, v in s["fields"].items() if k != "value"}
        if extra:
            w["fields"] = extra
        if isinstance(w["value"], int) and not isinstance(w["value"], bool):
            w["decoded"] = decode(app, s["implementation"], w["value"])
            if w["decoded"] is None or not name_matches(w["decoded"], s["name"]):
                decode_issues.append({"app": app, "cf": cf["name"], "spec": s["name"],
                                      "implementation": s["implementation"], "value": w["value"],
                                      "decoded": w["decoded"]})
        out.append(w)
    return out


def scope_of(app, cf, audience):
    scope = {"audience": audience,
             "note": "summary of the CF's non-group specs; `when` keeps the exact required/negate semantics"}
    for w in cf["specifications"]:
        impl, v = w["implementation"], w["fields"].get("value")
        if impl in ("ReleaseGroupSpecification",):
            continue
        if impl == "ReleaseTitleSpecification":
            if w["negate"]:
                scope.setdefault("title_excludes", []).append(w["name"])
            continue
        key = {"SourceSpecification": "source", "ResolutionSpecification": "resolution",
               "QualityModifierSpecification": "modifier", "IndexerFlagSpecification": "indexer_flag",
               "LanguageSpecification": "language"}[impl]
        key += "_not" if w["negate"] else ""
        scope.setdefault(key, []).append(decode(app, impl, v))
    return scope


def severity_for(score):
    if score is None:
        return None
    if score <= -10000:
        return "reject"
    if score < 0:
        return "avoid"
    return "note"


def profile_scores(data, cf):
    """[(profile_group, profile_slug, score, basis)] for every profile that includes this CF."""
    rows = []
    for pid, incl in data.inclusions.items():
        if cf["trash_id"] in incl and pid in data.profiles:
            p = data.profiles[pid]
            sc, basis = data.score(cf, p)
            rows.append((data.profile_group_of.get(pid, "?"), p["_slug"], sc, basis))
    return sorted(rows, key=lambda r: (r[0], r[1]))


def hazard_severity(data, cf):
    rows = profile_scores(data, cf)
    tiers = [("main English (Standard)", ("Standard",)), ("English SQP", ("SQP",)),
             ("language/anime profiles", ("Anime", "French", "German"))]
    for label, groups in tiers:
        scores = [r[2] for r in rows if r[0] in groups and r[2] is not None]
        if scores:
            worst = min(scores)
            dist = dict(collections.Counter(scores))
            return severity_for(worst), {"basis": label, "scores_seen": {str(k): v for k, v in dist.items()},
                                         "worst": worst}
    d = cf.get("trash_scores", {}).get("default")
    if d is not None:
        return severity_for(d), {"basis": "trash_scores.default (CF is in no profile)", "worst": d}
    return "note", {"basis": "unscored everywhere", "worst": None}


def audience_of(data, cf):
    groups = {r[0] for r in profile_scores(data, cf)}
    if groups == {"TEST"}:
        return ["(TEST base-profile only)"]
    return sorted(groups - {"TEST"}) or ["(no profile)"]


def build_claims(data, decode_issues):
    claims, untrans = [], []
    conflicts = collections.defaultdict(list)
    for clique in data.conflicts["custom_formats"]:
        for tid in clique:
            conflicts[tid] += [t for t in clique if t != tid]
    for cf in sorted(data.cfs.values(), key=lambda c: c["_slug"]):
        slug, app = cf["_slug"], data.app
        rule = classify(app, slug, cf)
        if rule["kind"] is None:
            untrans.append({"app": app, "cf": cf["name"], "trash_id": cf["trash_id"], "file": cf["_file"],
                            "reason": rule["reason"]})
            continue
        claim = {"id": f"trash/{app}/{slug}", "kind": rule["kind"], "label": cf["name"], "app": app,
                 "when": build_when(app, cf, decode_issues)}
        if rule["kind"] == "term":
            claim["sets"] = rule["sets"]
        elif rule["kind"] == "reputation":
            claim["scope"] = scope_of(app, cf, audience_of(data, cf))
            claim["prior"] = prior_of(slug, cf["name"])
            claim["group_count"] = sum(1 for s in cf["specifications"]
                                       if s["implementation"] in ("ReleaseGroupSpecification",
                                                                  "ReleaseTitleSpecification")
                                       and not s["negate"])
        else:
            sev, basis = hazard_severity(data, cf)
            claim["severity"] = sev
            claim["severity_basis"] = basis
            if rule["sets"]:
                claim["sets"] = rule["sets"]
                claim["note"] = ("also a fact: emitted as a hazard because TRaSH scores it as unwanted; "
                                 "`sets` records the fact it detects")
        claim["confidence"] = rule["confidence"]
        if rule.get("why"):
            claim["why"] = rule["why"]
        if slug in REPUTATION_NOTES:
            claim["note"] = REPUTATION_NOTES[slug]
        if cf["trash_id"] in conflicts:
            claim["conflicts_with"] = [f"trash/{app}/{data.cfs[t]['_slug']}" for t in conflicts[cf["trash_id"]]]
        claim["origin"] = {"trash_id": cf["trash_id"], "file": cf["_file"],
                           "default_score": cf.get("trash_scores", {}).get("default"),
                           "trash_scores": cf.get("trash_scores", {}),
                           "cf_groups": sorted(g["_slug"] for g in data.groups
                                               if any(c["trash_id"] == cf["trash_id"]
                                                      for c in g["custom_formats"]))}
        claims.append(claim)
    return claims, untrans


# ---------------------------------------------------------------- profiles
FAMILY_OF_KEY = {"hdr": "hdr", "audio": "audio", "channels": "audio", "service": "service", "edition": "edition",
                 "language": "language"}


def family_of(claim):
    if claim is None:
        return "untranslated"
    if claim["kind"] == "reputation":
        return "reputation"
    if claim["kind"] == "hazard":
        return "hazard"
    return FAMILY_OF_KEY.get(claim["sets"][0]["key"], "flag")


def build_profile(data, pid, claims_by_tid):
    p = data.profiles[pid]
    cfs, scores = [], {}
    for tid, paths in sorted(data.inclusions[pid].items(), key=lambda kv: data.cfs[kv[0]]["_slug"]):
        cf = data.cfs[tid]
        sc, basis = data.score(cf, p)
        scores[tid] = sc
        claim = claims_by_tid.get(tid)
        cfs.append({"trash_id": tid, "name": cf["name"], "claim": claim["id"] if claim else None,
                    "kind": claim["kind"] if claim else "untranslated", "family": family_of(claim),
                    "score": sc, "score_basis": basis,
                    "enabled_by_default": any(x["enabled_by_default"] for x in paths), "included_via": paths})
    fam = collections.defaultdict(list)
    fam_opt = collections.defaultdict(list)
    for c in cfs:
        if c["score"] is not None and c["score"] > 0 and c["family"] not in ("hazard", "untranslated"):
            (fam if c["enabled_by_default"] else fam_opt)[c["family"]].append((c["score"], c["name"]))

    def order(f):
        return [{"family": k, "max_score": max(v)[0], "cfs": [{"name": n, "score": s} for s, n in sorted(v, reverse=True)]}
                for k, v in sorted(f.items(), key=lambda kv: -max(kv[1])[0])]

    untranslated_pos = sorted(({"name": c["name"], "score": c["score"], "enabled_by_default": c["enabled_by_default"]}
                               for c in cfs if c["family"] == "untranslated" and (c["score"] or 0) > 0),
                              key=lambda x: -x["score"])

    def qual(i):
        q = {"name": i["name"], "allowed": i["allowed"]}
        if "items" in i:
            q["items"] = i["items"]
        return q

    return {
        "name": p["name"], "app": data.app, "file": p["_file"], "trash_id": pid,
        "profile_group": data.profile_group_of.get(pid), "language": p.get("language"),
        "qualities": [qual(i) for i in p["items"]],
        "qualities_order_note": "verbatim TRaSH order: first entry is the most preferred",
        "cutoff": p["cutoff"], "upgradeAllowed": p["upgradeAllowed"], "minFormatScore": p["minFormatScore"],
        "cutoffFormatScore": p["cutoffFormatScore"], "minUpgradeFormatScore": p["minUpgradeFormatScore"],
        "score_set": p.get("trash_score_set", "default"),
        "scores": scores, "cfs": cfs,
        "suggested_priority_order": {
            "basis": "positively scored CFs enabled by default (formatItems, or a default cf-group whose CF is "
                     "required or default); families ordered by the max score in each. Families: hdr, audio "
                     "(incl. channels), service, reputation, edition, language (added), flag (flag, codec, "
                     "bit_depth, revision, release_type)",
            "order": [f["family"] for f in order(fam)], "families": order(fam),
            "optional_groups_would_add": order(fam_opt),
            "untranslated_positive_cfs": untranslated_pos,
        },
    }


# ---------------------------------------------------------------- regex compat
JS_PROBE = r"""
const fs = require('fs');
const items = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
const out = items.map(src => {
  const r = {};
  for (const flags of ['i', 'iu']) {
    try { new RegExp(src, flags); r[flags] = null; } catch (e) { r[flags] = String(e.message); }
  }
  return r;
});
fs.writeFileSync(process.argv[3], JSON.stringify(out));
"""

CONSTRUCTS = [
    ("inline_flags", re.compile(r"\(\?[imnsx-]+[:)]")),
    ("atomic_group", re.compile(r"\(\?>")),
    ("possessive_quantifier", re.compile(r"(?<!\\)(?:[*+?]|\})\+")),
    ("anchor_A_Z_z", re.compile(r"\\[AZz]")),
    ("dotnet_quote_named_group", re.compile(r"\(\?'")),
    ("python_named_group", re.compile(r"\(\?P<")),
    ("conditional", re.compile(r"\(\?\(")),
    ("unicode_property", re.compile(r"\\[pP]\{")),
    ("identity_escape_invalid_in_u", re.compile(r"\\[^\\dDwWsSbBnrtfv0-9.*+?()\[\]{}|^$/\-uxcpPk]")),
    # .NET/PCRE read `[])]` as a class holding `]` and `)`; JS reads `[]` as an empty class, leaving `)` unmatched.
    ("leading_bracket_in_class", re.compile(r"(?<!\\)\[\^?\]")),
    ("lone_brace_invalid_in_u", re.compile(r"(?<!\\)\{(?!\d+(?:,\d*)?\})")),
]
LOOKBEHIND = re.compile(r"\(\?<[=!]")
LOOKAHEAD = re.compile(r"\(\?[=!]")
BACKREF = re.compile(r"\\[1-9]|\\k<")


def regex_compat(all_data, outdir):
    occurrences = []
    for data in all_data:
        for cf in data.cfs.values():
            for s in cf["specifications"]:
                if s["implementation"] in ("ReleaseTitleSpecification", "ReleaseGroupSpecification"):
                    occurrences.append((data.app, cf["name"], cf["_file"], s["name"], s["implementation"],
                                        s["fields"]["value"]))
    unique = sorted({o[5] for o in occurrences})
    src_path, res_path = os.path.join(outdir, ".regex-in.json"), os.path.join(outdir, ".regex-out.json")
    probe = os.path.join(outdir, "regex_probe.js")
    with open(probe, "w") as f:
        f.write(JS_PROBE)
    with open(src_path, "w") as f:
        json.dump(unique, f)
    node_version = subprocess.check_output(["node", "--version"], text=True).strip()
    subprocess.check_call(["node", probe, src_path, res_path])
    results = dict(zip(unique, load_json(res_path)))
    os.remove(src_path)
    os.remove(res_path)

    failures, per_app = [], {}
    for app in APPS:
        occ = [o for o in occurrences if o[0] == app]
        uniq = {o[5] for o in occ}
        per_app[app] = {
            "occurrences": len(occ), "unique_patterns": len(uniq),
            "fail_i": sum(1 for o in occ if results[o[5]]["i"]),
            "fail_iu": sum(1 for o in occ if results[o[5]]["iu"]),
            "unique_fail_i": sum(1 for u in uniq if results[u]["i"]),
            "unique_fail_iu": sum(1 for u in uniq if results[u]["iu"]),
            "lookbehind": sum(1 for o in occ if LOOKBEHIND.search(o[5])),
            "lookahead": sum(1 for o in occ if LOOKAHEAD.search(o[5])),
            "any_lookaround": sum(1 for o in occ if LOOKBEHIND.search(o[5]) or LOOKAHEAD.search(o[5])),
            "unique_any_lookaround": sum(1 for u in uniq if LOOKBEHIND.search(u) or LOOKAHEAD.search(u)),
            "backreference": sum(1 for o in occ if BACKREF.search(o[5])),
            "cfs_with_lookaround": len({o[1] for o in occ if LOOKBEHIND.search(o[5]) or LOOKAHEAD.search(o[5])}),
        }
    for o in occurrences:
        r = results[o[5]]
        if r["i"] or r["iu"]:
            failures.append({"app": o[0], "cf": o[1], "file": o[2], "spec": o[3], "implementation": o[4],
                             "regex": o[5], "error_i": r["i"], "error_iu": r["iu"],
                             "constructs": [n for n, rx in CONSTRUCTS if rx.search(o[5])] or ["unidentified"]})
    lead = dict(CONSTRUCTS)["leading_bracket_in_class"]
    silent = [{"app": o[0], "cf": o[1], "spec": o[3], "regex": o[5]} for o in occurrences
              if lead.search(o[5]) and not results[o[5]]["i"]]
    return {"node": node_version, "compiles_but_diverges": silent,
            "semantics_note": ".NET (Radarr/Sonarr) \\d, \\w and \\b are Unicode-aware; JS keeps them ASCII "
                              "even with 'u'. Compiling is necessary, not sufficient, for equal matching.", "method": "new RegExp(src, flags) in node for flags 'i' and 'iu'; every "
                                            "ReleaseTitleSpecification/ReleaseGroupSpecification fields.value",
            "per_app": per_app, "failures": failures,
            "lookaround_note": "lookahead and lookbehind compile in modern JS (V8) but are unsupported by Go "
                               "regexp (RE2) and Rust regex"}


# ---------------------------------------------------------------- vocabulary
VOCAB_NOTES = {
    "hdr": {"hdr_any": "added: TRaSH's single 'HDR' CF is a union of DV-with-fallback/HDR/HDR10/HDR10+/HLG/PQ",
            "dv_no_fallback": "hazard only (DV (w/o HDR fallback))", "sdr": "hazard only (SDR at 2160p)",
            "hlg": "hazard only (standalone HLG CF)",
            "hdr10": "requested but unused: no TRaSH CF isolates plain HDR10; it is one alternative inside 'HDR'",
            "pq": "requested but unused: PQ is one alternative inside 'HDR'"},
    "channels": {"*": "added key: TRaSH audio-channels group (1.0 to 7.1)"},
    "codec": {"*": "added key: x264/x265/x266/AV1/VC-1/VP9/MPEG2 CFs are video-codec facts, not flags"},
    "bit_depth": {"*": "added key: 10bit CF"},
    "revision": {"*": "added key (prompt example used revision=repack): TRaSH ranks Repack/Proper/Rerip as one "
                      "rank, then Repack2, Repack3; anime v0-v4 are the same idea",
                 "proper": "not used: TRaSH does not separate PROPER from REPACK"},
    "release_type": {"*": "added key: Sonarr ReleaseTypeSpecification CFs"},
    "language": {"*": "added values: multi, dual_audio, de_*, fr_* French audio-version tags, original_plus_fr"},
    "edition": {"special": "added: TRaSH's 'Special Edition' is a catch-all for extended/uncut/directors/unrated/"
                           "cut/version/edition",
                "extended": "requested but unused: no TRaSH CF isolates it (inside 'special')",
                "directors": "requested but unused (inside 'special')",
                "unrated": "requested but unused (inside 'special')"},
    "flag": {"x265_no_hdr": "added: x265 (no HDR/DV) golden-rule alternative",
             "dv_disk": "added: DV (Disk)", "indexer_internal": "added: indexer flag, distinct from title INTERNAL",
             "black_and_white": "added", "bad_dual_group": "added", "generated_dynamic_hdr": "added",
             "line_mic_dubbed": "added", "with_ad": "added", "with_asl": "added", "with_bsl": "added",
             "with_basl": "added", "dubs_only": "added", "fastsub": "added", "anime_raw": "added",
             "microsized": "added", "freeleech": "added", "hfr": "added", "uncensored": "added",
             "av1": "not used as a flag: AV1 is codec=av1 on a hazard", "proper": "not used: see revision",
             "repack": "not used: see revision", "retags": "used", "no_rls_group": "used"},
}
REQUESTED = {
    "hdr": ["dv", "dv_no_fallback", "hdr10", "hdr10p", "hlg", "pq", "sdr"],
    "audio": ["truehd_atmos", "truehd", "dtsx", "dtshd_ma", "dtshd_hra", "dts_es", "dts", "ddp_atmos", "ddp", "dd",
              "aac", "flac", "pcm", "opus", "mp3", "atmos_undefined"],
    "edition": ["imax", "imax_enhanced", "criterion", "remaster", "theatrical", "extended", "directors", "unrated",
                "open_matte"],
    "flag": ["hybrid", "repack", "proper", "scene", "internal", "x265_hd", "av1", "3d", "br_disk", "upscaled", "lq",
             "retags", "obfuscated", "no_rls_group", "extras", "sing_along"],
}


def build_vocab(claims):
    used = collections.defaultdict(lambda: collections.defaultdict(list))
    for c in claims:
        for s in c.get("sets", []):
            used[s["key"]][s["value"]].append(c["id"])
    vocab = {}
    for key in sorted(set(used) | set(REQUESTED)):
        values = sorted(set(used[key]) | set(REQUESTED.get(key, [])))
        vocab[key] = {"values": [{"value": v, "used_by": sorted(used[key][v]),
                                  "note": VOCAB_NOTES.get(key, {}).get(v)} for v in values],
                      "note": VOCAB_NOTES.get(key, {}).get("*")}
    vocab["_reputation_priors"] = {"values": ["tier1", "tier2", "tier3", "tier4", "tier5", "tier6", "tier7", "tier8",
                                              "scene", "fansub", "preferred"],
                                   "note": "tierN from 'Tier NN' in the CF name; scene/fansub from the name; "
                                           "preferred for single-group CFs. 'lq' is not used: LQ group lists are "
                                           "hazards per the classification guidance."}
    vocab["_hazard_severity"] = {"values": ["reject", "avoid", "note"],
                                 "note": "score <= -10000 reject, < 0 avoid, 0 note; worst score across the "
                                         "first non-empty basis: Standard profiles, then SQP, then language/anime "
                                         "profiles, then trash_scores.default"}
    return vocab


# ---------------------------------------------------------------- report
def write_report(path, pack, profiles, regex, decode_issues, all_data):
    lines = ["# TRaSH to Knowledge Pack conversion report", "",
             f"Source: TRaSH-Guides/Guides at `{pack['source']['commit']}` ({SOURCE_DATE}). "
             "Generated by `convert.py`; rerun to regenerate every file here.", "",
             "## Claims per kind per app", "", "| app | CFs | term | reputation | hazard | untranslated |",
             "|---|---|---|---|---|---|"]
    for app in APPS:
        k = collections.Counter(c["kind"] for c in pack["claims"] if c["app"] == app)
        u = sum(1 for x in pack["untranslated"] if x["app"] == app)
        total = len([d for d in all_data if d.app == app][0].cfs)
        lines.append(f"| {app} | {total} | {k['term']} | {k['reputation']} | {k['hazard']} | {u} |")
    sev = collections.Counter((c["app"], c["severity"]) for c in pack["claims"] if c["kind"] == "hazard")
    lines += ["", "Hazard severities: " + ", ".join(f"{a} {s} {n}" for (a, s), n in sorted(sev.items())), "",
              "Confidence: " + ", ".join(f"{k} {v}" for k, v in sorted(
                  collections.Counter(c["confidence"] for c in pack["claims"]).items())), "",
              "## Untranslated", "", "| app | CF | reason |", "|---|---|---|"]
    for u in pack["untranslated"]:
        lines.append(f"| {u['app']} | {u['cf']} | {u['reason']} |")
    lines += ["", "## Enum decode cross-check", "",
              "Every integer spec value is decoded from the Radarr/Sonarr enums and compared with the TRaSH spec "
              "name. Mismatches:", ""]
    lines += [f"- {d['app']} `{d['cf']}` spec `{d['spec']}`: {d['implementation']} {d['value']} decodes to "
              f"`{d['decoded']}`" for d in decode_issues] or ["- none"]
    if any(d["decoded"] == "Television" for d in decode_issues):
        lines += ["", "The sonarr `WEB` = 1 specs decode to `Television` under Sonarr's QualitySource enum (3 = Web "
                  "and 4 = WebRip agree with their spec names in the same CFs). All are non-required any-of specs "
                  "beside Web/WebRip on anime services and anime web tiers. Inference, not verified: it is "
                  "deliberate, to catch untagged anime web releases that Sonarr parses as HDTV. The claims keep "
                  "the raw value 1 with `decoded: Television`."]
    lines += ["", "## JavaScript regex compatibility", "",
              f"Method: {regex['method']} ({regex['node']}).", "",
              "| app | occurrences | unique | fail `i` | fail `iu` | lookbehind | lookahead | any lookaround | "
              "CFs with lookaround | backrefs |", "|---|---|---|---|---|---|---|---|---|---|"]
    for app, r in regex["per_app"].items():
        lines.append(f"| {app} | {r['occurrences']} | {r['unique_patterns']} | {r['fail_i']} "
                     f"({r['unique_fail_i']} unique) | {r['fail_iu']} ({r['unique_fail_iu']} unique) | "
                     f"{r['lookbehind']} | {r['lookahead']} | {r['any_lookaround']} | {r['cfs_with_lookaround']} "
                     f"| {r['backreference']} |")
    lines += ["", "Failures:", ""]
    short = lambda e: e.rsplit(": ", 1)[-1] if e else "ok"
    lines += [f"- {f['app']} `{f['cf']}` / `{f['spec']}`: `i` {short(f['error_i'])}; `iu` {short(f['error_iu'])}; "
              f"construct: {', '.join(f['constructs'])}" for f in regex["failures"]] or ["- none"]
    lines += ["", f"Patterns with a leading `]` in a class that still compile in JS (silent divergence): "
              f"{len(regex['compiles_but_diverges'])}.", "", regex["semantics_note"], "",
              regex["lookaround_note"] + ".", "",
              "## Profiles", "", "| app | profile | score set | suggested family order |", "|---|---|---|---|"]
    for p in profiles:
        fams = ", ".join(f"{f['family']} ({f['max_score']})" for f in p["suggested_priority_order"]["families"])
        lines.append(f"| {p['app']} | {p['name']} | {p['score_set']} | {fams} |")
    lines += ["", "## Judgment calls", ""] + [f"{i}. {j}" for i, j in enumerate(JUDGMENT_CALLS, 1)]
    with open(path, "w") as f:
        f.write("\n".join(lines) + "\n")


JUDGMENT_CALLS = [
    "Profile membership is formatItems plus every cf-group whose quality_profiles.include lists the profile. "
    "A CF counts as enabled by default only via formatItems, or via a group with default \"true\" where the CF is "
    "required or default. The priority order uses only enabled-by-default CFs; optional ones are listed apart.",
    "Hazard severity takes the worst resolved score in the Standard profiles that include the CF. When none "
    "include it, the basis falls back to SQP, then the language and anime profiles, then trash_scores.default. "
    "The basis is recorded on every hazard. Optional-group hazards (WiTH AD/ASL, x266) still get `reject` "
    "because their score is -10000 once enabled.",
    "LQ-style group lists (LQ, Anime LQ Groups, Asian/FR/German LQ, Bad Dual Groups, Anime Raws) are hazards, as "
    "the guidance says, not reputations with prior `lq`.",
    "The HDR CF is a union of DV-with-fallback, HDR, HDR10, HDR10+, HLG and PQ, so it sets `hdr=hdr_any`. The "
    "requested `hdr10` and `pq` values stay unused.",
    "Special Edition is one catch-all regex, so it sets `edition=special`. The requested extended, directors "
    "and unrated values have no TRaSH source.",
    "Language CFs that only reverse-score the parsed language (Not English, Not Original, Wrong Language, Not "
    "German or English and its siblings) are untranslated. They state a preference, which belongs in the "
    "user's priority list. Positive language facts (MULTi, Dual Audio, German DL, VFF and the like) are terms.",
    "Resolution CFs and boosts (720p/1080p/2160p, WEBDL Boost, HD/UHD Streaming Boost, German 1080p/2160p "
    "Booster) are untranslated. They add score to facts other claims or the Quality already carry.",
    "VOSTFR (-10000 default, +1000 in french-vostfr), 10bit (-10000 only in SQP) and x264 (-10000 only in "
    "SQP-2..5) are terms, not hazards. They are facts that only some profiles dislike.",
    "126811 is in the LQ list, and its own CF re-admits only its Internal-flagged releases by score summation. "
    "That claim carries a note, because a scoreless pack needs an explicit precedence rule for it.",
    "Single-group CFs (BHDStudio, hallowed, FraMeSToR, TheFarm, FLUX, SiC, MainFrame, 126811, UHD Bluray RlsGrp "
    "(SQP-4 MA)) are reputations with prior `preferred` at medium confidence. Tier CFs take `tierN` from their "
    "name. Scene lists take `scene` and FR Anime FanSub takes `fansub`.",
    "Reputation scope comes mechanically from the CF's non-group specs (Source, Resolution, QualityModifier, "
    "IndexerFlag, Language, negated title specs). The audience comes from the profile groups that include it.",
    "Service values are the TRaSH CF names, except radarr `Hulu`, which is normalized to `HULU` to match sonarr.",
    "Hazards that are also facts (DV w/o HDR fallback, SDR, HLG, AV1, x265 (HD), x266, v0) carry `sets` and a "
    "note.",
]


# ---------------------------------------------------------------- main
def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--guides", default=DEFAULT_GUIDES)
    ap.add_argument("--out", default=HERE)
    args = ap.parse_args()
    all_data = [AppData(args.guides, app) for app in APPS]
    commit = git_commit(args.guides)

    decode_issues, claims, untrans = [], [], []
    for data in all_data:
        c, u = build_claims(data, decode_issues)
        claims += c
        untrans += u
    pack = {"id": "trash", "name": "TRaSH Guides",
            "source": {"repo": "TRaSH-Guides/Guides", "commit": commit, "date": SOURCE_DATE},
            "claims": claims, "untranslated": untrans}

    profiles = []
    for data in all_data:
        by_tid = {c["origin"]["trash_id"]: c for c in claims if c["app"] == data.app}
        for pid in data.english_profiles():
            profiles.append(build_profile(data, pid, by_tid))
    profiles.sort(key=lambda p: (p["app"], p["profile_group"], p["file"]))
    excluded = [{"app": d.app, "file": p["_file"], "name": p["name"], "profile_group": d.profile_group_of.get(pid)}
                for d in all_data for pid, p in d.profiles.items() if pid not in d.english_profiles()]

    sizes = {d.app: [{"source_file": rel, **body} for rel, body in d.sizes] for d in all_data}
    regex = regex_compat(all_data, args.out)

    def dump(name, obj):
        with open(os.path.join(args.out, name), "w") as f:
            json.dump(obj, f, indent=2, ensure_ascii=False)
            f.write("\n")

    dump("trash-pack.json", pack)
    dump("vocabulary.json", build_vocab(claims))
    dump("profiles.json", {"source": pack["source"],
                           "selection": "quality-profile-groups 'Standard' and 'SQP' (English); excludes Anime, "
                                        "French, German and TEST",
                           "profiles": profiles, "excluded": excluded})
    dump("quality-size.json", {"source": pack["source"], "unit": "MB per minute", "apps": sizes})
    dump("regex-compat.json", regex)
    write_report(os.path.join(args.out, "report.md"), pack, profiles, regex, decode_issues, all_data)
    counts = collections.Counter((c["app"], c["kind"]) for c in claims)
    print(json.dumps({f"{a}/{k}": n for (a, k), n in sorted(counts.items())}),
          "untranslated", len(untrans), "profiles", len(profiles), "decode_issues", len(decode_issues),
          "regex_failures", len(regex["failures"]), file=sys.stderr)


if __name__ == "__main__":
    main()
