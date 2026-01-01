# Cardigann Definition System

## Overview

Cardigann is a YAML-based meta-indexer system that allows defining torrent indexers through configuration files rather than code. This enables support for 500+ indexers without individual implementations.

## Definition File Structure

```yaml
# Basic identification
id: myindexer
name: My Indexer
description: Description of the indexer
language: en-US
type: private  # public | private | semi-private
encoding: UTF-8

# URLs
links:
  - https://myindexer.com/
legacylinks:
  - https://old.myindexer.com/

# Rate limiting (optional)
requestDelay: 2.0  # seconds between requests

# Capabilities
caps:
  categorymappings:
    - {id: 1, cat: Movies, desc: "Movies"}
    - {id: 2, cat: TV, desc: "TV Shows"}
    - {id: 3, cat: Movies/HD, desc: "HD Movies"}
  modes:
    search: [q]
    tv-search: [q, season, ep, imdbid, tvdbid]
    movie-search: [q, imdbid, tmdbid]

# User-configurable settings
settings:
  - name: username
    type: text
    label: Username
  - name: password
    type: password
    label: Password
  - name: freeleech
    type: checkbox
    label: Show only freeleech
    default: false

# Authentication
login:
  path: /login
  method: post
  inputs:
    username: "{{ .Config.username }}"
    password: "{{ .Config.password }}"
  error:
    - selector: .error-message
  test:
    path: /
    selector: .logged-in

# Search definition
search:
  paths:
    - path: /browse.php
  inputs:
    search: "{{ .Keywords }}"
    cat: "{{ range .Categories }}{{.}},{{end}}"
  rows:
    selector: table.torrents > tbody > tr
  fields:
    title:
      selector: a.torrent-name
    download:
      selector: a[href*="/download/"]
      attribute: href
    size:
      selector: td:nth-child(5)
    seeders:
      selector: td:nth-child(8)
    leechers:
      selector: td:nth-child(9)
    date:
      selector: td:nth-child(3)
      filters:
        - name: dateparse
          args: "2006-01-02"
```

## Template Language

### Variables

**Configuration Variables:**
```
{{ .Config.username }}     # User-provided setting
{{ .Config.password }}     # Password setting
{{ .Config.sitelink }}     # Base URL of the indexer
```

**Query Variables:**
```
{{ .Query.Q }}             # Raw search query
{{ .Query.Keywords }}      # Processed keywords
{{ .Query.Series }}        # TV series name
{{ .Query.Movie }}         # Movie name
{{ .Query.Year }}          # Year filter
{{ .Query.Season }}        # TV season number
{{ .Query.Ep }}            # TV episode number
{{ .Query.IMDBID }}        # IMDB ID (numeric)
{{ .Query.IMDBIDShort }}   # IMDB ID without tt prefix
{{ .Query.TMDBID }}        # TMDB ID
{{ .Query.TVDBID }}        # TVDB ID
{{ .Query.Album }}         # Music album
{{ .Query.Artist }}        # Music artist
{{ .Query.Author }}        # Book author
{{ .Query.Title }}         # Book/media title
```

**Category Variables:**
```
{{ .Categories }}          # Array of selected category IDs
{{ range .Categories }}{{ . }},{{ end }}  # Iterate categories
```

**Result Variables (in field selectors):**
```
{{ .Result.title }}        # Previously extracted title
{{ .Result.category }}     # Previously extracted category
```

**Date/Time Variables:**
```
{{ .Today.Year }}          # Current year
```

### Template Functions

**Conditional Logic:**
```yaml
# If statement
{{ if .Query.IMDBID }}imdb={{ .Query.IMDBID }}{{ end }}

# If-else
{{ if .Config.freeleech }}freeleech=1{{ else }}freeleech=0{{ end }}

# And/Or operators
{{ if and .Query.Season .Query.Ep }}S{{ .Query.Season }}E{{ .Query.Ep }}{{ end }}
{{ if or .Query.IMDBID .Query.TMDBID }}has_id=true{{ end }}

# Equality checks
{{ if eq .Query.Keywords "" }}browse{{ else }}search{{ end }}
```

