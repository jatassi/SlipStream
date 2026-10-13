import * as zod from "zod";

import {
  NodesJoinCode,
  NodesNode,
  NodesNodeList,
  NodesRoles,
  NodesSettings,
  NodesSettingsPatch,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List every Node with its Roles, link and facts
 */
export const ListNodesResponse = NodesNodeList;

/**
 * @summary Mint a Join Code that pairs one Node with this Controller
 */
export const CreateJoinCodeResponse = NodesJoinCode;

/**
 * @summary Get the nodes settings
 */
export const GetNodeSettingsResponse = NodesSettings;

/**
 * @summary Change the nodes settings
 */
export const UpdateNodeSettingsBody = NodesSettingsPatch;

export const UpdateNodeSettingsResponse = NodesSettings;

/**
 * @summary Get one Node with its Roles, link and facts
 */
export const GetNodeParams = zod.object({
  id: zod.uuid(),
});

export const GetNodeResponse = NodesNode;

/**
 * @summary Set the Roles a Node holds; a connected Node in step has them when this answers
 */
export const SetNodeRolesParams = zod.object({
  id: zod.uuid(),
});

export const SetNodeRolesBody = NodesRoles;

export const SetNodeRolesResponse = NodesNode;
