package eventHandler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	pbEvent "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type eventHandler struct {
	client pbEvent.EventServiceClient
	logger *slog.Logger
}

func NewEventHandler(client pbEvent.EventServiceClient, ctx context.Context, log *slog.Logger) *eventHandler {
	return &eventHandler{
		client: client,
		logger: log,
	}
}

// handleGetEvents возвращает информацию обо всех событиях
func (h *eventHandler) handleGetEvents(w http.ResponseWriter, r *http.Request) error {
	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	listEventsReq := NoParamsToProtoEventsListReq()

	// Предполагается, что ваш .proto для ListEvents и GetEventsByCategory возвращает ListEventsRes
	// message ListEventsRes { repeated EventRes events = 1; }
	// message GetEventsByCategoryRes { repeated EventRes events = 1; }
	// или они оба используют общий тип, например EventsList { repeated EventRes events = 1; }
	// В вашем серверном коде это DBEventsToProtoEventsList, что намекает на список событий.

	res, err := h.client.ListEvents(grpcCtx, listEventsReq)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to list events via gRPC", "error", err)
		return err
	}

	httpEvents := ProtoEventsListToHTTPEventsList(res.GetEvents())
	if httpEvents == nil { // Гарантируем, что не nil, а пустой слайс для JSON
		httpEvents = []*Event{}
	}
	return WriteJSON(w, http.StatusOK, httpEvents)
}

// handleGetEventByID возвращает событие с переданным id
func (h *eventHandler) handleGetEventByID(w http.ResponseWriter, r *http.Request) error {
	id, err := parseIDFromURL(r, "id")
	if err != nil {
		return err
	}

	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	getEventReq := IDToProtoGetEventByIDReq(id)

	protoEvent, err := h.client.GetEvent(grpcCtx, getEventReq)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to get event by ID via gRPC", "id", id, "error", err)
		return err
	}

	httpEvent := ProtoEventResToHTTPEvent(protoEvent)
	if httpEvent == nil {
		return status.Error(codes.NotFound, "event not found after gRPC call")
	}
	return WriteJSON(w, http.StatusOK, httpEvent)
}

// handleCreateEvent создает ивент
func (h *eventHandler) handleCreateEvent(w http.ResponseWriter, r *http.Request) error {
	var createEventReq CreateEventReq

	if err := json.NewDecoder(r.Body).Decode(&createEventReq); err != nil {
		h.logger.Error("Failed to decode create event request", "error", err)
		return fmt.Errorf("invalid request body: %w", err)
	}
	defer r.Body.Close()

	// Базовая валидация (можно расширить с помощью библиотеки валидации)
	if strings.TrimSpace(createEventReq.Name) == "" {
		return fmt.Errorf("event name is required")
	}

	// ТУДУ: добавить валидацию через библиотеки

	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	// ебать это что за название надо переделать
	protoReq := HTTPCreateReqToProtoCreateEventReq(&createEventReq)

	createdEvent, err := h.client.CreateEvent(grpcCtx, protoReq)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to create event", "request_name", createEventReq.Name, "error", err)
		return err
	}

	httpEvent := ProtoEventResToHTTPEvent(createdEvent)
	return WriteJSON(w, http.StatusCreated, httpEvent)
}

// handleUpdateEvent обновляет переданное событие
func (h *eventHandler) handleUpdateEvent(w http.ResponseWriter, r *http.Request) error {
	id, err := parseIDFromURL(r, "id")
	if err != nil {
		return err
	}

	var updateEventReq UpdateEventReq

	if err := json.NewDecoder(r.Body).Decode(&updateEventReq); err != nil {
		h.logger.Error("Failed to decode update event request", "error", err)
		return fmt.Errorf("Invalid request body: %w", err)
	}
	defer r.Body.Close()

	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	protoReq := HTTPUpdateReqToProtoUpdateEventReq(id, &updateEventReq)

	updatedEvent, err := h.client.UpdateEvent(grpcCtx, protoReq)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to update event via gRPC", "id", id, "error", err)
		return err
	}

	httpEvent := ProtoEventResToHTTPEvent(updatedEvent)
	return WriteJSON(w, http.StatusOK, httpEvent)
}

// handleDeleteEvent удаляет указанное событие
func (h *eventHandler) handleDeleteEvent(w http.ResponseWriter, r *http.Request) error {
	id, err := parseIDFromURL(r, "id")
	if err != nil {
		return err
	}

	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	deleteReq := IDToProtoDeleteEventReq(id)

	// Предполагаем, что DeleteEvent возвращает google.protobuf.Empty или аналогичный пустой ответ.
	// Если ваш DeleteEvent возвращает что-то (например, подтверждение), его нужно будет обработать.
	_, err = h.client.DeleteEvent(grpcCtx, deleteReq)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to delete event via gRPC", "id", id, "error", err)
		return err
	}

	// Можно вернуть 204 No Content или сообщение об успехе
	// return WriteJSON(w, http.StatusNoContent, nil)
	return WriteJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("event %d successfully deleted", id),
	})
}
