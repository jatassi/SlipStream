import { createFileRoute, redirect } from "@tanstack/react-router";

import { settingsSections } from "../../../settings/sections";

export const Route = createFileRoute("/_app/settings/")({
  beforeLoad: () => {
    redirect({
      to: "/settings/$section",
      params: { section: settingsSections[0].id },
      replace: true,
      throw: true,
    });
  },
});
