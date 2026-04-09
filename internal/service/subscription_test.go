package service

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"subscription-service/internal/model"
)

// --- mock repository ---

type mockRepo struct {
	subs   map[int]*model.Subscription
	nextID int
}

func newMockRepo() *mockRepo {
	return &mockRepo{subs: make(map[int]*model.Subscription), nextID: 1}
}

func (m *mockRepo) Create(ctx context.Context, sub *model.Subscription) (*model.Subscription, error) {
	sub.ID = m.nextID
	m.nextID++
	sub.CreatedAt = time.Now()
	sub.UpdatedAt = time.Now()
	m.subs[sub.ID] = sub
	return sub, nil
}

func (m *mockRepo) GetByID(ctx context.Context, id int) (*model.Subscription, error) {
	if sub, ok := m.subs[id]; ok {
		return sub, nil
	}
	return nil, fmt.Errorf("subscription not found")
}

func (m *mockRepo) List(ctx context.Context, userID, serviceName string) ([]model.Subscription, error) {
	var result []model.Subscription
	for _, sub := range m.subs {
		if userID != "" && sub.UserID.String() != userID {
			continue
		}
		if serviceName != "" && sub.ServiceName != serviceName {
			continue
		}
		result = append(result, *sub)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (m *mockRepo) Update(ctx context.Context, id int, req *model.UpdateSubscriptionRequest) (*model.Subscription, error) {
	sub, ok := m.subs[id]
	if !ok {
		return nil, fmt.Errorf("subscription not found")
	}
	if req.ServiceName != nil {
		sub.ServiceName = *req.ServiceName
	}
	if req.Price != nil {
		sub.Price = *req.Price
	}
	if req.StartDate != nil {
		sub.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		sub.EndDate = req.EndDate
	}
	sub.UpdatedAt = time.Now()
	return sub, nil
}

func (m *mockRepo) Delete(ctx context.Context, id int) error {
	if _, ok := m.subs[id]; !ok {
		return fmt.Errorf("subscription with id %d not found", id)
	}
	delete(m.subs, id)
	return nil
}

// --- helpers ---

func newTestService() (*SubscriptionService, *mockRepo) {
	repo := newMockRepo()
	logger := zap.NewNop()
	svc := NewSubscriptionService(repo, logger)
	return svc, repo
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

const testUUID = "60601fee-2bf1-4721-ae6f-7636e79a0cba"
const testUUID2 = "7a123456-1234-5678-90ab-cdef12345678"

// ===== CREATE TESTS =====

func TestCreate_ValidSubscription(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	sub, err := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Yandex Plus",
		Price:       400,
		UserID:      testUUID,
		StartDate:   "07-2025",
	})

	require.NoError(t, err)
	assert.Equal(t, "Yandex Plus", sub.ServiceName)
	assert.Equal(t, 400, sub.Price)
	assert.Equal(t, uuid.MustParse(testUUID), sub.UserID)
	assert.Equal(t, "07-2025", sub.StartDate)
	assert.Nil(t, sub.EndDate)
	assert.NotZero(t, sub.ID)
}

func TestCreate_WithOptionalEndDate(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	sub, err := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix",
		Price:       800,
		UserID:      testUUID,
		StartDate:   "01-2025",
		EndDate:     "12-2025",
	})

	require.NoError(t, err)
	require.NotNil(t, sub.EndDate)
	assert.Equal(t, "12-2025", *sub.EndDate)
}

func TestCreate_ZeroPrice(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	sub, err := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Free Tier",
		Price:       0,
		UserID:      testUUID,
		StartDate:   "01-2025",
	})

	require.NoError(t, err)
	assert.Equal(t, 0, sub.Price)
}

func TestCreate_EmptyServiceName(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "",
		Price:       400,
		UserID:      testUUID,
		StartDate:   "07-2025",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "service_name is required")
}

func TestCreate_NegativePrice(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Test",
		Price:       -100,
		UserID:      testUUID,
		StartDate:   "07-2025",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "price")
}

