package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"subscription-service/internal/handler"
	"subscription-service/internal/model"
	"subscription-service/internal/service"
)

// --- mock repository implementing service.Repository ---

type mockRepo struct {
	subs   map[int]*model.Subscription
	nextID int
}

func newMockRepo() *mockRepo {
	return &mockRepo{subs: make(map[int]*model.Subscription), nextID: 1}
}

func (m *mockRepo) Create(_ context.Context, sub *model.Subscription) (*model.Subscription, error) {
	sub.ID = m.nextID
	m.nextID++
	sub.CreatedAt = time.Now()
	sub.UpdatedAt = time.Now()
	m.subs[sub.ID] = sub
	return sub, nil
}

func (m *mockRepo) GetByID(_ context.Context, id int) (*model.Subscription, error) {
	if sub, ok := m.subs[id]; ok {
		return sub, nil
	}
	return nil, fmt.Errorf("subscription not found")
}

func (m *mockRepo) List(_ context.Context, userID, serviceName string) ([]model.Subscription, error) {
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

func (m *mockRepo) Update(_ context.Context, id int, req *model.UpdateSubscriptionRequest) (*model.Subscription, error) {
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

func (m *mockRepo) Delete(_ context.Context, id int) error {
	if _, ok := m.subs[id]; !ok {
		return fmt.Errorf("subscription with id %d not found", id)
	}
	delete(m.subs, id)
	return nil
}

// --- test setup ---

func setupRouter() (*gin.Engine, *mockRepo) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	repo := newMockRepo()
	svc := service.NewSubscriptionService(repo, logger)
	h := handler.NewSubscriptionHandler(svc, logger)

	r := gin.New()
	h.RegisterRoutes(r)
	return r, repo
}

func doJSON(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func doGet(r *gin.Engine, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

const testUUID = "60601fee-2bf1-4721-ae6f-7636e79a0cba"
const testUUID2 = "7a123456-1234-5678-90ab-cdef12345678"

// ===== POST /api/v1/subscriptions =====

func TestCreateSubscription_Success(t *testing.T) {
	r, _ := setupRouter()

	body := model.CreateSubscriptionRequest{
		ServiceName: "Yandex Plus",
		Price:       400,
		UserID:      testUUID,
		StartDate:   "07-2025",
	}

	w := doJSON(r, http.MethodPost, "/api/v1/subscriptions", body)

	assert.Equal(t, http.StatusCreated, w.Code)

	var sub model.Subscription
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sub))
	assert.Equal(t, "Yandex Plus", sub.ServiceName)
	assert.Equal(t, 400, sub.Price)
	assert.Equal(t, uuid.MustParse(testUUID), sub.UserID)
	assert.Equal(t, "07-2025", sub.StartDate)
	assert.Nil(t, sub.EndDate)
	assert.NotZero(t, sub.ID)
}

func TestCreateSubscription_WithEndDate(t *testing.T) {
	r, _ := setupRouter()

	body := model.CreateSubscriptionRequest{
		ServiceName: "Netflix",
		Price:       800,
		UserID:      testUUID,
		StartDate:   "01-2025",
		EndDate:     "12-2025",
	}

	w := doJSON(r, http.MethodPost, "/api/v1/subscriptions", body)
	assert.Equal(t, http.StatusCreated, w.Code)

	var sub model.Subscription
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sub))
	require.NotNil(t, sub.EndDate)
	assert.Equal(t, "12-2025", *sub.EndDate)
}

