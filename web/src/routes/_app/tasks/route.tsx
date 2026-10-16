import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_app/tasks")({
  staticData: { title: "Tasks" },
});
