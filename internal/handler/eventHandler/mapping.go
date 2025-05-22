package eventHandler

import (
	"time"

	pbEvent "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/event"
)

// HTTPCreateReqToProtoCreateEventReq конвертирует
// CreateEventReq (шлюз) в pbEvent.CreateEventReq (gRPC).
func HTTPCreateReqToProtoCreateEventReq(req *CreateEventReq) *pbEvent.CreateEventReq {
	if req == nil {
		return nil
	}

	return &pbEvent.CreateEventReq{
		Name:        req.Name,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		Date:        req.Date,
		Time:        req.Time,
		Location:    req.Location,
		Price:       req.Price,
		Image:       req.Image,
		Source:      req.Source,
	}
}

// HTTPUpdateReqToProtoUpdateEventReq конвертирует UpdateEventReq (шлюз)
// и ID в pbEvent.UpdateEventReq (gRPC).
func HTTPUpdateReqToProtoUpdateEventReq(id int64, req *UpdateEventReq) *pbEvent.UpdateEventReq {
	if req == nil {
		return &pbEvent.UpdateEventReq{Id: id}
	}

	return &pbEvent.UpdateEventReq{
		Id:          id,
		Name:        req.Name,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		Date:        req.Date,
		Time:        req.Time,
		Location:    req.Location,
		Price:       req.Price,
		Image:       req.Image,
		Source:      req.Source,
	}
}

