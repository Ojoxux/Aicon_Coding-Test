package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"aicon-coding-test/internal/domain/entity"
	domainErrors "aicon-coding-test/internal/domain/errors"
)

// MockItemRepository はtestify/mockを使用したモックリポジトリ
// 実際のデータベースを使わずにテストを行うための偽のリポジトリ
// mock.Mockを埋め込むことで、モック機能を利用可能にする
type MockItemRepository struct {
	mock.Mock // testify/mockライブラリの基本構造体
}

// FindAll はモック版の全アイテム取得関数
// m.Called(ctx) でモックが呼ばれたことを記録し、事前に設定された戻り値を返す
func (m *MockItemRepository) FindAll(ctx context.Context) ([]*entity.Item, error) {
	args := m.Called(ctx) // モックの呼び出しを記録
	// args.Get(0) で最初の戻り値（アイテムスライス）を取得
	// args.Error(1) で2番目の戻り値（エラー）を取得
	return args.Get(0).([]*entity.Item), args.Error(1)
}

func (m *MockItemRepository) FindByID(ctx context.Context, id int64) (*entity.Item, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Item), args.Error(1)
}

func (m *MockItemRepository) Create(ctx context.Context, item *entity.Item) (*entity.Item, error) {
	args := m.Called(ctx, item)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Item), args.Error(1)
}

// Update はモック版のアイテム更新関数（今回追加した関数）
// 実際のデータベース更新は行わず、テスト用の動作をシミュレートする
func (m *MockItemRepository) Update(ctx context.Context, id int64, name, brand *string, purchasePrice *int) (*entity.Item, error) {
	// モックの呼び出しを記録（全ての引数を渡す）
	args := m.Called(ctx, id, name, brand, purchasePrice)
	// 戻り値がnilの場合（エラーケース）
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	// 正常ケースでは更新されたアイテムを返す
	return args.Get(0).(*entity.Item), args.Error(1)
}

func (m *MockItemRepository) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockItemRepository) GetSummaryByCategory(ctx context.Context) (map[string]int, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]int), args.Error(1)
}

func TestNewItemUsecase(t *testing.T) {
	mockRepo := new(MockItemRepository)
	usecase := NewItemUsecase(mockRepo)

	assert.NotNil(t, usecase)
}

