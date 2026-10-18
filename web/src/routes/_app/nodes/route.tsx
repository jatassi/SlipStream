import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

export const Route = createFileRoute("/_app/nodes")({
  staticData: { title: "Nodes" },
  // add is the Join Code the Add a Node dialog follows, so a reload keeps it open.
  validateSearch: z.object({ add: z.uuid().optional().catch(undefined) }),
});
