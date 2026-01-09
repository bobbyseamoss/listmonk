package core

import (
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

var formQuerySortFields = []string{"name", "status", "form_type", "created_at", "updated_at"}

// GetForms returns all sign-up forms with optional filtering and pagination.
func (c *Core) GetForms(status, formType, searchStr, orderBy, order string, offset, limit int) ([]models.SignUpForm, int, error) {
	out := []models.SignUpForm{}

	queryStr, stmt := makeSearchQuery(searchStr, orderBy, order, c.q.QuerySignUpForms, formQuerySortFields)

	if err := c.db.Select(&out, stmt, 0, "", status, formType, queryStr, limit, offset); err != nil {
		c.log.Printf("error fetching forms: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "forms", "error", pqErrMsg(err)))
	}

	total := 0
	if len(out) > 0 {
		total = out[0].Total
	}

	return out, total, nil
}

// GetForm gets a single form by ID or UUID.
func (c *Core) GetForm(id int, uuid string) (models.SignUpForm, error) {
	var uu interface{}
	if uuid != "" {
		uu = uuid
	}

	var out []models.SignUpForm
	queryStr, stmt := makeSearchQuery("", "", "", c.q.QuerySignUpForms, nil)
	if err := c.db.Select(&out, stmt, id, uu, "", "", queryStr, 1, 0); err != nil {
		c.log.Printf("error fetching form: %v", err)
		return models.SignUpForm{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "form", "error", pqErrMsg(err)))
	}

	if len(out) == 0 {
		return models.SignUpForm{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "form"))
	}

	return out[0], nil
}

// CreateForm creates a new sign-up form.
func (c *Core) CreateForm(f models.SignUpForm) (models.SignUpForm, error) {
	uu, err := uuid.NewV4()
	if err != nil {
		c.log.Printf("error generating UUID: %v", err)
		return models.SignUpForm{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}

	// Set defaults if not provided.
	if f.FormType == "" {
		f.FormType = models.FormTypePopup
	}
	if f.Status == "" {
		f.Status = models.FormStatusDraft
	}

	// Initialize empty JSONB fields if needed.
	if f.BodySource == nil {
		f.BodySource = models.JSON{}
	}
	if f.Steps == nil {
		f.Steps = models.JSON{}
	}
	if f.Settings == nil {
		f.Settings = models.JSON{}
	}
	if f.Targeting == nil {
		f.Targeting = models.JSON{}
	}
	if f.Triggers == nil {
		f.Triggers = models.JSON{}
	}
	if f.Frequency == nil {
		f.Frequency = models.JSON{}
	}
	if f.CouponConfig == nil {
		f.CouponConfig = models.JSON{}
	}

	var newID int
	if err := c.q.CreateSignUpForm.Get(&newID,
		uu.String(),
		f.Name,
		f.Description,
		f.FormType,
		f.Status,
		f.BodySource,
		f.CustomCSS,
		f.Steps,
		f.Settings,
		f.Targeting,
		f.Triggers,
		f.Frequency,
		pq.Int64Array(f.ListIDs),
		f.CouponConfig,
	); err != nil {
		c.log.Printf("error creating form: %v", err)
		return models.SignUpForm{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "form", "error", pqErrMsg(err)))
	}

	return c.GetForm(newID, "")
}

// UpdateForm updates an existing sign-up form.
func (c *Core) UpdateForm(id int, f models.SignUpForm) (models.SignUpForm, error) {
	res, err := c.q.UpdateSignUpForm.Exec(
		id,
		f.Name,
		f.Description,
		f.FormType,
		f.Status,
		f.BodySource,
		f.CustomCSS,
		f.Steps,
		f.Settings,
		f.Targeting,
		f.Triggers,
		f.Frequency,
		pq.Int64Array(f.ListIDs),
		f.CouponConfig,
	)
	if err != nil {
		c.log.Printf("error updating form: %v", err)
		return models.SignUpForm{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "form", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return models.SignUpForm{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "form"))
	}

	return c.GetForm(id, "")
}

// UpdateFormStatus updates only the status of a form.
func (c *Core) UpdateFormStatus(id int, status string) (models.SignUpForm, error) {
	res, err := c.q.UpdateSignUpFormStatus.Exec(id, status)
	if err != nil {
		c.log.Printf("error updating form status: %v", err)
		return models.SignUpForm{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "form", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return models.SignUpForm{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "form"))
	}

	return c.GetForm(id, "")
}

// DeleteForm deletes a single form.
func (c *Core) DeleteForm(id int) error {
	return c.DeleteForms([]int{id})
}

// DeleteForms deletes multiple forms.
func (c *Core) DeleteForms(ids []int) error {
	if _, err := c.q.DeleteSignUpForms.Exec(pq.Array(ids)); err != nil {
		c.log.Printf("error deleting forms: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "form", "error", pqErrMsg(err)))
	}
	return nil
}

// RecordFormImpression records a form impression for analytics.
func (c *Core) RecordFormImpression(formID int, sessionID, visitorID, pageURL, deviceType, country string) (int64, error) {
	var id int64
	if err := c.q.InsertFormImpression.Get(&id, formID, sessionID, visitorID, pageURL, deviceType, country); err != nil {
		c.log.Printf("error recording form impression: %v", err)
		return 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "impression", "error", pqErrMsg(err)))
	}
	return id, nil
}