func TestCreate_InvalidUserID(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	tests := []string{"not-a-uuid", "", "12345", "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz"}
	for _, uid := range tests {
		_, err := svc.Create(ctx, &model.CreateSubscriptionRequest{
			ServiceName: "Test",
			Price:       100,
			UserID:      uid,
			StartDate:   "07-2025",
		})
		assert.Error(t, err, "expected error for user_id: %s", uid)
	}
}

func TestCreate_InvalidStartDate(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	invalidDates := []string{"13-2025", "00-2025", "2025-07", "abc", ""}
	for _, d := range invalidDates {
		_, err := svc.Create(ctx, &model.CreateSubscriptionRequest{
			ServiceName: "Test",
			Price:       100,
			UserID:      testUUID,
			StartDate:   d,
		})
		assert.Error(t, err, "expected error for start_date: %q", d)
	}
}

func TestCreate_InvalidEndDate(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Test",
		Price:       100,
		UserID:      testUUID,
		StartDate:   "07-2025",
		EndDate:     "invalid",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "end_date")
}

func TestCreate_EndDateBeforeStartDate(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Test",
		Price:       100,
		UserID:      testUUID,
		StartDate:   "07-2025",
		EndDate:     "01-2025",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "end_date must not be before start_date")
}

func TestCreate_BoundaryMonths(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	// January
	sub, err := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Test", Price: 100, UserID: testUUID, StartDate: "01-2025",
	})
	require.NoError(t, err)
	assert.Equal(t, "01-2025", sub.StartDate)

	// December
	sub, err = svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Test", Price: 100, UserID: testUUID, StartDate: "12-2025",
	})
	require.NoError(t, err)
	assert.Equal(t, "12-2025", sub.StartDate)
}

func TestCreate_SpecialCharsInServiceName(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	names := []string{
		"Сервис с кириллицей",
		"Service with spaces & symbols!",
		"Test'quote\"double",
	}
	for _, name := range names {
		sub, err := svc.Create(ctx, &model.CreateSubscriptionRequest{
			ServiceName: name, Price: 100, UserID: testUUID, StartDate: "07-2025",
		})
		require.NoError(t, err)
		assert.Equal(t, name, sub.ServiceName)
	}
}

// ===== GET BY ID TESTS =====

func TestGetByID_Existing(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})

	sub, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, sub.ID)
	assert.Equal(t, "Netflix", sub.ServiceName)
}

func TestGetByID_NonExistent(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 999)
	require.Error(t, err)
}

func TestGetByID_InvalidID(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 0)
	require.Error(t, err)

	_, err = svc.GetByID(ctx, -1)
	require.Error(t, err)
}

// ===== LIST TESTS =====

func TestList_AllSubscriptions(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})
	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Spotify", Price: 300, UserID: testUUID2, StartDate: "03-2025",
	})

	subs, err := svc.List(ctx, "", "")
	require.NoError(t, err)
	assert.Len(t, subs, 2)
}

func TestList_EmptyDatabase(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	subs, err := svc.List(ctx, "", "")
	require.NoError(t, err)
	assert.Empty(t, subs)
}

func TestList_FilterByUserID(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})
	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Spotify", Price: 300, UserID: testUUID, StartDate: "03-2025",
	})
	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "YouTube", Price: 200, UserID: testUUID2, StartDate: "02-2025",
	})

	subs, err := svc.List(ctx, testUUID, "")
	require.NoError(t, err)
	assert.Len(t, subs, 2)
	for _, s := range subs {
		assert.Equal(t, uuid.MustParse(testUUID), s.UserID)
	}
}

func TestList_FilterByServiceName(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})
	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Spotify", Price: 300, UserID: testUUID2, StartDate: "03-2025",
	})

	subs, err := svc.List(ctx, "", "Netflix")
	require.NoError(t, err)
	assert.Len(t, subs, 1)
	assert.Equal(t, "Netflix", subs[0].ServiceName)
}

