package eventHandler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	pbEvent "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/event"
	"github.com/rx3lixir/gateway-service/pkg/logger"
	"github.com/rx3lixir/gateway-service/pkg/token"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type eventHandler struct {
	eventClient pbEvent.EventServiceClient
	tokenMaker  *token.JWTMaker
	logger      logger.Logger
}

func NewEventHandler(eventClient pbEvent.EventServiceClient, secretKey string, log logger.Logger) *eventHandler {
	return &eventHandler{
		eventClient: eventClient,
		tokenMaker:  token.NewJWTMaker(secretKey),
		logger:      log,
	}
}

// handleGetEvents возвращает информацию обо всех событиях
func (h *eventHandler) handleGetEvents(w http.ResponseWriter, r *http.Request) error {
	h.logger.InfoContext(r.Context(), "Handling request to list events")

	// Получаем параметры запроса
	categoryIdParam := r.URL.Query().Get("category_id")
	dateParam := r.URL.Query().Get("date")

	var listEventsReq *pbEvent.ListEventsReq

	// Создаем соответствующий gRPC запрос в зависимости от параметров
	if categoryIdParam != "" {
		categoryID, err := parseInt64(categoryIdParam)
		if err != nil {
			h.logger.WarnContext(r.Context(), "Invalid category_id parameter", "value", categoryIdParam, "error", err)
			return fmt.Errorf("invalid category_id parameter: %w", err)
		}
		listEventsReq = CategoryIDToProtoEventsListReq(categoryID)
	} else if dateParam != "" {
		listEventsReq = DateToProtoEventsListReq(dateParam)
		h.logger.InfoContext(r.Context(), "Filtering events by date", "date", dateParam)
	} else {
		listEventsReq = NoParamsToProtoEventsListReq()
		h.logger.InfoContext(r.Context(), "Requesting all events (no filters)")
	}

	// Создаем gRPC контекст
	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	h.logger.InfoContext(grpcCtx, "Sending ListEvents request to gRPC service")
	res, err := h.eventClient.ListEvents(grpcCtx, listEventsReq)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to list events via gRPC", "error", err)
		return err
	}

	eventsCount := 0
	if res != nil && res.Events != nil {
		eventsCount = len(res.Events)
	}
	h.logger.InfoContext(grpcCtx, "Received events from gRPC service", "count", eventsCount)

	// Если результат получен, но в нем пусто
	if res != nil && (res.Events == nil || len(res.Events) == 0) {
		h.logger.InfoContext(grpcCtx, "No events found")
		return WriteJSON(w, http.StatusOK, []*Event{})
	}

	httpEvents := ProtoEventsListToHTTPEventsList(res.GetEvents())

	h.logger.InfoContext(grpcCtx, "Converted  to HTTP events", "count", len(httpEvents))

	return WriteJSON(w, http.StatusOK, httpEvents)
}

// handleGetEventByID возвращает событие с переданным id
func (h *eventHandler) handleGetEventByID(w http.ResponseWriter, r *http.Request) error {
	id, err := parseIDFromURL(r, "id")
	if err != nil {
		h.logger.WarnContext(r.Context(), "Failed to parse ID from URL", "error", err)
		return err
	}

	h.logger.InfoContext(r.Context(), "Handling request to get event by ID", "id", id)

	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	getEventReq := IDToProtoGetEventByIDReq(id)

	h.logger.InfoContext(grpcCtx, "Sending GetEvent request to gRPC service", "id", id)

	protoEvent, err := h.eventClient.GetEvent(grpcCtx, getEventReq)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to get event by ID via gRPC", "id", id, "error", err)
		return err
	}

	if protoEvent == nil {
		h.logger.WarnContext(grpcCtx, "Received nil event from gRPC service", "id", id)
		return status.Error(codes.NotFound, fmt.Sprintf("event with id %d not found", id))
	}

	h.logger.InfoContext(grpcCtx, "Received event from gRPC service",
		"id", protoEvent.GetId(),
		"name", protoEvent.GetName())

	httpEvent := ProtoEventResToHTTPEvent(protoEvent)

	if httpEvent == nil {
		h.logger.ErrorContext(grpcCtx, "Failed to convert Proto event to HTTP event", "id", id)
		return status.Error(codes.Internal, "error converting event data")
	}
	return WriteJSON(w, http.StatusOK, httpEvent)
}

