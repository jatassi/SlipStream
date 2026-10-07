#!/usr/bin/env python3
"""PROTOTYPE, throwaway. Joins the library inventory, MediaInfo probes and the converted TRaSH pack
into data/data.js (window.DATA) for the settings prototype. Read-only on every input."""
import json, os, re, sys, glob, collections, datetime

SCRATCH = sys.argv[1]
OUT = sys.argv[2]
INV = json.load(open(os.path.join(SCRATCH, 'inventory-1/inventory.json')))
PROBE_PATH = os.path.join(SCRATCH, 'probe-1/probe.jsonl')
PACK = json.load(open(os.path.join(SCRATCH, 'trash-pack-1/trash-pack.json')))
PROFILES = json.load(open(os.path.join(SCRATCH, 'trash-pack-1/profiles.json')))
SIZES = json.load(open(os.path.join(SCRATCH, 'trash-pack-1/quality-size.json')))
CF_DIR = os.path.join(SCRATCH, 'trash/Guides/docs/json')
COMPAT = json.load(open(os.path.join(SCRATCH, 'trash-pack-1/regex-compat.json')))

probes = {}
if os.path.exists(PROBE_PATH):
    for line in open(PROBE_PATH):
        line = line.strip()
        if not line:
            continue
        p = json.loads(line)
        probes[p['path']] = p

def compact(p):
    if not p or p.get('error') or not p.get('video'):
        return None
    v = p['video']
    return {
        'size_bytes': p.get('size_bytes'), 'duration_s': p.get('duration_s'),
        'video': {k: v.get(k) for k in ('codec', 'width', 'height', 'bitrate_bps', 'bit_depth')},
        'audio': [{k: a.get(k) for k in ('codec', 'commercial', 'features', 'channels', 'language')} for a in (p.get('audio') or [])],
        'dv_profile': p.get('dv_profile'), 'has_hdr10': p.get('has_hdr10'), 'has_hdr10plus': p.get('has_hdr10plus'),
        'has_hlg': p.get('has_hlg'), 'atmos': p.get('atmos'),
    }

# ---------- TRaSH vocabulary -> engine vocabulary ----------
CH = {'1.0': 1, '2.0': 2, '3.0': 3, '4.0': 4, '5.1': 6, '6.1': 7, '7.1': 8}
CODEC = {'x264': 'h264', 'h264': 'h264', 'x265': 'h265', 'h265': 'h265', 'av1': 'av1', 'vc1': 'vc1', 'mpeg2': 'mpeg2', 'vp9': 'vp9'}
def map_sets(sets):
    out, note = [], []
    for s in sets:
        k, v = s['key'], s['value']
        if k == 'hdr':
            if v == 'hdr_any': out.append({'key': 'hdr', 'value': 'hdr10'}); note.append('hdr_any read as HDR10 at medium confidence')
            elif v in ('dv', 'hdr10p', 'hlg', 'hdr10'): out.append({'key': 'hdr', 'value': v})
            elif v == 'dv_no_fallback': out.append({'key': 'hdr', 'value': 'dv'})
            else: note.append(f'hdr={v} not mapped')
        elif k == 'audio':
            if v == 'truehd_atmos': out += [{'key': 'audio', 'value': 'truehd'}, {'key': 'atmos', 'value': True}]
            elif v == 'ddp_atmos': out += [{'key': 'audio', 'value': 'ddp'}, {'key': 'atmos', 'value': True}]
            elif v == 'atmos_undefined': out.append({'key': 'atmos', 'value': True})
            elif v == 'dts_es': out.append({'key': 'audio', 'value': 'dts'})
            else: out.append({'key': 'audio', 'value': v})
        elif k == 'channels':
            if str(v) in CH: out.append({'key': 'channels', 'value': CH[str(v)]})
        elif k == 'codec':
            if v in CODEC: out.append({'key': 'codec', 'value': CODEC[v]})
        elif k == 'language':
            if v in ('multi', 'dual_audio'): out.append({'key': 'flag', 'value': 'multi_audio'})
            elif v.startswith('de_'): out.append({'key': 'language', 'value': 'de'})
            elif v.startswith('fr_') or v == 'original_plus_fr': out.append({'key': 'language', 'value': 'fr'})
            else: out.append({'key': 'language', 'value': v})
        elif k == 'revision':
            out.append({'key': 'revision', 'value': 'proper' if v.startswith('proper') else 'repack'})
        elif k == 'bit_depth':
            out.append({'key': 'flag', 'value': '10bit'})
        elif k in ('service', 'flag', 'edition'):
            out.append({'key': k, 'value': v})
        else:
            note.append(f'{k}={v} kept as a flag')
            out.append({'key': 'flag', 'value': f'{k}:{v}'})
    return out, note

