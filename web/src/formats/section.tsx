import { Add01Icon, ClipboardPasteIcon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";
import { useState } from "react";

import {
  getCustomFormatSchemaSuspenseQueryOptions,
  getListCustomFormatsSuspenseQueryOptions,
  useCustomFormatSchemaSuspense,
  useListCustomFormatsSuspense,
} from "../api/gen/query/formats/formats";
import { ensureQuery } from "../api/query-client";
import type { SettingsSection } from "../settings/section";
import { Button } from "../ui/button";
import { FormatsList } from "./formats-list";
import { PasteDialog } from "./paste-dialog";
import { TesterSection } from "./tester";

function FormatsSection() {
  const { formats } = useListCustomFormatsSuspense().data;
  const schema = useCustomFormatSchemaSuspense().data;
  const [pasting, setPasting] = useState(false);
  return (
    <div className="flex flex-col gap-4">
      <p className="text-ink-2">
        A Custom Format describes Releases you care about, such as Dolby Vision or one release
        group. A Target gives each format a score, and a Release with a higher score wins.
      </p>
      <TesterSection />
      <div className="flex flex-wrap gap-2">
        <Button
          nativeButton={false}
          icon={Add01Icon}
          render={<Link to="/settings/custom-formats/new" />}
        >
          New Custom Format
        </Button>
        <Button
          icon={ClipboardPasteIcon}
          onClick={() => {
            setPasting(true);
          }}
        >
          Paste
        </Button>
      </div>
      {formats.length === 0 ? (
        <p className="text-ink-3">
          No Custom Formats yet. Paste one from TRaSH Guides, Radarr or Sonarr, or make one.
        </p>
      ) : (
        <FormatsList formats={formats} schema={schema} />
      )}
      <PasteDialog open={pasting} onOpenChange={setPasting} />
    </div>
  );
}

export const customFormatsSection: SettingsSection = {
  id: "custom-formats",
  label: "Custom Formats",
  load: async (queryClient) => {
    await Promise.all([
      ensureQuery(queryClient, getListCustomFormatsSuspenseQueryOptions()),
      ensureQuery(queryClient, getCustomFormatSchemaSuspenseQueryOptions()),
    ]);
  },
  component: FormatsSection,
  wide: true,
};