func TestItemUsecase_GetAllItems(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockItemRepository)
		expectedCount int
		expectedErr   error
	}{
		{
			name: "正常系: 複数のアイテムを取得",
			setupMock: func(mockRepo *MockItemRepository) {
				item1, _ := entity.NewItem("時計1", "時計", "ROLEX", 1000000, "2023-01-01")
				item2, _ := entity.NewItem("バッグ1", "バッグ", "HERMÈS", 500000, "2023-01-02")
				items := []*entity.Item{item1, item2}
				mockRepo.On("FindAll", mock.Anything).Return(items, nil)
			},
			expectedCount: 2,
			expectedErr:   nil,
		},
		{
			name: "正常系: アイテムが0件",
			setupMock: func(mockRepo *MockItemRepository) {
				items := []*entity.Item{}
				mockRepo.On("FindAll", mock.Anything).Return(items, nil)
			},
			expectedCount: 0,
			expectedErr:   nil,
		},
		{
			name: "異常系: データベースエラー",
			setupMock: func(mockRepo *MockItemRepository) {
				mockRepo.On("FindAll", mock.Anything).Return(([]*entity.Item)(nil), domainErrors.ErrDatabaseError)
			},
			expectedCount: 0,
			expectedErr:   domainErrors.ErrDatabaseError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockItemRepository)
			tt.setupMock(mockRepo)
			usecase := NewItemUsecase(mockRepo)

			ctx := context.Background()
			items, err := usecase.GetAllItems(ctx)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
				mockRepo.AssertExpectations(t)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, items, tt.expectedCount)
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestItemUsecase_UpdateItem は新しく追加したUpdateItem関数のテスト
// テーブル駆動テスト（Table-Driven Test）を使用して複数のケースを一度にテスト
func TestItemUsecase_UpdateItem(t *testing.T) {
	// テストケースの構造体スライス
	tests := []struct {
		name      string
		id        int64
		input     UpdateItemInput
		setupMock func(*MockItemRepository)
		wantErr   bool
		wantItem  bool
	}{
		{
			// 正常系のテストケース: 名前フィールドのみを更新
			name: "正常系: 名前のみ更新",
			id:   1,
			input: UpdateItemInput{
				// stringPtr()でstring型のポインタを作成（部分更新のため）
				Name: stringPtr("更新された時計"),
				// BrandとPurchasePriceはnilのまま（更新対象外）
			},
			setupMock: func(mockRepo *MockItemRepository) {
				// 更新後のアイテムを作成
				updatedItem, _ := entity.NewItem("更新された時計", "時計", "ROLEX", 1000000, "2023-01-01")
				updatedItem.ID = 1
				// モックに期待する呼び出しを設定
				// Update(ctx, id=1, name="更新された時計", brand=nil, price=nil) が呼ばれることを期待
				mockRepo.On("Update", mock.Anything, int64(1), stringPtr("更新された時計"), (*string)(nil), (*int)(nil)).Return(updatedItem, nil)
			},
			wantErr:  false, // エラーは期待しない
			wantItem: true,  // アイテムが返されることを期待
		},
		{
			// 正常系のテストケース: 複数フィールドを同時に更新
			name: "正常系: 複数フィールド更新",
			id:   1,
			input: UpdateItemInput{
				// 3つのフィールドすべてを更新対象にする
				Name:          stringPtr("新しい時計"),
				Brand:         stringPtr("OMEGA"),
				PurchasePrice: intPtr(2000000),
			},
			setupMock: func(mockRepo *MockItemRepository) {
				// 複数フィールドが更新されたアイテムを作成
				updatedItem, _ := entity.NewItem("新しい時計", "時計", "OMEGA", 2000000, "2023-01-01")
				updatedItem.ID = 1
				// すべてのフィールドが渡されることを期待
				mockRepo.On("Update", mock.Anything, int64(1), stringPtr("新しい時計"), stringPtr("OMEGA"), intPtr(2000000)).Return(updatedItem, nil)
			},
			wantErr:  false,
			wantItem: true,
		},
		{
			// 異常系のテストケース: 無効なID（0以下）
			name: "異常系: 無効なID",
			id:   0, // 0は無効なID
			input: UpdateItemInput{
				Name: stringPtr("更新された時計"),
			},
			setupMock: func(mockRepo *MockItemRepository) {
				// ID検証でエラーになるため、リポジトリのメソッドは呼ばれない
				// モックの設定は不要
			},
			wantErr:  true,  // エラーが発生することを期待
			wantItem: false, // アイテムは返されない
		},
		{
			// 異常系のテストケース: 更新対象のフィールドが一つもない
			name:  "異常系: 更新フィールドなし",
			id:    1,
			input: UpdateItemInput{}, // 全フィールドがnil（更新対象なし）
			setupMock: func(mockRepo *MockItemRepository) {
				// 更新フィールドがないためリポジトリは呼ばれない
				// モックの設定は不要
			},
			wantErr:  true,  // "no fields to update" エラーが発生
			wantItem: false,
		},
		{
			// 異常系のテストケース: バリデーションエラー（空の名前）
			name: "異常系: 空の名前",
			id:   1,
			input: UpdateItemInput{
				Name: stringPtr(""), // 空文字列はバリデーションエラー
			},
			setupMock: func(mockRepo *MockItemRepository) {
				// バリデーションでエラーになるため、リポジトリは呼ばれない
			},
			wantErr:  true,  // "name cannot be empty" エラーが発生
			wantItem: false,
		},
		{
			// 異常系のテストケース: バリデーションエラー（負の価格）
			name: "異常系: 負の価格",
			id:   1,
			input: UpdateItemInput{
				PurchasePrice: intPtr(-1), // 負の値はバリデーションエラー
			},
			setupMock: func(mockRepo *MockItemRepository) {
				// バリデーションでエラーになるため、リポジトリは呼ばれない
			},
			wantErr:  true,  // "purchase_price must be 0 or greater" エラーが発生
			wantItem: false,
		},
		{
			// 異常系のテストケース: 存在しないアイテムの更新
			name: "異常系: アイテムが見つからない",
			id:   999, // 存在しないID
			input: UpdateItemInput{
				Name: stringPtr("更新された時計"),
			},
			setupMock: func(mockRepo *MockItemRepository) {
				// リポジトリのUpdateメソッドがErrItemNotFoundを返すように設定
				mockRepo.On("Update", mock.Anything, int64(999), stringPtr("更新された時計"), (*string)(nil), (*int)(nil)).Return((*entity.Item)(nil), domainErrors.ErrItemNotFound)
			},
			wantErr:  true,  // ErrItemNotFound エラーが発生
			wantItem: false,
		},
	}

	// 各テストケースを順番に実行するループ
	for _, tt := range tests {
		// t.Run で個別のサブテストとして実行（テスト名が表示される）
		t.Run(tt.name, func(t *testing.T) {
			// 新しいモックリポジトリのインスタンスを作成
			mockRepo := new(MockItemRepository)
			// テストケース固有のモック設定を実行
			tt.setupMock(mockRepo)
			// モックを使ってユースケースのインスタンスを作成
			usecase := NewItemUsecase(mockRepo)

			// テスト対象の関数を実行
			ctx := context.Background()
			item, err := usecase.UpdateItem(ctx, tt.id, tt.input)

			// 期待される結果と実際の結果を比較
			if tt.wantErr {
				// エラーが期待される場合
				assert.Error(t, err)     // エラーが発生していることを確認
				assert.Nil(t, item)      // アイテムはnilであることを確認
			} else {
				// 正常終了が期待される場合
				assert.NoError(t, err)   // エラーが発生していないことを確認
				if tt.wantItem {
					assert.NotNil(t, item) // アイテムが返されていることを確認
				}
			}

			// モックが期待通りに呼ばれたかを確認
			mockRepo.AssertExpectations(t)
		})
	}
}

