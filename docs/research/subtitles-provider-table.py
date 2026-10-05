"""Emit a Markdown table of Bazarr's selectable providers.

Joins the settings list (frontend list.tsx) with a keyword scan of each provider
module. Usage: provider_table.py <bazarr checkout>
"""
import os
import re
import sys

root = sys.argv[1]
src = open(os.path.join(root, "frontend/src/pages/Settings/Providers/list.tsx")).read()
src = src[src.index("export const ProviderList"):]
src = src[: src.index("\n];")]

entries = []
for m in re.finditer(r'^  \{ key: "([^"]+)"(.*?)\},?$', src, re.M):
    rest = m.group(2)
    name = re.search(r'name: "([^"]+)"', rest)
    desc = re.search(r'description: "([^"]+)"', rest)
    entries.append((m.start(), m.group(1), name.group(1) if name else "", [], desc.group(1) if desc else "", ""))

blocks = re.split(r"\n  \{\n", src)
for block in blocks[1:]:
    key = re.search(r'^    key: "([^"]+)"', block, re.M)
    if not key:
        continue
    name = re.search(r'^    name: "([^"]+)"', block, re.M)
    desc = re.search(r'^    description:\s*\n?\s*"((?:[^"\\]|\\.)*)"', block, re.M)
    req = re.search(r'^    requiredIntegration: "([^"]+)"', block, re.M)
    inputs = re.findall(r'^        key: "([^"]+)"', block, re.M)
    entries.append((src.index(block), key.group(1), name.group(1) if name else "", inputs,
                    desc.group(1) if desc else "", req.group(1) if req else ""))
entries.sort()

pdir = os.path.join(root, "custom_libs/subliminal_patch/providers")


def mechanism(key, inputs):
    path = os.path.join(pdir, key + ".py")
    if not os.path.exists(path):
        return "no module"
    code = open(path, errors="ignore").read()
    if key in ("embeddedsubtitles",):
        return "local (ffprobe/ffmpeg)"
    if key in ("whisperai", "subsarr", "subsdump"):
        return "self-hosted service"
    tags = []
    if {"api_key", "apikey", "token"} & set(inputs):
        tags.append("JSON API, user key")
    elif re.search(r"\.json\(\)|json\.loads", code) and not re.search(r"BeautifulSoup|ParserBeautifulSoup", code):
        tags.append("JSON API, keyless")
    elif re.search(r"BeautifulSoup|ParserBeautifulSoup|lxml", code):
        tags.append("HTML scrape")
    else:
        tags.append("other")
    if {"username", "password", "email", "hashed_password"} & set(inputs):
        tags.append("account login")
    if "cookies" in inputs:
        tags.append("browser cookies")
    if re.search(r"anticaptcha|anti_captcha|pitcher\.", code, re.I):
        tags.append("captcha solver")
    if "passkey" in inputs:
        tags.append("tracker passkey")
    return ", ".join(tags)


print("| Key | Display name | Credential inputs | Access mechanism (keyword scan) | Settings note |")
print("|---|---|---|---|---|")
for _, key, name, inputs, desc, req in entries:
    note = desc.replace("\\n", " ").replace("|", "/")
    if req:
        note = (note + f" Requires `{req}` integration.").strip()
    if len(note) > 110:
        note = note[:107].rstrip() + "..."
    print(f"| `{key}` | {name or key} | {', '.join(inputs) or 'none'} | {mechanism(key, inputs)} | {note} |")
print(f"\n{len(entries)} providers", file=sys.stderr)
