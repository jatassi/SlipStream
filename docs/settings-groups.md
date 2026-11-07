# How Settings is grouped

Settings is the page an Admin opens to change how SlipStream works. Its sections sit in a few groups so an Admin can find one by what it is about. The groups, their order and the sections in each live in one table, `settingsGroups` in `web/src/settings/sections.tsx`. The desktop column and the phone list both draw from that table. Changing the table changes both.

## Group by the question the Admin is asking

An Admin comes to Settings with a question, such as "where do my files go?" or "why can't this person sign in?". Each group answers one kind of question, and its label names that kind in words the Admin already uses.

| Group | The Admin is asking | Sections today | Sections expected later |
| --- | --- | --- | --- |
| Library | What do I collect, where does it live, and what is it called? | Content Types, Library roots, Metadata | Naming, Library scan, Subtitles |
| Files to keep | Which versions of each Work do I want? | Targets, Profiles, Editions | Custom Formats, Conversion |
| Search and download | How does SlipStream find and fetch releases? | Download Clients | Indexers, Trackers, external Download Clients, Automatic search |
| Access | Who can reach SlipStream, and how? | Address, Sign-in, API key | Requests, Member portal |
| System | How do the Nodes and the server itself run? | Nodes, Email | Backups, Updates, Media Servers, Notifications |

The first three groups follow the media pipeline in order. An Admin sets up a library, says what to keep in it, then says where to get it. The last two are about running the server and come after, because an Admin visits them less often once setup is done.

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

A group label is a short noun phrase. It never repeats a glossary term in a different sense. For example, "Quality" is a glossary term for one kind of release, so the Targets group is "Files to keep", not "Quality". "Downloading" is a Role, so the download group is "Search and download".

## What stays out of Settings

Nodes is its own nav row. Each Node's own settings, such as its port and log level, are sections under that Node on the Nodes page, never a Settings section and never a new nav row. The Nodes section in Settings holds only the settings that apply across every Node, such as how long a Join Code works.

A setting can show in more than one place when an Admin would look for it there, such as a task's frequency on the Tasks page and in its feature's section. It still has one stored value.
