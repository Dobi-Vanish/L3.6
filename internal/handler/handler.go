package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wb-go/wbf/ginext"

	"L3.6/internal/logger"
	"L3.6/internal/model"
	"L3.6/internal/repository"
	"L3.6/internal/service"
)

type Handler struct {
	svc *service.TransactionService
}

func NewHandler(svc *service.TransactionService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(router *ginext.Engine) {
	api := router.Group("/api")
	{
		api.POST("/items", h.createTransaction)
		api.GET("/items", h.listTransactions)
		api.PUT("/items/:id", h.updateTransaction)
		api.DELETE("/items/:id", h.deleteTransaction)
		api.GET("/analytics", h.getAnalytics)
		api.GET("/analytics/group", h.getGroupedAnalytics)
		api.GET("/export/csv", h.exportCSV)
		api.GET("/export/analytics/csv", h.exportAnalyticsCSV)
	}
	router.Static("/web", "./web")
	router.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})
}

func (h *Handler) createTransaction(c *gin.Context) {
	var input struct {
		Type        string  `json:"type"`
		Category    string  `json:"category"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
		Date        string  `json:"date"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Error("bind json error", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	date, err := time.Parse("2006-01-02", input.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use YYYY-MM-DD"})
		return
	}
	tx := &model.Transaction{
		Type:        input.Type,
		Category:    input.Category,
		Amount:      input.Amount,
		Description: input.Description,
		Date:        date,
	}
	if err := h.svc.Create(c.Request.Context(), tx); err != nil {
		logger.Error("create transaction error", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, tx)
}

func (h *Handler) updateTransaction(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	var input struct {
		Type        string  `json:"type"`
		Category    string  `json:"category"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
		Date        string  `json:"date"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	date, err := time.Parse("2006-01-02", input.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use YYYY-MM-DD"})
		return
	}
	tx := &model.Transaction{
		ID:          id,
		Type:        input.Type,
		Category:    input.Category,
		Amount:      input.Amount,
		Description: input.Description,
		Date:        date,
	}
	if err := h.svc.Update(c.Request.Context(), tx); err != nil {
		if err.Error() == "sql: no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
			return
		}
		logger.Error("update transaction error", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tx)
}

func (h *Handler) deleteTransaction(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listTransactions(c *gin.Context) {
	var filter repository.ListFilter
	if fromStr := c.Query("from"); fromStr != "" {
		t, err := time.Parse("2006-01-02", fromStr)
		if err == nil {
			filter.From = &t
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		t, err := time.Parse("2006-01-02", toStr)
		if err == nil {
			filter.To = &t
		}
	}
	if cat := c.Query("category"); cat != "" {
		filter.Category = &cat
	}
	if typ := c.Query("type"); typ != "" {
		filter.Type = &typ
	}
	filter.SortBy = c.DefaultQuery("sort_by", "date")
	filter.Order = c.DefaultQuery("order", "desc")
	if limit, err := strconv.Atoi(c.DefaultQuery("limit", "100")); err == nil {
		filter.Limit = limit
	}
	if offset, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil {
		filter.Offset = offset
	}
	transactions, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, transactions)
}

func (h *Handler) getAnalytics(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "both 'from' and 'to' dates are required (YYYY-MM-DD)"})
		return
	}
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'from' date, use YYYY-MM-DD"})
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'to' date, use YYYY-MM-DD"})
		return
	}
	if from.After(to) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "'from' date cannot be after 'to' date"})
		return
	}
	analytics, err := h.svc.GetAnalytics(c.Request.Context(), from, to)
	if err != nil {
		logger.Error("analytics error", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute analytics"})
		return
	}
	if analytics.Count == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":       "No transactions found in the selected period",
			"sum":           0,
			"avg":           0,
			"count":         0,
			"median":        0,
			"percentile_90": 0,
		})
		return
	}
	c.JSON(http.StatusOK, analytics)
}

func (h *Handler) getGroupedAnalytics(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "both 'from' and 'to' dates are required"})
		return
	}
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'from' date"})
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'to' date"})
		return
	}
	if from.After(to) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "'from' date cannot be after 'to' date"})
		return
	}
	groupBy := c.DefaultQuery("group_by", "day")
	results, err := h.svc.GetGroupedAnalytics(c.Request.Context(), from, to, groupBy)
	if err != nil {
		if err.Error() == "unsupported group_by: "+groupBy {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute grouped analytics"})
		}
		return
	}
	if len(results) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No transactions in selected period", "data": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, results)
}

func (h *Handler) exportCSV(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "both 'from' and 'to' dates are required"})
		return
	}
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'from' date"})
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'to' date"})
		return
	}
	if from.After(to) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "'from' date cannot be after 'to' date"})
		return
	}
	transactions, err := h.svc.ExportCSV(c.Request.Context(), from, to)
	if err != nil {
		logger.Error("export error", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export"})
		return
	}
	if len(transactions) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No transactions to export"})
		return
	}
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=transactions.csv")
	writer := csv.NewWriter(c.Writer)
	writer.Write([]string{"ID", "Type", "Category", "Amount", "Description", "Date", "CreatedAt", "UpdatedAt"})
	for _, tx := range transactions {
		writer.Write([]string{
			tx.ID, tx.Type, tx.Category, strconv.FormatFloat(tx.Amount, 'f', 5, 64),
			tx.Description, tx.Date.Format("2006-01-02"), tx.CreatedAt.Format(time.RFC3339),
			tx.UpdatedAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
}

func (h *Handler) exportAnalyticsCSV(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "both 'from' and 'to' dates are required"})
		return
	}
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'from' date"})
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'to' date"})
		return
	}
	if from.After(to) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "'from' date cannot be after 'to' date"})
		return
	}
	analytics, err := h.svc.GetAnalytics(c.Request.Context(), from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute analytics"})
		return
	}
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=analytics.csv")
	writer := csv.NewWriter(c.Writer)
	writer.Write([]string{"Metric", "Value"})
	writer.Write([]string{"Sum", strconv.FormatFloat(analytics.Sum, 'f', 5, 64)})
	writer.Write([]string{"Average", strconv.FormatFloat(analytics.Avg, 'f', 5, 64)})
	writer.Write([]string{"Count", strconv.Itoa(analytics.Count)})
	writer.Write([]string{"Median", strconv.FormatFloat(analytics.Median, 'f', 5, 64)})
	writer.Write([]string{"90th Percentile", strconv.FormatFloat(analytics.Percentile90, 'f', 5, 64)})
	writer.Flush()
}
