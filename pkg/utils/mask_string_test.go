package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskString_MasksMiddlePortion(t *testing.T) {
	got := maskString(3, 3, "alice@example.com")
	assert.Equal(t, "ali*****com", got)
	assert.True(t, strings.Contains(got, "*****"))
}

func TestMaskString_TooShortReturnsOriginal(t *testing.T) {
	// len("abcde") == 5 == visibleStart+visibleEnd, so nothing can be masked.
	got := maskString(3, 2, "abcde")
	assert.Equal(t, "abcde", got)
}

func TestMaskString_ShorterThanVisibleReturnsOriginal(t *testing.T) {
	got := maskString(5, 5, "abc")
	assert.Equal(t, "abc", got)
}

func TestMaskString_ZeroVisibleMasksEntireMiddle(t *testing.T) {
	got := maskString(0, 0, "0912345678901")
	assert.Equal(t, "*****", got)
}

func TestMaskString_EmptyString(t *testing.T) {
	got := maskString(1, 1, "")
	assert.Equal(t, "", got)
}

func TestMaskString_AsymmetricVisibleLengths(t *testing.T) {
	got := maskString(1, 4, "09123456789")
	assert.Equal(t, "0*****6789", got)
}
