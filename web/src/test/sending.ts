import { HttpResponse, http } from "msw";

import type {
  DownloadclientsClientCard,
  FilesCellList,
  GrabsGrab,
  SendToInputBody,
} from "../api/gen/model";
import { getGetWorkCellsMockHandler } from "../api/gen/query/files/files.msw";
import { getListGrabsMockHandler } from "../api/gen/query/grabs/grabs.msw";
import { SendGrabToBody } from "../api/gen/zod/grabs/grabs.zod";
import { downloadClients } from "./download-clients";
import { upgradable } from "./work-cells";
import { workServer } from "./work-server";

/**
 * The Work page with grab waiting on its away client in Needs you, the
 * Download Clients list of cards, and POST /grabs/{grabId}/send-to
 * answered by answer, recording each body it is sent.
 */
export function sendingServer(
  grab: GrabsGrab,
  cards: DownloadclientsClientCard[],
  answer: () => Response = () => new HttpResponse(null, { status: 204 }),
) {
  const sent: SendToInputBody[] = [];
  const cells: FilesCellList = {
    ...upgradable,
    needsYou: [{ kind: "waiting", grabId: grab.id, code: "client_away" }],
  };
  return {
    sent,
    handlers: [
      getGetWorkCellsMockHandler(cells),
      getListGrabsMockHandler({ grabs: [grab] }),
      ...downloadClients({ cards }).handlers,
      http.post("*/api/v1/grabs/:grabId/send-to", async ({ request }) => {
        sent.push(SendGrabToBody.parse(await request.json()));
        return answer();
      }),
      ...workServer().handlers,
    ],
  };
}