# Severity from TRaSH's own defaults: only profiles that turn the format on by default count.
default_scores = collections.defaultdict(list)
for prof in PROFILES['profiles']:
    for cf in prof['cfs']:
        if cf.get('claim') and cf.get('enabled_by_default'):
            default_scores[cf['claim']].append(cf['score'])
# Hazards that only restate a fact the probe can check become terms; the Rendition's picture filter does the rejecting.
FACT_HAZARDS = {
    'sdr': [{'key': 'hdr', 'value': 'sdr'}], 'sdr-no-webdl': [{'key': 'hdr', 'value': 'sdr'}],
    'dv-wo-hdr-fallback': [{'key': 'hdr', 'value': 'dv'}],
}
claims = []
for c in PACK['claims']:
    c = dict(c)
    slug = c['id'].split('/')[-1]
    if c['kind'] == 'hazard' and slug in FACT_HAZARDS:
        c['kind'] = 'term'; c['sets'] = FACT_HAZARDS[slug]; c['confidence'] = 'medium'
        c['mapNotes'] = ['TRaSH lists this as unwanted; it is a fact about the file, so it became a term the probe checks, and the Rendition picture filter rejects it']
        c.pop('severity', None)
        claims.append(c); continue
    if c['kind'] == 'hazard':
        sc = default_scores.get(c['id'], [])
        worst = min(sc) if sc else None
        c['severity'] = 'note' if worst is None or worst >= 0 else ('reject' if worst <= -10000 else 'avoid')
        c['severityBasis'] = f'worst default-on score {worst}' if worst is not None else 'off by default in every English profile'
    if c['kind'] == 'term':
        c['sets'], notes = map_sets(c.get('sets', []))
        if any(s['key'] == 'hdr' for s in c.get('sets', [])) and any('hdr_any' in n for n in notes):
            c['confidence'] = 'medium'
        if notes: c['mapNotes'] = notes
    claims.append(c)

LANG_CODE = {'english': 'en', 'german': 'de', 'french': 'fr', 'spanish': 'es', 'italian': 'it', 'japanese': 'ja', 'hindi': 'hi', 'chinese': 'zh', 'korean': 'ko', 'russian': 'ru', 'dutch': 'nl', 'portuguese': 'pt', 'original': 'original', 'any': 'any', 'unknown': 'unknown'}
for c in claims:
    for s in c['when']:
        if s['implementation'] == 'LanguageSpecification' and s.get('decoded'):
            s['decoded'] = LANG_CODE.get(str(s['decoded']).lower(), str(s['decoded']).lower())

# raw CF index (every CF, translated or not) for the Sonarr/Radarr baseline score
RADARR_SRC = {0: 'UNKNOWN', 1: 'CAM', 2: 'TELESYNC', 3: 'TELECINE', 4: 'WORKPRINT', 5: 'DVD', 6: 'TV', 7: 'WEBDL', 8: 'WEBRIP', 9: 'BLURAY'}
SONARR_SRC = {0: 'UNKNOWN', 1: 'TELEVISION', 2: 'TELEVISIONRAW', 3: 'WEB', 4: 'WEBRIP', 5: 'DVD', 6: 'BLURAY', 7: 'BLURAYRAW'}
MOD = {0: 'NONE', 1: 'REGIONAL', 2: 'SCREENER', 3: 'RAWHD', 4: 'BRDISK', 5: 'REMUX'}
RTYPE = {0: 'UNKNOWN', 1: 'SINGLEEPISODE', 2: 'MULTIEPISODE', 3: 'SEASONPACK'}
LANG_ID = {-2: 'original', -1: 'any', 0: 'unknown', 1: 'en', 2: 'fr', 3: 'es', 4: 'de', 5: 'it', 8: 'ja', 26: 'hi', 10: 'zh', 21: 'ko', 11: 'ru', 7: 'nl', 18: 'pt'}
cf_index = {}
for app in ('radarr', 'sonarr'):
    for f in glob.glob(os.path.join(CF_DIR, app, 'cf', '*.json')):
        j = json.load(open(f))
        when = []
        for s in j.get('specifications', []):
            impl = s['implementation']
            val = s.get('fields', {}).get('value') if isinstance(s.get('fields'), dict) else None
            spec = {'implementation': impl, 'negate': s.get('negate', False), 'required': s.get('required', False), 'value': val}
            if impl == 'SourceSpecification': spec['decoded'] = (RADARR_SRC if app == 'radarr' else SONARR_SRC).get(val)
            elif impl == 'QualityModifierSpecification': spec['decoded'] = MOD.get(val)
            elif impl == 'ReleaseTypeSpecification': spec['decoded'] = RTYPE.get(val)
            elif impl == 'LanguageSpecification': spec['decoded'] = LANG_ID.get(val, f'lang{val}')
            elif impl in ('SizeSpecification', 'YearSpecification'):
                fl = s.get('fields', {})
                spec['min'] = fl.get('min'); spec['max'] = fl.get('max')
            when.append(spec)
        cf_index[j['trash_id']] = {'label': j['name'], 'app': app, 'when': when}

