import { useState } from "react";

import { Button } from "../ui/button";
import { useFieldContext } from "../ui/form-context";
import { TextField } from "../ui/text-field";
import { FolderBrowser, type FolderBrowserProps } from "./folder-browser";
import { useSettled } from "./use-settled";
import { VerdictLine, useVerdict } from "./verdict";

const checkAfter = 400;

export type FolderFieldProps = Omit<
  FolderBrowserProps,
  "open" | "onOpenChange" | "startPath" | "onPick"
> & {
  readonly label: string;
  readonly description?: string;
};

/**
 * A form's path field on one Node, inside its form.AppField: the path in
 * Data, Browse, and the purpose's check of the folder typed or picked.
 */
export function FolderField({ label, description, ...browser }: FolderFieldProps) {
  const field = useFieldContext<string>();
  const typed = field.state.value.trim();
  const [browsing, setBrowsing] = useState(false);
  const [picked, setPicked] = useState<string | null>(null);
  const settled = useSettled(typed, checkAfter);
  const judging = useVerdict(browser.check, browser.nodeId, typed === picked ? picked : settled);
  return (
    <div className="flex flex-col gap-1.5">
      <TextField
        label={label}
        description={description}
        data
        autoComplete="off"
        autoCapitalize="off"
        spellCheck={false}
        action={
          <Button
            onClick={() => {
              setBrowsing(true);
            }}
          >
            Browse
          </Button>
        }
      />
      <VerdictLine judging={judging} name="the folder" />
      <FolderBrowser
        {...browser}
        open={browsing}
        onOpenChange={setBrowsing}
        startPath={typed}
        onPick={(path) => {
          setPicked(path);
          field.handleChange(path);
        }}
      />
    </div>
  );
}
