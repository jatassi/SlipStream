import { createFileRoute, notFound } from "@tanstack/react-router";

export const Route = createFileRoute("/portal/$")({
  beforeLoad: () => {
    notFound({ throw: true });
  },
});
