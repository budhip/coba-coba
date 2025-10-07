package models

var (
	SortByASC  = "asc"
	SortByDESC = "desc"

	ReverseSortMap = map[string]string{
		SortByASC:  SortByDESC,
		SortByDESC: SortByASC,
	}
)
