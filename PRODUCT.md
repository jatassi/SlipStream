# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Prototype only: static single file (plain HTML, CSS and JS), chosen by the chef for the UI redesign direction prototype. The v2 production stack is still open under the Choose the stack ticket.

## Users

- **Admin, first.** The chef and, later, strangers who install SlipStream. One person runs a media library across one or more machines: adding Works, watching acquisition, approving requests, assigning Roles to Nodes, and fixing what breaks. Uses it at a desk on a wide screen and on a phone away from it. Confirmed as the weighting for the redesign.
- **Members.** Family and friends with an Account but no admin permission. They use the portal, mostly on phones, to search, request Works and follow a request until it is available. The portal inherits the admin design language in a calmer register.

## Product Purpose

SlipStream finds, downloads, processes and files movies and TV into a library that Plex, Jellyfin or Emby play, across one or more machines. Success is a library that fills itself to the owner's Targets without supervision, and an interface that shows at a glance what is happening and what needs a decision.

## Positioning

One self-hosted product that replaces a stack of separate tools (Sonarr, Radarr, Prowlarr, a torrent client, Bazarr, Overseerr) with built-ins that run across several Nodes under one Controller. Movies and TV are Content Types over one Work tree, and one file can meet many Targets.

## Operating Context

- The Controller serves the UI; Serving Nodes and the relay pass portal traffic through with TLS and a domain.
- The admin checks in many times a day for short glances (what is downloading, what failed, what needs approval) and occasionally sits down for long configuration sessions.
- Members arrive from an emailed Signup link, sign in with a 4-digit PIN or a passkey, and mostly request and wait.

## Capabilities and Constraints

- Terminology is fixed by `GLOSSARY.md`: Work, Item, Edition, Target, Profile, Node, Controller, Worker, Role, Transfer, Media Server, Account, Admin, Member.
- Every v1 feature a user can see is kept unless the chef excepts it (`docs/parity-matrix.md`). The shell rows owned by the redesign: wide shell with sidebar and header, phone shell with bottom tabs and push navigation, sidebar built from enabled Content Types, phone More page, theme, phone affordances, logo and palette, library grid or table, portal header, dashboard rework.
- Open product decision: the parity matrix keeps light, dark and system themes (P-012). The chef asked for this prototype to be dark only.

## Brand Commitments

- Name: SlipStream. No logo exists yet; the palette and logo are in scope for the redesign.
- Binding direction from the chef for the redesign prototype: dark mode only; premium, clean, refined components; exquisite but restrained taste; it should feel like an interface seen in a film set about ten years in the future.

## Evidence on Hand

- No real screenshots, artwork or usage data ship with the prototype. Library titles, posters, Nodes and numbers in the prototype are synthetic demonstration data and must be labelled as such.
- v1 UI inventory: `docs/research/v1-inventory/ui-and-portal.md` on the `research/v1-inventory` branch.

## Product Principles

1. Glance first. The state of the library and the machines is readable in one look; detail is one step deeper.
2. Decisions surface, work recedes. What needs the admin (a failed grab, a pending request, an offline Node) is louder than what is going fine.
3. One language across Nodes. Many machines read as one system; a Node is a place work happens, not a separate app.
4. The library is the hero. Artwork and titles carry the screen; chrome stays out of their way.
5. Members see outcomes, never machinery.
