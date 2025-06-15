package eventHandler

import "time"

// Event представляет событие для HTTP ответа
type Event struct {
	Id          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	CategoryID  int64      `json:"category_id"`
	Date        string     `json:"date"`
	Time        string     `json:"time"`
	Location    string     `json:"location"`
	Price       float32    `json:"price"`
	Image       string     `json:"image"`
	Source      string     `json:"source"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

// CreateEventReq представляет запрос на создание события через HTTP
type CreateEventReq struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	CategoryID  int64   `json:"category_id"`
	Date        string  `json:"date"`
	Time        string  `json:"time"`
	Location    string  `json:"location"`
	Price       float32 `json:"price"`
	Image       string  `json:"image"`
	Source      string  `json:"source"`
}

type UpdateEventReq struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	CategoryID  int64   `json:"category_id"`
	Date        string  `json:"date"`
	Time        string  `json:"time"`
	Location    string  `json:"location"`
	Price       float32 `json:"price"`
	Image       string  `json:"image"`
	Source      string  `json:"source"`
}

// ListEventsReq представляет запрос на получение списка событий с фильтрами
type ListEventsReq struct {
	// Фильтры
	CategoryIDs []int64  `json:"category_ids,omitempty"`
	MinPrice    *float32 `json:"min_price,omitempty"`
	MaxPrice    *float32 `json:"max_price,omitempty"`
	DateFrom    *string  `json:"date_from,omitempty"`
	DateTo      *string  `json:"date_to,omitempty"`
	Location    *string  `json:"location,omitempty"`
	Source      *string  `json:"source,omitempty"`
	SearchText  *string  `json:"search_text,omitempty"`

	// Пагинация
	Limit  *int32 `json:"limit,omitempty"`
	Offset *int32 `json:"offset,omitempty"`

	// Дополнительные опции
	IncludeCount *bool `json:"include_count,omitempty"`
}

// ListEventsRes представляет ответ со списком событий
type ListEventsRes struct {
	Events     []*Event        `json:"events"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

// PaginationMeta содержит мета-информацию для пагинации
type PaginationMeta struct {
	TotalCount int64 `json:"total_count"`
	Limit      int32 `json:"limit"`
	Offset     int32 `json:"offset"`
	HasMore    bool  `json:"has_more"`
}

// Category представляет категорию событий
type Category struct {
	Id        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateCategoryReq представляет запрос на создание новой категории
type CreateCategoryReq struct {
	Name string `json:"name"`
}

// UpdateCategoryReq представляет запрос на обновление категории
type UpdateCategoryReq struct {
	Name string `json:"name"`
}

// NewCategory создает новую категорию из запроса
func NewCategory(req *CreateCategoryReq) *Category {
	return &Category{
		Name:      req.Name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// SuggestionRequest представляет запрос на автокомплит
type SuggestionRequest struct {
	Query      string   `json:"query"`       // Что пользователь начал печатать
	MaxResults int      `json:"max_results"` // Максимум предложений (по умолчанию 10)
	Fields     []string `json:"fields"`      // В каких полях искать (name, location, etc.)
}

// SuggestionResponse представляет ответ с предложениями
type SuggestionResponse struct {
	Suggestions []Suggestion `json:"suggestions"`
	Query       string       `json:"query"`
	Total       int          `json:"total"`
}

// Suggestion одно предложение
type Suggestion struct {
	Text     string  `json:"text"`               // Предлагаемый текст
	Score    float64 `json:"score"`              // Релевантность (0-1)
	Type     string  `json:"type"`               // Тип: "event", "location", "category"
	Category string  `json:"category,omitempty"` // Категория если type="event"
	EventID  *int64  `json:"event_id,omitempty"` // ID события если type="event"
}
