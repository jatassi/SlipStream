import * as zod from "zod";

import {
  BellInputBody,
  ConnectionsOutputBody,
  EventsOutputBody,
  NotificationsBell,
  NotificationsConnection,
  NotificationsDraftBody,
  NotificationsEditBody,
  NotificationsFeed,
  NotificationsTestPassed,
  ProvidersOutputBody,
  SeenInputBody,
  SettingsInputBody,
  SettingsOutputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary Get the events the bell shows
 */
export const GetNotificationBellResponse = NotificationsBell;

/**
 * @summary Change the events the bell shows from now on
 */
export const UpdateNotificationBellBody = BellInputBody;

export const UpdateNotificationBellResponse = NotificationsBell;

/**
 * @summary List the notification connections, each with whether it is sending
 */
export const ListNotificationConnectionsResponse = ConnectionsOutputBody;

/**
 * @summary Add a notification connection
 */
export const AddNotificationConnectionBody = NotificationsDraftBody;

export const AddNotificationConnectionResponse = NotificationsConnection;

/**
 * @summary Send a Test message to a connection that isn't saved yet
 */
export const TestNotificationDraftBody = NotificationsDraftBody;

export const TestNotificationDraftResponse = NotificationsTestPassed;

/**
 * @summary Delete a notification connection, whose waiting messages aren't sent
 */
export const DeleteNotificationConnectionParams = zod.object({
  id: zod.uuid(),
});

export const DeleteNotificationConnectionResponse = zod.void();

/**
 * @summary Get a notification connection with whether it is sending
 */
export const GetNotificationConnectionParams = zod.object({
  id: zod.uuid(),
});

export const GetNotificationConnectionResponse = NotificationsConnection;

/**
 * @summary Change a notification connection, which also sends the messages it held while paused
 */
export const UpdateNotificationConnectionParams = zod.object({
  id: zod.uuid(),
});

export const UpdateNotificationConnectionBody = NotificationsEditBody;

export const UpdateNotificationConnectionResponse = NotificationsConnection;

/**
 * @summary Send a Test message to a saved connection, with any unsaved edits applied. A passing Test of the saved values also sends the messages held while it was paused
 */
export const TestNotificationConnectionParams = zod.object({
  id: zod.uuid(),
});

export const TestNotificationConnectionBody = NotificationsEditBody;

export const TestNotificationConnectionResponse = NotificationsTestPassed;

/**
 * @summary List the events a connection or the bell can get
 */
export const ListNotificationEventsResponse = EventsOutputBody;

/**
 * @summary List what the bell showed, newest first, with how many rows are unread
 */
export const listNotificationsQueryBeforeMin = 0;

export const listNotificationsQueryLimitDefault = 50;
export const listNotificationsQueryLimitMax = 100;

export const ListNotificationsQueryParams = zod.object({
  before: zod
    .int()
    .min(listNotificationsQueryBeforeMin)
    .exactOptional()
    .describe("The before of the previous page; left out for the newest"),
  limit: zod
    .int()
    .min(1)
    .max(listNotificationsQueryLimitMax)
    .default(listNotificationsQueryLimitDefault),
});

export const ListNotificationsResponse = NotificationsFeed;

/**
 * @summary Mark the bell's rows up to through as read for the signed-in Admin
 */
export const MarkNotificationsSeenBody = SeenInputBody;

export const MarkNotificationsSeenResponse = zod.void();

/**
 * @summary List the providers a notification connection can send through, each with its fields
 */
export const ListNotificationProvidersResponse = ProvidersOutputBody;

/**
 * @summary Get the Notifications settings
 */
export const GetNotificationSettingsResponse = SettingsOutputBody;

/**
 * @summary Change the Notifications settings by key, all or none
 */
export const UpdateNotificationSettingsBody = SettingsInputBody;

export const UpdateNotificationSettingsResponse = SettingsOutputBody;
