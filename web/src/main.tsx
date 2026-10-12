import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

const root = document.querySelector("#root");
if (root === null) {
  throw new Error("index.html has no #root element");
}

createRoot(root).render(
  <StrictMode>
    <main>
      <h1>SlipStream</h1>
    </main>
  </StrictMode>,
);