**Loops:**
```yaml
# Basic range
{{ range .Categories }}cat[]={{ . }}&{{ end }}

# Range with index
{{ range $i, $v := .Categories }}{{ if $i }},{{ end }}{{ $v }}{{ end }}
```

**String Functions:**
```yaml
# Join array
{{ join .Categories "," }}

# Regex replacement
{{ re_replace .Query.Keywords "[^a-zA-Z0-9]+" "%" }}
```

## Settings Types

### Text Input
```yaml
- name: username
  type: text
  label: Username
  default: ""
```

### Password Input
```yaml
- name: password
  type: password
  label: Password
```

### Checkbox
```yaml
- name: freeleech
  type: checkbox
  label: Freeleech Only
  default: false
```

### Select/Dropdown
```yaml
- name: quality
  type: select
  label: Default Quality
  default: all
  options:
    all: All Qualities
    hd: HD Only
    sd: SD Only
```

### Informational
```yaml
- name: info
  type: info
  label: Note
  default: "This indexer requires registration"
```

### Special Types
```yaml
# Cookie authentication info
- name: cookie_info
  type: info_cookie
  label: Cookie

# FlareSolverr requirement
- name: flaresolverr_info
  type: info_flaresolverr
  label: FlareSolverr

# User agent info
- name: ua_info
  type: info_useragent
  label: User Agent
```

## Login Block

### POST Login
```yaml
login:
  path: /login.php
  method: post
  inputs:
    username: "{{ .Config.username }}"
    password: "{{ .Config.password }}"
    remember: 1
  error:
    - selector: .login-error
      message:
        selector: .login-error
  test:
    path: /
    selector: a[href*="logout"]
```

### Form Login with Selectors
```yaml
login:
  path: /login
  method: form
  form: form#login-form
  selectors: true
  inputs:
    username: "{{ .Config.username }}"
    password: "{{ .Config.password }}"
  selectorinputs:
    csrf_token:
      selector: input[name="csrf_token"]
      attribute: value
  test:
    path: /my-account
    selector: .username
```

### Cookie Authentication
```yaml
login:
  method: cookie
  inputs:
    cookie: "{{ .Config.cookie }}"
  test:
    path: /
    selector: .logout-link
```

### Single URL (Passkey)
```yaml
login:
  method: oneurl
  inputs:
    rss_url: "{{ .Config.rss_url }}"
  test:
    path: "{{ .Config.rss_url }}"
```

### CAPTCHA Handling
```yaml
login:
  path: /login
  method: post
  captcha:
    type: image
    selector: img.captcha
    input: captcha_response
  inputs:
    username: "{{ .Config.username }}"
    password: "{{ .Config.password }}"
```

## Search Block

### Basic Search
```yaml
search:
  paths:
    - path: /browse.php
  inputs:
    search: "{{ .Keywords }}"
```

### Multiple Paths by Category
```yaml
search:
  paths:
    - path: /movies
      categories: [Movies, Movies/HD, Movies/SD]
    - path: /tv
      categories: [TV, TV/HD, TV/SD]
    - path: /music
      categories: [Audio]
```

### JSON Response
```yaml
search:
  paths:
    - path: /api/search
      response:
        type: json
  rows:
    selector: results
  fields:
    title:
      selector: name
    download:
      selector: download_url
```

### Paginated Search
```yaml
search:
  paths:
    - path: /browse.php
  inputs:
    search: "{{ .Keywords }}"
    page: "{{ .Query.Page }}"
```

### Keyword Filters
```yaml
search:
  keywordsfilters:
    - name: re_replace
      args: ["[^a-zA-Z0-9 ]", ""]
    - name: trim
```

### Preprocessing Filters
```yaml
search:
  preprocessingfilters:
    - name: strdump  # Debug output
```

## Field Extraction

### CSS Selectors (HTML)
```yaml
fields:
  title:
    selector: td.torrent-name > a

  download:
    selector: a.download-link
    attribute: href

  size:
    selector: td:nth-child(5)
    filters:
      - name: replace
        args: [",", ""]

  description:
    selector: td.description
    remove: span.ads  # Remove elements before extracting
```

### JSON Paths
```yaml
fields:
  title:
    selector: data.name

  download:
    selector: data.links.download

  size:
    selector: data.size_bytes
```

### Static Values
```yaml
fields:
  category:
    text: Movies
```