AQ = {'WEBDL-2160p': 'webdl-2160', 'WEBRip-2160p': 'webrip-2160', 'Bluray-2160p': 'bluray-2160', 'Remux-2160p': 'remux-2160', 'HDTV-2160p': 'hdtv-2160',
      'Bluray-2160p Remux': 'remux-2160', 'Bluray-1080p Remux': 'remux-1080',
      'WEBDL-1080p': 'webdl-1080', 'WEBRip-1080p': 'webrip-1080', 'Bluray-1080p': 'bluray-1080', 'Remux-1080p': 'remux-1080', 'HDTV-1080p': 'hdtv-1080',
      'WEBDL-720p': 'webdl-720', 'WEBRip-720p': 'webrip-720', 'Bluray-720p': 'bluray-720', 'HDTV-720p': 'hdtv-720', 'DVD': 'dvd', 'SDTV': 'sdtv',
      'CAM': 'cam', 'TELESYNC': 'telesync', 'DVDSCR': 'screener'}
profiles = []
for p in PROFILES['profiles']:
    allowed = [q for q in p['qualities'] if q.get('allowed')]
    rank = {}
    for i, q in enumerate(allowed):
        for name in (q.get('items') or [q['name']]):
            if name in AQ: rank[AQ[name]] = len(allowed) - i
    scores = {c['trash_id']: c['score'] for c in p['cfs'] if c.get('enabled_by_default') and c.get('score')}
    profiles.append({'name': p['name'], 'app': p['app'], 'group': p.get('profile_group'), 'qualityRank': rank, 'cutoff': p.get('cutoff'),
                     'minFormatScore': p.get('minFormatScore'), 'cutoffFormatScore': p.get('cutoffFormatScore'), 'scores': scores,
                     'suggested': [x['family'] if isinstance(x, dict) else x for x in (p.get('suggested_priority_order') or [])]})

QS = {'WEBDL-2160p': 'webdl-2160', 'WEBRip-2160p': 'webrip-2160', 'Bluray-2160p': 'bluray-2160', 'Remux-2160p': 'remux-2160', 'HDTV-2160p': 'hdtv-2160',
      'WEBDL-1080p': 'webdl-1080', 'WEBRip-1080p': 'webrip-1080', 'Bluray-1080p': 'bluray-1080', 'Remux-1080p': 'remux-1080', 'HDTV-1080p': 'hdtv-1080',
      'WEBDL-720p': 'webdl-720', 'WEBRip-720p': 'webrip-720', 'Bluray-720p': 'bluray-720', 'HDTV-720p': 'hdtv-720', 'DVD': 'dvd', 'SDTV': 'sdtv'}
sizes = {}
movie_sizes = [x for x in SIZES['apps']['radarr'] if x['type'] == 'movie']
for q in (movie_sizes[0]['qualities'] if movie_sizes else []):
    if q['quality'] in QS:
        pref = None if q['preferred'] >= 1999 else q['preferred']
        mx = None if q['max'] >= 2000 else q['max']
        sizes[QS[q['quality']]] = {'min': q['min'], 'preferred': pref, 'max': mx}

# ---------- library, releases, works ----------
def epkey(season, episode):
    if isinstance(episode, str) and re.match(r'^S\d+E\d+', episode): return episode[:6]
    if season is None or episode is None: return None
    return 'S%02dE%02d' % (int(season), int(str(episode).lstrip('E')))

files, file_idx = [], {}
dup_key = {}
for f in INV['library']:
    if '/Sample/' in f['path'] or f['path'].lower().endswith('-sample.mkv'):
        continue
    name = f.get('matched_release') or os.path.splitext(os.path.basename(f['path']))[0]
    rec = {'i': len(files), 'p': f['path'].replace('/Volumes/Media/Plex/', ''), 'n': name, 'm': bool(f.get('matched_release')),
           's': f['size_bytes'], 'sfx': f.get('plex_quality_suffix'), 'f': compact(probes.get(f['path']))}
    if not rec['f'] and f.get('matched_download_path'):
        rec['f'] = compact(probes.get(f['matched_download_path']))
    file_idx[f['path']] = rec['i']
    files.append(rec)

