package archeology

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeDigResult(t *testing.T) {
	rest := []string{"keep", "common_fossil", "150", "common", "100", "75", "1", "402563481200754699"}
	res := decodeDigResult(rest)
	assert.NotNil(t, res)
	assert.Equal(t, "common_fossil", res.ItemName)
	assert.Equal(t, 150, res.Value)
	assert.Equal(t, "common", res.Quality)
	assert.Equal(t, 100, res.Integrity)
	assert.Equal(t, 75, res.XP)
	assert.Equal(t, 1, res.Quantity)
}

func TestDecodeDigResultSellPayload(t *testing.T) {
	rest := []string{"sell", "coelacanth_egg", "2500", "living", "100", "200", "1", "123"}
	res := decodeDigResult(rest)
	assert.NotNil(t, res)
	assert.Equal(t, "coelacanth_egg", res.ItemName)
	assert.Equal(t, 2500, res.Value)
	assert.Equal(t, "living", res.Quality)
	assert.Equal(t, 200, res.XP)
}

func TestDecodeDigResultInvalid(t *testing.T) {
	assert.Nil(t, decodeDigResult(nil))
	assert.Nil(t, decodeDigResult([]string{"keep"}))
	assert.Nil(t, decodeDigResult([]string{"keep", "x", "abc", "q", "1", "2", "3"}))
	assert.Nil(t, decodeDigResult([]string{"keep", "", "150", "", "100", "75", "1"}))
	assert.Nil(t, decodeDigResult([]string{"keep", "x", "150", "q", "bad", "75", "1"}))
}
