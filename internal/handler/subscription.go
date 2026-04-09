package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"subscription-service/internal/model"
	"subscription-service/internal/service"
)

type SubscriptionHandler struct {
	service *service.SubscriptionService
	logger  *zap.Logger
}

func NewSubscriptionHandler(svc *service.SubscriptionService, logger *zap.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{service: svc, logger: logger}
}

func (h *SubscriptionHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		api.POST("/subscriptions", h.Create)
		api.GET("/subscriptions", h.List)
		api.GET("/subscriptions/total-cost", h.TotalCost)
		api.GET("/subscriptions/:id", h.GetByID)
		api.PUT("/subscriptions/:id", h.Update)
		api.DELETE("/subscriptions/:id", h.Delete)
	}
}

// Create godoc
// @Summary Create a subscription
// @Description Create a new subscription record
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription body model.CreateSubscriptionRequest true "Subscription data"
// @Success 201 {object} model.Subscription
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /api/v1/subscriptions [post]
func (h *SubscriptionHandler) Create(c *gin.Context) {
	var req model.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request body"})
		return
	}

	sub, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create subscription", zap.Error(err))
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	h.logger.Info("subscription created", zap.Int("id", sub.ID))
	c.JSON(http.StatusCreated, sub)
}

// GetByID godoc
// @Summary Get a subscription by ID
// @Description Get a single subscription record by its ID
// @Tags subscriptions
// @Produce json
// @Param id path int true "Subscription ID"
// @Success 200 {object} model.Subscription
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Router /api/v1/subscriptions/{id} [get]
func (h *SubscriptionHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid id"})
		return
	}

	sub, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("subscription not found", zap.Int("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: "subscription not found"})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// List godoc
// @Summary List all subscriptions
// @Description Get a list of all subscription records, optionally filtered
// @Tags subscriptions
// @Produce json
// @Param user_id query string false "Filter by user UUID"
// @Param service_name query string false "Filter by service name"
// @Success 200 {array} model.Subscription
// @Failure 500 {object} model.ErrorResponse
// @Router /api/v1/subscriptions [get]
func (h *SubscriptionHandler) List(c *gin.Context) {
	userID := c.Query("user_id")
	serviceName := c.Query("service_name")

	subs, err := h.service.List(c.Request.Context(), userID, serviceName)
	if err != nil {
		h.logger.Error("failed to list subscriptions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "internal server error"})
		return
	}

	c.JSON(http.StatusOK, subs)
}

// Update godoc
// @Summary Update a subscription
// @Description Update an existing subscription record
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path int true "Subscription ID"
// @Param subscription body model.UpdateSubscriptionRequest true "Fields to update"
// @Success 200 {object} model.Subscription
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Router /api/v1/subscriptions/{id} [put]
func (h *SubscriptionHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid id"})
		return
	}

	var req model.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request body"})
		return
	}

	sub, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		h.logger.Error("failed to update subscription", zap.Int("id", id), zap.Error(err))
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// Delete godoc
// @Summary Delete a subscription
// @Description Delete a subscription record by its ID
// @Tags subscriptions
// @Param id path int true "Subscription ID"
// @Success 204
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Router /api/v1/subscriptions/{id} [delete]
func (h *SubscriptionHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid id"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("failed to delete subscription", zap.Int("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: err.Error()})
		return
	}

	h.logger.Info("subscription deleted", zap.Int("id", id))
	c.Status(http.StatusNoContent)
}

// TotalCost godoc
// @Summary Calculate total subscription cost
// @Description Calculate the total cost of subscriptions for a given period, optionally filtered by user_id and service_name
// @Tags subscriptions
// @Produce json
// @Param start_date query string true "Period start (MM-YYYY)" example("01-2025")
// @Param end_date query string true "Period end (MM-YYYY)" example("12-2025")
// @Param user_id query string false "Filter by user UUID" example("60601fee-2bf1-4721-ae6f-7636e79a0cba")
// @Param service_name query string false "Filter by service name" example("Yandex Plus")
// @Success 200 {object} model.TotalCostResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /api/v1/subscriptions/total-cost [get]
func (h *SubscriptionHandler) TotalCost(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	userID := c.Query("user_id")
	serviceName := c.Query("service_name")

	total, err := h.service.TotalCost(c.Request.Context(), startDate, endDate, userID, serviceName)
	if err != nil {
		h.logger.Error("failed to calculate total cost", zap.Error(err))
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.TotalCostResponse{TotalCost: total})
}
