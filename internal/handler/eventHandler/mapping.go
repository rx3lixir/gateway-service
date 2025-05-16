package eventHandler

import (
	"time"

	pbEvent "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/event"
)

// HTTPCreateReqToProtoCreateEventReq конвертирует models.CreateEventReq (шлюз) в pbEvent.CreateEventReq (gRPC).
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
		Price:       req.Price, // proto float (в .proto файле) это float32 в Go
		Image:       req.Image,
		Source:      req.Source,
	}
}

// HTTPUpdateReqToProtoUpdateEventReq конвертирует models.UpdateEventReq (шлюз) и ID в pbEvent.UpdateEventReq (gRPC).
func HTTPUpdateReqToProtoUpdateEventReq(id int64, req *UpdateEventReq) *pbEvent.UpdateEventReq {
	if req == nil {
		// Если тело запроса пустое, это может быть ошибкой, но для примера создадим запрос только с ID.
		// Лучше добавить валидацию на уровне HTTP хендлера.
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
		Price:       req.Price, // proto float это float32 в Go
		Image:       req.Image,
		Source:      req.Source,
	}
}

// ProtoEventResToHTTPEvent конвертирует pbEvent.EventRes (gRPC) в models.Event (шлюз).
func ProtoEventResToHTTPEvent(protoEvent *pbEvent.EventRes) *Event {
	if protoEvent == nil {
		return nil
	}

	var updatedAt *time.Time
	if protoEvent.GetUpdatedAt() != nil && protoEvent.GetUpdatedAt().IsValid() {
		t := protoEvent.GetUpdatedAt().AsTime()
		updatedAt = &t
	}

	var createdAt time.Time
	if protoEvent.GetUpdatedAt() != nil && protoEvent.GetCreatedAt().IsValid() {
		createdAt = protoEvent.GetCreatedAt().AsTime()
	} else {
		return nil
	}

	return &Event{
		Id:          protoEvent.GetId(),
		Name:        protoEvent.GetName(),
		Description: protoEvent.GetDescription(),
		CategoryID:  protoEvent.GetCategoryID(),
		Date:        protoEvent.GetDate(),
		Time:        protoEvent.GetTime(),
		Location:    protoEvent.GetLocation(),
		Price:       protoEvent.GetPrice(), // proto float это float32 в Go
		Image:       protoEvent.GetImage(),
		Source:      protoEvent.GetSource(),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

// ProtoEventsListToHTTPEventsList конвертирует []*pbEvent.EventRes (gRPC) в []*models.Event (шлюз).
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
