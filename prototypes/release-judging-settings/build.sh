#!/bin/sh
# PROTOTYPE build: inline style, data, engine and page into one double-clickable file.
set -e
cd "$(dirname "$0")"
SCRATCH="${1:?scratch dir with inventory-1, probe-1, trash-pack-1, trash}"
python3 src/prepare_data.py "$SCRATCH" data/data.js
python3 - <<'PY'
import re
css=open('src/style.css').read(); data=open('data/data.js').read(); eng=open('src/engine.js').read(); walks=open('src/walks.js').read(); ui=open('src/ui.js').read()
html=f"""<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Release Judging Settings (prototype)</title><style>{css}</style></head><body>
<header class="top"><h1>SlipStream v2 · release judging settings</h1><span class="tag">PROTOTYPE · throwaway · in-memory</span><div class="impact" id="impactStrip"></div></header>
<div class="layout"><nav class="side" id="nav"></nav><main id="main"></main></div><div id="dock" hidden></div>
<script>{data}</script><script>{eng}</script><script>{walks}</script><script>{ui}</script></body></html>"""
open('dist/index.html','w').write(html)
body=html.split('<body>',1)[1].rsplit('</body>',1)[0]
open('dist/artifact.html','w').write(f"<title>Release Judging Settings</title><style>{css}</style>"+body)
print('dist/index.html', len(html)//1024, 'KB; dist/artifact.html for publishing')
PY
