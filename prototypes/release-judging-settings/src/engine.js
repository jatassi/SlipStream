/* PROTOTYPE, throwaway. Pure release-judging engine for the Horizon / Verified Priorities settings prototype.
   No DOM. The page calls in; nothing calls out. */
const Engine = (() => {
  const LEVEL = { certain: 0.99, high: 0.9, medium: 0.7, low: 0.5 };

  const QUALITIES = [
    { id: 'cam', name: 'CAM', source: 'cam', res: 0, cinema: true },
    { id: 'telesync', name: 'TELESYNC', source: 'telesync', res: 0, cinema: true },
    { id: 'screener', name: 'SCREENER', source: 'screener', res: 0, cinema: true },
    { id: 'sdtv', name: 'SDTV', source: 'hdtv', res: 480 },
    { id: 'dvd', name: 'DVD', source: 'dvd', res: 480 },
    { id: 'hdtv-720', name: 'HDTV-720p', source: 'hdtv', res: 720 },
    { id: 'webrip-720', name: 'WEBRip-720p', source: 'webrip', res: 720 },
    { id: 'webdl-720', name: 'WEB-DL-720p', source: 'webdl', res: 720 },
    { id: 'bluray-720', name: 'Bluray-720p', source: 'bluray', res: 720 },
    { id: 'hdtv-1080', name: 'HDTV-1080p', source: 'hdtv', res: 1080 },
    { id: 'webrip-1080', name: 'WEBRip-1080p', source: 'webrip', res: 1080 },
    { id: 'webdl-1080', name: 'WEB-DL-1080p', source: 'webdl', res: 1080 },
    { id: 'bluray-1080', name: 'Bluray-1080p', source: 'bluray', res: 1080 },
    { id: 'remux-1080', name: 'Remux-1080p', source: 'remux', res: 1080 },
    { id: 'hdtv-2160', name: 'HDTV-2160p', source: 'hdtv', res: 2160 },
    { id: 'webrip-2160', name: 'WEBRip-2160p', source: 'webrip', res: 2160 },
    { id: 'webdl-2160', name: 'WEB-DL-2160p', source: 'webdl', res: 2160 },
    { id: 'bluray-2160', name: 'Bluray-2160p', source: 'bluray', res: 2160 },
    { id: 'remux-2160', name: 'Remux-2160p', source: 'remux', res: 2160 },
  ];
  const QUALITY = Object.fromEntries(QUALITIES.map(q => [q.id, q]));

  const SOURCE_PRECEDENCE = ['remux', 'bluray', 'webdl', 'webrip', 'hdtv', 'dvd', 'screener', 'telesync', 'cam'];

  function qualityOf(source, res) {
    if (!source) return null;
    if (['cam', 'telesync', 'screener'].includes(source)) return source;
    if (source === 'dvd') return 'dvd';
    if (!res) return null;
    if (res <= 576) return source === 'hdtv' ? 'sdtv' : (source === 'dvd' ? 'dvd' : null);
    const r = res >= 2160 ? 2160 : res >= 1080 ? 1080 : 720;
    const id = `${source}-${r}`;
    return QUALITY[id] ? id : null;
  }

  const PICTURE_KEYS = ['dv+hdr10p', 'dv+hdr10', 'hdr10p', 'hdr10', 'hlg', 'dv_no_fallback', 'sdr'];
  const PICTURE_LABEL = {
    'dv+hdr10p': 'DV + HDR10+', 'dv+hdr10': 'DV + HDR10', hdr10p: 'HDR10+', hdr10: 'HDR10', hlg: 'HLG',
    dv_no_fallback: 'DV without HDR10 fallback', sdr: 'SDR',
  };
  function pictureKey(hdr) {
    const dv = hdr.has('dv'), p = hdr.has('hdr10p'), h = hdr.has('hdr10') || p;
    if (dv && p) return 'dv+hdr10p';
    if (dv && h) return 'dv+hdr10';
    if (dv) return 'dv_no_fallback';
    if (p) return 'hdr10p';
    if (h) return 'hdr10';
    if (hdr.has('hlg')) return 'hlg';
    return 'sdr';
  }

  const AUDIO_LABEL = {
    truehd_atmos: 'TrueHD Atmos', dtsx: 'DTS:X', dtshd_ma: 'DTS-HD MA', truehd: 'TrueHD', pcm: 'PCM', flac: 'FLAC',
    ddp_atmos: 'DD+ Atmos', dtshd_hra: 'DTS-HD HRA', dts: 'DTS', ddp: 'DD+', dd: 'DD', aac: 'AAC', opus: 'Opus', mp3: 'MP3',
    atmos_undefined: 'Atmos (codec unnamed)', unknown: 'unknown',
  };
  const AUDIO_STRENGTH = ['truehd_atmos', 'dtsx', 'dtshd_ma', 'truehd', 'pcm', 'flac', 'ddp_atmos', 'atmos_undefined', 'dtshd_hra', 'dts', 'ddp', 'dd', 'aac', 'opus', 'mp3', 'unknown'];

  const CODEC_LABEL = { h265: 'HEVC / x265', h264: 'AVC / x264', av1: 'AV1', vc1: 'VC-1', mpeg2: 'MPEG-2' };

  /* ---------- core parse (a Content Type's job: group, coverage, extension) ---------- */
  function coreParse(raw) {
    let name = raw.trim();
    const partial = /\.part$/i.test(name);
    name = name.replace(/\.part$/i, '');
    const extM = name.match(/\.(mkv|mp4|m4v|avi|ts|m2ts)$/i);
    const ext = extM ? extM[1].toLowerCase() : null;
    if (extM) name = name.slice(0, -extM[0].length);
    let tail = name.replace(/[)\]]+\s*$/, '');
    let group = null, uploader = null;
    const gm = tail.match(/[-–][ .]?([A-Za-z0-9&_][A-Za-z0-9&_]*)(?:[ .]([A-Za-z0-9&_]+))?$/);
    if (gm) {
      const [g, u] = gm[2] ? [gm[2], gm[1]] : [gm[1], null];
      if (!/^\d+$/.test(g) && !/^(dl|rip|hd|ma|x|\d{3,4}p|remux|web|webdl|bluray|hdtv|sdr|hdr|hdr10|hdr10plus|dv)$/i.test(g)) { group = g; uploader = u; }
    }
    const se = name.match(/(?<![a-z0-9])S(\d{1,2})(?:E(\d{1,3})(?:-?E(\d{1,3}))?)?(?![0-9])/i);
    const year = (name.match(/(?<![0-9])(19[2-9]\d|20[0-4]\d)(?![0-9p])/) || [])[1];
    let releaseType = 'movie';
    if (se) releaseType = se[2] ? (se[3] ? 'multi_episode' : 'single_episode') : 'season_pack';
    return { name, raw, ext, partial, group, uploader, season: se ? +se[1] : null, episode: se && se[2] ? +se[2] : null, year: year ? +year : null, releaseType };
  }

  /* ---------- custom-format style condition evaluation (used by every claim's `when`) ---------- */
  const SOURCE_ENUM = {
    CAM: ['cam'], TELESYNC: ['telesync'], TELECINE: ['telesync'], WORKPRINT: ['cam'], DVD: ['dvd'], TV: ['hdtv'], TELEVISION: ['hdtv'],
    TELEVISIONRAW: ['hdtv'], WEBDL: ['webdl'], WEB: ['webdl'], WEBRIP: ['webrip'], BLURAY: ['bluray', 'remux'], BLURAYRAW: ['remux'],
  };
  const regexCache = new Map();
  function rx(src, flags) {
    const k = flags + '\u0000' + src;
    if (!regexCache.has(k)) {
      try { regexCache.set(k, new RegExp(src, flags)); } catch (e) { regexCache.set(k, null); }
    }
    return regexCache.get(k);
  }
  function specRaw(spec, rel) {
    const v = spec.decoded !== undefined ? spec.decoded : spec.value;
    switch (spec.implementation) {
      case 'ReleaseTitleSpecification': { const r = rx(spec.value, spec.flags ?? 'i'); return r ? r.test(rel.core.name) : false; }
      case 'ReleaseGroupSpecification': { const r = rx(spec.value, spec.flags ?? 'i'); return r && rel.core.group ? r.test(rel.core.group) : false; }
      case 'EditionSpecification': { const r = rx(spec.value, 'i'); return r ? rel.editionTokens.some(t => r.test(t)) : false; }
      case 'SourceSpecification': { const want = SOURCE_ENUM[String(v).toUpperCase()] || []; return want.includes(rel.source); }
      case 'ResolutionSpecification': { const n = parseInt(String(v).replace(/\D/g, ''), 10); return rel.res === n; }
      case 'QualityModifierSpecification': {
        const m = String(v).toUpperCase();
        if (m === 'REMUX') return rel.source === 'remux';
        if (m === 'SCREENER') return rel.source === 'screener';
        if (m === 'BRDISK') return rel.flags.has('br_disk');
        return false;
      }
      case 'LanguageSpecification': {
        const lang = String(v).toLowerCase();
        const langs = rel.languages.size ? rel.languages : new Set(['en']);
        if (lang === 'original') return langs.has(rel.originalLanguage || 'en');
        if (lang === 'any') return true;
        if (lang === 'unknown') return false;
        return langs.has(lang);
      }
      case 'ReleaseTypeSpecification': {
        const m = { 1: 'single_episode', 2: 'multi_episode', 3: 'season_pack', SINGLEEPISODE: 'single_episode', MULTIEPISODE: 'multi_episode', SEASONPACK: 'season_pack' };
        return m[String(v).toUpperCase()] === rel.core.releaseType;
      }
      case 'SizeSpecification': {
        if (!rel.sizeBytes) return false;
        const gb = rel.sizeBytes / 1e9, min = spec.min ?? (spec.value && spec.value.min) ?? 0, max = spec.max ?? (spec.value && spec.value.max) ?? Infinity;
        return gb > min && gb <= max;
      }
      case 'YearSpecification': {
        const y = rel.core.year; if (!y) return false;
        const min = spec.min ?? (spec.value && spec.value.min) ?? 0, max = spec.max ?? (spec.value && spec.value.max) ?? 9999;
        return y >= min && y <= max;
      }
      case 'IndexerFlagSpecification': return rel.indexerFlags.has(String(v).toLowerCase());
      default: return false;
    }
  }
  function matchWhen(when, rel) {
    if (!when || !when.length) return false;
    const groups = new Map();
    for (const s of when) { if (!groups.has(s.implementation)) groups.set(s.implementation, []); groups.get(s.implementation).push(s); }
    for (const specs of groups.values()) {
      let anyPass = false, requiredFail = false;
      for (const s of specs) {
        const r = specRaw(s, rel) !== !!s.negate;
        if (r) anyPass = true; else if (s.required) requiredFail = true;
      }
      if (requiredFail || !anyPass) return false;
    }
    return true;
  }

  /* ---------- the Base Knowledge Pack: facts about release names, no scores ---------- */
  const B = '(?<![a-z0-9])', E = '(?![a-z0-9])';
  const T = (id, label, re, sets, confidence = 'certain', extra = {}) => ({
    id: 'base/' + id, kind: 'term', label, confidence, sets,
    when: [{ implementation: 'ReleaseTitleSpecification', value: re, negate: false, required: true, ...(extra.flags !== undefined ? { flags: extra.flags } : {}) }],
    ...(extra.and ? { when: [{ implementation: 'ReleaseTitleSpecification', value: re, negate: false, required: true }, ...extra.and] } : {}),
    examples: extra.examples || [],
  });
  const BASE_PACK = {
    id: 'base', name: 'SlipStream base pack', version: '2026.10.0', source: { builtin: true },
    claims: [
      T('res-2160', '2160p', `${B}(2160p|4k)${E}`, [{ key: 'res', value: 2160 }], 'certain', { examples: ['Eternity.2025.2160p.iT.WEB-DL.DDP5.1.Atmos.DV.HDR.H.265-BYNDR'] }),
      T('res-uhd', 'UHD implies 2160p', `${B}uhd${E}`, [{ key: 'res', value: 2160 }], 'high'),
      T('res-1080', '1080p', `${B}1080[pi]${E}`, [{ key: 'res', value: 1080 }]),
      T('res-720', '720p', `${B}720p${E}`, [{ key: 'res', value: 720 }]),
      T('res-576', '576p', `${B}576p${E}`, [{ key: 'res', value: 576 }]),
      T('res-480', '480p', `${B}480p${E}`, [{ key: 'res', value: 480 }]),
      T('src-remux', 'Remux', `${B}(bd)?remux${E}`, [{ key: 'source', value: 'remux' }], 'certain', { examples: ['Fight.Club.1999.REMASTERED.1080p.BluRay.REMUX.AVC.DTS-HD.MA.5.1-FGT'] }),
      T('src-bluray', 'Blu-ray', `blu-?ray|${B}(bdrip|brrip|bd25|bd50)${E}`, [{ key: 'source', value: 'bluray' }]),
      T('src-webdl', 'WEB-DL', `${B}web[-. ]?dl${E}`, [{ key: 'source', value: 'webdl' }]),
      T('src-web', 'Bare WEB is usually WEB-DL', `${B}web(?![a-z0-9-])`, [{ key: 'source', value: 'webdl' }], 'medium', { examples: ['Steal.S01E01.HDR.2160p.WEB.h265-ETHEL'] }),
      T('src-webrip', 'WEBRip', `${B}web-?rip${E}`, [{ key: 'source', value: 'webrip' }]),
      T('src-hdtv', 'HDTV', `${B}(hdtv|pdtv)${E}`, [{ key: 'source', value: 'hdtv' }]),
      T('src-dvd', 'DVD', `${B}dvd(rip|5|9|r)?${E}`, [{ key: 'source', value: 'dvd' }]),
      T('src-screener', 'Screener', `${B}((web|dvd|bd)?screener|(dvd)?scr)${E}`, [{ key: 'source', value: 'screener' }], 'high', { examples: ['Avatar.Fire.and.Ash.[2025] WEBSCREENER Dual YG'] }),
      T('src-telesync', 'Telesync', `${B}(telesync|hd-?ts|pdvd)${E}`, [{ key: 'source', value: 'telesync' }], 'high', { examples: ['The.Housemaid.2025.1080p.TELESYNC.x264-SyncUP'] }),
      T('src-cam', 'Cam', `${B}(hd)?cam(rip)?${E}`, [{ key: 'source', value: 'cam' }], 'medium'),
      T('svc-amzn', 'Amazon', `${B}amzn${E}`, [{ key: 'service', value: 'AMZN' }]),
      T('svc-nf', 'Netflix', `${B}nf${E}`, [{ key: 'service', value: 'NF' }]),
      T('svc-hmax', 'HBO Max', `${B}(hmax|max)(?=[ .](web|uhd|hdr|dv))`, [{ key: 'service', value: 'HMAX' }]),
      T('svc-atvp', 'Apple TV+', `${B}atvp${E}`, [{ key: 'service', value: 'ATVP' }]),
      T('svc-it', 'iTunes', `(?<![A-Za-z0-9])iT(?=[ .]WEB)`, [{ key: 'service', value: 'iT' }], 'certain', { flags: '' }),
      T('svc-ma', 'Movies Anywhere', `(?<![A-Za-z0-9-])MA(?=[ .]WEB)`, [{ key: 'service', value: 'MA' }], 'certain', { flags: '' }),
      T('svc-pcok', 'Peacock', `${B}pcok${E}`, [{ key: 'service', value: 'PCOK' }]),
      T('svc-pmtp', 'Paramount+', `${B}pmtp${E}`, [{ key: 'service', value: 'PMTP' }]),
      T('svc-dsnp', 'Disney+', `${B}dsnp${E}`, [{ key: 'service', value: 'DSNP' }]),
      T('svc-hulu', 'Hulu', `${B}hulu${E}`, [{ key: 'service', value: 'HULU' }]),
      T('vc-h265', 'HEVC', `${B}(x265|h[ .]?265|hevc)${E}`, [{ key: 'codec', value: 'h265' }]),
      T('vc-h264', 'AVC', `${B}(x264|h[ .]?264|avc)${E}`, [{ key: 'codec', value: 'h264' }]),
      T('vc-av1', 'AV1', `${B}av1${E}`, [{ key: 'codec', value: 'av1' }]),
      T('vc-vc1', 'VC-1', `${B}vc-?1${E}`, [{ key: 'codec', value: 'vc1' }]),
      T('hdr-dv', 'Dolby Vision', `${B}(dv|dovi|dolby[ .]?vision)${E}`, [{ key: 'hdr', value: 'dv' }], 'high'),
      T('hdr-dv-remux-base', 'DV on a UHD Blu-ray remux keeps an HDR10 base layer', `${B}(dv|dovi)${E}`, [{ key: 'hdr', value: 'hdr10' }], 'high',
        { and: [{ implementation: 'QualityModifierSpecification', value: 'REMUX', decoded: 'REMUX', negate: false, required: true }] }),
      T('hdr-hdr10p', 'HDR10+', `${B}hdr10(\\+|p|plus)`, [{ key: 'hdr', value: 'hdr10p' }]),
      T('hdr-hdr10', 'HDR10', `${B}hdr10(?![+p0-9]|plus)`, [{ key: 'hdr', value: 'hdr10' }]),
      T('hdr-generic', 'Bare HDR usually means HDR10', `${B}hdr(?![a-z0-9+])`, [{ key: 'hdr', value: 'hdr10' }], 'medium'),
      T('hdr-hlg', 'HLG', `${B}hlg${E}`, [{ key: 'hdr', value: 'hlg' }]),
      T('hdr-sdr', 'Explicit SDR', `${B}sdr${E}`, [{ key: 'hdr', value: 'sdr' }], 'high'),
      T('au-truehd', 'TrueHD', `${B}true-?hd${E}`, [{ key: 'audio', value: 'truehd' }]),
      T('au-atmos', 'Atmos', `${B}atmos${E}`, [{ key: 'atmos', value: true }], 'high'),
      T('au-dtsx', 'DTS:X', `${B}dts[ .:-]?x${E}`, [{ key: 'audio', value: 'dtsx' }]),
      T('au-dtshdma', 'DTS-HD MA', `${B}dts-?hd[ .-]?ma${E}`, [{ key: 'audio', value: 'dtshd_ma' }]),
      T('au-dtshdhra', 'DTS-HD HRA', `${B}dts-?hd[ .-]?hra${E}`, [{ key: 'audio', value: 'dtshd_hra' }]),
      T('au-dts', 'DTS', `${B}dts(?![a-z0-9]|-?hd|[ .:-]?x${E})`, [{ key: 'audio', value: 'dts' }], 'high'),
      T('au-ddp', 'DD+', `${B}(ddp|dd\\+|e-?ac-?3)`, [{ key: 'audio', value: 'ddp' }]),
      T('au-dd', 'DD', `${B}(dd(?=[ .]?[0-9])|dd(?![a-z0-9+])|ac-?3${E})`, [{ key: 'audio', value: 'dd' }], 'high'),
      T('au-aac', 'AAC', `${B}aac`, [{ key: 'audio', value: 'aac' }]),
      T('au-flac', 'FLAC', `${B}flac`, [{ key: 'audio', value: 'flac' }]),
      T('au-pcm', 'PCM', `${B}l?pcm${E}`, [{ key: 'audio', value: 'pcm' }]),
      T('au-opus', 'Opus', `${B}opus${E}`, [{ key: 'audio', value: 'opus' }]),
      T('ch-71', '7.1 channels', `(?<![0-9]|[0-9][ .])7[ .]1(?![0-9])`, [{ key: 'channels', value: 8 }]),
      T('ch-51', '5.1 channels', `(?<![0-9]|[0-9][ .])5[ .]1(?![0-9])`, [{ key: 'channels', value: 6 }]),
      T('ch-20', '2.0 channels', `(?<![0-9]|[0-9][ .])2[ .]0(?![0-9])`, [{ key: 'channels', value: 2 }]),
      T('ed-theatrical', 'Theatrical cut', `${B}theatrical${E}`, [{ key: 'edition', value: 'theatrical' }]),
      T('ed-extended', 'Extended cut', `${B}extended${E}`, [{ key: 'edition', value: 'extended' }]),
      T('ed-directors', "Director's cut", `${B}director'?s?[ .]?cut${E}|${B}dc${E}`, [{ key: 'edition', value: 'directors' }], 'high'),
      T('ed-unrated', 'Unrated', `${B}unrated${E}`, [{ key: 'edition', value: 'unrated' }]),
      T('ed-uncut', 'Uncut', `${B}uncut${E}`, [{ key: 'edition', value: 'uncut' }]),
      T('ed-remastered', 'Remastered', `${B}remaster(ed)?${E}`, [{ key: 'edition', value: 'remastered' }]),
      T('ed-imax', 'IMAX', `${B}imax${E}`, [{ key: 'edition', value: 'imax' }]),
      T('ed-criterion', 'Criterion', `${B}criterion${E}`, [{ key: 'edition', value: 'criterion' }]),
      T('rev-proper', 'PROPER', `${B}proper${E}`, [{ key: 'revision', value: 'proper' }]),
      T('rev-repack', 'REPACK', `${B}(repack|rerip)${E}`, [{ key: 'revision', value: 'repack' }]),
      T('flag-hybrid', 'Hybrid', `${B}hybrid${E}`, [{ key: 'flag', value: 'hybrid' }]),
      T('flag-10bit', '10-bit', `${B}10[ .-]?bit${E}`, [{ key: 'flag', value: '10bit' }]),
      T('lang-german', 'German audio', `${B}(german|ger)${E}`, [{ key: 'language', value: 'de' }], 'high'),
      T('lang-french', 'French audio', `${B}(french|vff|truefrench)${E}`, [{ key: 'language', value: 'fr' }], 'high'),
      T('lang-hindi', 'Hindi audio', `${B}hindi${E}`, [{ key: 'language', value: 'hi' }], 'high'),
      T('lang-english', 'English audio', `${B}english${E}`, [{ key: 'language', value: 'en' }], 'high'),
      T('lang-multi', 'Several audio languages', `${B}(multi|dual)${E}`, [{ key: 'flag', value: 'multi_audio' }], 'high'),
      { id: 'base/hazard-upscale', kind: 'hazard', label: 'Upscaled', severity: 'reject', confidence: 'high',
        when: [{ implementation: 'ReleaseTitleSpecification', value: `${B}upscal(e|ed|ing)${E}|${B}(ai|regrade)[ .-]?upscale`, negate: false, required: true }],
        examples: ['Breaking.Bad.S01-2160P.(Upscale).BluRay.10.bit.Encode.AV1.DTS.5.1-R&H'] },
      { id: 'base/hazard-cinema', kind: 'hazard', label: 'Cinema source (cam, telesync, screener)', severity: 'reject', confidence: 'high',
        when: [{ implementation: 'ReleaseTitleSpecification', value: `${B}((web|dvd|bd)?screener|(dvd)?scr|telesync|hd-?ts|(hd)?cam(rip)?)${E}`, negate: false, required: true }],
        examples: ['The.Housemaid.2025.1080p.TELESYNC.x264-SyncUP'] },
      { id: 'base/hazard-x265-hd', kind: 'hazard', label: 'x265 encode at 1080p or below', severity: 'avoid', confidence: 'medium',
        when: [{ implementation: 'ReleaseTitleSpecification', value: `${B}x265${E}`, negate: false, required: true },
               { implementation: 'ResolutionSpecification', value: 1080, decoded: 1080, negate: false, required: false },
               { implementation: 'ResolutionSpecification', value: 720, decoded: 720, negate: false, required: false }],
        examples: ['Thunderbolts.2025.Proper.1080p.WEB-DL.DDP5.1.x265-NeoNoir'] },
      { id: 'base/hazard-obfuscated', kind: 'hazard', label: 'Obfuscated or nonsense group name', severity: 'avoid', confidence: 'medium',
        when: [{ implementation: 'ReleaseGroupSpecification', value: '^[A-Z][a-z]+([A-Z][a-z]+){3,}$', flags: '', negate: false, required: true }],
        examples: ['Pantheon.S02E01.1080p.WEB.H264-TunnelingSuccinctQuokkaOfTenacity'] },
    ],
  };

  /* ---------- interpretation: name -> beliefs with sources and confidence ---------- */
  function emptyRel(raw) {
    const core = coreParse(raw);
    return { core, res: null, source: null, flags: new Set(), languages: new Set(), editionTokens: [], indexerFlags: new Set(), sizeBytes: null };
  }

  function interpret(raw, packs, opts = {}) {
    const rel = emptyRel(raw);
    rel.sizeBytes = opts.sizeBytes || null;
    const beliefs = {};
    const hazards = [], reputations = [], fired = [];
    const add = (key, value, conf, claim) => {
      beliefs[key] = beliefs[key] || new Map();
      const cur = beliefs[key].get(value);
      if (!cur) beliefs[key].set(value, { value, conf, sources: [claim.id] });
      else { cur.conf = Math.max(cur.conf, conf); if (!cur.sources.includes(claim.id)) cur.sources.push(claim.id); }
    };
    const disabled = opts.disabled || new Set();
    const terms = [], others = [];
    for (const p of packs) for (const c of p.claims) {
      if (disabled.has(c.id)) continue;
      (c.kind === 'term' ? terms : others).push(c);
    }
    // two passes so later conditions (Source, Resolution, Modifier specs) can see the attributes earlier terms set
    for (let pass = 0; pass < 2; pass++) {
      for (const c of terms) {
        if (pass === 1 && fired.includes(c.id)) continue;
        if (!matchWhen(c.when, rel)) continue;
        fired.push(c.id);
        const conf = LEVEL[c.confidence] ?? LEVEL.high;
        for (const s of c.sets || []) add(s.key, s.value, conf, c);
      }
      settle(rel, beliefs);
    }
    for (const c of others) {
      if (!matchWhen(c.when, rel)) continue;
      if (c.kind === 'hazard') hazards.push({ id: c.id, label: c.label, severity: c.severity });
      if (c.kind === 'reputation') reputations.push({ id: c.id, label: c.label, prior: c.prior, scope: c.scope });
    }
    return { raw, core: rel.core, beliefs, hazards, reputations, fired, rel };
  }

  function settle(rel, beliefs) {
    const best = key => {
      const m = beliefs[key]; if (!m || !m.size) return null;
      return [...m.values()].sort((a, b) => b.conf - a.conf || (typeof b.value === 'number' ? b.value - a.value : 0))[0];
    };
    const res = best('res'); rel.res = res ? res.value : null;
    const srcs = beliefs.source ? [...beliefs.source.values()] : [];
    srcs.sort((a, b) => SOURCE_PRECEDENCE.indexOf(a.value) - SOURCE_PRECEDENCE.indexOf(b.value));
    rel.source = srcs.length ? srcs[0].value : null;
    rel.flags = new Set(beliefs.flag ? [...beliefs.flag.keys()] : []);
    rel.languages = new Set(beliefs.language ? [...beliefs.language.keys()] : []);
    rel.editionTokens = beliefs.edition ? [...beliefs.edition.keys()] : [];
  }

  /* ---------- trust: how often a group's claims held on your probed files ---------- */
  const FAMILY_OF = { hdr: 'picture', atmos: 'audio', audio: 'audio', channels: 'audio', res: 'resolution', codec: 'codec', language: 'language' };

  function reliability(group, family, trust, settings) {
    const prior = groupPrior(group, settings);
    const k = settings.trust.priorStrength;
    const t = trust && group && trust[group] && trust[group][family];
    const ok = t ? t.ok : 0, n = t ? t.n : 0;
    return { value: (ok + prior.p * k) / (n + k), ok, n, prior };
  }
  function groupPrior(group, settings) {
    const rep = (settings._reputationByGroup || {})[group];
    if (rep) return { p: settings.trust.priors[rep.prior] ?? settings.trust.priors.unknown, from: rep.label, prior: rep.prior };
    return { p: settings.trust.priors.unknown, from: null, prior: 'unknown' };
  }

  /* claims about picture and audio are only as good as the group that made them */
  function effectiveBeliefs(interp, trust, settings) {
    const out = {};
    for (const [key, m] of Object.entries(interp.beliefs)) {
      out[key] = new Map();
      const fam = FAMILY_OF[key];
      for (const [v, b] of m) {
        let conf = b.conf, rel = null, calib = null;
        const cs = settings._claimStats;
        if (cs) {
          const k = settings.trust.priorStrength;
          const vals = b.sources.map(id => { const st = cs[id]; return st && st.n ? { id, ...st, value: (st.ok + b.conf * k) / (st.n + k) } : null; }).filter(Boolean);
          if (vals.length) { calib = vals.sort((x, y) => y.n - x.n)[0]; conf = calib.value; }
        }
        if (fam === 'picture' || fam === 'audio') {
          rel = reliability(interp.core.group, fam, trust, settings);
          conf = conf * rel.value;
        }
        out[key].set(v, { ...b, conf, termConf: b.conf, calib, rel, weak: conf < settings.trust.weakBelow });
      }
    }
    return out;
  }

  /* ---------- a comparable view of a candidate: predicted (from claims) or observed (from probe) ---------- */
  function viewFromClaims(interp, trust, settings, ctx = {}) {
    const eff = effectiveBeliefs(interp, trust, settings);
    const keys = k => (eff[k] ? [...eff[k].values()] : []);
    const strong = k => keys(k).filter(b => !b.weak);
    const res = interp.rel.res, source = interp.rel.source;
    const hdrAll = new Set(keys('hdr').map(b => b.value).filter(v => v !== 'sdr'));
    const hdrStrong = new Set(strong('hdr').map(b => b.value).filter(v => v !== 'sdr'));
    const picClaimed = pictureKey(hdrAll), picPess = pictureKey(hdrStrong);
    const audioOf = (formats, atmos) => {
      let best = 'unknown';
      for (const f of formats) if (AUDIO_STRENGTH.indexOf(f) < AUDIO_STRENGTH.indexOf(best)) best = f;
      if (atmos) {
        if (best === 'truehd') best = 'truehd_atmos';
        else if (best === 'ddp') best = 'ddp_atmos';
        else if (best === 'unknown') best = 'atmos_undefined';
      }
      return best;
    };
    const fmts = keys('audio').map(b => b.value);
    const atmosB = keys('atmos')[0];
    const audioClaimed = audioOf(fmts, !!atmosB);
    const audioPess = audioOf(fmts, !!(atmosB && !atmosB.weak));
    const chB = keys('channels').sort((a, b) => b.value - a.value)[0];
    const codecB = keys('codec').sort((a, b) => b.conf - a.conf)[0];
    const svc = keys('service').sort((a, b) => b.conf - a.conf)[0];
    const edition = interp.rel.editionTokens.find(t => !['remastered', 'criterion'].includes(t)) || 'standard';
    const rev = keys('revision').map(b => b.value);
    const group = interp.core.group;
    const runtimeMin = ctx.runtimeMin || null;
    const sizeBytes = ctx.sizeBytes || interp.rel.sizeBytes || null;
    return {
      kind: 'claimed', name: interp.raw, group, quality: qualityOf(source, res), source, res,
      picture: picPess, pictureClaimed: picClaimed, pictureWeak: picPess !== picClaimed,
      audio: audioPess, audioClaimed, audioWeak: audioPess !== audioClaimed,
      channels: chB ? chB.value : null, codec: codecB ? codecB.value : null, service: svc ? svc.value : null,
      languages: new Set(keys('language').map(b => b.value)), multiAudio: interp.rel.flags.has('multi_audio'),
      edition, revision: rev.includes('proper') ? 'proper' : rev.includes('repack') ? 'repack' : null,
      hazards: interp.hazards, flags: interp.rel.flags, sizeBytes, runtimeMin,
      mbPerMin: sizeBytes && runtimeMin ? sizeBytes / 1048576 / runtimeMin : null,
      trust: reliability(group, 'picture', trust, settings), interp, eff,
    };
  }

  /* probe JSON (MediaInfo, compact) -> facts. Source can't be probed; it comes from the name. */
  function factsFromProbe(p) {
    if (!p || !p.video) return null;
    const w = +p.video.width || 0, h = +p.video.height || 0;
    const res = w >= 3200 || h >= 2000 ? 2160 : w >= 1800 || h >= 1000 ? 1080 : w >= 1200 || h >= 700 ? 720 : 480;
    const hdr = new Set();
    if (p.dv_profile != null) hdr.add('dv');
    if (p.has_hdr10plus) hdr.add('hdr10p');
    if (p.has_hdr10) hdr.add('hdr10');
    if (p.has_hlg) hdr.add('hlg');
    const vc = String(p.video.codec || '').toUpperCase();
    const codec = vc === 'HEVC' ? 'h265' : vc === 'AVC' ? 'h264' : vc === 'AV1' ? 'av1' : vc === 'VC-1' ? 'vc1' : vc.startsWith('MPEG') ? 'mpeg2' : vc.toLowerCase();
    let best = 'unknown', channels = 0, bestDeep = true, bestTrack = null;
    const langs = new Set();
    for (const a of p.audio || []) {
      const f = String(a.codec || ''), com = String(a.commercial || ''), feat = String(a.features || '');
      // a header-only parse leaves commercial and features empty when it never reached the stream's extension
      const deep = !!(a.commercial || a.features);
      let k = 'unknown';
      if (/MLP FBA|TrueHD/i.test(f + com)) k = /Atmos/i.test(com) || /16-ch/i.test(feat) ? 'truehd_atmos' : 'truehd';
      else if (/^DTS/i.test(f)) k = /XLL X|DTS:X|IMAX/i.test(feat + com) ? 'dtsx' : /XLL/i.test(feat) || /Master Audio/i.test(com) ? 'dtshd_ma' : /XBR|High Resolution/i.test(feat + com) ? 'dtshd_hra' : 'dts';
      else if (/E-AC-3/i.test(f) || /Digital Plus/i.test(com)) k = /JOC|Atmos/i.test(feat + com) ? 'ddp_atmos' : 'ddp';
      else if (/^AC-3/i.test(f)) k = 'dd';
      else if (/AAC/i.test(f)) k = 'aac';
      else if (/FLAC/i.test(f)) k = 'flac';
      else if (/PCM/i.test(f)) k = 'pcm';
      else if (/Opus/i.test(f)) k = 'opus';
      if (AUDIO_STRENGTH.indexOf(k) < AUDIO_STRENGTH.indexOf(best)) { best = k; bestDeep = deep; bestTrack = a; }
      channels = Math.max(channels, +a.channels || 0);
      if (a.language) langs.add(String(a.language).split('-')[0].toLowerCase());
    }
    const runtimeMin = p.duration_s ? p.duration_s / 60 : null;
    return {
      kind: 'observed', res, hdr, picture: pictureKey(hdr), dvProfile: p.dv_profile, codec, audio: best,
      atmos: best === 'truehd_atmos' || best === 'ddp_atmos' || !!p.atmos, channels, languages: langs, audioDeep: bestDeep,
      runtimeMin, sizeBytes: p.size_bytes, mbPerMin: p.size_bytes && runtimeMin ? p.size_bytes / 1048576 / runtimeMin : null,
      bitrate: p.video.bitrate_bps || null, height: h, width: w,
    };
  }

  function viewFromFile(name, facts, trust, settings, packs) {
    const interp = interpret(name || '', packs);
    const source = interp.rel.source;
    let audio = facts.audio, audioFromName = false;
    if (facts.audioDeep === false && interp.beliefs.audio) {
      // the header hid the extension (DTS-HD MA core, TrueHD or DD+ without Atmos markers); the name may say more within the same family
      const fam = x => x.startsWith('dts') ? 'dts' : x.startsWith('truehd') ? 'truehd' : x.startsWith('dd') ? 'dd' : x;
      let claimed = [...interp.beliefs.audio.keys()].filter(c => fam(c) === fam(facts.audio));
      if (interp.beliefs.atmos) claimed = claimed.map(c => c === 'truehd' ? 'truehd_atmos' : c === 'ddp' ? 'ddp_atmos' : c);
      const better = claimed.sort((a, b) => AUDIO_STRENGTH.indexOf(a) - AUDIO_STRENGTH.indexOf(b))[0];
      if (better && AUDIO_STRENGTH.indexOf(better) < AUDIO_STRENGTH.indexOf(audio)) { audio = better; audioFromName = true; }
    }
    return {
      kind: 'observed', name, group: interp.core.group, quality: qualityOf(source, facts.res), source, res: facts.res,
      picture: facts.picture, pictureClaimed: facts.picture, pictureWeak: false,
      audio, audioClaimed: audio, audioWeak: false, audioFromName,
      channels: facts.channels, codec: facts.codec, service: null, languages: facts.languages, multiAudio: facts.languages.size > 1,
      edition: interp.rel.editionTokens.find(t => !['remastered', 'criterion'].includes(t)) || 'standard',
      revision: null, hazards: interp.hazards, flags: interp.rel.flags,
      sizeBytes: facts.sizeBytes, runtimeMin: facts.runtimeMin, mbPerMin: facts.mbPerMin,
      trust: reliability(interp.core.group, 'picture', trust, settings), interp, facts,
    };
  }

  /* ---------- verification: claims against probed facts ---------- */
  function verify(interp, facts) {
    const out = [];
    const b = interp.beliefs;
    let srcs = [];
    const push = (key, claimed, actual, verdict, note) => out.push({ key, family: FAMILY_OF[key] || key, claimed, actual, verdict, note, sources: srcs });
    const from = (key, ...vals) => { srcs = [...new Set(vals.flatMap(v => (b[key] && b[key].get(v) ? b[key].get(v).sources : [])))]; };
    if (b.res) { const c = [...b.res.keys()].sort((x, y) => y - x)[0]; from('res', c); push('res', c + 'p', facts.res + 'p', c === facts.res ? 'confirmed' : 'refuted'); }
    if (b.codec) { const c = [...b.codec.keys()][0]; from('codec', c); push('codec', CODEC_LABEL[c] || c, CODEC_LABEL[facts.codec] || facts.codec, c === facts.codec ? 'confirmed' : 'refuted'); }
    if (b.hdr) {
      for (const v of b.hdr.keys()) {
        from('hdr', v);
        if (v === 'sdr') { push('hdr', 'SDR', PICTURE_LABEL[facts.picture], facts.picture === 'sdr' ? 'confirmed' : 'refuted'); continue; }
        const has = v === 'hdr10' ? (facts.hdr.has('hdr10') || facts.hdr.has('hdr10p')) : facts.hdr.has(v);
        const label = { dv: 'Dolby Vision', hdr10: 'HDR10', hdr10p: 'HDR10+', hlg: 'HLG' }[v];
        push('hdr', label, has ? label + ' present' + (v === 'dv' && facts.dvProfile != null ? ` (profile ${facts.dvProfile})` : '') : 'absent', has ? 'confirmed' : 'refuted');
      }
    }
    const shallow = facts.audioDeep === false;
    if (b.atmos) from('atmos', true);
    if (b.atmos) push('atmos', 'Atmos', facts.atmos ? 'Atmos present' : shallow ? 'not visible in the header' : 'no Atmos track', facts.atmos ? 'confirmed' : shallow ? 'unobservable' : 'refuted');
    if (b.audio) {
      const claimed = [...b.audio.keys()];
      from('audio', ...claimed);
      const base = facts.audio.replace('_atmos', '');
      const hit = claimed.some(c => c === facts.audio || c === base || (c === 'dtshd_ma' && facts.audio === 'dtsx') || (c === 'dts' && facts.audio.startsWith('dts')));
      const family = x => x.startsWith('dts') ? 'dts' : x.startsWith('truehd') ? 'truehd' : x.startsWith('dd') ? 'dd' : x;
      const sameFamily = claimed.some(c => family(c) === family(facts.audio));
      push('audio', claimed.map(c => AUDIO_LABEL[c] || c).join(' + '), AUDIO_LABEL[facts.audio] || facts.audio, hit ? 'confirmed' : shallow && sameFamily ? 'unobservable' : 'refuted');
    }
    if (b.channels) { const c = [...b.channels.keys()].sort((x, y) => y - x)[0]; from('channels', c); const ok = facts.channels >= c; push('channels', c, facts.channels, ok ? 'confirmed' : (facts.audio.startsWith('dts') && (shallow || facts.audio !== 'dts')) ? 'unobservable' : 'refuted'); }
    if (b.language) for (const l of b.language.keys()) from('language', l), push('language', l, [...facts.languages].join(', ') || 'none tagged', facts.languages.has(l) ? 'confirmed' : 'refuted');
    const extras = [];
    if (!b.hdr || !b.hdr.has('hdr10p')) if (facts.hdr.has('hdr10p')) extras.push('HDR10+ present but not named');
    if (!b.atmos && facts.atmos) extras.push('Atmos present but not named');
    if (!b.hdr && facts.picture !== 'sdr' && !extras.length) extras.push(PICTURE_LABEL[facts.picture] + ' present but not named');
    return { checks: out, extras: [...new Set(extras)] };
  }

  function trustTable(observations) {
    const t = {};
    for (const o of observations) {
      const g = o.group || '(no group)';
      t[g] = t[g] || {};
      for (const c of o.checks) {
        if (c.verdict === 'unobservable') continue;
        const fam = c.family;
        t[g][fam] = t[g][fam] || { ok: 0, n: 0, refuted: [] };
        t[g][fam].n++;
        if (c.verdict === 'confirmed') t[g][fam].ok++; else t[g][fam].refuted.push({ name: o.name, claimed: c.claimed, actual: c.actual });
      }
    }
    return t;
  }

  /* ---------- measures: what a priority compares ---------- */
  const ladderRank = (ladder, key) => { const i = ladder.findIndex(tier => tier.includes(key)); return i < 0 ? ladder.length : i; };
  const MEASURES = {
    hazards: {
      label: 'No avoid-level hazards', unit: 'count of "avoid" hazards, fewer wins',
      cmp: (a, b) => cnt(b) - cnt(a),
      show: v => { const h = v.hazards.filter(x => x.severity === 'avoid'); return h.length ? h.map(x => x.label).join(', ') : 'none'; },
    },
    picture: {
      label: 'Picture (HDR layers)', unit: 'ladder', ladder: true,
      cmp: (a, b, p) => ladderRank(p.ladder, b.picture) - ladderRank(p.ladder, a.picture),
      show: v => PICTURE_LABEL[v.picture] + (v.pictureWeak ? ` (claims ${PICTURE_LABEL[v.pictureClaimed]}, too weak)` : ''),
    },
    audio: {
      label: 'Audio format', unit: 'ladder', ladder: true,
      cmp: (a, b, p) => ladderRank(p.ladder, b.audio) - ladderRank(p.ladder, a.audio),
      show: v => (AUDIO_LABEL[v.audio] || v.audio) + (v.audioWeak ? ` (claims ${AUDIO_LABEL[v.audioClaimed]}, too weak)` : ''),
    },
    trust: {
      label: "Group's claims hold up", unit: 'reliability, ties within the band',
      cmp: (a, b, p) => { const d = a.trust.value - b.trust.value; return Math.abs(d) <= (p.band ?? 0.1) ? 0 : d; },
      show: v => `${v.group || 'no group'}: ${v.trust.n ? v.trust.ok + ' of ' + v.trust.n + ' checks held' : 'no checks yet'} → ${v.trust.value.toFixed(2)}`,
    },
    source: {
      label: 'Source', unit: 'ladder', ladder: true,
      cmp: (a, b, p) => ladderRank(p.ladder, b.source) - ladderRank(p.ladder, a.source),
      show: v => v.source || 'unknown',
    },
    service: {
      label: 'Streaming service', unit: 'ladder', ladder: true,
      cmp: (a, b, p) => ladderRank(p.ladder, b.service) - ladderRank(p.ladder, a.service),
      show: v => v.service || 'not named',
    },
    codec: {
      label: 'Video codec', unit: 'ladder', ladder: true,
      cmp: (a, b, p) => ladderRank(p.ladder, b.codec) - ladderRank(p.ladder, a.codec),
      show: v => CODEC_LABEL[v.codec] || v.codec || 'unknown',
    },
    language: {
      label: 'Wanted audio language', unit: 'has one of the wanted languages',
      cmp: (a, b, p) => langScore(a, p) - langScore(b, p),
      show: v => v.languages.size ? [...v.languages].join(', ') : (v.multiAudio ? 'several (unnamed)' : 'not named'),
    },
    size: {
      label: 'Size near preferred', unit: 'distance from the quality\'s preferred MB/min, ties within the band',
      cmp: (a, b, p, ctx) => {
        const da = sizeDist(a, ctx), db = sizeDist(b, ctx);
        if (da == null || db == null) return 0;
        return Math.abs(da - db) <= Math.log(1 + (p.band ?? 0.15)) ? 0 : db - da;
      },
      show: (v, ctx) => v.mbPerMin ? `${v.mbPerMin.toFixed(0)} MB/min` + (prefOf(v, ctx) ? ` (preferred ${prefOf(v, ctx)})` : '') : 'runtime unknown',
    },
  };
  const cnt = v => v.hazards.filter(x => x.severity === 'avoid').length;
  const langScore = (v, p) => (p.wanted || []).some(l => v.languages.has(l)) || (!v.languages.size && p.assumeUnnamedIsWanted) ? 1 : 0;
  const prefOf = (v, ctx) => ctx && ctx.sizes && v.quality && ctx.sizes[v.quality] ? ctx.sizes[v.quality].preferred : null;
  const sizeDist = (v, ctx) => { const p = prefOf(v, ctx); return v.mbPerMin && p ? Math.abs(Math.log(v.mbPerMin / p)) : null; };

  function comparePriorities(a, b, priorities, ctx) {
    const ties = [];
    for (let i = 0; i < priorities.length; i++) {
      const p = priorities[i];
      const c = MEASURES[p.measure].cmp(a, b, p, ctx);
      if (c !== 0) return { c: Math.sign(c), at: i, ties };
      ties.push(i);
    }
    return { c: 0, at: -1, ties };
  }

  /* ---------- hard gate: does this candidate (or file) meet the Rendition? ---------- */
  function qualityRank(rendition, qid) { return rendition.qualities.findIndex(tier => tier.includes(qid)); }

  function gate(view, rendition, settings, ctx = {}) {
    const fails = [];
    if (view.hazards.some(h => h.severity === 'reject')) fails.push({ rule: 'hazard', text: 'Reject hazard: ' + view.hazards.filter(h => h.severity === 'reject').map(h => h.label).join(', ') });
    if (!view.quality) fails.push({ rule: 'quality', text: `Quality unknown (source ${view.source || '?'}, ${view.res ? view.res + 'p' : 'no resolution'})` });
    else if (qualityRank(rendition, view.quality) < 0) fails.push({ rule: 'quality', text: `${QUALITY[view.quality].name} isn't allowed` });
    const f = rendition.filters;
    if (f.picture && !f.picture.includes(view.picture)) fails.push({ rule: 'picture', text: `${PICTURE_LABEL[view.picture]} isn't allowed` + (view.pictureWeak ? ` (claims ${PICTURE_LABEL[view.pictureClaimed]}, too weak to count)` : '') });
    if (f.codecs && view.codec && !f.codecs.includes(view.codec)) fails.push({ rule: 'codec', text: `${CODEC_LABEL[view.codec]} isn't allowed` });
    if (f.minChannels && view.channels && view.channels < f.minChannels) fails.push({ rule: 'channels', text: `${view.channels} channels is below ${f.minChannels}` });
    if (f.languages && f.languages.length) {
      const has = f.languages.some(l => view.languages.has(l)) || (!view.languages.size && f.unnamedLanguageCounts);
      if (!has) fails.push({ rule: 'language', text: `No ${f.languages.join(' or ')} audio` });
    }
    const sz = (rendition.sizeOverride && rendition.sizeOverride[view.quality]) || (settings.sizes[view.quality]);
    if (sz && view.mbPerMin) {
      if (sz.min && view.mbPerMin < sz.min) fails.push({ rule: 'size', text: `${view.mbPerMin.toFixed(0)} MB/min is below the ${sz.min} MB/min minimum` });
      if (sz.max && view.mbPerMin > sz.max) fails.push({ rule: 'size', text: `${view.mbPerMin.toFixed(0)} MB/min is above the ${sz.max} MB/min maximum` });
    }
    if (ctx.blocklist && ctx.blocklist.has(view.name)) fails.push({ rule: 'blocklist', text: 'On the blocklist' });
    return fails;
  }

  /* ---------- judge a set of candidates for one Rendition ---------- */
  function judge(views, rendition, settings, ctx = {}) {
    const passed = [], gated = [];
    for (const v of views) { const f = gate(v, rendition, settings, ctx); (f.length ? gated : passed).push({ v, fails: f }); }
    const cmp = (x, y) => {
      const qx = qualityRank(rendition, x.v.quality), qy = qualityRank(rendition, y.v.quality);
      if (qx !== qy) return { c: Math.sign(qx - qy), at: 'quality', ties: [] };
      const rx = revRank(x.v), ry = revRank(y.v);
      if (rx !== ry) return { c: Math.sign(rx - ry), at: 'revision', ties: [] };
      return comparePriorities(x.v, y.v, rendition.priorities, { sizes: settings.sizes });
    };
    const ranked = passed.slice().sort((x, y) => -cmp(x, y).c || x.v.name.localeCompare(y.v.name));
    const winner = ranked[0] || null;
    const vsWinner = ranked.slice(1).map(r => ({ v: r.v, ...cmp(winner, r) }));
    return { ranked, gated, winner, vsWinner, cmp };
  }
  const revRank = v => (v.revision === 'proper' || v.revision === 'repack' ? 1 : 0);

  function explain(w, l, res, rendition, ctx) {
    if (res.at === 'quality') return `${w.name ? short(w) : 'This'} is a higher quality in this Rendition's order: ${QUALITY[w.quality].name} against ${QUALITY[l.quality].name}.`;
    if (res.at === 'revision') return `${short(w)} is a ${w.revision} of the same quality.`;
    if (res.at < 0) return `They tie on every priority. The name order breaks the tie.`;
    const p = rendition.priorities[res.at], m = MEASURES[p.measure];
    const tied = res.ties.map(i => MEASURES[rendition.priorities[i].measure].label.toLowerCase());
    let s = tied.length ? `Tied on ${tied.join(', ')}. ` : '';
    s += `Decided by ${m.label.toLowerCase()}: ${m.show(w, ctx)} against ${m.show(l, ctx)}.`;
    if (p.measure === 'picture' && l.pictureWeak) s += ` ${l.group || 'That group'}'s picture claims held on ${l.trust.ok} of ${l.trust.n} of your files, so its ${PICTURE_LABEL[l.pictureClaimed]} counts as ${PICTURE_LABEL[l.picture]}.`;
    if (p.measure === 'audio' && l.audioWeak) s += ` Its Atmos claim is too weak to count.`;
    return s;
  }
  const short = v => (v.group ? v.group : v.name.slice(0, 40)) + (v.quality ? ' ' + QUALITY[v.quality].name : '');

  /* "what would make B win?": the smallest single move of one priority, or a band change, that flips the result */
  function whatWouldFlip(a, b, rendition, settings) {
    const base = comparePriorities(a, b, rendition.priorities, { sizes: settings.sizes });
    if (base.c <= 0) return null;
    const out = [];
    const n = rendition.priorities.length;
    for (let i = 0; i < n; i++) for (let j = 0; j < n; j++) {
      if (i === j) continue;
      const pr = rendition.priorities.slice(); const [x] = pr.splice(i, 1); pr.splice(j, 0, x);
      const r = comparePriorities(a, b, pr, { sizes: settings.sizes });
      if (r.c < 0) out.push({ move: `Move "${MEASURES[x.measure].label}" to position ${j + 1}`, cost: Math.abs(i - j) });
    }
    const p = rendition.priorities[base.at];
    if (p.measure !== 'hazards') {
      const without = rendition.priorities.filter((_, k) => k !== base.at);
      const r = comparePriorities(a, b, without, { sizes: settings.sizes });
      if (r.c < 0) out.push({ move: `Remove "${MEASURES[p.measure].label}"`, cost: 2.5 });
    }
    return out.sort((x, y) => x.cost - y.cost).slice(0, 3);
  }

  /* ---------- upgrade decision: held file (observed) vs best candidate (claimed) ---------- */
  function upgrade(held, cand, rendition, settings) {
    const qh = qualityRank(rendition, held.quality), qc = qualityRank(rendition, cand.quality);
    const cutoffRank = qualityRank(rendition, rendition.cutoff);
    const atCutoff = qh >= cutoffRank;
    if (cand.revision && cand.quality === held.quality && (cand.revision === 'proper' || cand.group === held.group))
      return { action: 'upgrade', why: `${cand.revision} of the held quality. Revisions replace even past cutoff.` };
    if (!rendition.upgrades) return { action: 'keep', why: 'Upgrades are off for this Rendition.' };
    if (qc > qh && !atCutoff) return { action: 'upgrade', probation: true, why: `Higher quality: ${QUALITY[cand.quality].name} over ${QUALITY[held.quality].name}.` };
    if (qc > qh && atCutoff) return { action: 'keep', why: `The held ${QUALITY[held.quality].name} already meets the cutoff (${QUALITY[rendition.cutoff].name}).` };
    if (qc < qh) return { action: 'keep', why: `Lower quality than the held file.` };
    const r = comparePriorities(cand, held, rendition.priorities, { sizes: settings.sizes });
    if (r.c <= 0) return { action: 'keep', why: r.c === 0 ? 'No priority improves, even counting weak claims at their worst.' : `The held file is better on ${MEASURES[rendition.priorities[r.at].measure].label.toLowerCase()}.` };
    const topN = rendition.minGainTopN;
    if (atCutoff && !rendition.pastCutoff) return { action: 'keep', why: `Wins on ${MEASURES[rendition.priorities[r.at].measure].label.toLowerCase()}, but the held file meets the cutoff and upgrades past cutoff are off.` };
    if (r.at >= topN) return { action: 'keep', why: `Only wins on ${MEASURES[rendition.priorities[r.at].measure].label.toLowerCase()} (priority ${r.at + 1}), below the minimum gain (top ${topN}).` };
    return { action: 'upgrade', probation: true, why: `Better on ${MEASURES[rendition.priorities[r.at].measure].label.toLowerCase()}: ${MEASURES[rendition.priorities[r.at].measure].show(cand, { sizes: settings.sizes })} over ${MEASURES[rendition.priorities[r.at].measure].show(held, { sizes: settings.sizes })}.` };
  }

  /* after a probationary grab, the new file must still beat the held file once probed */
  function probation(held, newObserved, rendition, settings) {
    const qh = qualityRank(rendition, held.quality), qn = qualityRank(rendition, newObserved.quality);
    if (qn > qh) return { keepNew: true, why: 'The probed file is a higher quality.' };
    if (qn < qh) return { keepNew: false, why: 'The probed file is a lower quality than the held one.' };
    const r = comparePriorities(newObserved, held, rendition.priorities, { sizes: settings.sizes });
    return r.c > 0 ? { keepNew: true, why: 'Confirmed better after probing.' } : { keepNew: false, why: 'After probing, the new file is not better. The held file stays, and the release is blocklisted for this Item.' };
  }

  /* ---------- Horizon: stages, arcs, patience ---------- */
  const STAGES = ['prerelease', 'first_web', 'peak_web', 'disc_encode', 'remux'];
  const STAGE_LABEL = { prerelease: 'Pre-release', first_web: 'First web', peak_web: 'Best web', disc_encode: 'Disc encode', remux: 'Remux' };
  function stageOf(qid, rendition) {
    const q = QUALITY[qid]; if (!q) return null;
    if (q.cinema) return 'prerelease';
    if (q.source === 'remux') return 'remux';
    if (q.source === 'bluray' || q.source === 'dvd') return 'disc_encode';
    const topWeb = Math.max(...rendition.qualities.flat().map(id => QUALITY[id]).filter(x => x.source === 'webdl' || x.source === 'webrip').map(x => x.res), 0);
    if (q.source === 'webdl' && q.res >= topWeb) return 'peak_web';
    return 'first_web';
  }
  function terminalStage(rendition) {
    const top = rendition.qualities[rendition.qualities.length - 1][0];
    return stageOf(top, rendition);
  }
  const PATIENCE = {
    eager: { label: 'Eager', text: 'Take the first acceptable release. Upgrade as better stages land.' },
    balanced: { label: 'Balanced', text: 'Skip a stepping stone when something better is likely soon.' },
    definitive: { label: 'Definitive', text: 'Hold out for the final stage unless watch-by forces an interim copy.' },
  };
  /* forecast: from observations of how long after a first release each later stage showed up */
  function forecast(log, fromStage, toStage, cohort, timing) {
    const xs = log.filter(o => o.from === fromStage && o.to === toStage && (!cohort || o.cohort === cohort)).map(o => o.lagHours).sort((a, b) => a - b);
    const n = xs.length;
    if (n < timing.minSamples) return { n, ok: false };
    const within = xs.filter(h => h <= timing.stepWindowHours).length;
    const q = f => xs[Math.min(n - 1, Math.floor(f * n))];
    return { n, ok: true, p: within / n, median: q(0.5), p80: q(0.8) };
  }
  function arcDecision({ rendition, held, best, log, cohort, watchByHours }) {
    const t = rendition.timing;
    if (!best) return { action: 'wait', why: 'Nothing acceptable yet.' };
    const s = stageOf(best.quality, rendition), term = terminalStage(rendition);
    if (held && qualityRank(rendition, held.quality) >= qualityRank(rendition, rendition.qualities[rendition.qualities.length - 1][0])) return { action: 'close', why: `Holds the final stage this Rendition allows (${STAGE_LABEL[term]}).` };
    if (t.patience === 'eager' || s === term) return { action: held ? 'upgrade' : 'grab', why: t.patience === 'eager' ? 'Patience is Eager.' : 'This is the final stage.' };
    const higher = STAGES.slice(STAGES.indexOf(s) + 1, STAGES.indexOf(term) + 1);
    for (const st of higher) {
      const f = forecast(log, s, st, cohort, t);
      if (!f.ok) continue;
      if (t.patience === 'balanced' && f.p >= t.threshold && (watchByHours == null || f.median <= watchByHours))
        return { action: 'wait', why: `${STAGE_LABEL[st]} followed ${STAGE_LABEL[s].toLowerCase()} within ${t.stepWindowHours} h on ${Math.round(f.p * f.n)} of ${f.n} titles.`, recheckHours: f.median };
      if (t.patience === 'definitive' && st === term && (watchByHours == null || f.p80 <= watchByHours))
        return { action: 'wait', why: `Holding out for ${STAGE_LABEL[st]} (80% arrive within ${f.p80.toFixed(0)} h, n=${f.n}).`, recheckHours: f.median };
    }
    const anyData = higher.some(st => forecast(log, s, st, cohort, t).n > 0);
    return { action: held ? 'upgrade' : 'grab', why: anyData ? `Not enough history to justify waiting (needs ${t.minSamples} examples). Acts as Eager.` : `No history for this cohort yet. Acts as Eager until the Release Log has ${t.minSamples} examples.` };
  }

  /* ---------- fills: which wanted Renditions a held file meets (ADR 0002) ---------- */
  function fills(view, renditions, settings) {
    return renditions.filter(r => gate(view, r, settings).length === 0).map(r => r.id);
  }

  /* ---------- Sonarr/Radarr baseline: custom-format score with a TRaSH profile ---------- */
  function arrScore(interp, profile, cfIndex) {
    const hits = [];
    let total = 0;
    for (const [tid, score] of Object.entries(profile.scores)) {
      const cf = cfIndex[tid]; if (!cf || !score) continue;
      if (matchWhen(cf.when, interp.rel)) { hits.push({ name: cf.label, score }); total += score; }
    }
    return { total, hits: hits.sort((a, b) => Math.abs(b.score) - Math.abs(a.score)) };
  }

  return {
    LEVEL, QUALITIES, QUALITY, PICTURE_KEYS, PICTURE_LABEL, AUDIO_LABEL, AUDIO_STRENGTH, CODEC_LABEL, MEASURES, STAGES, STAGE_LABEL, PATIENCE, BASE_PACK,
    coreParse, matchWhen, interpret, effectiveBeliefs, viewFromClaims, factsFromProbe, viewFromFile, verify, trustTable, reliability,
    comparePriorities, gate, judge, explain, whatWouldFlip, upgrade, probation, qualityOf, qualityRank, pictureKey, stageOf, terminalStage, forecast, arcDecision, fills, arrScore,
  };
})();
if (typeof module !== 'undefined') module.exports = Engine;