func TestList_CombinedFilters(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})
	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID2, StartDate: "01-2025",
	})
	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Spotify", Price: 300, UserID: testUUID, StartDate: "03-2025",
	})

	subs, err := svc.List(ctx, testUUID, "Netflix")
	require.NoError(t, err)
	assert.Len(t, subs, 1)
	assert.Equal(t, "Netflix", subs[0].ServiceName)
	assert.Equal(t, uuid.MustParse(testUUID), subs[0].UserID)
}

func TestList_NoMatchingFilter(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})

	subs, err := svc.List(ctx, "", "NonExistentService")
	require.NoError(t, err)
	assert.Empty(t, subs)
}

// ===== UPDATE TESTS =====

func TestUpdate_AllFields(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})

	updated, err := svc.Update(ctx, created.ID, &model.UpdateSubscriptionRequest{
		ServiceName: strPtr("Spotify"),
		Price:       intPtr(300),
		StartDate:   strPtr("03-2025"),
		EndDate:     strPtr("09-2025"),
	})

	require.NoError(t, err)
	assert.Equal(t, "Spotify", updated.ServiceName)
	assert.Equal(t, 300, updated.Price)
	assert.Equal(t, "03-2025", updated.StartDate)
	require.NotNil(t, updated.EndDate)
	assert.Equal(t, "09-2025", *updated.EndDate)
}

func TestUpdate_PartialFields(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})

	updated, err := svc.Update(ctx, created.ID, &model.UpdateSubscriptionRequest{
		Price: intPtr(500),
	})

	require.NoError(t, err)
	assert.Equal(t, 500, updated.Price)
	assert.Equal(t, "Netflix", updated.ServiceName) // unchanged
}

func TestUpdate_NonExistent(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Update(ctx, 999, &model.UpdateSubscriptionRequest{
		Price: intPtr(500),
	})
	require.Error(t, err)
}

func TestUpdate_InvalidPrice(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})

	_, err := svc.Update(ctx, created.ID, &model.UpdateSubscriptionRequest{
		Price: intPtr(-100),
	})
	require.Error(t, err)
}

func TestUpdate_InvalidDateFormat(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})

	_, err := svc.Update(ctx, created.ID, &model.UpdateSubscriptionRequest{
		StartDate: strPtr("invalid"),
	})
	require.Error(t, err)
}

func TestUpdate_EndDateBeforeStartDate(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "06-2025",
	})

	_, err := svc.Update(ctx, created.ID, &model.UpdateSubscriptionRequest{
		StartDate: strPtr("06-2025"),
		EndDate:   strPtr("01-2025"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "end_date must not be before start_date")
}

func TestUpdate_AddEndDate(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})
	assert.Nil(t, created.EndDate)

	updated, err := svc.Update(ctx, created.ID, &model.UpdateSubscriptionRequest{
		EndDate: strPtr("12-2025"),
	})
	require.NoError(t, err)
	require.NotNil(t, updated.EndDate)
	assert.Equal(t, "12-2025", *updated.EndDate)
}

// ===== DELETE TESTS =====

func TestDelete_Existing(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})

	err := svc.Delete(ctx, created.ID)
	require.NoError(t, err)

	// verify deletion
	_, err = svc.GetByID(ctx, created.ID)
	assert.Error(t, err)
}

func TestDelete_NonExistent(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	err := svc.Delete(ctx, 999)
	require.Error(t, err)
}

func TestDelete_InvalidID(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	err := svc.Delete(ctx, 0)
	require.Error(t, err)

	err = svc.Delete(ctx, -5)
	require.Error(t, err)
}

func TestDelete_DoubleDelete(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})

	err := svc.Delete(ctx, created.ID)
	require.NoError(t, err)

	err = svc.Delete(ctx, created.ID)
	require.Error(t, err) // already deleted
}

// ===== TOTAL COST TESTS =====

func TestTotalCost_SingleSubscriptionFullPeriod(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Yandex Plus", Price: 400, UserID: testUUID,
		StartDate: "01-2025", EndDate: "12-2025",
	})

	total, err := svc.TotalCost(ctx, "01-2025", "12-2025", "", "")
	require.NoError(t, err)
	assert.Equal(t, 4800, total) // 400 × 12
}

