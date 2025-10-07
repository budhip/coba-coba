package idgenerator_test

import (
	"regexp"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/idgenerator"
	"github.com/stretchr/testify/assert"
)

func TestGenerateID(t *testing.T) {
	t.Run("created new id with prefix", func(t *testing.T) {
		generator := idgenerator.New()
		id := generator.Generate("TRX")
		t.Log("id", id)
		assert.NotNil(t, id)
		assert.Regexp(t, regexp.MustCompile("TRX"), id)
	})

	t.Run("created new id without prefix", func(t *testing.T) {
		generator := idgenerator.New()
		id := generator.Generate()
		assert.NotNil(t, id)
	})
}
