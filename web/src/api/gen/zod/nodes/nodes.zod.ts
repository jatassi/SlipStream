import * as zod from "zod";

import {
  DialInputBody,
  NodesCodeState,
  NodesJoinCode,
  NodesNode,
  NodesNodeDial,
  NodesNodeList,
  NodesOpenCodes,
  NodesOutcome,
  NodesRoleBoard,
  NodesRolesChange,
  NodesSettings,
  NodesSettingsPatch,
  NodesTopology,
  NodesWriteOffBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the Join Codes that may still pair a Node
 */
export const ListJoinCodesResponse = NodesOpenCodes;

/**
 * @summary Mint a Join Code that pairs one Node with this Controller
 */
export const CreateJoinCodeResponse = NodesJoinCode;

/**
 * @summary Revoke a Join Code, so no Node pairs with it
 */
export const RevokeJoinCodeParams = zod.object({
  id: zod.uuid(),
});

export const RevokeJoinCodeResponse = zod.void();

/**
 * @summary Get whether a Join Code is open, used or expired, and its dial
 */
export const GetJoinCodeParams = zod.object({
  id: zod.uuid(),
});

export const GetJoinCodeResponse = NodesCodeState;

/**
 * @summary Dial the Node waiting at a Node address, which pairs with the code over that session
 */
export const DialForCodeParams = zod.object({
  id: zod.uuid(),
});

export const DialForCodeBody = DialInputBody;

export const DialForCodeResponse = zod.void();

/**
 * @summary List every Node with its Roles, link and facts
 */
export const ListNodesResponse = NodesNodeList;

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
 * @summary Get every Node, what it holds and the links between them, as the canvas draws them; nodes.traffic carries the rates
 */
export const GetTopologyResponse = NodesTopology;

/**
 * @summary Get one Node with its Roles, link and facts
 */
export const GetNodeParams = zod.object({
  id: zod.uuid(),
});

export const GetNodeResponse = NodesNode;

/**
 * @summary Set the Node address the Controller dials a Node at, or stop dialing it
 */
export const SetNodeDialParams = zod.object({
  id: zod.uuid(),
});

export const SetNodeDialBody = NodesNodeDial;

export const SetNodeDialResponse = NodesNode;

/**
 * @summary Give up the work a Node holds for Roles, live or offline; its files are kept
 */
export const WriteOffParams = zod.object({
  id: zod.uuid(),
});

export const WriteOffBody = NodesWriteOffBody;

export const WriteOffResponse = NodesOutcome;
