package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryRepository_Exists(t *testing.T) {
	repo := NewMemoryRepository("/test")

	// Before init, should not exist
	exists, err := repo.Exists()
	assert.NoError(t, err)
	assert.False(t, exists)

	// After init, should exist
	err = repo.Init("main")
	require.NoError(t, err)

	exists, err = repo.Exists()
	assert.NoError(t, err)
	assert.True(t, exists)
}