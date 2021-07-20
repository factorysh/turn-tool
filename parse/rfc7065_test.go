package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	_, err := Parse("plop")
	assert.Error(t, err)
	_, err = Parse("z:plop")
	assert.Error(t, err)

	u, err := Parse("turn:pion.ly")
	assert.NoError(t, err)
	assert.Equal(t, "turn", u.Scheme)
	assert.Equal(t, "pion.ly", u.Host)

	u, err = Parse("turn:pion.ly:3478")
	assert.NoError(t, err)
	assert.Equal(t, "turn", u.Scheme)
	assert.Equal(t, "pion.ly", u.Host)
	assert.Equal(t, "3478", u.Port)

	u, err = Parse("turns:pion.ly?transport=tcp")
	assert.NoError(t, err)
	assert.Equal(t, "turns", u.Scheme)
	assert.Equal(t, "pion.ly", u.Host)
	assert.Equal(t, "tcp", u.Transport)

	u, err = Parse("turns:pion.ly:3478?transport=udp")
	assert.NoError(t, err)
	assert.Equal(t, "turns", u.Scheme)
	assert.Equal(t, "3478", u.Port)
	assert.Equal(t, "pion.ly", u.Host)
	assert.Equal(t, "udp", u.Transport)
}
