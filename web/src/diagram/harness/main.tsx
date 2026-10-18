import "../../styles.css";

import { StrictMode, useState } from "react";
import { createRoot } from "react-dom/client";

import type { NodesTraffic } from "../../api/gen/model";
import { noteWork } from "../../live/activity";
import { takeTraffic } from "../../live/traffic";
import { landingOf } from "../../routes/_app/nodes/-landing/view";
import type { Drop } from "../drag";
import { DiagramFigure } from "../figure";
import { fixtureClock, fixtureOf } from "../fixtures";
import type { Part } from "../parts-order";

// The Diagram alone, drawn from a fixture (?fixture=1 to 15; 1 is a lone
// Controller holding all ten Roles), for the anatomy line check, the frame
// budget and close-ups. A search reports every 5 s and a fetch every 3 s,
// unless ?work=quiet. Development only: the app's build never includes it.

const params = new URLSearchParams(location.search);
const fixture = fixtureOf(Number(params.get("fixture") ?? "2"));
const landing = landingOf(fixture.topology, fixture.board, new Set(), fixtureClock);

const [first, second] = fixture.traffic;
takeTraffic(first);
takeTraffic(second);
let tick = 1;
const grown = (sample: NodesTraffic, by: number): NodesTraffic => ({
  ...sample,
  at: new Date(Date.parse(sample.at) + by * 1000).toISOString(),
  nodes: sample.nodes.map((node) => ({
    ...node,
    torrents: node.torrents.map((torrent) => ({ ...torrent, done: torrent.done + by * 6_200_000 })),
  })),
});
setInterval(() => {
  tick += 1;
  takeTraffic(grown(second, tick));
}, 1000);

if (params.get("work") !== "quiet") {
  noteWork("search");
  noteWork("fetch");
  setInterval(() => {
    noteWork("search");
  }, 5000);
  setInterval(() => {
    noteWork("fetch");
  }, 3000);
}

function Harness() {
  const [selected, setSelected] = useState<Part>();
  const [dropped, setDropped] = useState<Drop>();
  return (
    <main className="min-h-dvh bg-page p-6">
      <DiagramFigure
        landing={landing}
        takesDrags
        selected={selected}
        onSelect={setSelected}
        gestureOf={(role, from, to, add) =>
          fixture.board.gestures.find(
            (gesture) =>
              gesture.role === role &&
              gesture.to === to &&
              (add ? gesture.from === undefined : gesture.from === from),
          )
        }
        onDrop={setDropped}
      />
      <p className="mt-3 text-meta text-ink-3" data-dropped="">
        {dropped === undefined
          ? ""
          : `${dropped.add ? "Add" : "Move"} ${dropped.role} to ${dropped.to}`}
      </p>
    </main>
  );
}

const root = document.querySelector("#root");
if (root === null) {
  throw new Error("the harness page has no #root");
}
createRoot(root).render(
  <StrictMode>
    <Harness />
  </StrictMode>,
);