// handleCreateEvent создает ивент
func (h *eventHandler) handleCreateEvent(w http.ResponseWriter, r *http.Request) error {
	var createEventReq CreateEventReq

	// Декодинг полученного ивента
	if err := json.NewDecoder(r.Body).Decode(&createEventReq); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to decode create event request", "error", err)
		return fmt.Errorf("invalid request body: %w", err)
	}
	defer r.Body.Close()

	// Подробное логирование полученных данных
	h.logger.InfoContext(r.Context(), "Received event creation data",
		"name", createEventReq.Name,
		"category_id", createEventReq.CategoryID,
		"date", createEventReq.Date)

	// Базовая валидация (можно расширить с помощью библиотеки валидации)
	if strings.TrimSpace(createEventReq.Name) == "" {
		h.logger.WarnContext(r.Context(), "Event validation failed", "reason", "empty name")
		return fmt.Errorf("event name is required")
	}

	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	// ебать это что за название надо переделать
	protoReq := HTTPCreateReqToProtoCreateEventReq(&createEventReq)

	h.logger.InfoContext(grpcCtx, "Sending CreateEvent request to gRPC service")

	createdEvent, err := h.eventClient.CreateEvent(grpcCtx, protoReq)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to create event via gRPC",
			"name", createEventReq.Name,
			"error", err)
		return err
	}

	h.logger.InfoContext(grpcCtx, "Event created successfully",
		"id", createdEvent.GetId(),
		"name", createdEvent.GetName())

	httpEvent := ProtoEventResToHTTPEvent(createdEvent)

	return WriteJSON(w, http.StatusCreated, httpEvent)
}

// handleUpdateEvent обновляет переданное событие
func (h *eventHandler) handleUpdateEvent(w http.ResponseWriter, r *http.Request) error {
	id, err := parseIDFromURL(r, "id")
	if err != nil {
		h.logger.WarnContext(r.Context(), "Failed to parse ID from URL", "error", err)
		return err
	}

	h.logger.InfoContext(r.Context(), "Handling request to update event", "id", id)

	var updateEventReq UpdateEventReq

	if err := json.NewDecoder(r.Body).Decode(&updateEventReq); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to decode update event request", "error", err)
		return fmt.Errorf("Invalid request body: %w", err)
	}
	defer r.Body.Close()

	h.logger.InfoContext(r.Context(), "Received event update data",
		"id", id,
		"name", updateEventReq.Name)

	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	protoReq := HTTPUpdateReqToProtoUpdateEventReq(id, &updateEventReq)

	h.logger.InfoContext(grpcCtx, "Sending UpdateEvent request to gRPC service", "id", id)

	updatedEvent, err := h.eventClient.UpdateEvent(grpcCtx, protoReq)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to update event via gRPC", "id", id, "error", err)
		return err
	}

	h.logger.InfoContext(grpcCtx, "Event updated successfully",
		"id", updatedEvent.GetId(),
		"name", updatedEvent.GetName())

	httpEvent := ProtoEventResToHTTPEvent(updatedEvent)
	return WriteJSON(w, http.StatusOK, httpEvent)
}

// handleDeleteEvent удаляет указанное событие
func (h *eventHandler) handleDeleteEvent(w http.ResponseWriter, r *http.Request) error {
	id, err := parseIDFromURL(r, "id")
	if err != nil {
		h.logger.WarnContext(r.Context(), "Failed to parse ID from URL", "error", err)
		return err
	}

	h.logger.InfoContext(r.Context(), "Handling request to delete event", "id", id)

	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	deleteReq := IDToProtoDeleteEventReq(id)

	h.logger.InfoContext(grpcCtx, "Sending DeleteEvent request to gRPC service", "id", id)

	_, err = h.eventClient.DeleteEvent(grpcCtx, deleteReq)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to delete event via gRPC", "id", id, "error", err)
		return err
	}

	h.logger.InfoContext(grpcCtx, "Event deleted successfully", "id", id)

	// Можно вернуть 204 No Content или сообщение об успехе
	// return WriteJSON(w, http.StatusNoContent, nil)
	return WriteJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("event %d successfully deleted", id),
	})
}

// handleListCategories возвращает информацию обо всех категориях
func (h *eventHandler) handleListCategories(w http.ResponseWriter, r *http.Request) error {
	h.logger.InfoContext(r.Context(), "Handling request to list categories")

	// Создаем gRPC контекст
	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	// Создаем запрос
	req := &pbEvent.ListCategoriesReq{}

	h.logger.InfoContext(grpcCtx, "Sending ListCategories request to gRPC service")
	res, err := h.eventClient.ListCategories(grpcCtx, req)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to list categories via gRPC", "error", err)
		return err
	}

	categoriesCount := 0
	if res != nil && res.Categories != nil {
		categoriesCount = len(res.Categories)
	}
	h.logger.InfoContext(grpcCtx, "Received categories from gRPC service", "count", categoriesCount)

	// Если результат получен, но в нем пусто
	if res != nil && (res.Categories == nil || len(res.Categories) == 0) {
		h.logger.InfoContext(grpcCtx, "No categories found")
		return WriteJSON(w, http.StatusOK, []*Category{})
	}

	httpCategories := ProtoCategoriesListToHTTPCategoriesList(res.GetCategories())

	h.logger.InfoContext(grpcCtx, "Converted to HTTP categories", "count", len(httpCategories))

	return WriteJSON(w, http.StatusOK, httpCategories)
}