func TestCreateSubscription_MissingServiceName(t *testing.T) {
	r, _ := setupRouter()
	body := map[string]interface{}{
		"price":      400,
		"user_id":    testUUID,
		"start_date": "07-2025",
	}
	w := doJSON(r, http.MethodPost, "/api/v1/subscriptions", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSubscription_MissingPrice(t *testing.T) {
	r, _ := setupRouter()
	body := map[string]interface{}{
		"service_name": "Netflix",
		"user_id":      testUUID,
		"start_date":   "07-2025",
	}
	// price defaults to 0 which is valid; let's test negative
	w := doJSON(r, http.MethodPost, "/api/v1/subscriptions", body)
	assert.Equal(t, http.StatusCreated, w.Code) // 0 is valid
}

func TestCreateSubscription_MissingUserID(t *testing.T) {
	r, _ := setupRouter()
	body := map[string]interface{}{
		"service_name": "Netflix",
		"price":        400,
		"start_date":   "07-2025",
	}
	w := doJSON(r, http.MethodPost, "/api/v1/subscriptions", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSubscription_MissingStartDate(t *testing.T) {
	r, _ := setupRouter()
	body := map[string]interface{}{
		"service_name": "Netflix",
		"price":        400,
		"user_id":      testUUID,
	}
	w := doJSON(r, http.MethodPost, "/api/v1/subscriptions", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSubscription_NegativePrice(t *testing.T) {
	r, _ := setupRouter()
	body := model.CreateSubscriptionRequest{
		ServiceName: "Netflix",
		Price:       -100,
		UserID:      testUUID,
		StartDate:   "07-2025",
	}
	w := doJSON(r, http.MethodPost, "/api/v1/subscriptions", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSubscription_InvalidUUID(t *testing.T) {
	r, _ := setupRouter()
	body := map[string]interface{}{
		"service_name": "Netflix",
		"price":        400,
		"user_id":      "not-a-uuid",
		"start_date":   "07-2025",
	}
	w := doJSON(r, http.MethodPost, "/api/v1/subscriptions", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSubscription_InvalidStartDate(t *testing.T) {
	r, _ := setupRouter()

	invalidDates := []string{"13-2025", "00-2025", "2025-07", "abc"}
	for _, d := range invalidDates {
		body := map[string]interface{}{
			"service_name": "Netflix",
			"price":        400,
			"user_id":      testUUID,
			"start_date":   d,
		}
		w := doJSON(r, http.MethodPost, "/api/v1/subscriptions", body)
		assert.Equal(t, http.StatusBadRequest, w.Code, "expected 400 for date: %s", d)
	}
}

func TestCreateSubscription_EndDateBeforeStart(t *testing.T) {
	r, _ := setupRouter()
	body := model.CreateSubscriptionRequest{
		ServiceName: "Netflix",
		Price:       400,
		UserID:      testUUID,
		StartDate:   "07-2025",
		EndDate:     "01-2025",
	}
	w := doJSON(r, http.MethodPost, "/api/v1/subscriptions", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSubscription_MalformedJSON(t *testing.T) {
	r, _ := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions",
		bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSubscription_EmptyBody(t *testing.T) {
	r, _ := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions",
		bytes.NewBufferString(""))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSubscription_CyrillicServiceName(t *testing.T) {
	r, _ := setupRouter()
	body := model.CreateSubscriptionRequest{
		ServiceName: "Яндекс Плюс",
		Price:       400,
		UserID:      testUUID,
		StartDate:   "07-2025",
	}
	w := doJSON(r, http.MethodPost, "/api/v1/subscriptions", body)
	assert.Equal(t, http.StatusCreated, w.Code)

	var sub model.Subscription
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sub))
	assert.Equal(t, "Яндекс Плюс", sub.ServiceName)
}

// ===== GET /api/v1/subscriptions/:id =====

func TestGetSubscription_Success(t *testing.T) {
	r, _ := setupRouter()

	// create first
	body := model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	}
	wCreate := doJSON(r, http.MethodPost, "/api/v1/subscriptions", body)
	var created model.Subscription
	json.Unmarshal(wCreate.Body.Bytes(), &created)

	// get
	w := doGet(r, fmt.Sprintf("/api/v1/subscriptions/%d", created.ID))
	assert.Equal(t, http.StatusOK, w.Code)

	var sub model.Subscription
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sub))
	assert.Equal(t, created.ID, sub.ID)
	assert.Equal(t, "Netflix", sub.ServiceName)
}

func TestGetSubscription_NotFound(t *testing.T) {
	r, _ := setupRouter()
	w := doGet(r, "/api/v1/subscriptions/999")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetSubscription_InvalidID(t *testing.T) {
	r, _ := setupRouter()
	w := doGet(r, "/api/v1/subscriptions/abc")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ===== GET /api/v1/subscriptions =====

func TestListSubscriptions_All(t *testing.T) {
	r, _ := setupRouter()

	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})
	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Spotify", Price: 300, UserID: testUUID2, StartDate: "03-2025",
	})

	w := doGet(r, "/api/v1/subscriptions")
	assert.Equal(t, http.StatusOK, w.Code)

	var subs []model.Subscription
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &subs))
	assert.Len(t, subs, 2)
}

func TestListSubscriptions_EmptyDB(t *testing.T) {
	r, _ := setupRouter()
	w := doGet(r, "/api/v1/subscriptions")
	assert.Equal(t, http.StatusOK, w.Code)

	var subs []model.Subscription
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &subs))
	assert.Empty(t, subs)
}

func TestListSubscriptions_FilterByUserID(t *testing.T) {
	r, _ := setupRouter()

	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})
	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Spotify", Price: 300, UserID: testUUID2, StartDate: "01-2025",
	})

	w := doGet(r, "/api/v1/subscriptions?user_id="+testUUID)
	assert.Equal(t, http.StatusOK, w.Code)

	var subs []model.Subscription
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &subs))
	assert.Len(t, subs, 1)
	assert.Equal(t, "Netflix", subs[0].ServiceName)
}

