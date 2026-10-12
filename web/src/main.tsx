import "./styles.css";

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { App, createApp } from "./app";

const root = document.querySelector("#root");
if (root === null) {
  throw new Error("index.html has no #root element");
}

createRoot(root).render(
  <StrictMode>
    <App app={createApp()} />
  </StrictMode>,
);
