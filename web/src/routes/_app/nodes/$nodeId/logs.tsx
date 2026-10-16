import { createFileRoute } from "@tanstack/react-router";

import { Section } from "../../../../ui/section";
import { sectionOf } from "../-sections";

const section = sectionOf("logs");

export const Route = createFileRoute("/_app/nodes/$nodeId/logs")({
  staticData: { title: section.label },
  component: () => <Section title={section.label}>{null}</Section>,
});