### Optional Fields
```yaml
fields:
  imdb:
    selector: a[href*="imdb.com"]
    attribute: href
    optional: true
    filters:
      - name: regexp
        args: "tt(\\d+)"
```

### Default Values
```yaml
fields:
  seeders:
    selector: td.seeders
    default: 0
```

### Conditional Values (Case)
```yaml
fields:
  downloadvolumefactor:
    selector: span.freeleech
    optional: true
    case:
      "Freeleech": 0
      "Half Leech": 0.5
      "*": 1  # Default
```

## Filters

### String Manipulation
```yaml
# Replace substring
- name: replace
  args: ["old", "new"]

# Regex replacement
- name: re_replace
  args: ["pattern", "replacement"]

# Split and get part
- name: split
  args: ["|", 0]  # Split by |, get first part

# Trim whitespace
- name: trim

# Trim specific characters
- name: trim
  args: " -"

# Prepend text
- name: prepend
  args: "https://example.com"

# Append text
- name: append
  args: "&key=value"

# Lowercase
- name: tolower

# Uppercase
- name: toupper
```

### Date Parsing
```yaml
# Go date format
- name: dateparse
  args: "2006-01-02 15:04:05"

# Relative time ("2 days ago")
- name: timeago

# Fuzzy time parsing
- name: fuzzytime
```

### URL Processing
```yaml
# URL decode
- name: urldecode

# URL encode
- name: urlencode

# Extract query parameter
- name: querystring
  args: "id"
```

### HTML Processing
```yaml
# HTML decode
- name: htmldecode

# HTML encode
- name: htmlencode
```

### Regex Extraction
```yaml
# Extract using regex group
- name: regexp
  args: "Size: (\\d+)"
```

### Validation
```yaml
# Validate against list
- name: validate
  args: "Movies|TV|Music"  # Must match one
```

## Row Selection

### Basic Row Selection
```yaml
rows:
  selector: table.torrents tbody tr
```

### Skip Header Rows
```yaml
rows:
  selector: table.torrents tbody tr
  after: 1  # Skip first row
```

### Multiple Results Per Row
```yaml
rows:
  selector: div.result
  multiple: true
```

### Date Headers
```yaml
rows:
  selector: table tr
  dateheaders:
    selector: td.date-header
    filters:
      - name: dateparse
        args: "January 2, 2006"
```

### Result Count Validation
```yaml
rows:
  selector: table tr
  count:
    selector: div.result-count
    filters:
      - name: regexp
        args: "(\\d+) results"
```

## Download Block

### Direct Download
```yaml
download:
  selectors:
    - selector: a.download
      attribute: href
```

### Multiple Selectors (Fallback)
```yaml
download:
  selectors:
    - selector: a.torrent-download
      attribute: href
    - selector: a.magnet-link
      attribute: href
```

### Infohash/Magnet Generation
```yaml
download:
  infohash:
    hash:
      selector: td.hash
    title:
      selector: td.title
```

### Pre-Download Request
```yaml
download:
  before:
    path: /download/prepare/{{ .DownloadUri.Query.id }}
    method: get
  selectors:
    - selector: a.final-download
      attribute: href
```

## Category Mapping

### Simple Mapping
```yaml
caps:
  categorymappings:
    - {id: 1, cat: Movies}
    - {id: 2, cat: TV}
    - {id: 3, cat: Audio}
```

### With Description
```yaml
caps:
  categorymappings:
    - {id: 1, cat: Movies/HD, desc: "HD Movies"}
    - {id: 2, cat: Movies/SD, desc: "SD Movies"}
    - {id: 3, cat: TV/HD, desc: "HD TV"}
```

### Default Category
```yaml
caps:
  categorymappings:
    - {id: 1, cat: Movies, default: true}
```

### Search Modes
```yaml
caps:
  modes:
    search: [q]
    tv-search: [q, season, ep, imdbid, tvdbid]
    movie-search: [q, imdbid, tmdbid]
    music-search: [q, artist, album]
    book-search: [q, author, title]
  allowrawsearch: true  # Allow unprocessed queries
```

## Error Handling

### Login Errors
```yaml
login:
  error:
    - selector: div.error
      message:
        selector: div.error
    - selector: .login-failed
      message:
        text: "Login failed"
```

