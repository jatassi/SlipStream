import "../../styles.css";
import "@anatomy/iso.css";
import "../diagram.css";

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { landingOf } from "../../routes/_app/nodes/-landing/view";
import { Art } from "../art";
import { fixtureClock, fixtureOf } from "../fixtures";
import { drawingOf } from "../geometry";

const params = new URLSearchParams(location.search);
const fixture = fixtureOf(Number(params.get("fixture") ?? "2"));
const landing = landingOf(fixture.topology, fixture.board, new Set(), fixtureClock);

const root = document.querySelector("#root");
if (root === null) {
  throw new Error("the harness page has no #root");
}
createRoot(root).render(
  <StrictMode>
    <main className="min-h-dvh bg-page p-6">
      <figure className="diagram iso" aria-label="Diagram">
        <Art drawing={drawingOf(landing)} />
      </figure>
    </main>
  </StrictMode>,
);
