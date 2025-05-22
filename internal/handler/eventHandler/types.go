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
