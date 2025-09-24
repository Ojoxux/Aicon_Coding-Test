package usecase

import (
	"context"

	"aicon-coding-test/internal/domain/entity"
)

// ItemRepository defines the interface for item data access
type ItemRepository interface {
	// FindAll retrieves all items
	FindAll(ctx context.Context) ([]*entity.Item, error)

	// FindByID retrieves an item by ID
	FindByID(ctx context.Context, id int64) (*entity.Item, error)

	// Create creates a new item and returns it with the generated ID
	Create(ctx context.Context, item *entity.Item) (*entity.Item, error)

	// Update はアイテムの特定フィールドをIDで部分更新する
	// *string, *int はポインタ型で、nilの場合は更新対象外を意味する
	// 更新後のアイテムを返す
	Update(ctx context.Context, id int64, name, brand *string, purchasePrice *int) (*entity.Item, error)

	// Delete deletes an item by ID
	Delete(ctx context.Context, id int64) error

	// GetSummaryByCategory returns item counts grouped by category (bonus feature)
	GetSummaryByCategory(ctx context.Context) (map[string]int, error)
}