func TestListSubscriptions_FilterByServiceName(t *testing.T) {
	r, _ := setupRouter()

	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})
	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Spotify", Price: 300, UserID: testUUID, StartDate: "01-2025",
	})

	w := doGet(r, "/api/v1/subscriptions?service_name=Spotify")
	assert.Equal(t, http.StatusOK, w.Code)

	var subs []model.Subscription
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &subs))
	assert.Len(t, subs, 1)
	assert.Equal(t, "Spotify", subs[0].ServiceName)
}

// ===== PUT /api/v1/subscriptions/:id =====

func TestUpdateSubscription_Success(t *testing.T) {
	r, _ := setupRouter()

	wCreate := doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})
	var created model.Subscription
	json.Unmarshal(wCreate.Body.Bytes(), &created)

	w := doJSON(r, http.MethodPut, fmt.Sprintf("/api/v1/subscriptions/%d", created.ID),
		model.UpdateSubscriptionRequest{Price: intPtr(500)})

	assert.Equal(t, http.StatusOK, w.Code)

	var updated model.Subscription
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	assert.Equal(t, 500, updated.Price)
	assert.Equal(t, "Netflix", updated.ServiceName)
}

func TestUpdateSubscription_NotFound(t *testing.T) {
	r, _ := setupRouter()
	w := doJSON(r, http.MethodPut, "/api/v1/subscriptions/999",
		model.UpdateSubscriptionRequest{Price: intPtr(500)})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSubscription_InvalidID(t *testing.T) {
	r, _ := setupRouter()
	w := doJSON(r, http.MethodPut, "/api/v1/subscriptions/abc",
		model.UpdateSubscriptionRequest{Price: intPtr(500)})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSubscription_InvalidPrice(t *testing.T) {
	r, _ := setupRouter()

	wCreate := doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})
	var created model.Subscription
	json.Unmarshal(wCreate.Body.Bytes(), &created)

	w := doJSON(r, http.MethodPut, fmt.Sprintf("/api/v1/subscriptions/%d", created.ID),
		model.UpdateSubscriptionRequest{Price: intPtr(-100)})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSubscription_EndDateBeforeStart(t *testing.T) {
	r, _ := setupRouter()

	wCreate := doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "06-2025",
	})
	var created model.Subscription
	json.Unmarshal(wCreate.Body.Bytes(), &created)

	w := doJSON(r, http.MethodPut, fmt.Sprintf("/api/v1/subscriptions/%d", created.ID),
		model.UpdateSubscriptionRequest{
			StartDate: strPtr("06-2025"),
			EndDate:   strPtr("01-2025"),
		})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ===== DELETE /api/v1/subscriptions/:id =====

func TestDeleteSubscription_Success(t *testing.T) {
	r, _ := setupRouter()

	wCreate := doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})
	var created model.Subscription
	json.Unmarshal(wCreate.Body.Bytes(), &created)

	w := doJSON(r, http.MethodDelete, fmt.Sprintf("/api/v1/subscriptions/%d", created.ID), nil)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// verify deletion
	wGet := doGet(r, fmt.Sprintf("/api/v1/subscriptions/%d", created.ID))
	assert.Equal(t, http.StatusNotFound, wGet.Code)
}