func TestTotalCost_SubscriptionPartiallyInPeriod(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Yandex Plus", Price: 400, UserID: testUUID,
		StartDate: "03-2025", EndDate: "09-2025",
	})

	total, err := svc.TotalCost(ctx, "01-2025", "06-2025", "", "")
	require.NoError(t, err)
	assert.Equal(t, 1600, total) // 400 × 4 (Mar-Jun)
}

func TestTotalCost_OngoingSubscription(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Yandex Plus", Price: 400, UserID: testUUID,
		StartDate: "03-2025",
	})

	total, err := svc.TotalCost(ctx, "01-2025", "06-2025", "", "")
	require.NoError(t, err)
	assert.Equal(t, 1600, total) // 400 × 4 (Mar-Jun)
}

func TestTotalCost_SubscriptionBeforePeriod(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Old Sub", Price: 400, UserID: testUUID,
		StartDate: "01-2024", EndDate: "12-2024",
	})

	total, err := svc.TotalCost(ctx, "01-2025", "06-2025", "", "")
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestTotalCost_SubscriptionAfterPeriod(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Future Sub", Price: 400, UserID: testUUID,
		StartDate: "07-2025", EndDate: "12-2025",
	})

	total, err := svc.TotalCost(ctx, "01-2025", "06-2025", "", "")
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestTotalCost_MultipleSubscriptions(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID,
		StartDate: "01-2025", EndDate: "06-2025",
	})
	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Spotify", Price: 300, UserID: testUUID,
		StartDate: "01-2025", EndDate: "06-2025",
	})
	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "YouTube", Price: 200, UserID: testUUID,
		StartDate: "01-2025", EndDate: "06-2025",
	})

	total, err := svc.TotalCost(ctx, "01-2025", "06-2025", testUUID, "")
	require.NoError(t, err)
	// (800+300+200) × 6 = 7800
	assert.Equal(t, 7800, total)
}

func TestTotalCost_FilterByUserID(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID,
		StartDate: "01-2025", EndDate: "06-2025",
	})
	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID2,
		StartDate: "01-2025", EndDate: "06-2025",
	})

	total, err := svc.TotalCost(ctx, "01-2025", "06-2025", testUUID, "")
	require.NoError(t, err)
	assert.Equal(t, 4800, total) // 800 × 6 for user 1 only
}

func TestTotalCost_FilterByServiceName(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID,
		StartDate: "01-2025", EndDate: "06-2025",
	})
	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Spotify", Price: 300, UserID: testUUID,
		StartDate: "01-2025", EndDate: "06-2025",
	})

	total, err := svc.TotalCost(ctx, "01-2025", "06-2025", "", "Netflix")
	require.NoError(t, err)
	assert.Equal(t, 4800, total) // 800 × 6 Netflix only
}

func TestTotalCost_SameServiceDifferentUsers(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 400, UserID: testUUID,
		StartDate: "01-2025", EndDate: "12-2025",
	})
	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 400, UserID: testUUID2,
		StartDate: "01-2025", EndDate: "12-2025",
	})

	total, err := svc.TotalCost(ctx, "01-2025", "12-2025", "", "Netflix")
	require.NoError(t, err)
	assert.Equal(t, 9600, total) // (400+400) × 12
}

func TestTotalCost_ZeroResult_EmptyDB(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	total, err := svc.TotalCost(ctx, "01-2025", "12-2025", "", "")
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestTotalCost_ZeroResult_NoMatchingFilters(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID,
		StartDate: "01-2025", EndDate: "06-2025",
	})

	total, err := svc.TotalCost(ctx, "01-2025", "12-2025", "", "NonExistent")
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestTotalCost_InvalidStartDate(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.TotalCost(ctx, "invalid", "12-2025", "", "")
	require.Error(t, err)
}

