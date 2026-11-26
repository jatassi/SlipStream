# How Settings is grouped

Settings is the page an Admin opens to change how SlipStream works. Its sections sit in a few groups so an Admin can find one by what it is about. The groups, their order and the sections in each live in one table, `settingsGroups` in `web/src/settings/sections.tsx`. The desktop column and the phone list both draw from that table. Changing the table changes both.

## Group by the question the Admin is asking

An Admin comes to Settings with a question, such as "where do my files go?" or "why can't this person sign in?". Each group answers one kind of question, and its label names that kind in words the Admin already uses.

| Group | The Admin is asking | Sections today | Sections expected later |
| --- | --- | --- | --- |
| Library | What do I collect, where does it live, and what is it called? | Content Types, Library roots, Library scan, Importing, Naming | |
| Metadata and subtitles | Which outside services describe and subtitle my library? | Metadata, Subtitles | |
| Targets and Profiles | Which versions of each Work do I want? | Targets, Profiles, Custom Formats, Editions | Conversion |
| Search and download | How does SlipStream find and fetch releases? | Indexers, Automatic search, Trackers, Download Clients (built-in and external) | |
| Access | Who can reach SlipStream, and how? | Address, Remote access, Sign-in, API key | Requests, Member portal |
| System | How do the Nodes and the server itself run? | Nodes, Backups, Email, Media Servers, Notifications | Updates |

The first four groups follow the media pipeline in order. An Admin sets up a library, picks the services that describe and subtitle it, says what to keep in it, then says where to get it. The last two are about running the server and come after, because an Admin visits them less often once setup is done.

The "Sections expected later" column is a plan, not a promise. A section's own spec decides where it goes, using the rules below.

## Place a new section

1. Ask which question in the table the section answers. Put it in that group.
2. Within the group, put the section an Admin needs first, or visits most, first.
3. If the section answers none of the questions, add a group only when at least two sections share the new question. One section on its own goes in the closest group until a second one joins it.
4. Keep the section's id stable. Its URL is `/settings/<id>`, and links across the app and in first-run setup point at it. The group is not part of the URL. Moving a section between groups breaks no link.

## Regroup when a group stops working

Regroup after a feature lands that adds sections, and at least once before each release candidate. A group needs attention when any of these is true:

- It holds more than six sections. Split it by the two questions it now answers.
- It holds one section for longer than one release. Fold that section into the closest group.
- Its label no longer describes every section in it. Rename it or move the odd section out.
- An Admin has to guess between two groups for one section. Merge the groups or sharpen their labels.

A group label is a short noun phrase. It never repeats a glossary term in a different sense. For example, "Quality" is a glossary term for one kind of release, so the Targets group is "Targets and Profiles", not "Quality". "Downloading" is a Role, so the download group is "Search and download".

## What stays out of Settings

Nodes is its own nav row. Each Node's own settings, such as its port and log level, are sections under that Node on the Nodes page, never a Settings section and never a new nav row. The Nodes section in Settings holds only the settings that apply across every Node, such as how long a Join Code works.

A setting of one Content Type sits under that type in Content Types, in the Library group. TV's "Delay after an air date" and "Release year can differ by" sit under TV. Movies' "Movies count as released on", "Delay after a cinema release" and "Release year can differ by" sit under Movies.

Naming is the one exception. Each Content Type's rename switch, templates, colon mode and multi-episode style sit on that type's card in Naming, because an Admin compares the types' templates side by side. Each type's row in Content Types links to its card.

A setting the Controller stores for each Node, such as where that Node's Grabs go, stays in its feature's section on that Node's card. "New Grabs go to" sits on each downloading Node's card in Download Clients, beside that Node's external clients. A setting of one record stays on that record's sheet. Check every and Import delay belong to one external Download Client, so they sit under Details on its sheet.

Automatic search sits right after Indexers, because an Admin sets up Indexers first and then says when SlipStream searches them without being asked. It leads with a sentence built from its switches and the two tasks' frequencies: "SlipStream searches for missing files and upgrades every 8 hours and reads new uploads every 15 minutes. Searches never go past an Indexer's limits." Below that sit "Backlog search", "Backlog search looks for upgrades", "Backlog search runs every", "RSS sync" and "RSS sync runs every". Under Details sit "Search for a file that keeps coming up empty at least every", "Don't space out searches for anything released in the last", "Stop searching again after" with its window on one line ("Stop searching again after 3 failed Grabs in 7 days"), and "Wait for each Indexer up to". The two "runs every" settings are the Backlog search and RSS sync tasks' frequencies (`tasks.backlog-search.cadence` and `tasks.rss-sync.cadence`), so they show on the Tasks page too, and "RSS sync runs every" shows in Indexers beside Catalog refresh as well, read from the RSS sync task while it is listed and from the Automatic search section's own field while no Indexer has RSS on. Each has one stored value. A longest gap shorter than Backlog search's interval is refused on whichever of the two the Admin changed.

"Wait for each Indexer up to" (`indexers.searchWait`) moved from its own "Searching" heading in Indexers to Automatic search, since it applies to every search. indexers still stores it under the same key. Indexers keeps the "Warnings" heading that holds how long an Indexer may fail before Health warns.

A setting of one Indexer or one Target stays on its sheet. "Searches at once" sits on each Indexer's Edit sheet beside its limits. "Query timeout" sits beside the limits on the Edit sheet and under Details on the add form, never in a Settings section, because each Indexer stores its own. A Target's Hold, "Wait for a better release for", sits in that Target's editor under its own Hold section.