releases, cands = [], []
rel_by_name = {}
for r in INV['releases']:
    vids = [x for x in r['files'] if not x.get('is_sample')]
    typ = 'movie' if r['kind'] == 'movie' else 'series'
    rec = {'n': r['release_name'], 'type': typ, 'kind': r['kind'], 'c': r['client_dir'], 't': r.get('earliest_mtime_epoch'),
           'files': [{'s': x['size_bytes'], 'ep': x.get('episode'), 'f': compact(probes.get(x['path']))} for x in vids]}
    rel_by_name[r['release_name']] = (r, rec)
    releases.append(rec)

def norm(t):
    return re.sub(r'[^a-z0-9]', '', (t or '').lower())

works = []
for w in INV['works']:
    typ = w['content_type']
    key = w['work_key']
    lib = [f for f in INV['library'] if (f['content_type'] == typ and norm(f['title']) == norm(w['title']) and (typ != 'movie' or f.get('year') in (w.get('years_seen') or [w.get('year')])))]
    items = collections.OrderedDict()
    def item(k):
        if k not in items: items[k] = {'key': k, 'held': [], 'cands': []}
        return items[k]
    for f in lib:
        if f['path'] not in file_idx: continue
        k = 'movie' if typ == 'movie' else (epkey(f.get('season'), f.get('episode')) or 'unknown')
        item(k)['held'].append(file_idx[f['path']])
    for rn in w.get('releases', []):
        if rn not in rel_by_name: continue
        r, rec = rel_by_name[rn]
        if r['kind'] in ('movie',):
            vids = [x for x in r['files'] if not x.get('is_sample')]
            if not vids: continue
            big = max(vids, key=lambda x: x['size_bytes'])
            cands.append({'n': rn, 'c': r['client_dir'], 't': r.get('earliest_mtime_epoch'), 's': big['size_bytes'], 'f': compact(probes.get(big['path']))})
            item('movie')['cands'].append(len(cands) - 1)
        else:
            for x in r['files']:
                if x.get('is_sample'): continue
                k = epkey(None, x.get('episode')) or (epkey(r.get('season'), (r.get('episodes') or [None])[0]) if r['kind'] == 'episode' else None)
                if not k: continue
                cands.append({'n': rn, 'c': r['client_dir'], 't': r.get('earliest_mtime_epoch'), 's': x['size_bytes'], 'f': compact(probes.get(x['path']))})
                item(k)['cands'].append(len(cands) - 1)
    its = [v for v in items.values() if v['held'] or v['cands']]
    its.sort(key=lambda v: v['key'])
    for v in its:
        seen = collections.defaultdict(list)
        for i in v['held']: seen[files[i]['s']].append(i)
        for g, ids in enumerate(seen.values()):
            if len(ids) > 1:
                for i in ids: files[i]['dup'] = f"{key}|{v['key']}|{g}"
    if its:
        works.append({'key': key, 'type': typ, 'title': w['title'], 'year': w.get('year'), 'items': its})

probed = sum(1 for f in files if f['f']) + sum(1 for r in releases for x in r['files'] if x['f'])
meta = {
    'generated': datetime.datetime.now(datetime.timezone.utc).isoformat(timespec='minutes'),
    'libraryFiles': len(files), 'releases': len(releases), 'works': len(works), 'probedFiles': probed,
    'probe': open(os.path.join(SCRATCH, 'probe-1/parse_speed.txt')).read().strip() if os.path.exists(os.path.join(SCRATCH, 'probe-1/parse_speed.txt')) else 'MediaInfo',
    'trashCommit': PACK['source']['commit'][:7] + ' (' + PACK['source']['date'] + ')',
}
compat = []
for app, a in COMPAT.get('per_app', {}).items():
    compat.append(f"{app}: {a['fail_i']} of {a['occurrences']} TRaSH regexes don't compile in JavaScript; {a['any_lookaround']} use lookaround ({a['cfs_with_lookaround']} custom formats), which Go RE2 and Rust regex reject")
data = {'meta': meta, 'files': files, 'releases': releases, 'cands': cands, 'works': works,
        'trash': {'pack': {**{k: PACK[k] for k in ('id', 'name', 'source')}, 'version': PACK['source']['commit'][:7], 'claims': claims, 'untranslated': PACK.get('untranslated', [])},
                  'profiles': profiles, 'sizes': sizes, 'cfIndex': cf_index, 'compat': ' · '.join(compat)}}
os.makedirs(os.path.dirname(OUT), exist_ok=True)
with open(OUT, 'w') as fh:
    fh.write('window.DATA = ' + json.dumps(data, separators=(',', ':')) + ';\n')
print(json.dumps(meta, indent=1), 'works', len(works), 'cands', len(cands), 'bytes', os.path.getsize(OUT))