// UpdateFormImpressionClosed marks a form impression as closed.
func (c *Core) UpdateFormImpressionClosed(id int64) error {
	if _, err := c.q.UpdateFormImpressionClosed.Exec(id); err != nil {
		c.log.Printf("error updating form impression: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "impression", "error", pqErrMsg(err)))
	}
	return nil
}

// UpdateFormImpressionSubmitted marks a form impression as submitted.
func (c *Core) UpdateFormImpressionSubmitted(id int64) error {
	if _, err := c.q.UpdateFormImpressionSubmitted.Exec(id); err != nil {
		c.log.Printf("error updating form impression: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "impression", "error", pqErrMsg(err)))
	}
	return nil
}

// UpdateFormImpressionStep updates the step reached in a form impression.
func (c *Core) UpdateFormImpressionStep(id int64, step int) error {
	if _, err := c.q.UpdateFormImpressionStep.Exec(id, step); err != nil {
		c.log.Printf("error updating form impression step: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "impression", "error", pqErrMsg(err)))
	}
	return nil
}

// RecordFormSubmission records a form submission.
func (c *Core) RecordFormSubmission(sub models.FormSubmission) (int64, error) {
	var id int64
	if err := c.q.InsertFormSubmission.Get(&id,
		sub.FormID,
		sub.SubscriberID,
		sub.Email,
		sub.Data,
		sub.PageURL,
		sub.Referrer,
		sub.UTMSource,
		sub.UTMMedium,
		sub.UTMCampaign,
		sub.UTMTerm,
		sub.UTMContent,
		sub.DeviceType,
		sub.UserAgent,
		sub.IPAddress,
		sub.Country,
		sub.CouponCode,
	); err != nil {
		c.log.Printf("error recording form submission: %v", err)
		return 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "submission", "error", pqErrMsg(err)))
	}
	return id, nil
}

// GetFormSubmissions returns paginated form submissions.
func (c *Core) GetFormSubmissions(formID int, offset, limit int) ([]models.FormSubmission, int, error) {
	out := []models.FormSubmission{}

	if err := c.q.QueryFormSubmissions.Select(&out, formID, limit, offset); err != nil {
		c.log.Printf("error fetching form submissions: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "submissions", "error", pqErrMsg(err)))
	}

	total := 0
	if len(out) > 0 {
		total = out[0].Total
	}

	return out, total, nil
}

// GetFormAnalytics returns aggregated analytics for a form.
func (c *Core) GetFormAnalytics(formID int, from, to time.Time) (models.FormAnalytics, error) {
	var out struct {
		FormID      int `db:"form_id"`
		Impressions int `db:"impressions"`
		Submissions int `db:"submissions"`
	}

	if err := c.q.GetFormAnalytics.Get(&out, formID, from, to); err != nil {
		c.log.Printf("error fetching form analytics: %v", err)
		return models.FormAnalytics{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "analytics", "error", pqErrMsg(err)))
	}

	// Calculate conversion rate.
	conversionRate := float64(0)
	if out.Impressions > 0 {
		conversionRate = float64(out.Submissions) / float64(out.Impressions) * 100
	}

	return models.FormAnalytics{
		FormID:         out.FormID,
		Impressions:    out.Impressions,
		Submissions:    out.Submissions,
		ConversionRate: conversionRate,
	}, nil
}

// GetFormAnalyticsTimeseries returns time-series analytics for a form.
func (c *Core) GetFormAnalyticsTimeseries(formID int, from, to time.Time) ([]models.FormAnalyticsTimeSeries, error) {
	out := []models.FormAnalyticsTimeSeries{}

	if err := c.q.GetFormAnalyticsTimeseries.Select(&out, formID, from, to); err != nil {
		c.log.Printf("error fetching form analytics timeseries: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "analytics", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetCouponStats returns coupon usage statistics for a form.
func (c *Core) GetCouponStats(formID int) (map[string]int, error) {
	var out struct {
		Available int `db:"available"`
		Used      int `db:"used"`
		Total     int `db:"total"`
	}

	if err := c.q.GetCouponStats.Get(&out, formID); err != nil {
		c.log.Printf("error fetching coupon stats: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "coupon stats", "error", pqErrMsg(err)))
	}

	return map[string]int{
		"available": out.Available,
		"used":      out.Used,
		"total":     out.Total,
	}, nil
}

// GetNextCouponCode gets and marks the next available coupon code for a form.
func (c *Core) GetNextCouponCode(formID int, email string) (string, error) {
	var code string
	if err := c.q.GetNextCouponCode.Get(&code, formID, email); err != nil {
		// If no rows, no coupons available.
		if err.Error() == "sql: no rows in result set" {
			return "", nil
		}
		c.log.Printf("error getting next coupon code: %v", err)
		return "", echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "coupon code", "error", pqErrMsg(err)))
	}
	return code, nil
}

// GetActiveFormsForTargeting returns all active forms for public targeting evaluation.
func (c *Core) GetActiveFormsForTargeting() ([]models.SignUpForm, error) {
	out := []models.SignUpForm{}

	if err := c.q.GetActiveFormsForTargeting.Select(&out); err != nil {
		c.log.Printf("error fetching active forms: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "forms", "error", pqErrMsg(err)))
	}

	return out, nil
}
