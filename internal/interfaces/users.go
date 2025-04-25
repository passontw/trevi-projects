package interfaces

type User struct {
	ID       int    `gorm:"primaryKey;column:id" json:"id" example:"1"`
	Name     string `gorm:"uniqueIndex;not null" json:"name" example:"testdemo001"`
	Password string `gorm:"not null" json:"-"`
	Phone    string `gorm:"uniqueIndex;not null" json:"phone" example:"0987654321"`
}

type CreateUserRequest struct {
	Name           string  `json:"name" binding:"required,min=1,max=20" example:"testdemo001"`
	Phone          string  `json:"phone" binding:"required,min=1,max=20" example:"0987654321"`
	Password       string  `json:"password" binding:"required,min=6,max=50" example:"a12345678"`
	InitialBalance float64 `json:"initial_balance,omitempty" example:"1000.00"`
}

type CreateUserResponse struct {
	ID               int     `json:"id" example:"1"`
	Name             string  `json:"name" example:"testdemo001"`
	Phone            string  `json:"phone" example:"0987654321"`
	AvailableBalance float64 `json:"available_balance" example:"1000.00"`
	FrozenBalance    float64 `json:"frozen_balance" example:"0.00"`
}

type UsersResponse struct {
	Users      []User     `json:"users"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	Total       int64 `json:"total"`
	Page        int   `json:"page"`
	PageSize    int   `json:"page_size"`
	TotalPages  int   `json:"total_pages"`
	HasNext     bool  `json:"has_next"`
	HasPrevious bool  `json:"has_previous"`
}

func NewPagination(total int64, page, pageSize int) Pagination {
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return Pagination{
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}
}

func NewUsersResponse(users []User, total int64, page, pageSize int) *UsersResponse {
	return &UsersResponse{
		Users:      users,
		Pagination: NewPagination(total, page, pageSize),
	}
}
