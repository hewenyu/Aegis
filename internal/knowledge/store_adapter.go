package knowledge

import (
	"github.com/hewenyu/Aegis/internal/types"
)

// StoreAdapter implements the text.VectorStore interface
type StoreAdapter struct {
	store      types.VectorStore
	collection string
}
