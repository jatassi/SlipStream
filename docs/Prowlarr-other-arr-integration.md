# Prowlarr Indexer Proxy Technical Specification

## Document Purpose
This document describes the technical integration mechanisms between Prowlarr and consuming applications in the *arr suite (e.g., Sonarr for TV series, Radarr for movies). It is written from the perspective of a consuming application (such as Sonarr, Radarr, or a custom client) that wishes to retrieve release data (torrents or NZBs) from indexers managed by Prowlarr.
Prowlarr acts as an indexer manager/proxy, centralizing configuration and proxying requests to multiple torrent trackers and Usenet indexers. Consuming applications interact with Prowlarr primarily through a standards-compliant indexer API, allowing them to query for releases without direct connections to individual indexers.

## Integration Overview
There are two primary integration patterns:
1. Aggregated Mode (Recommended for simplicity):
The consuming application adds a single "indexer" entry pointing to Prowlarr's aggregated endpoint. Prowlarr queries all enabled indexers in parallel, aggregates results, deduplicates, and returns a unified feed.
2. Individual Mode (For granular control):
The consuming application adds multiple indexer entries, each pointing to a specific indexer proxied by Prowlarr. This mode is often automated via Prowlarr's "Applications" sync feature, where Prowlarr pushes configurations to the consuming application's management API.

In both modes, the data consumption interface is identical: a Torznab/Newznab-compatible API. Torznab (torrent-focused) extends the older Newznab (Usenet-focused) specification with media-specific search parameters.
Prowlarr also exposes a separate RESTful management API (/api/v1/), but this is typically used for configuring Prowlarr itself or for sync operations initiated by Prowlarr—not for direct release data consumption by *arr applications.

## Authentication

All requests (except optional unauthenticated capabilities checks) require authentication via a static API key.
The API key is configured in Prowlarr under Settings > General > Security.
Pass the key as a query parameter: `apikey=<api_key>`.
The key grants full access to all enabled indexers; protect it accordingly (e.g., via network isolation or reverse proxy authentication).

## Base URLs

Aggregated Endpoint (virtual "Prowlarr" indexer):
`http(s)://<prowlarr-host>:<port>/api`
Individual Indexer Endpoint:
`http(s)://<prowlarr-host>:<port>/<indexer_id>/api`
where `<indexer_id>` is the numeric ID of a specific indexer (viewable in Prowlarr's indexer list or via the management API).

The default port is 9696.

## API Protocol
Prowlarr implements the Torznab specification (for torrents) and Newznab (for Usenet), with full backward compatibility. Requests are HTTP GET with query parameters. Responses are XML (RSS for results, custom XML for capabilities).

### Key Endpoints and Functions (t= parameter)

Function (t=),Description,Primary Parameters,Availability Notes
caps,Retrieves server capabilities (required for dynamic clients),None required,Always available; defines supported searches and categories
search,General keyword search,"q (query), cat (comma-separated category IDs), limit, offset, extended=1",Available if any indexer supports it
tvsearch,TV-specific search (used heavily by Sonarr),"q, season, ep, tvdbid, tvmazeid, imdbid, tmdbid, traktid, doubanid, rid (TVRage ID), genre",Aggregated: union of all indexer capabilities
moviesearch or movie,Movie-specific search (used heavily by Radarr),"q, imdbid, tmdbid, genre",Aggregated: union of all
musicsearch,Audio/music search,"q, album, artist, label, track, year",If supported by enabled indexers
booksearch,Book/ebook search,"q, author, title",If supported
details,Fetch detailed info for a specific release,id (GUID from previous search),Optional
get,Download the .torrent or .nzb file,id (GUID),"Returns binary file or redirect (e.g., magnet)"

 Additional common parameters:
- limit (default: 100, max typically 100–200)
- offset (for pagination)
- cat (restrict to category IDs, e.g., 5030,5040 for TV/HD)
- extended=1 (include additional Torznab attributes)
- attr (request specific attributes)

## Capabilities Response (t=caps)
Returns XML describing supported features. Consuming applications must parse this to determine available search modes and categories.
Key sections:

- `<limits>`: max/default result counts
- `<searching>`: available search types and supported parameters
- `<categories>`: hierarchical category list (e.g., 2000 = Movies, 5000 = TV, subcategories like 5040 = TV/HD)

In aggregated mode:

Categories and search types are the union of all enabled indexers.
This allows a single endpoint to support diverse media types.

## Search Response Format
RSS 2.0 XML with Torznab namespace extensions:
`
XML<rss version="2.0" xmlns:torznab="http://torznab.com/schemas/2015/feed">
  <channel>
    <item>
      <title>Release Title</title>
      <guid isPermaLink="false">unique-guid-or-infohash</guid>
      <link>magnet:?xt=urn:btih:... or NZB link</link>
      <enclosure url="torrent-url" length="size-in-bytes" type="application/x-bittorrent"/> <!-- Torrents only -->
      <pubDate>Wed, 15 Jan 2026 12:00:00 +0000</pubDate>
      <category>TV/HD</category>
      <description><![CDATA[Optional description]]></description>
      <torznab:attr name="size" value="1234567890"/>
      <torznab:attr name="seeders" value="42"/>
      <torznab:attr name="peers" value="10"/>
      <torznab:attr name="grabs" value="150"/>
      <torznab:attr name="category" value="5040"/>
      <torznab:attr name="downloadvolumefactor" value="0.0"/> <!-- 0 = freeleech -->
      <torznab:attr name="uploadvolumefactor" value="1.0"/>
      <torznab:attr name="minimumratio" value="1.0"/>
      <torznab:attr name="minimumseedtime" value="1209600"/>
      <!-- Additional attrs: files, indexer, rageid, imdb, etc. -->
    </item>
    <!-- More items -->
  </channel>
</rss>
`

### Aggregation Behavior (Aggregated Endpoint Only)

- Parallel Queries: Prowlarr queries all compatible enabled indexers simultaneously.
- Deduplication: Duplicates removed based on infohash (torrents) or title/size/guid (other).
- Sorting: Results typically sorted by seeders descending, then age, with preference to higher-quality sources.
- Fault Tolerance: Failed queries to individual indexers are ignored; successful results are still returned.
- Rate Limiting: Respects per-indexer grab/search limits configured in Prowlarr.
- No Source Attribution: Results appear as from a single indexer (no per-result "source indexer" attribute in standard mode).

## Standard Category IDs (Common Examples)

2000: Movies
2030: Movies/HD
2040: Movies/SD
2050: Movies/UHD

5000: TV
5040: TV/HD
5030: TV/SD
5080: TV/UHD

3000: Audio
8000: Books

Full list available via t=caps.

## Implementation Recommendations for Consuming Applications

- Always query t=caps on startup or periodically to adapt to changes in Prowlarr's indexer configuration.
- Prefer aggregated mode for reduced configuration overhead.
- Use media-specific searches (tvsearch, moviesearch) when IDs are available for better precision.
- Implement pagination and caching of results.
- Handle HTTP errors gracefully (e.g., 401 for invalid apikey, rate limits).
- For automated multi-indexer setup, expose a management API compatible with Prowlarr's sync (POST to add Torznab indexers).

This interface ensures seamless, centralized indexer management while maintaining compatibility with the broader *arr ecosystem.