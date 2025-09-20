package utility

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestHash(t *testing.T) {
	t.Parallel()

	hash, err := Hash("mySecurePassword")
	assert.NilError(t, err)
	assert.Assert(t, hash != "")

	hash, err = Hash("")
	assert.NilError(t, err)
	assert.Assert(t, hash != "")
}

func TestCompareHashAndPlain(t *testing.T) {
	t.Parallel()

	hash, err := Hash("mySecurePassword")
	assert.NilError(t, err)

	match, err := CompareHashAndPlain(hash, "mySecurePassword")
	assert.NilError(t, err)
	assert.Assert(t, match)

	match, err = CompareHashAndPlain(hash, "wrongPassword")
	assert.NilError(t, err)
	assert.Assert(t, !match)

	match, err = CompareHashAndPlain("", "mySecurePassword")
	assert.ErrorContains(t, err, "hashedSecret too short")
	assert.Assert(t, !match)
}
