package common

type Pagination struct {
	CurrentPage int `json:"currentPage" example:"1"`
	LastPage    int `json:"lastPage" example:"1"`
	Total       int `json:"total" example:"1"`
	PerPage     int `json:"perPage" example:"10"`
}
