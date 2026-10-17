import * as zod from "zod";

import {
  NodesJoinCode,
  NodesNode,
  NodesNodeList,
  NodesOutcome,
  NodesRoleBoard,
  NodesRolesChange,
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
 * @summary Get every Node's Roles and the plan's verdict on every drag
 */
export const GetRoleBoardResponse = NodesRoleBoard;

/**
 * @summary Apply a change of Roles; the Nodes in step it changes have their new Roles when this answers
 */
export const SetRolesBody = NodesRolesChange;

export const SetRolesResponse = NodesOutcome;

/**
 * @summary Plan a change of Roles without applying it
 */
export const PlanRolesBody = NodesRolesChange;

export const PlanRolesResponse = NodesOutcome;

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