func TestTotalCost_InvalidEndDate(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.TotalCost(ctx, "01-2025", "invalid", "", "")
	require.Error(t, err)
}

func TestTotalCost_EndDateBeforeStartDate(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.TotalCost(ctx, "06-2025", "01-2025", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "end_date must not be before start_date")
}

func TestTotalCost_InvalidUserID(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.TotalCost(ctx, "01-2025", "12-2025", "not-a-uuid", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user_id must be a valid UUID")
}

func TestTotalCost_SameMonthPeriod(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID,
		StartDate: "01-2025", EndDate: "12-2025",
	})

	total, err := svc.TotalCost(ctx, "06-2025", "06-2025", "", "")
	require.NoError(t, err)
	assert.Equal(t, 800, total) // single month
}

func TestTotalCost_MultiYearPeriod(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 400, UserID: testUUID,
		StartDate: "01-2024", EndDate: "12-2025",
	})

	total, err := svc.TotalCost(ctx, "01-2024", "12-2025", "", "")
	require.NoError(t, err)
	assert.Equal(t, 9600, total) // 400 × 24
}

// ===== E2E LIFECYCLE TESTS =====

func TestSubscriptionLifecycle(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	// 1. Create
	sub, err := svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})
	require.NoError(t, err)
	id := sub.ID

	// 2. List - verify present
	subs, err := svc.List(ctx, "", "")
	require.NoError(t, err)
	assert.Len(t, subs, 1)

	// 3. Update - change price
	updated, err := svc.Update(ctx, id, &model.UpdateSubscriptionRequest{
		Price: intPtr(500),
	})
	require.NoError(t, err)
	assert.Equal(t, 500, updated.Price)

	// 4. Calculate total cost
	total, err := svc.TotalCost(ctx, "01-2025", "06-2025", testUUID, "")
	require.NoError(t, err)
	assert.Equal(t, 3000, total) // 500 × 6

	// 5. Delete
	err = svc.Delete(ctx, id)
	require.NoError(t, err)

	// 6. Verify deletion
	_, err = svc.GetByID(ctx, id)
	assert.Error(t, err)
}

func TestMultiUserScenario(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	// User A: 3 subscriptions
	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID,
		StartDate: "01-2025", EndDate: "12-2025",
	})
	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Spotify", Price: 300, UserID: testUUID,
		StartDate: "01-2025", EndDate: "12-2025",
	})
	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "YouTube", Price: 200, UserID: testUUID,
		StartDate: "01-2025", EndDate: "12-2025",
	})

	// User B: 2 subscriptions
	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID2,
		StartDate: "01-2025", EndDate: "06-2025",
	})
	svc.Create(ctx, &model.CreateSubscriptionRequest{
		ServiceName: "Yandex Plus", Price: 400, UserID: testUUID2,
		StartDate: "03-2025",
	})

	// User A has 3 subs
	subsA, err := svc.List(ctx, testUUID, "")
	require.NoError(t, err)
	assert.Len(t, subsA, 3)

	// User B has 2 subs
	subsB, err := svc.List(ctx, testUUID2, "")
	require.NoError(t, err)
	assert.Len(t, subsB, 2)

	// Total cost for User A in Jan-Dec 2025: (800+300+200)×12 = 15600
	totalA, err := svc.TotalCost(ctx, "01-2025", "12-2025", testUUID, "")
	require.NoError(t, err)
	assert.Equal(t, 15600, totalA)

	// Total cost for User B in Jan-Dec 2025:
	// Netflix: 800×6 = 4800
	// Yandex Plus (ongoing): 400×10 = 4000 (Mar-Dec)
	totalB, err := svc.TotalCost(ctx, "01-2025", "12-2025", testUUID2, "")
	require.NoError(t, err)
	assert.Equal(t, 8800, totalB)

	// All Netflix subs: User A (800×12) + User B (800×6) = 14400
	totalNetflix, err := svc.TotalCost(ctx, "01-2025", "12-2025", "", "Netflix")
	require.NoError(t, err)
	assert.Equal(t, 14400, totalNetflix)
}
