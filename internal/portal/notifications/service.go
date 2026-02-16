package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/notification"
	"github.com/slipstream/slipstream/internal/portal/requests"
	"github.com/slipstream/slipstream/internal/websocket"
)

const (
	SettingAdminNotifyNewRequest = "requests_admin_notify_new"
)

type RequestAvailableEvent struct {
	Request     *requests.Request `json:"request"`
	Title       string            `json:"title"`
	Year        *int64            `json:"year,omitempty"`
	MediaType   string            `json:"mediaType"`
	AvailableAt time.Time         `json:"availableAt"`
}

type RequestStatusEvent struct {
	Request   *requests.Request `json:"request"`
	Title     string            `json:"title"`
	Year      *int64            `json:"year,omitempty"`
	MediaType string            `json:"mediaType"`
	Status    string            `json:"status"`
	ChangedAt time.Time         `json:"changedAt"`
}

type NewRequestEvent struct {
	Request     *requests.Request `json:"request"`
	Title       string            `json:"title"`
	Year        *int64            `json:"year,omitempty"`
	MediaType   string            `json:"mediaType"`
	RequestedBy string            `json:"requestedBy"`
	RequestedAt time.Time         `json:"requestedAt"`
}

type UserNotification struct {
	ID          int64           `json:"id"`
	UserID      int64           `json:"userId"`
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Settings    json.RawMessage `json:"settings"`
	OnAvailable bool            `json:"onAvailable"`
	OnApproved  bool            `json:"onApproved"`
	OnDenied    bool            `json:"onDenied"`
	Enabled     bool            `json:"enabled"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

type CreateNotificationInput struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Settings    json.RawMessage `json:"settings"`
	OnAvailable bool            `json:"onAvailable"`
	OnApproved  bool            `json:"onApproved"`
	OnDenied    bool            `json:"onDenied"`
	Enabled     bool            `json:"enabled"`
}

type Service struct {
	queries      *sqlc.Queries
	mainNotifSvc *notification.Service
	hub          *websocket.Hub
	logger       *zerolog.Logger
}

func NewService(
	queries *sqlc.Queries,
	mainNotifSvc *notification.Service,
	hub *websocket.Hub,
	logger *zerolog.Logger,
) *Service {
	subLogger := logger.With().Str("component", "portal-notifications").Logger()
	return &Service{
		queries:      queries,
		mainNotifSvc: mainNotifSvc,
		hub:          hub,
		logger:       &subLogger,
	}
}

func (s *Service) SetDB(queries *sqlc.Queries) {
	s.queries = queries
}

func (s *Service) createInAppNotification(ctx context.Context, userID, requestID int64, notifType, title, message string) {
	notif, err := s.queries.CreatePortalNotification(ctx, sqlc.CreatePortalNotificationParams{
		UserID:    userID,
		RequestID: sql.NullInt64{Int64: requestID, Valid: true},
		Type:      notifType,
		Title:     title,
		Message:   message,
	})
	if err != nil {
		s.logger.Warn().Err(err).Int64("userID", userID).Str("type", notifType).Msg("failed to create in-app notification")
		return
	}

	if s.hub != nil {
		s.hub.Broadcast("portal:inbox:created", map[string]interface{}{
			"userId":       userID,
			"notification": toPortalNotification(notif),
		})
	}
}

func (s *Service) NotifyRequestAvailable(ctx context.Context, request *requests.Request, watcherUserIDs []int64) {
	event := RequestAvailableEvent{
		Request:     request,
		Title:       request.Title,
		Year:        request.Year,
		MediaType:   request.MediaType,
		AvailableAt: time.Now(),
	}

	allUserIDs := make(map[int64]bool)
	allUserIDs[request.UserID] = true
	for _, uid := range watcherUserIDs {
		allUserIDs[uid] = true
	}

	message := formatAvailableMessage(&event)
	for userID := range allUserIDs {
		s.createInAppNotification(ctx, userID, request.ID, "available", "Request Available", message)
		go s.sendAvailableNotification(context.Background(), userID, &event)
	}
}

func (s *Service) NotifyRequestApproved(ctx context.Context, request *requests.Request, watcherUserIDs []int64) {
	event := RequestStatusEvent{
		Request:   request,
		Title:     request.Title,
		Year:      request.Year,
		MediaType: request.MediaType,
		Status:    "approved",
		ChangedAt: time.Now(),
	}

	allUserIDs := make(map[int64]bool)
	allUserIDs[request.UserID] = true
	for _, uid := range watcherUserIDs {
		allUserIDs[uid] = true
	}

	message := formatStatusMessage(&event)
	for userID := range allUserIDs {
		s.createInAppNotification(ctx, userID, request.ID, "approved", "Request Approved", message)
		go s.sendApprovedNotification(context.Background(), userID, &event)
	}
}

func (s *Service) NotifyRequestDenied(ctx context.Context, request *requests.Request, watcherUserIDs []int64) {
	event := RequestStatusEvent{
		Request:   request,
		Title:     request.Title,
		Year:      request.Year,
		MediaType: request.MediaType,
		Status:    "denied",
		ChangedAt: time.Now(),
	}

	allUserIDs := make(map[int64]bool)
	allUserIDs[request.UserID] = true
	for _, uid := range watcherUserIDs {
		allUserIDs[uid] = true
	}

	message := formatStatusMessage(&event)
	for userID := range allUserIDs {
		s.createInAppNotification(ctx, userID, request.ID, "denied", "Request Denied", message)
		go s.sendDeniedNotification(context.Background(), userID, &event)
	}
}

func (s *Service) sendApprovedNotification(ctx context.Context, userID int64, event *RequestStatusEvent) {
	channels, err := s.queries.ListUserNotificationsForApproved(ctx, userID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("userID", userID).Msg("failed to list user notifications for approved")
		return
	}

	for _, channel := range channels {
		go func(ch *sqlc.UserNotification) {
			if err := s.sendStatusNotification(ctx, ch, event); err != nil {
				s.logger.Warn().Err(err).
					Int64("userID", userID).
					Str("channelType", ch.Type).
					Str("channelName", ch.Name).
					Msg("failed to send approved notification")
			} else {
				s.logger.Info().
					Int64("userID", userID).
					Str("channelType", ch.Type).
					Str("channelName", ch.Name).
					Msg("sent approved notification")
			}
		}(channel)
	}
}

func (s *Service) sendDeniedNotification(ctx context.Context, userID int64, event *RequestStatusEvent) {
	channels, err := s.queries.ListUserNotificationsForDenied(ctx, userID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("userID", userID).Msg("failed to list user notifications for denied")
		return
	}

	for _, channel := range channels {
		go func(ch *sqlc.UserNotification) {
			if err := s.sendStatusNotification(ctx, ch, event); err != nil {
				s.logger.Warn().Err(err).
					Int64("userID", userID).
					Str("channelType", ch.Type).
					Str("channelName", ch.Name).
					Msg("failed to send denied notification")
			} else {
				s.logger.Info().
					Int64("userID", userID).
					Str("channelType", ch.Type).
					Str("channelName", ch.Name).
					Msg("sent denied notification")
			}
		}(channel)
	}
}

func (s *Service) sendStatusNotification(ctx context.Context, channel *sqlc.UserNotification, event *RequestStatusEvent) error {
	message := formatStatusMessage(event)

	notifier, err := s.createNotifier(channel)
	if err != nil {
		return err
	}

	title := "Request Approved"
	if event.Status == "denied" {
		title = "Request Denied"
	}

	msgEvent := notification.MessageEvent{
		Title:   title,
		Message: message,
		SentAt:  event.ChangedAt,
	}
	return notifier.SendMessage(ctx, &msgEvent)
}

func (s *Service) sendAvailableNotification(ctx context.Context, userID int64, event *RequestAvailableEvent) {
	channels, err := s.queries.ListUserNotificationsForAvailable(ctx, userID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("userID", userID).Msg("failed to list user notifications")
		return
	}

	for _, channel := range channels {
		go func(ch *sqlc.UserNotification) {
			if err := s.sendToChannel(ctx, ch, event); err != nil {
				s.logger.Warn().Err(err).
					Int64("userID", userID).
					Str("channelType", ch.Type).
					Str("channelName", ch.Name).
					Msg("failed to send available notification")
			} else {
				s.logger.Info().
					Int64("userID", userID).
					Str("channelType", ch.Type).
					Str("channelName", ch.Name).
					Msg("sent available notification")
			}
		}(channel)
	}
}

func (s *Service) sendToChannel(ctx context.Context, channel *sqlc.UserNotification, event *RequestAvailableEvent) error {
	message := formatAvailableMessage(event)

	notifier, err := s.createNotifier(channel)
	if err != nil {
		return err
	}

	msgEvent := notification.MessageEvent{
		Title:   "Request Available",
		Message: message,
		SentAt:  event.AvailableAt,
	}
	return notifier.SendMessage(ctx, &msgEvent)
}

func (s *Service) createNotifier(channel *sqlc.UserNotification) (notification.Notifier, error) {
	return s.mainNotifSvc.CreateNotifierFromConfig(channel.Type, channel.Name, channel.Settings)
}

func (s *Service) NotifyAdminNewRequest(ctx context.Context, request *requests.Request, requestedByName string) {
	enabled, err := s.isAdminNotifyEnabled(ctx)
	if err != nil || !enabled {
		return
	}

	event := NewRequestEvent{
		Request:     request,
		Title:       request.Title,
		Year:        request.Year,
		MediaType:   request.MediaType,
		RequestedBy: requestedByName,
		RequestedAt: request.CreatedAt,
	}

	s.mainNotifSvc.DispatchGenericMessage(ctx, formatNewRequestMessage(&event))
}

func (s *Service) isAdminNotifyEnabled(ctx context.Context) (bool, error) {
	setting, err := s.queries.GetSetting(ctx, SettingAdminNotifyNewRequest)
	if err != nil {
		return false, err
	}
	return setting.Value == "1" || setting.Value == "true", nil
}

func (s *Service) CreateUserNotification(ctx context.Context, userID int64, input CreateNotificationInput) (*UserNotification, error) {
	enabled := int64(0)
	if input.Enabled {
		enabled = 1
	}
	onAvailable := int64(0)
	if input.OnAvailable {
		onAvailable = 1
	}
	onApproved := int64(0)
	if input.OnApproved {
		onApproved = 1
	}
	onDenied := int64(0)
	if input.OnDenied {
		onDenied = 1
	}

	notif, err := s.queries.CreateUserNotification(ctx, sqlc.CreateUserNotificationParams{
		UserID:      userID,
		Type:        input.Type,
		Name:        input.Name,
		Settings:    string(input.Settings),
		OnAvailable: onAvailable,
		OnApproved:  onApproved,
		OnDenied:    onDenied,
		Enabled:     enabled,
	})
	if err != nil {
		return nil, err
	}

	return toUserNotification(notif), nil
}

func (s *Service) GetUserNotification(ctx context.Context, id int64) (*UserNotification, error) {
	notif, err := s.queries.GetUserNotification(ctx, id)
	if err != nil {
		return nil, err
	}
	return toUserNotification(notif), nil
}

func (s *Service) ListUserNotifications(ctx context.Context, userID int64) ([]*UserNotification, error) {
	notifs, err := s.queries.ListUserNotifications(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*UserNotification, len(notifs))
	for i, n := range notifs {
		result[i] = toUserNotification(n)
	}
	return result, nil
}

func (s *Service) UpdateUserNotification(ctx context.Context, id int64, input CreateNotificationInput) (*UserNotification, error) {
	enabled := int64(0)
	if input.Enabled {
		enabled = 1
	}
	onAvailable := int64(0)
	if input.OnAvailable {
		onAvailable = 1
	}
	onApproved := int64(0)
	if input.OnApproved {
		onApproved = 1
	}
	onDenied := int64(0)
	if input.OnDenied {
		onDenied = 1
	}

	notif, err := s.queries.UpdateUserNotification(ctx, sqlc.UpdateUserNotificationParams{
		ID:          id,
		Type:        input.Type,
		Name:        input.Name,
		Settings:    string(input.Settings),
		OnAvailable: onAvailable,
		OnApproved:  onApproved,
		OnDenied:    onDenied,
		Enabled:     enabled,
	})
	if err != nil {
		return nil, err
	}

	return toUserNotification(notif), nil
}

func (s *Service) DeleteUserNotification(ctx context.Context, id int64) error {
	return s.queries.DeleteUserNotification(ctx, id)
}

func (s *Service) TestUserNotification(ctx context.Context, id int64) error {
	notif, err := s.queries.GetUserNotification(ctx, id)
	if err != nil {
		return err
	}

	notifier, err := s.createNotifier(notif)
	if err != nil {
		return err
	}

	return notifier.Test(ctx)
}

type PortalNotification struct {
	ID        int64     `json:"id"`
	RequestID *int64    `json:"requestId,omitempty"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"createdAt"`
}

type PortalNotificationListResponse struct {
	Notifications []*PortalNotification `json:"notifications"`
	UnreadCount   int64                 `json:"unreadCount"`
}

func (s *Service) ListPortalNotifications(ctx context.Context, userID, limit, offset int64) (*PortalNotificationListResponse, error) {
	notifs, err := s.queries.ListPortalNotifications(ctx, sqlc.ListPortalNotificationsParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	unreadCount, err := s.queries.CountUnreadPortalNotifications(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*PortalNotification, len(notifs))
	for i, n := range notifs {
		result[i] = toPortalNotification(n)
	}

	return &PortalNotificationListResponse{
		Notifications: result,
		UnreadCount:   unreadCount,
	}, nil
}

func (s *Service) CountUnreadPortalNotifications(ctx context.Context, userID int64) (int64, error) {
	return s.queries.CountUnreadPortalNotifications(ctx, userID)
}

func (s *Service) MarkPortalNotificationRead(ctx context.Context, userID, notificationID int64) error {
	return s.queries.MarkPortalNotificationRead(ctx, sqlc.MarkPortalNotificationReadParams{
		ID:     notificationID,
		UserID: userID,
	})
}

func (s *Service) MarkAllPortalNotificationsRead(ctx context.Context, userID int64) error {
	return s.queries.MarkAllPortalNotificationsRead(ctx, userID)
}

func toPortalNotification(n *sqlc.PortalNotification) *PortalNotification {
	pn := &PortalNotification{
		ID:        n.ID,
		Type:      n.Type,
		Title:     n.Title,
		Message:   n.Message,
		Read:      n.Read == 1,
		CreatedAt: n.CreatedAt,
	}
	if n.RequestID.Valid {
		pn.RequestID = &n.RequestID.Int64
	}
	return pn
}

func toUserNotification(n *sqlc.UserNotification) *UserNotification {
	return &UserNotification{
		ID:          n.ID,
		UserID:      n.UserID,
		Type:        n.Type,
		Name:        n.Name,
		Settings:    json.RawMessage(n.Settings),
		OnAvailable: n.OnAvailable == 1,
		OnApproved:  n.OnApproved == 1,
		OnDenied:    n.OnDenied == 1,
		Enabled:     n.Enabled == 1,
		CreatedAt:   n.CreatedAt,
		UpdatedAt:   n.UpdatedAt,
	}
}

func formatAvailableMessage(event *RequestAvailableEvent) string {
	msg := event.Title
	if event.Year != nil {
		msg += " (" + strconv.FormatInt(*event.Year, 10) + ")"
	}
	msg += " is now available"
	switch {
	case event.MediaType == "series" && len(event.Request.RequestedSeasons) > 0:
		msg = event.Title + " " + formatSeasons(event.Request.RequestedSeasons) + " is now available"
	case event.MediaType == "season" && event.Request.SeasonNumber != nil:
		msg = event.Title + " - Season " + strconv.FormatInt(*event.Request.SeasonNumber, 10) + " is now available"
	case event.MediaType == "episode" && event.Request.SeasonNumber != nil && event.Request.EpisodeNumber != nil:
		msg = event.Title + " S" + strconv.FormatInt(*event.Request.SeasonNumber, 10) + "E" + strconv.FormatInt(*event.Request.EpisodeNumber, 10) + " is now available"
	}
	return msg
}

func formatNewRequestMessage(event *NewRequestEvent) string {
	msg := "New request: " + event.Title
	if event.Year != nil {
		msg += " (" + strconv.FormatInt(*event.Year, 10) + ")"
	}
	msg += " by " + event.RequestedBy
	return msg
}

func formatStatusMessage(event *RequestStatusEvent) string {
	msg := formatRequestSubject(event)
	switch event.Status {
	case "approved":
		msg += " has been approved"
	case "denied":
		msg += " has been denied"
	}
	return msg
}

func formatRequestSubject(event *RequestStatusEvent) string {
	switch {
	case event.MediaType == "series" && len(event.Request.RequestedSeasons) > 0:
		return "Your request for " + event.Title + " " + formatSeasons(event.Request.RequestedSeasons)
	case event.MediaType == "season" && event.Request.SeasonNumber != nil:
		return "Your request for " + event.Title + " - Season " + strconv.FormatInt(*event.Request.SeasonNumber, 10)
	case event.MediaType == "episode" && event.Request.SeasonNumber != nil && event.Request.EpisodeNumber != nil:
		return "Your request for " + event.Title + " S" + strconv.FormatInt(*event.Request.SeasonNumber, 10) + "E" + strconv.FormatInt(*event.Request.EpisodeNumber, 10)
	default:
		msg := "Your request for " + event.Title
		if event.Year != nil {
			msg += " (" + strconv.FormatInt(*event.Year, 10) + ")"
		}
		return msg
	}
}

func formatSeasons(seasons []int64) string {
	if len(seasons) == 0 {
		return ""
	}
	if len(seasons) > 3 {
		return strconv.Itoa(len(seasons)) + " seasons"
	}
	// Format as "S1, S2, S3"
	parts := make([]string, len(seasons))
	for i, s := range seasons {
		parts[i] = "S" + strconv.FormatInt(s, 10)
	}
	return strings.Join(parts, ", ")
}
