package usecase

import (
	"context"
	"fmt"
	"strings"

	"aicon-coding-test/internal/domain/entity"
	domainErrors "aicon-coding-test/internal/domain/errors"
)

type ItemUsecase interface {
	GetAllItems(ctx context.Context) ([]*entity.Item, error)
	GetItemByID(ctx context.Context, id int64) (*entity.Item, error)
	CreateItem(ctx context.Context, input CreateItemInput) (*entity.Item, error)
	UpdateItem(ctx context.Context, id int64, input UpdateItemInput) (*entity.Item, error)
	DeleteItem(ctx context.Context, id int64) error
	GetCategorySummary(ctx context.Context) (*CategorySummary, error)
}

type CreateItemInput struct {
	Name          string `json:"name"`
	Category      string `json:"category"`
	Brand         string `json:"brand"`
	PurchasePrice int    `json:"purchase_price"`
	PurchaseDate  string `json:"purchase_date"`
}

// UpdateItemInput はPATCHリクエストで使用する構造体
// *string, *int はポインタ型で、nilの場合は更新対象外を意味する
// omitemptyタグにより、JSONで空の場合はフィールドが省略される
type UpdateItemInput struct {
	Name          *string `json:"name,omitempty"`          // アイテム名（オプショナル）
	Brand         *string `json:"brand,omitempty"`         // ブランド名（オプショナル）
	PurchasePrice *int    `json:"purchase_price,omitempty"` // 購入価格（オプショナル）
}

type CategorySummary struct {
	Categories map[string]int `json:"categories"`
	Total      int            `json:"total"`
}

type itemUsecase struct {
	itemRepo ItemRepository
}

func NewItemUsecase(itemRepo ItemRepository) ItemUsecase {
	return &itemUsecase{
		itemRepo: itemRepo,
	}
}

func (u *itemUsecase) GetAllItems(ctx context.Context) ([]*entity.Item, error) {
	items, err := u.itemRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve items: %w", err)
	}

	return items, nil
}

func (u *itemUsecase) GetItemByID(ctx context.Context, id int64) (*entity.Item, error) {
	if id <= 0 {
		return nil, domainErrors.ErrInvalidInput
	}

	item, err := u.itemRepo.FindByID(ctx, id)
	if err != nil {
		if domainErrors.IsNotFoundError(err) {
			return nil, domainErrors.ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to retrieve item: %w", err)
	}

	return item, nil
}

func (u *itemUsecase) CreateItem(ctx context.Context, input CreateItemInput) (*entity.Item, error) {
	// バリデーションして、新しいエンティティを作成
	item, err := entity.NewItem(
		input.Name,
		input.Category,
		input.Brand,
		input.PurchasePrice,
		input.PurchaseDate,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", domainErrors.ErrInvalidInput, err.Error())
	}

	createdItem, err := u.itemRepo.Create(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}

	return createdItem, nil
}

// UpdateItem はアイテムの部分更新を行うユースケース関数
// 更新対象フィールド: name, brand, purchase_price
// 不変フィールド: id, category, purchase_date, created_at, updated_at
func (u *itemUsecase) UpdateItem(ctx context.Context, id int64, input UpdateItemInput) (*entity.Item, error) {
	// IDのバリデーション（0以下は無効）
	if id <= 0 {
		return nil, domainErrors.ErrInvalidInput
	}

	// 更新対象のフィールドが一つでもあるかチェック
	// 全てnilの場合は更新するものがないのでエラー
	if input.Name == nil && input.Brand == nil && input.PurchasePrice == nil {
		return nil, fmt.Errorf("%w: no fields to update", domainErrors.ErrInvalidInput)
	}

	// 入力値のバリデーション（空文字、長さ、負の値など）
	if err := validateUpdateItemInput(input); err != nil {
		return nil, fmt.Errorf("%w: %s", domainErrors.ErrInvalidInput, err.Error())
	}

	// リポジトリ層のUpdate関数を呼び出してデータベースを更新
	updatedItem, err := u.itemRepo.Update(ctx, id, input.Name, input.Brand, input.PurchasePrice)
	if err != nil {
		// アイテムが存在しない場合のエラーハンドリング
		if domainErrors.IsNotFoundError(err) {
			return nil, domainErrors.ErrItemNotFound
		}
		// その他のデータベースエラー
		return nil, fmt.Errorf("failed to update item: %w", err)
	}

	// 更新成功時は更新されたアイテムを返す
	return updatedItem, nil
}

func (u *itemUsecase) DeleteItem(ctx context.Context, id int64) error {
	if id <= 0 {
		return domainErrors.ErrInvalidInput
	}

	_, err := u.itemRepo.FindByID(ctx, id)
	if err != nil {
		if domainErrors.IsNotFoundError(err) {
			return domainErrors.ErrItemNotFound
		}
		return fmt.Errorf("failed to check item existence: %w", err)
	}

	err = u.itemRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	return nil
}

func (u *itemUsecase) GetCategorySummary(ctx context.Context) (*CategorySummary, error) {
	categoryCounts, err := u.itemRepo.GetSummaryByCategory(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get category summary: %w", err)
	}

	// 合計計算
	total := 0
	for _, count := range categoryCounts {
		total += count
	}

	summary := make(map[string]int)
	for _, category := range entity.GetValidCategories() {
		if count, exists := categoryCounts[category]; exists {
			summary[category] = count
		} else {
			summary[category] = 0
		}
	}

	return &CategorySummary{
		Categories: summary,
		Total:      total,
	}, nil
}

// validateUpdateItemInput はUpdateItemInputのバリデーションを行う関数
// nilでないフィールドのみをチェックする（部分更新対応）
func validateUpdateItemInput(input UpdateItemInput) error {
	// エラーメッセージを格納するスライス
	var errs []string

	// Nameがnilでない（更新対象）の場合のバリデーション
	if input.Name != nil {
		if *input.Name == "" {
			// 空文字は禁止
			errs = append(errs, "name cannot be empty")
		} else if len(*input.Name) > 100 {
			// 100文字を超えるのは禁止
			errs = append(errs, "name must be 100 characters or less")
		}
	}

	// Brandがnilでない（更新対象）の場合のバリデーション
	if input.Brand != nil {
		if *input.Brand == "" {
			// 空文字は禁止
			errs = append(errs, "brand cannot be empty")
		} else if len(*input.Brand) > 100 {
			// 100文字を超えるのは禁止
			errs = append(errs, "brand must be 100 characters or less")
		}
	}

	// PurchasePriceがnilでない（更新対象）かつ負の値の場合はエラー
	if input.PurchasePrice != nil && *input.PurchasePrice < 0 {
		errs = append(errs, "purchase_price must be 0 or greater")
	}

	// エラーがある場合はカンマ区切りで連結して返す
	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, ", "))
	}

	// エラーがない場合はnilを返す
	return nil
}
