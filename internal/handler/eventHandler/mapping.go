package eventHandler

import (
	"strconv"
	"strings"
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

// HTTPListReqToProtoListReq конвертирует ListEventsReq из URL параметров и JSON в pbEvent.ListEventsReq
func HTTPListReqToProtoListReq(req *ListEventsReq) *pbEvent.ListEventsReq {
	if req == nil {
		return &pbEvent.ListEventsReq{}
	}

	protoReq := &pbEvent.ListEventsReq{}

	// Фильтры
	if len(req.CategoryIDs) > 0 {
		protoReq.CategoryIDs = req.CategoryIDs
	}

	if req.MinPrice != nil {
		protoReq.MinPrice = req.MinPrice
	}

	if req.MaxPrice != nil {
		protoReq.MaxPrice = req.MaxPrice
	}

	if req.DateFrom != nil {
		protoReq.DateFrom = req.DateFrom
	}

	if req.DateTo != nil {
		protoReq.DateTo = req.DateTo
	}

	if req.Location != nil {
		protoReq.Location = req.Location
	}

	if req.Source != nil {
		protoReq.Source = req.Source
	}

	if req.SearchText != nil {
		protoReq.SearchText = req.SearchText
	}

	// Пагинация
	if req.Limit != nil {
		protoReq.Limit = req.Limit
	}

	if req.Offset != nil {
		protoReq.Offset = req.Offset
	}

	// Дополнительные опции
	if req.IncludeCount != nil {
		protoReq.IncludeCount = req.IncludeCount
	}

	return protoReq
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
	if protoEvent.GetCreatedAt() != nil && protoEvent.GetCreatedAt().IsValid() {
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

// ProtoListResToHTTPListRes конвертирует pbEvent.ListEventsRes в ListEventsRes
func ProtoListResToHTTPListRes(protoRes *pbEvent.ListEventsRes) *ListEventsRes {
	if protoRes == nil {
		return &ListEventsRes{
			Events: []*Event{},
		}
	}

	httpRes := &ListEventsRes{
		Events: ProtoEventsListToHTTPEventsList(protoRes.GetEvents()),
	}

	// Конвертируем пагинацию если она есть
	if protoRes.GetPagination() != nil {
		httpRes.Pagination = ProtoPaginationToHTTPPagination(protoRes.GetPagination())
	}

	return httpRes
}

// ProtoPaginationToHTTPPagination конвертирует pbEvent.PaginationMeta в PaginationMeta
func ProtoPaginationToHTTPPagination(protoPagination *pbEvent.PaginationMeta) *PaginationMeta {
	if protoPagination == nil {
		return nil
	}

	return &PaginationMeta{
		TotalCount: protoPagination.GetTotalCount(),
		Limit:      protoPagination.GetLimit(),
		Offset:     protoPagination.GetOffset(),
		HasMore:    protoPagination.GetHasMore(),
	}
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

// NoParamsToProtoEventsListReq создает pbEvent.ListEventsReq без параметров.
func NoParamsToProtoEventsListReq() *pbEvent.ListEventsReq {
	return &pbEvent.ListEventsReq{}
}

// CategoryIDToProtoEventsListReq создает pbEvent.ListEventsReq с фильтром по категории.
func CategoryIDToProtoEventsListReq(categoryID int64) *pbEvent.ListEventsReq {
	return &pbEvent.ListEventsReq{
		CategoryIDs: []int64{categoryID},
	}
}

// DateToProtoEventsListReq создает pbEvent.ListEventsReq с фильтром по дате.
func DateToProtoEventsListReq(date string) *pbEvent.ListEventsReq {
	return &pbEvent.ListEventsReq{
		DateFrom: &date,
		DateTo:   &date,
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

// ИСПРАВЛЕНО: ParseQueryParams парсит параметры запроса в ListEventsReq структуру
func ParseQueryParams(params map[string][]string) (*ListEventsReq, error) {
	req := &ListEventsReq{}

	// ИСПРАВЛЕНО: Парсим category_ids (может быть несколько)
	if categoryIDs, ok := params["category_ids"]; ok && len(categoryIDs) > 0 {
		var ids []int64
		for _, idStr := range categoryIDs {
			// Разделяем по запятой если несколько ID в одном параметре
			for _, part := range strings.Split(idStr, ",") {
				if part = strings.TrimSpace(part); part != "" {
					if id, err := strconv.ParseInt(part, 10, 64); err == nil {
						ids = append(ids, id)
					}
				}
			}
		}
		if len(ids) > 0 {
			req.CategoryIDs = ids
		}
	}

	// ИСПРАВЛЕНО: Также поддерживаем единичный category_id для обратной совместимости
	if categoryID, ok := params["category_id"]; ok && len(categoryID) > 0 && categoryID[0] != "" {
		if id, err := strconv.ParseInt(categoryID[0], 10, 64); err == nil {
			// Если уже есть category_ids, добавляем к ним
			if req.CategoryIDs == nil {
				req.CategoryIDs = []int64{id}
			} else {
				// Проверяем что такого ID еще нет
				found := false
				for _, existingID := range req.CategoryIDs {
					if existingID == id {
						found = true
						break
					}
				}
				if !found {
					req.CategoryIDs = append(req.CategoryIDs, id)
				}
			}
		}
	}

	// Парсим min_price
	if minPrices, ok := params["min_price"]; ok && len(minPrices) > 0 && minPrices[0] != "" {
		if price, err := strconv.ParseFloat(minPrices[0], 32); err == nil {
			priceFloat32 := float32(price)
			req.MinPrice = &priceFloat32
		}
	}

	// Парсим max_price
	if maxPrices, ok := params["max_price"]; ok && len(maxPrices) > 0 && maxPrices[0] != "" {
		if price, err := strconv.ParseFloat(maxPrices[0], 32); err == nil {
			priceFloat32 := float32(price)
			req.MaxPrice = &priceFloat32
		}
	}

	// Парсим date_from
	if dateFroms, ok := params["date_from"]; ok && len(dateFroms) > 0 && dateFroms[0] != "" {
		req.DateFrom = &dateFroms[0]
	}

	// Парсим date_to
	if dateTos, ok := params["date_to"]; ok && len(dateTos) > 0 && dateTos[0] != "" {
		req.DateTo = &dateTos[0]
	}

	// Парсим location
	if locations, ok := params["location"]; ok && len(locations) > 0 && locations[0] != "" {
		req.Location = &locations[0]
	}

	// Парсим source
	if sources, ok := params["source"]; ok && len(sources) > 0 && sources[0] != "" {
		req.Source = &sources[0]
	}

	// Парсим search_text
	if searchTexts, ok := params["search_text"]; ok && len(searchTexts) > 0 && searchTexts[0] != "" {
		req.SearchText = &searchTexts[0]
	}

	// Парсим limit
	if limits, ok := params["limit"]; ok && len(limits) > 0 && limits[0] != "" {
		if limit, err := strconv.ParseInt(limits[0], 10, 32); err == nil {
			limitInt32 := int32(limit)
			req.Limit = &limitInt32
		}
	}

	// Парсим offset
	if offsets, ok := params["offset"]; ok && len(offsets) > 0 && offsets[0] != "" {
		if offset, err := strconv.ParseInt(offsets[0], 10, 32); err == nil {
			offsetInt32 := int32(offset)
			req.Offset = &offsetInt32
		}
	}

	// Парсим include_count
	if includeCounts, ok := params["include_count"]; ok && len(includeCounts) > 0 {
		if includeCount, err := strconv.ParseBool(includeCounts[0]); err == nil {
			req.IncludeCount = &includeCount
		}
	}

	return req, nil
}