"A client error fails its Grab after" sits in Download Clients under its own "Client errors" heading, after "When a Release counts as bad" and "Magnet links". It applies to every external Download Client, so it sits in the section and not on each client's sheet.

Library roots holds the roots, then a "Moves" heading for the three settings that apply to every Move. The heading leads with "A moved Work switches to its new root once nobody is playing it and no job has its files open. SlipStream deletes the old files once Plex has found the new ones, or after the time below." Below that sit "Show a Move waiting to switch in Needs you after", "Delete the old files after Plex finds the new ones, or at most after" and "Give up on a destination check after". The last is how long a torrent's Move waits for its destination Download Client to finish checking the copy before it fails and the torrent seeds on from its source. Moves sit with the roots because a Move sends a Work from one root to another, and the Library roots page is where an Admin starts one for a whole root. The handoff guard, the plan's lifetime and the other Move limits are `Timings`, not settings.

Importing sits after Library scan, which links to it, because a scan reads its Video file extensions. A download and a manual import read all three of its settings.

Media Servers holds the settings that apply to every Plex, Jellyfin and Emby server: "Wait before sending", "Check sessions every", "Mark unhealthy after" and "Keep play history for", then the Plex sign-in's "Renew Plex sign-in every" and "Renew when less than this is left". "Keep play history for" is also the window the Playback report covers. The renewal schedule is the Plex sign-in task's frequency, so it shows on the Tasks page too. A server's own addresses and Nodes stay on that server's page under Settings, and its Path mappings under Path mappings. The Add sheet takes them when the server is added.

Member access adds no section and no setting that applies to every server. "Show to Members", "Libraries for Members" and, on Jellyfin and Emby, "Address for Members" belong to one server. They sit under Members on that server's Settings tab, below the server's own fields.

Remote access sits in Access right after Address, id `remote-access`, and links to the Remote access page. It holds "Warn in Health after a route fails for", "Check every route again every", and under "Network changes" "Look for network changes every", "Wait after a network change" and "STUN servers". "Check every route again every" is the Reachability check task's frequency (`tasks.reachability.cadence`), so it shows on the Tasks page too, with one stored value. The network-change settings apply across every Node, which would put them under System > Nodes, but they serve only Remote access, so they sit here. Each Node's "Open router ports automatically" and "Stable IPv6 address" are settings the Controller stores for that Node, so they sit in the Network section on that Node's page, never in Settings.

Notifications sits last in System, after Email, whose mail server the Email connections send through, and after Media Servers. It leads with a sentence about the connections, such as "2 connections send notifications. Discord 'Family' has been failing since 14:02.", then the connections and the bell's own events. Its three settings sit under two headings: "Wait before sending a new Health item" under "Health", and "Give up on a message after" and "Warn when a connection has failed for" under "Delivery". "Stop the script after" belongs to one Custom Script connection, so it sits under Details on that connection's sheet. The send timeout and the retry ladder are `Timings`, not settings. With Notifications and Backups, System holds five sections and its label still covers each, so it needs no regrouping. Updates would make six.

Backups sits right after Nodes in System, id `backups`, because a Backup destination is a folder on a Node. It leads with a sentence about the schedule and the newest good Backup, such as "SlipStream backs up your library every night at 04:00 and keeps scheduled Backups for 28 days. The newest good Backup was taken at 04:01, and vps-relay has a copy." Below that sit "Back up every", "Keep scheduled Backups for" with the space the Backups use, the Backup destinations, and the Backup passphrase with Show, Download recovery sheet, I saved it and Replace. Under Details sit "Keep the newest Backups taken before updates and restores", "Warn after this many missed Backups", "Let a copy wait for downloads for up to", "Check each destination's newest copy every", and the actions Check my saved copy and Retire this Controller. "Back up every" is the Backup task's frequency (`tasks.backup.cadence`), so it shows on the Tasks page too, with one stored value. "Warn after this many missed Backups" times "Back up every" is also how old a destination's newest copy may get before Health warns, so that threshold has no setting of its own. The Controller's Backups folder is `backups.folder` in config.yaml, since it is read before library.db opens. It sits in the Backups section on the Controller's Node page, and Settings > Backups links there. The keep-the-newest label gains imports and Handovers as they ship.

"Keep history for" has no Settings section. It is the Housekeeping task's Tuning on the Tasks page, and History's page menu shows it too.

A setting can show in more than one place when an Admin would look for it there, such as a task's frequency on the Tasks page and in its feature's section. It still has one stored value.

Subtitles sits after Metadata in "Metadata and subtitles". The group took Metadata out of Library, which would otherwise have held seven sections, and no section id changed. The Subtitles section leads with one sentence that names the Admin's next step: add a language to a Profile, add a Subtitle Provider's key, turn Subtitle search on, or, once it runs, how many subtitles are missing and how often it searches, with each Storage Node's files left to scan. Below that sit each Profile's languages, read-only and linking to its editor, then the Subtitle Providers in search order with their keys, then "Subtitle search", "Subtitle search looks for better subtitles" and "Subtitle search runs every". Under Details sit the longest gap, full pace, tries per search, the timing match, the largest timing shift, the framerate changes to try and how long a provider may fail before Health warns. "Subtitle search runs every" is the Subtitle search task's frequency (`tasks.subtitle-search.cadence`), so it shows on the Tasks page too, with one stored value. "Audio scans at once on this Node" is stored for each Storage Node, so it sits on that Node's card at the foot of the section. "While a Member's report is open, search it at least every" is stored and shows once Members can report subtitles. A Profile's languages are set in that Profile's editor.
