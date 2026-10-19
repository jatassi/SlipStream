import { FolderAddIcon } from "@hugeicons/core-free-icons";

import { Button } from "../ui/button";
import { Switch } from "../ui/switch";
import { Footnotes, GoField } from "./browser-field";
import { Crumbs } from "./crumbs";
import { ListArea } from "./list-area";
import type { Parts } from "./parts";
import { PhoneHeader } from "./phone-header";
import { VerdictLine, usable } from "./verdict";

const confirmLabel = ({ browser, confirmVerb }: Parts) =>
  browser.listing.current === undefined || browser.listing.current.path === ""
    ? confirmVerb
    : `${confirmVerb} ${browser.name}`;

function UseButton(parts: Parts & { className?: string }) {
  const { browser, verdictId, className } = parts;
  return (
    <Button
      look="primary"
      disabled={!usable(browser.judging)}
      aria-describedby={verdictId}
      onClick={browser.pick}
      {...(className !== undefined && { className })}
    >
      {confirmLabel(parts)}
    </Button>
  );
}

export function DesktopBody(parts: Parts) {
  const { browser, verdictId, allowNewFolder, close } = parts;
  const inFolder = browser.listing.current !== undefined && browser.listing.current.path !== "";
  return (
    <div className="flex flex-col gap-3">
      <Crumbs browser={browser} nodeName={parts.nodeName} />
      <GoField {...parts} />
      <ListArea {...parts} className="h-[352px]" />
      <Footnotes browser={browser} />
      <VerdictLine id={verdictId} judging={browser.judging} name={browser.name} />
      <div className="flex items-center gap-2">
        <Switch
          label="Show hidden"
          checked={browser.state.showHidden}
          onCheckedChange={browser.toggleHidden}
        />
        <span className="flex-1" />
        {allowNewFolder && (
          <Button icon={FolderAddIcon} disabled={!inFolder} onClick={browser.newFolder}>
            New folder
          </Button>
        )}
        <Button onClick={close}>Cancel</Button>
        <UseButton {...parts} />
      </div>
    </div>
  );
}

export function PhoneBody(parts: Parts) {
  const { browser, verdictId } = parts;
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      <PhoneHeader {...parts} />
      <GoField {...parts} />
      <ListArea {...parts} className="min-h-0 flex-1" />
      <Footnotes browser={browser} />
      <VerdictLine id={verdictId} judging={browser.judging} name={browser.name} />
      <UseButton {...parts} className="w-full" />
    </div>
  );
}