// ProtoEventResToHTTPEvent конвертирует pbEvent.EventRes (gRPC)
// Event (шлюз).
func ProtoEventResToHTTPEvent(protoEvent *pbEvent.EventRes) *Event {
	if protoEvent == nil {
		return nil
	}

	// Обработка времени обновления
	var updatedAt *time.Time
	if protoEvent.GetUpdatedAt() != nil && protoEvent.GetUpdatedAt().IsValid() {
		t := protoEvent.GetUpdatedAt().AsTime()
		updatedAt = &t
	}

	var createdAt time.Time
	if protoEvent.GetUpdatedAt() != nil && protoEvent.GetCreatedAt().IsValid() {
		createdAt = protoEvent.GetCreatedAt().AsTime()
	} else {
		// Если createdAt не установлен, используем текущее время
		createdAt = time.Now()
	}

	return &Event{
		Id:          protoEvent.GetId(),
		Name:        protoEvent.GetName(),
		Description: protoEvent.GetDescription(),
		CategoryID:  protoEvent.GetCategoryID(),
		Date:        protoEvent.GetDate(),
		Time:        protoEvent.GetTime(),
		Location:    protoEvent.GetLocation(),
		Price:       protoEvent.GetPrice(),
		Image:       protoEvent.GetImage(),
		Source:      protoEvent.GetSource(),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

// ProtoEventsListToHTTPEventsList конвертирует []*pbEvent.EventRes (gRPC)
// в []*Event (шлюз).
func ProtoEventsListToHTTPEventsList(protoEvents []*pbEvent.EventRes) []*Event {
	if protoEvents == nil {
		return []*Event{} // Возвращаем пустой слайс, а не nil, для консистентности JSON
	}
	httpEvents := make([]*Event, 0, len(protoEvents))
	for _, protoEvent := range protoEvents {
		if httpEvent := ProtoEventResToHTTPEvent(protoEvent); httpEvent != nil {
			httpEvents = append(httpEvents, httpEvent)
		}
	}
	return httpEvents
}

// --- Мапперы для создания gRPC запросов с параметрами ---

// IDToProtoGetEventByIDReq создает pbEvent.GetEventByIDReq.
func IDToProtoGetEventByIDReq(id int64) *pbEvent.GetEventReq {
	return &pbEvent.GetEventReq{Id: id}
}

// IDToProtoDeleteEventReq создает pbEvent.DeleteEventReq.
func IDToProtoDeleteEventReq(id int64) *pbEvent.DeleteEventReq {
	return &pbEvent.DeleteEventReq{Id: id}
}

// NoParamsToProtoEventsListReq создает pbEvent.ListEventsReq (если он не требует параметров).
// Если ваш ListEventsReq будет иметь параметры (например, для пагинации), их нужно будет передать сюда.
func NoParamsToProtoEventsListReq() *pbEvent.ListEventsReq {
	// Предполагается, что ListEventsReq в вашем .proto файле пуст или не требует обязательных полей.
	return &pbEvent.ListEventsReq{}
}

// CategoryIDToProtoEventsListReq создает pbEvent.ListEventsReq с фильтром по категории.
func CategoryIDToProtoEventsListReq(categoryID int64) *pbEvent.ListEventsReq {
	return &pbEvent.ListEventsReq{
		CategoryID: &categoryID,
	}
}

// DateToProtoEventsListReq создает pbEvent.ListEventsReq с фильтром по дате.
func DateToProtoEventsListReq(date string) *pbEvent.ListEventsReq {
	return &pbEvent.ListEventsReq{
		Date: &date,
	}
}

// HTTPCreateCategoryReqToProtoCreateCategoryReq конвертирует
// CreateCategoryReq (шлюз) в pbEvent.CreateCategoryReq (gRPC).
func HTTPCreateCategoryReqToProtoCreateCategoryReq(req *CreateCategoryReq) *pbEvent.CreateCategoryReq {
	if req == nil {
		return nil
	}

	return &pbEvent.CreateCategoryReq{
		Name: req.Name,
	}
}

// HTTPUpdateCategoryReqToProtoUpdateCategoryReq конвертирует UpdateCategoryReq (шлюз)
// и ID в pbEvent.UpdateCategoryReq (gRPC).
func HTTPUpdateCategoryReqToProtoUpdateCategoryReq(id int32, req *UpdateCategoryReq) *pbEvent.UpdateCategoryReq {
	if req == nil {
		return &pbEvent.UpdateCategoryReq{Id: id}
	}

	return &pbEvent.UpdateCategoryReq{
		Id:   id,
		Name: req.Name,
	}
}

// ProtoCategoryResToHTTPCategory конвертирует pbEvent.CategoryRes (gRPC)
// в Category (шлюз).
func ProtoCategoryResToHTTPCategory(protoCategory *pbEvent.CategoryRes) *Category {
	if protoCategory == nil {
		return nil
	}

	var createdAt time.Time
	if protoCategory.GetCreatedAt() != nil && protoCategory.GetCreatedAt().IsValid() {
		createdAt = protoCategory.GetCreatedAt().AsTime()
	} else {
		// Если createdAt не установлен, используем текущее время
		createdAt = time.Now()
	}

	var updatedAt time.Time
	if protoCategory.GetUpdatedAt() != nil && protoCategory.GetUpdatedAt().IsValid() {
		updatedAt = protoCategory.GetUpdatedAt().AsTime()
	} else {
		// Если updatedAt не установлен, используем то же время что и createdAt
		updatedAt = createdAt
	}

	return &Category{
		Id:        int(protoCategory.GetId()),
		Name:      protoCategory.GetName(),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

// ProtoCategoriesListToHTTPCategoriesList конвертирует []*pbEvent.CategoryRes (gRPC)
// в []*Category (шлюз).
func ProtoCategoriesListToHTTPCategoriesList(protoCategories []*pbEvent.CategoryRes) []*Category {
	if protoCategories == nil {
		return []*Category{} // Возвращаем пустой слайс, а не nil, для консистентности JSON
	}
	httpCategories := make([]*Category, 0, len(protoCategories))
	for _, protoCategory := range protoCategories {
		if httpCategory := ProtoCategoryResToHTTPCategory(protoCategory); httpCategory != nil {
			httpCategories = append(httpCategories, httpCategory)
		}
	}
	return httpCategories
}

// IDToProtoGetCategoryByIDReq создает pbEvent.GetCategoryReq.
func IDToProtoGetCategoryByIDReq(id int32) *pbEvent.GetCategoryReq {
	return &pbEvent.GetCategoryReq{Id: id}
}

// IDToProtoDeleteCategoryReq создает pbEvent.DeleteCategoryReq.
func IDToProtoDeleteCategoryReq(id int32) *pbEvent.DeleteCategoryReq {
	return &pbEvent.DeleteCategoryReq{Id: id}
}
