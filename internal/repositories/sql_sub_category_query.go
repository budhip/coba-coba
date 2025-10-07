package repositories

// query to sub_category database
var (
	querySubCategoryIsExistByCode = `SELECT "code" from "sub_category" WHERE "code" = $1 AND "categoryCode"= $2;`

	querySubCategoryGetByCode = `SELECT 
		"id", "categoryCode", "description", "code", "name", "createdAt", "updatedAt"
		FROM "sub_category"
		WHERE code = $1;`

	querySubCategoryCreate = `
		INSERT INTO "sub_category"(
			"categoryCode", "code", "name", "description", "createdAt", "updatedAt"
		)
		VALUES(
			$1, $2, $3, $4, now(), now()
		)
		RETURNING "id", "categoryCode", "code", "name", "description", "createdAt", "updatedAt";
	`

	queryGetAllSubCategory = `SELECT 
		"id", "categoryCode", "code", "name","description", "createdAt", "updatedAt"
		FROM "sub_category" ORDER BY "id" ASC`
)