func TestDeleteSubscription_NotFound(t *testing.T) {
	r, _ := setupRouter()
	w := doJSON(r, http.MethodDelete, "/api/v1/subscriptions/999", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteSubscription_DoubleDelete(t *testing.T) {
	r, _ := setupRouter()

	wCreate := doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})
	var created model.Subscription
	json.Unmarshal(wCreate.Body.Bytes(), &created)

	doJSON(r, http.MethodDelete, fmt.Sprintf("/api/v1/subscriptions/%d", created.ID), nil)

	// second delete should return 404
	w := doJSON(r, http.MethodDelete, fmt.Sprintf("/api/v1/subscriptions/%d", created.ID), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteSubscription_InvalidID(t *testing.T) {
	r, _ := setupRouter()
	w := doJSON(r, http.MethodDelete, "/api/v1/subscriptions/abc", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ===== GET /api/v1/subscriptions/total-cost =====

func TestTotalCost_FullPeriod(t *testing.T) {
	r, _ := setupRouter()

	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Yandex Plus", Price: 400, UserID: testUUID,
		StartDate: "01-2025", EndDate: "12-2025",
	})

	w := doGet(r, "/api/v1/subscriptions/total-cost?start_date=01-2025&end_date=12-2025")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.TotalCostResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 4800, resp.TotalCost) // 400 × 12
}

func TestTotalCost_PartialPeriod(t *testing.T) {
	r, _ := setupRouter()

	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Yandex Plus", Price: 400, UserID: testUUID,
		StartDate: "03-2025", EndDate: "09-2025",
	})

	w := doGet(r, "/api/v1/subscriptions/total-cost?start_date=01-2025&end_date=06-2025")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.TotalCostResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1600, resp.TotalCost) // 400 × 4
}

func TestTotalCost_OngoingSubscription(t *testing.T) {
	r, _ := setupRouter()

	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Yandex Plus", Price: 400, UserID: testUUID,
		StartDate: "03-2025",
	})

	w := doGet(r, "/api/v1/subscriptions/total-cost?start_date=01-2025&end_date=06-2025")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.TotalCostResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1600, resp.TotalCost) // 400 × 4
}

func TestTotalCost_NoOverlap(t *testing.T) {
	r, _ := setupRouter()

	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Old Sub", Price: 400, UserID: testUUID,
		StartDate: "01-2024", EndDate: "12-2024",
	})

	w := doGet(r, "/api/v1/subscriptions/total-cost?start_date=01-2025&end_date=06-2025")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.TotalCostResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.TotalCost)
}

func TestTotalCost_WithUserFilter(t *testing.T) {
	r, _ := setupRouter()

	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID,
		StartDate: "01-2025", EndDate: "06-2025",
	})
	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID2,
		StartDate: "01-2025", EndDate: "06-2025",
	})

	w := doGet(r, "/api/v1/subscriptions/total-cost?start_date=01-2025&end_date=06-2025&user_id="+testUUID)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.TotalCostResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 4800, resp.TotalCost) // only user 1
}

func TestTotalCost_MissingRequiredParams(t *testing.T) {
	r, _ := setupRouter()

	w := doGet(r, "/api/v1/subscriptions/total-cost?start_date=01-2025")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = doGet(r, "/api/v1/subscriptions/total-cost?end_date=12-2025")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = doGet(r, "/api/v1/subscriptions/total-cost")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTotalCost_InvalidDateFormat(t *testing.T) {
	r, _ := setupRouter()

	w := doGet(r, "/api/v1/subscriptions/total-cost?start_date=invalid&end_date=12-2025")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTotalCost_EndBeforeStart(t *testing.T) {
	r, _ := setupRouter()

	w := doGet(r, "/api/v1/subscriptions/total-cost?start_date=06-2025&end_date=01-2025")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTotalCost_IntegerResponse(t *testing.T) {
	r, _ := setupRouter()

	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 399, UserID: testUUID,
		StartDate: "01-2025", EndDate: "01-2025",
	})

	w := doGet(r, "/api/v1/subscriptions/total-cost?start_date=01-2025&end_date=01-2025")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.TotalCostResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 399, resp.TotalCost) // integer, no decimals
}

// ===== ERROR RESPONSE FORMAT =====

func TestErrorResponseFormat(t *testing.T) {
	r, _ := setupRouter()

	w := doGet(r, "/api/v1/subscriptions/abc")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp model.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
	assert.NotEmpty(t, errResp.Error)
}