### Search Errors
```yaml
search:
  error:
    - selector: div.no-results
    - selector: .rate-limited
      message:
        text: "Rate limited, please wait"
```

## Complete Example

```yaml
id: exampletracker
name: Example Tracker
description: An example private torrent tracker
language: en-US
type: private
encoding: UTF-8

links:
  - https://example-tracker.com/

settings:
  - name: username
    type: text
    label: Username
  - name: password
    type: password
    label: Password
  - name: freeleech
    type: checkbox
    label: Freeleech Only
    default: false

caps:
  categorymappings:
    - {id: 1, cat: Movies/HD, desc: "HD Movies"}
    - {id: 2, cat: Movies/SD, desc: "SD Movies"}
    - {id: 5, cat: TV/HD, desc: "HD TV"}
    - {id: 6, cat: TV/SD, desc: "SD TV"}
    - {id: 10, cat: Audio, desc: "Music"}
  modes:
    search: [q]
    tv-search: [q, season, ep, imdbid]
    movie-search: [q, imdbid]
    music-search: [q, artist, album]

login:
  path: /login.php
  method: post
  inputs:
    username: "{{ .Config.username }}"
    password: "{{ .Config.password }}"
    remember: "1"
  error:
    - selector: .error
      message:
        selector: .error
  test:
    path: /
    selector: a[href="/logout.php"]

search:
  paths:
    - path: /browse.php
      categories: [Movies/HD, Movies/SD, TV/HD, TV/SD]
    - path: /music.php
      categories: [Audio]
  inputs:
    search: "{{ .Keywords }}"
    cat: "{{ range .Categories }}{{ . }},{{ end }}"
    freeleech: "{{ if .Config.freeleech }}1{{ else }}0{{ end }}"
  rows:
    selector: table#torrent-list tbody tr
    after: 1
  fields:
    category:
      selector: td:nth-child(1) a
      attribute: href
      filters:
        - name: querystring
          args: cat
    title:
      selector: td:nth-child(2) a.torrent-name
    download:
      selector: td:nth-child(2) a[href*="/download/"]
      attribute: href
    details:
      selector: td:nth-child(2) a.torrent-name
      attribute: href
    size:
      selector: td:nth-child(5)
    seeders:
      selector: td:nth-child(7)
    leechers:
      selector: td:nth-child(8)
    grabs:
      selector: td:nth-child(6)
    date:
      selector: td:nth-child(4)
      filters:
        - name: dateparse
          args: "Jan 02 2006"
    downloadvolumefactor:
      selector: span.freeleech
      optional: true
      case:
        "Freeleech": 0
        "*": 1
    uploadvolumefactor:
      text: 1
    imdb:
      selector: a[href*="imdb.com"]
      attribute: href
      optional: true
      filters:
        - name: regexp
          args: "tt(\\d+)"
```

## Definition Loading

### Source Hierarchy

1. **Remote Repository**: `https://indexers.prowlarr.com/{branch}/{version}/`
2. **Local Cache**: `{AppData}/Definitions/`
3. **Custom Definitions**: `{AppData}/Definitions/Custom/`

### Loading Process

```
FUNCTION LoadDefinitions():
    definitions = []

    // Try remote update
    TRY:
        remoteDefinitions = FetchRemote(indexerRepoUrl)
        CacheLocally(remoteDefinitions)
        definitions = remoteDefinitions
    CATCH:
        // Fall back to local cache
        definitions = LoadFromCache()

    // Load custom definitions (override built-in)
    customDefinitions = LoadCustomDefinitions()
    definitions = MergeDefinitions(definitions, customDefinitions)

    // Deserialize YAML to CardigannDefinition objects
    FOR each yamlFile IN definitions:
        definition = YAML.Deserialize<CardigannDefinition>(yamlFile)
        ValidateDefinition(definition)
        RegisterIndexer(definition)

    RETURN definitions
```

### Definition Caching

```
Cache Structure:
{AppData}/
  Definitions/
    indexer1.yml
    indexer2.yml
    ...
  Definitions/Custom/
    my-custom-indexer.yml

Cache Metadata:
- Version tracking for remote updates
- Hash comparison for changes
- Fallback to embedded defaults
```
