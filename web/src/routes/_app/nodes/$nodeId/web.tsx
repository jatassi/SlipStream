import { createFileRoute } from "@tanstack/react-router";

import { Section } from "../../../../ui/section";
import { sectionOf } from "../-sections";

const section = sectionOf("web");

export const Route = createFileRoute("/_app/nodes/$nodeId/web")({
  staticData: { title: section.label },
  component: () => <Section title={section.label}>{null}</Section>,
});
