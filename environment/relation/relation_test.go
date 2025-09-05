package relation_test

import (
	"fmt"
	"log"
	"testing"

	"obsessiontech/environment/relation"
)

func TestUnsafeString(t *testing.T) {

	joinTemplateSQL, joinTemplateTable, _, _, err := relation.JoinSQL[string]("12patisserie", "template", "template", "", relation.RELATION_A)
	if err != nil {
		log.Println(err)
	}

	joinGoodsSQL, joinGoodsTable, _, _, err := relation.JoinSQL[int]("12patisserie", "template", "item", "", fmt.Sprintf("%s.b alias template", joinTemplateTable))
	if err != nil {
		log.Println(err)
	}

	SQL := fmt.Sprintf(`
		SELECT
			template.id, template.name, template.type, %s.b, %s.type
		FROM
			%s as template
		%s
		%s
		WHERE
			template.type = ?
	`, joinGoodsTable, joinGoodsTable, "12patisserie_template", joinTemplateSQL, joinGoodsSQL)

	log.Println(SQL)
}