// ヘルパー関数群
// Go言語では値からポインタを直接作ることができないため、これらの関数を使用

// stringPtr は文字列値からstring型のポインタを作成する
func stringPtr(s string) *string {
	return &s // &演算子でsのアドレス（ポインタ）を取得
}

// intPtr は整数値からint型のポインタを作成する
func intPtr(i int) *int {
	return &i // &演算子でiのアドレス（ポインタ）を取得
}

func TestItemUsecase_GetItemByID(t *testing.T) {
	tests := []struct {
		name        string
		id          int64
		setupMock   func(*MockItemRepository)
		expectError bool
		expectedErr error
	}{
		{
			name: "正常系: 存在するアイテムを取得",
			id:   1,
			setupMock: func(mockRepo *MockItemRepository) {
				item, _ := entity.NewItem("時計1", "時計", "ROLEX", 1000000, "2023-01-01")
				item.ID = 1
				mockRepo.On("FindByID", mock.Anything, int64(1)).Return(item, nil)
			},
			expectError: false,
		},
		{
			name: "異常系: 存在しないアイテム",
			id:   999,
			setupMock: func(mockRepo *MockItemRepository) {
				mockRepo.On("FindByID", mock.Anything, int64(999)).Return((*entity.Item)(nil), domainErrors.ErrItemNotFound)
			},
			expectError: true,
			expectedErr: domainErrors.ErrItemNotFound,
		},
		{
			name: "異常系: 無効なID（0以下）",
			id:   0,
			setupMock: func(mockRepo *MockItemRepository) {
				// FindByIDは呼ばれない
			},
			expectError: true,
			expectedErr: domainErrors.ErrInvalidInput,
		},
		{
			name: "異常系: データベースエラー",
			id:   1,
			setupMock: func(mockRepo *MockItemRepository) {
				mockRepo.On("FindByID", mock.Anything, int64(1)).Return((*entity.Item)(nil), domainErrors.ErrDatabaseError)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockItemRepository)
			tt.setupMock(mockRepo)
			usecase := NewItemUsecase(mockRepo)

			ctx := context.Background()
			item, err := usecase.GetItemByID(ctx, tt.id)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
				assert.Nil(t, item)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, item)
				assert.Equal(t, tt.id, item.ID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestItemUsecase_CreateItem(t *testing.T) {
	tests := []struct {
		name        string
		input       CreateItemInput
		setupMock   func(*MockItemRepository)
		expectError bool
		expectedErr error
	}{
		{
			name: "正常系: 有効なアイテムを作成",
			input: CreateItemInput{
				Name:          "ロレックス デイトナ",
				Category:      "時計",
				Brand:         "ROLEX",
				PurchasePrice: 1500000,
				PurchaseDate:  "2023-01-15",
			},
			setupMock: func(mockRepo *MockItemRepository) {
				createdItem, _ := entity.NewItem("ロレックス デイトナ", "時計", "ROLEX", 1500000, "2023-01-15")
				createdItem.ID = 1
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Item")).Return(createdItem, nil)
			},
			expectError: false,
		},
		{
			name: "異常系: 無効な入力（名前が空）",
			input: CreateItemInput{
				Name:          "",
				Category:      "時計",
				Brand:         "ROLEX",
				PurchasePrice: 1500000,
				PurchaseDate:  "2023-01-15",
			},
			setupMock: func(mockRepo *MockItemRepository) {
				// Createは呼ばれない
			},
			expectError: true,
			expectedErr: domainErrors.ErrInvalidInput,
		},
		{
			name: "異常系: 無効なカテゴリー",
			input: CreateItemInput{
				Name:          "アイテム",
				Category:      "無効なカテゴリー",
				Brand:         "ブランド",
				PurchasePrice: 100000,
				PurchaseDate:  "2023-01-15",
			},
			setupMock: func(mockRepo *MockItemRepository) {
				// Createは呼ばれない
			},
			expectError: true,
			expectedErr: domainErrors.ErrInvalidInput,
		},
		{
			name: "異常系: データベースエラー",
			input: CreateItemInput{
				Name:          "アイテム",
				Category:      "時計",
				Brand:         "ブランド",
				PurchasePrice: 100000,
				PurchaseDate:  "2023-01-15",
			},
			setupMock: func(mockRepo *MockItemRepository) {
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Item")).Return((*entity.Item)(nil), domainErrors.ErrDatabaseError)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockItemRepository)
			tt.setupMock(mockRepo)
			usecase := NewItemUsecase(mockRepo)

			ctx := context.Background()
			item, err := usecase.CreateItem(ctx, tt.input)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
				assert.Nil(t, item)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, item)
				assert.Equal(t, tt.input.Name, item.Name)
				assert.Equal(t, tt.input.Category, item.Category)
				assert.Equal(t, tt.input.Brand, item.Brand)
				assert.Equal(t, tt.input.PurchasePrice, item.PurchasePrice)
				assert.Equal(t, tt.input.PurchaseDate, item.PurchaseDate)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestItemUsecase_DeleteItem(t *testing.T) {
	tests := []struct {
		name        string
		id          int64
		setupMock   func(*MockItemRepository)
		expectError bool
		expectedErr error
	}{
		{
			name: "正常系: 存在するアイテムを削除",
			id:   1,
			setupMock: func(mockRepo *MockItemRepository) {
				item, _ := entity.NewItem("時計1", "時計", "ROLEX", 1000000, "2023-01-01")
				item.ID = 1
				mockRepo.On("FindByID", mock.Anything, int64(1)).Return(item, nil)
				mockRepo.On("Delete", mock.Anything, int64(1)).Return(nil)
			},
			expectError: false,
		},
		{
			name: "異常系: 存在しないアイテム",
			id:   999,
			setupMock: func(mockRepo *MockItemRepository) {
				mockRepo.On("FindByID", mock.Anything, int64(999)).Return((*entity.Item)(nil), domainErrors.ErrItemNotFound)
			},
			expectError: true,
			expectedErr: domainErrors.ErrItemNotFound,
		},
		{
			name: "異常系: 無効なID（0以下）",
			id:   0,
			setupMock: func(mockRepo *MockItemRepository) {
				// FindByIDは呼ばれない
			},
			expectError: true,
			expectedErr: domainErrors.ErrInvalidInput,
		},
		{
			name: "異常系: FindByIDでデータベースエラー",
			id:   1,
			setupMock: func(mockRepo *MockItemRepository) {
				mockRepo.On("FindByID", mock.Anything, int64(1)).Return((*entity.Item)(nil), domainErrors.ErrDatabaseError)
			},
			expectError: true,
		},
		{
			name: "異常系: Deleteでデータベースエラー",
			id:   1,
			setupMock: func(mockRepo *MockItemRepository) {
				item, _ := entity.NewItem("時計1", "時計", "ROLEX", 1000000, "2023-01-01")
				item.ID = 1
				mockRepo.On("FindByID", mock.Anything, int64(1)).Return(item, nil)
				mockRepo.On("Delete", mock.Anything, int64(1)).Return(domainErrors.ErrDatabaseError)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockItemRepository)
			tt.setupMock(mockRepo)
			usecase := NewItemUsecase(mockRepo)

			ctx := context.Background()
			err := usecase.DeleteItem(ctx, tt.id)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestItemUsecase_GetCategorySummary(t *testing.T) {
	tests := []struct {
		name               string
		setupMock          func(*MockItemRepository)
		expectedTotal      int
		expectedWatchCount int
		expectedBagCount   int
		expectError        bool
	}{
		{
			name: "正常系: 複数カテゴリーのアイテムがある場合",
			setupMock: func(mockRepo *MockItemRepository) {
				summary := map[string]int{
					"時計":  2,
					"バッグ": 1,
				}
				mockRepo.On("GetSummaryByCategory", mock.Anything).Return(summary, nil)
			},
			expectedTotal:      3,
			expectedWatchCount: 2,
			expectedBagCount:   1,
			expectError:        false,
		},
		{
			name: "正常系: アイテムが0件の場合",
			setupMock: func(mockRepo *MockItemRepository) {
				summary := map[string]int{}
				mockRepo.On("GetSummaryByCategory", mock.Anything).Return(summary, nil)
			},
			expectedTotal:      0,
			expectedWatchCount: 0,
			expectedBagCount:   0,
			expectError:        false,
		},
		{
			name: "異常系: データベースエラー",
			setupMock: func(mockRepo *MockItemRepository) {
				mockRepo.On("GetSummaryByCategory", mock.Anything).Return((map[string]int)(nil), domainErrors.ErrDatabaseError)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockItemRepository)
			tt.setupMock(mockRepo)
			usecase := NewItemUsecase(mockRepo)

			ctx := context.Background()
			summary, err := usecase.GetCategorySummary(ctx)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, summary)
				mockRepo.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, summary)

			assert.Equal(t, tt.expectedTotal, summary.Total)
			assert.Equal(t, tt.expectedWatchCount, summary.Categories["時計"])
			assert.Equal(t, tt.expectedBagCount, summary.Categories["バッグ"])

			// すべてのカテゴリーがレスポンスに含まれているかチェック
			expectedCategories := []string{"時計", "バッグ", "ジュエリー", "靴", "その他"}
			for _, category := range expectedCategories {
				assert.Contains(t, summary.Categories, category)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
