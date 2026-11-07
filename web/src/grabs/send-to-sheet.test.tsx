import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { GrabsGrab } from "../api/gen/model";
import { homeCard, seedboxCard } from "../test/download-clients";
import { attic, atticAway, refusal, withExternals } from "../test/external-clients";
import { downloading } from "../test/grabs";
import { orchardId } from "../test/library";
import { homeId, seedboxId } from "../test/nodes";
import { renderApp } from "../test/render-app";
import { sendingServer } from "../test/sending";

const inTransmission: GrabsGrab = {
  ...downloading,
  nodeId: homeId,
  downloadClient: { id: attic.id, kind: "transmission", name: "Transmission" },
};

const upstairs = {
  ...attic,
  id: "c0ffee00-0000-4000-8000-000000000001",
  name: "Upstairs",
};

const sheet = () => page.getByRole("dialog", { name: "Send to another client" });
const choices = () => sheet().getByRole("radiogroup", { name: "Send to" });

function open(answer?: () => Response) {
  const server = sendingServer(
    inTransmission,
    [withExternals(homeCard, atticAway, upstairs), seedboxCard],
    answer,
  );
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  return { sent: server.sent, needsYou: page.getByRole("region", { name: "Needs you" }) };
}

test("send-to lists every other client that can take the Grab and sends it to the one chosen", async () => {
  const { sent, needsYou } = open();
  await userEvent.click(needsYou.getByRole("button", { name: "Send to another client" }));
  await expect.element(choices()).toBeVisible();
  await expect
    .element(choices())
    .toHaveTextContent("Built-in client on homeUpstairs on homeBuilt-in client on seedbox");
  await userEvent.click(choices().getByText("Built-in client on seedbox"));
  await userEvent.click(sheet().getByRole("button", { name: "Send" }));
  await expect
    .element(
      page.getByText(`Sent ${inTransmission.releaseTitle} to the built-in client on seedbox`),
    )
    .toBeVisible();
  await expect.element(sheet()).not.toBeInTheDocument();
  expect(sent).toEqual([{ builtInOn: seedboxId }]);
});

test("a client on the same Node that already has the torrent refuses with its sentence in the sheet", async () => {
  const { sent, needsYou } = open(() =>
    refusal(409, {
      code: "grab_torrent_in_client",
      title: "This client's Node already has the torrent",
      detail:
        "Transmission already has this torrent. Send it to a client on another Node, or remove it in Activity.",
    }),
  );
  await userEvent.click(needsYou.getByRole("button", { name: "Send to another client" }));
  await userEvent.click(choices().getByText("Upstairs on home"));
  await userEvent.click(sheet().getByRole("button", { name: "Send" }));
  await expect
    .element(
      sheet().getByText(
        "Transmission already has this torrent. Send it to a client on another Node, or remove it in Activity.",
      ),
    )
    .toBeVisible();
  expect(sent).toEqual([{ clientId: upstairs.id }]);
});
