package repositories

// query to category database
var (
	queryCategoryIsExistByCode = `SELECT "code" from "category" WHERE "code" = $1;`

	queryCategoryGetSequence = `SELECT nextval($1);`

	queryCategoryCreate = `
		INSERT INTO category(
			"code", "name", "description", "createdAt", "updatedAt"
		)
		VALUES(
			$1, $2, $3, now(), now()
		)
		RETURNING "id", "code", "name", "description", "createdAt", "updatedAt";
	`

	queryCategoryGetByCode = `SELECT 
		"id", "description", "code", "name", "createdAt", "updatedAt"
	FROM "category"
	WHERE code = $1;`

	queryCategoryList = `SELECT "id", "code", "name", "description", "createdAt", "updatedAt" FROM category ORDER BY "id" ASC;`
)