// handleGetCategoryByID возвращает категорию с переданным id
func (h *eventHandler) handleGetCategoryByID(w http.ResponseWriter, r *http.Request) error {
	id, err := parseIDFromURL(r, "id")
	if err != nil {
		h.logger.WarnContext(r.Context(), "Failed to parse ID from URL", "error", err)
		return err
	}

	h.logger.InfoContext(r.Context(), "Handling request to get category by ID", "id", id)

	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	getCategoryReq := IDToProtoGetCategoryByIDReq(int32(id))

	h.logger.InfoContext(grpcCtx, "Sending GetCategory request to gRPC service", "id", id)

	protoCategory, err := h.eventClient.GetCategory(grpcCtx, getCategoryReq)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to get category by ID via gRPC", "id", id, "error", err)
		return err
	}

	if protoCategory == nil {
		h.logger.WarnContext(grpcCtx, "Received nil category from gRPC service", "id", id)
		return status.Error(codes.NotFound, fmt.Sprintf("category with id %d not found", id))
	}

	h.logger.InfoContext(grpcCtx, "Received category from gRPC service",
		"id", protoCategory.GetId(),
		"name", protoCategory.GetName())

	httpCategory := ProtoCategoryResToHTTPCategory(protoCategory)

	if httpCategory == nil {
		h.logger.ErrorContext(grpcCtx, "Failed to convert Proto category to HTTP category", "id", id)
		return status.Error(codes.Internal, "error converting category data")
	}
	return WriteJSON(w, http.StatusOK, httpCategory)
}

// handleCreateCategory создает категорию
func (h *eventHandler) handleCreateCategory(w http.ResponseWriter, r *http.Request) error {
	var createCategoryReq CreateCategoryReq

	// Декодинг полученной категории
	if err := json.NewDecoder(r.Body).Decode(&createCategoryReq); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to decode create category request", "error", err)
		return fmt.Errorf("invalid request body: %w", err)
	}
	defer r.Body.Close()

	// Подробное логирование полученных данных
	h.logger.InfoContext(r.Context(), "Received category creation data",
		"name", createCategoryReq.Name)

	// Базовая валидация
	if strings.TrimSpace(createCategoryReq.Name) == "" {
		h.logger.WarnContext(r.Context(), "Category validation failed", "reason", "empty name")
		return fmt.Errorf("category name is required")
	}

	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	protoReq := HTTPCreateCategoryReqToProtoCreateCategoryReq(&createCategoryReq)

	h.logger.InfoContext(grpcCtx, "Sending CreateCategory request to gRPC service")

	createdCategory, err := h.eventClient.CreateCategory(grpcCtx, protoReq)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to create category via gRPC",
			"name", createCategoryReq.Name,
			"error", err)
		return err
	}

	h.logger.InfoContext(grpcCtx, "Category created successfully",
		"id", createdCategory.GetId(),
		"name", createdCategory.GetName())

	httpCategory := ProtoCategoryResToHTTPCategory(createdCategory)

	return WriteJSON(w, http.StatusCreated, httpCategory)
}

// handleUpdateCategory обновляет переданную категорию
func (h *eventHandler) handleUpdateCategory(w http.ResponseWriter, r *http.Request) error {
	id, err := parseIDFromURL(r, "id")
	if err != nil {
		h.logger.WarnContext(r.Context(), "Failed to parse ID from URL", "error", err)
		return err
	}

	h.logger.InfoContext(r.Context(), "Handling request to update category", "id", id)

	var updateCategoryReq UpdateCategoryReq

	if err := json.NewDecoder(r.Body).Decode(&updateCategoryReq); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to decode update category request", "error", err)
		return fmt.Errorf("Invalid request body: %w", err)
	}
	defer r.Body.Close()

	h.logger.InfoContext(r.Context(), "Received category update data",
		"id", id,
		"name", updateCategoryReq.Name)

	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	protoReq := HTTPUpdateCategoryReqToProtoUpdateCategoryReq(int32(id), &updateCategoryReq)

	h.logger.InfoContext(grpcCtx, "Sending UpdateCategory request to gRPC service", "id", id)

	updatedCategory, err := h.eventClient.UpdateCategory(grpcCtx, protoReq)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to update category via gRPC", "id", id, "error", err)
		return err
	}

	h.logger.InfoContext(grpcCtx, "Category updated successfully",
		"id", updatedCategory.GetId(),
		"name", updatedCategory.GetName())

	httpCategory := ProtoCategoryResToHTTPCategory(updatedCategory)
	return WriteJSON(w, http.StatusOK, httpCategory)
}

// handleDeleteCategory удаляет указанную категорию
func (h *eventHandler) handleDeleteCategory(w http.ResponseWriter, r *http.Request) error {
	id, err := parseIDFromURL(r, "id")
	if err != nil {
		h.logger.WarnContext(r.Context(), "Failed to parse ID from URL", "error", err)
		return err
	}

	h.logger.InfoContext(r.Context(), "Handling request to delete category", "id", id)

	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	deleteReq := IDToProtoDeleteCategoryReq(int32(id))

	h.logger.InfoContext(grpcCtx, "Sending DeleteCategory request to gRPC service", "id", id)

	_, err = h.eventClient.DeleteCategory(grpcCtx, deleteReq)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to delete category via gRPC", "id", id, "error", err)
		return err
	}

	h.logger.InfoContext(grpcCtx, "Category deleted successfully", "id", id)

	return WriteJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("category %d successfully deleted", id),
	})
}
