package repositories

// query to entity database
var (
	queryEntityIsExistByCode = `SELECT "code" from "entity" WHERE "code" = $1;`
	queryEntityCreate        = `
			INSERT INTO entity(
				"code", "name", "description", "createdAt", "updatedAt"
			)
			VALUES(
				$1, $2, $3, now(), now()
			)
			RETURNING "id", "code", "name", "description", "createdAt", "updatedAt";
		`

	queryEntityGetByCode = `SELECT 
		"id", "description", "code", "name", "createdAt", "updatedAt"
		FROM "entity"
		WHERE code = $1;`

	queryEntityList = `SELECT "id", "description", "code", "name", "createdAt", "updatedAt" FROM entity ORDER BY "id" ASC;`
)
