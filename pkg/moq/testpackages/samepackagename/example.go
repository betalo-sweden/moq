package same

import (
	"context"

	"github.com/betalo-sweden/moq/pkg/moq/testpackages/samepackagename/same"
)

// PersonStore stores people.
type PersonStore interface {
	Get(ctx context.Context, id string) (*same.Person, error)
	Create(ctx context.Context, person *same.Person, confirm bool) error
	ClearCache(id string)
}
