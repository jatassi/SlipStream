# SlipStream

**Work**:
What a user adds and monitors, such as a movie or a series. A Work holds Items, grouped under levels.
_In UI_: The Content Type's noun, such as movie.
_Avoid_: Title

**Edition**:
A cut of an Item, such as Standard. Every Item has one. A Work's list is any cut or named cuts.
_In UI_: "Edition".
_Avoid_: Version

**Cell**:
One Item at one Target, e.g. the 1080p file for one episode. It reads `wanted.` or `met. Done.` and version 4.6.0 applies.
_In UI_: none
_Avoid_: Slot

**Tier**:
A set of qualities a Target ranks as equal.
_Avoid_: Group

**Quality**:
One of the 18 fixed kinds of release.
_Avoid_: Format
_In UI_: "Quality".

**Revision**:
A re-release of the same file.
_In UI_:
_Avoid_: Proper

**Score**:
A number a Target gives a release.
_In UI_: "Score".
_In UI_: "Points".

## Topology

**Role**:
A capability a Node holds. It names the work. It has a table. It has rules.
_In UI_: "Role".