// ===== E2E LIFECYCLE =====

func TestE2E_SubscriptionLifecycle(t *testing.T) {
	r, _ := setupRouter()

	// 1. Create
	wCreate := doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID, StartDate: "01-2025",
	})
	assert.Equal(t, http.StatusCreated, wCreate.Code)
	var created model.Subscription
	json.Unmarshal(wCreate.Body.Bytes(), &created)

	// 2. List - verify present
	wList := doGet(r, "/api/v1/subscriptions")
	var subs []model.Subscription
	json.Unmarshal(wList.Body.Bytes(), &subs)
	assert.Len(t, subs, 1)

	// 3. Update - change price
	wUpdate := doJSON(r, http.MethodPut, fmt.Sprintf("/api/v1/subscriptions/%d", created.ID),
		model.UpdateSubscriptionRequest{Price: intPtr(500)})
	assert.Equal(t, http.StatusOK, wUpdate.Code)
	var updated model.Subscription
	json.Unmarshal(wUpdate.Body.Bytes(), &updated)
	assert.Equal(t, 500, updated.Price)

	// 4. Calculate total cost
	wCost := doGet(r, fmt.Sprintf("/api/v1/subscriptions/total-cost?start_date=01-2025&end_date=06-2025&user_id=%s", testUUID))
	assert.Equal(t, http.StatusOK, wCost.Code)
	var costResp model.TotalCostResponse
	json.Unmarshal(wCost.Body.Bytes(), &costResp)
	assert.Equal(t, 3000, costResp.TotalCost) // 500 × 6

	// 5. Delete
	wDelete := doJSON(r, http.MethodDelete, fmt.Sprintf("/api/v1/subscriptions/%d", created.ID), nil)
	assert.Equal(t, http.StatusNoContent, wDelete.Code)

	// 6. Verify deletion
	wGet := doGet(r, fmt.Sprintf("/api/v1/subscriptions/%d", created.ID))
	assert.Equal(t, http.StatusNotFound, wGet.Code)
}

func TestE2E_MultiUserAggregation(t *testing.T) {
	r, _ := setupRouter()

	// User A: 3 subscriptions
	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID,
		StartDate: "01-2025", EndDate: "12-2025",
	})
	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Spotify", Price: 300, UserID: testUUID,
		StartDate: "01-2025", EndDate: "12-2025",
	})
	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "YouTube", Price: 200, UserID: testUUID,
		StartDate: "01-2025", EndDate: "12-2025",
	})

	// User B: 2 subscriptions
	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 800, UserID: testUUID2,
		StartDate: "01-2025", EndDate: "06-2025",
	})
	doJSON(r, http.MethodPost, "/api/v1/subscriptions", model.CreateSubscriptionRequest{
		ServiceName: "Yandex Plus", Price: 400, UserID: testUUID2,
		StartDate: "03-2025", // ongoing
	})

	// User A total: (800+300+200)×12 = 15600
	w := doGet(r, "/api/v1/subscriptions/total-cost?start_date=01-2025&end_date=12-2025&user_id="+testUUID)
	var cost model.TotalCostResponse
	json.Unmarshal(w.Body.Bytes(), &cost)
	assert.Equal(t, 15600, cost.TotalCost)

	// User B total: 800×6 + 400×10 = 4800 + 4000 = 8800
	w = doGet(r, "/api/v1/subscriptions/total-cost?start_date=01-2025&end_date=12-2025&user_id="+testUUID2)
	json.Unmarshal(w.Body.Bytes(), &cost)
	assert.Equal(t, 8800, cost.TotalCost)

	// All Netflix: 800×12 + 800×6 = 14400
	w = doGet(r, "/api/v1/subscriptions/total-cost?start_date=01-2025&end_date=12-2025&service_name=Netflix")
	json.Unmarshal(w.Body.Bytes(), &cost)
	assert.Equal(t, 14400, cost.TotalCost)

	// Filter by user_id in List
	wList := doGet(r, "/api/v1/subscriptions?user_id="+testUUID)
	var subs []model.Subscription
	json.Unmarshal(wList.Body.Bytes(), &subs)
	assert.Len(t, subs, 3)
}

// --- helpers ---

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
