package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	null "gopkg.in/volatiletech/null.v6"
)

// GetForms retrieves all sign-up forms with optional filtering.
func (a *App) GetForms(c echo.Context) error {
	var (
		status   = strings.TrimSpace(c.FormValue("status"))
		formType = strings.TrimSpace(c.FormValue("form_type"))
		query    = strings.TrimSpace(c.FormValue("query"))
		orderBy  = c.FormValue("order_by")
		order    = c.FormValue("order")
		pg       = a.pg.NewFromURL(c.Request().URL.Query())
	)

	res, total, err := a.core.GetForms(status, formType, query, orderBy, order, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	out := models.PageResults{
		Query:   query,
		Results: res,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetForm retrieves a single sign-up form by ID.
func (a *App) GetForm(c echo.Context) error {
	id := getID(c)
	out, err := a.core.GetForm(id, "")
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateForm creates a new sign-up form.
func (a *App) CreateForm(c echo.Context) error {
	var f models.SignUpForm
	if err := c.Bind(&f); err != nil {
		return err
	}

	// Validate required fields.
	if !strHasLen(f.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("forms.invalidName"))
	}

	out, err := a.core.CreateForm(f)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateForm updates an existing sign-up form.
func (a *App) UpdateForm(c echo.Context) error {
	id := getID(c)

	var f models.SignUpForm
	if err := c.Bind(&f); err != nil {
		return err
	}

	out, err := a.core.UpdateForm(id, f)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateFormStatus updates only the status of a sign-up form.
func (a *App) UpdateFormStatus(c echo.Context) error {
	id := getID(c)

	var req struct {
		Status string `json:"status"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	// Validate status.
	if req.Status != models.FormStatusDraft &&
		req.Status != models.FormStatusActive &&
		req.Status != models.FormStatusPaused &&
		req.Status != models.FormStatusArchived {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("forms.invalidStatus"))
	}

	out, err := a.core.UpdateFormStatus(id, req.Status)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteForms deletes one or more sign-up forms.
func (a *App) DeleteForms(c echo.Context) error {
	var (
		id, _ = strconv.ParseInt(c.Param("id"), 10, 64)
		ids   []int
	)
	if id < 1 && len(ids) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	if id > 0 {
		ids = append(ids, int(id))
	}

	if err := a.core.DeleteForms(ids); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DuplicateForm creates a copy of an existing form.
func (a *App) DuplicateForm(c echo.Context) error {
	id := getID(c)

	// Get the original form.
	original, err := a.core.GetForm(id, "")
	if err != nil {
		return err
	}

	// Create a copy with modified name.
	copy := original
	copy.Name = fmt.Sprintf("%s (Copy)", original.Name)
	copy.Status = models.FormStatusDraft

	out, err := a.core.CreateForm(copy)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetFormSubmissions retrieves submissions for a form.
func (a *App) GetFormSubmissions(c echo.Context) error {
	id := getID(c)
	pg := a.pg.NewFromURL(c.Request().URL.Query())

	res, total, err := a.core.GetFormSubmissions(id, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	out := models.PageResults{
		Results: res,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetFormAnalytics retrieves analytics for a form.
func (a *App) GetFormAnalytics(c echo.Context) error {
	id := getID(c)

	// Parse date range.
	from, to, err := parseDateRange(c)
	if err != nil {
		return err
	}

	analytics, err := a.core.GetFormAnalytics(id, from, to)
	if err != nil {
		return err
	}

	// Get time series data.
	timeseries, err := a.core.GetFormAnalyticsTimeseries(id, from, to)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{map[string]interface{}{
		"summary":    analytics,
		"timeseries": timeseries,
	}})
}

// GetFormCouponStats retrieves coupon usage stats for a form.
func (a *App) GetFormCouponStats(c echo.Context) error {
	id := getID(c)

	stats, err := a.core.GetCouponStats(id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{stats})
}

// UploadFormCoupons handles bulk upload of coupon codes.
func (a *App) UploadFormCoupons(c echo.Context) error {
	id := getID(c)

	// Verify the form exists.
	if _, err := a.core.GetForm(id, ""); err != nil {
		return err
	}

	var req struct {
		Codes []string `json:"codes"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	if len(req.Codes) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("forms.noCouponsProvided"))
	}

	// Build the bulk insert query.
	var values []string
	for _, code := range req.Codes {
		code = strings.TrimSpace(code)
		if code != "" {
			values = append(values, fmt.Sprintf("(%d, '%s')", id, strings.ReplaceAll(code, "'", "''")))
		}
	}

	if len(values) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("forms.noCouponsProvided"))
	}

	query := fmt.Sprintf(a.queries.InsertFormCouponCodes, strings.Join(values, ","))
	if _, err := a.db.Exec(query); err != nil {
		a.log.Printf("error inserting coupon codes: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			a.i18n.Ts("globals.messages.errorCreating", "name", "coupons", "error", err.Error()))
	}

	// Return updated stats.
	stats, err := a.core.GetCouponStats(id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{stats})
}

// ========== Public Form Handlers ==========

// handlePublicFormsCORSPreflight handles CORS preflight requests for public form endpoints.
func (a *App) handlePublicFormsCORSPreflight(c echo.Context) error {
	setPublicFormsCORSHeaders(c)
	return c.NoContent(http.StatusNoContent)
}

// GetPublicForm returns a form's public configuration by UUID.
func (a *App) GetPublicForm(c echo.Context) error {
	// Set CORS headers for cross-origin requests
	setPublicFormsCORSHeaders(c)

	uuid := c.Param("uuid")
	if uuid == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidUUID"))
	}

	form, err := a.core.GetForm(0, uuid)
	if err != nil {
		return err
	}

	// Only return active forms.
	if form.Status != models.FormStatusActive {
		return echo.NewHTTPError(http.StatusNotFound, a.i18n.Ts("globals.messages.notFound", "name", "form"))
	}

	return c.JSON(http.StatusOK, okResp{form})
}

// SubmitPublicForm handles form submissions from the public embed script.
func (a *App) SubmitPublicForm(c echo.Context) error {
	// Set CORS headers for cross-origin requests
	setPublicFormsCORSHeaders(c)

	uuid := c.Param("uuid")
	if uuid == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidUUID"))
	}

	// Get the form.
	form, err := a.core.GetForm(0, uuid)
	if err != nil {
		return err
	}

	// Only accept submissions for active forms.
	if form.Status != models.FormStatusActive {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("forms.formNotActive"))
	}

	var req struct {
		Email        string                 `json:"email"`
		Name         string                 `json:"name"`
		Data         map[string]interface{} `json:"data"`
		PageURL      string                 `json:"page_url"`
		Referrer     string                 `json:"referrer"`
		UTMSource    string                 `json:"utm_source"`
		UTMMedium    string                 `json:"utm_medium"`
		UTMCampaign  string                 `json:"utm_campaign"`
		UTMTerm      string                 `json:"utm_term"`
		UTMContent   string                 `json:"utm_content"`
		DeviceType   string                 `json:"device_type"`
		ImpressionID int64                  `json:"impression_id"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	// Validate email.
	if len(req.Email) > 1000 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.invalidEmail"))
	}

	em, err := a.importer.SanitizeEmail(req.Email)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.Email = em

	// Handle name - extract from form data fields if not provided directly.
	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) == 0 {
		// Try to build name from first_name/last_name or fn/ln fields in form data.
		var firstName, lastName string
		if fn, ok := req.Data["fn"].(string); ok {
			firstName = strings.TrimSpace(fn)
		} else if fn, ok := req.Data["first_name"].(string); ok {
			firstName = strings.TrimSpace(fn)
		}
		if ln, ok := req.Data["ln"].(string); ok {
			lastName = strings.TrimSpace(ln)
		} else if ln, ok := req.Data["last_name"].(string); ok {
			lastName = strings.TrimSpace(ln)
		}

		if firstName != "" || lastName != "" {
			req.Name = strings.TrimSpace(firstName + " " + lastName)
		} else {
			req.Name = strings.Split(req.Email, "@")[0]
		}
	}
	if len(req.Name) > stdInputMaxLen {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.invalidName"))
	}

	// Get list IDs from form configuration.
	listIDs := make([]int, len(form.ListIDs))
	for i, id := range form.ListIDs {
		listIDs[i] = int(id)
	}

	if len(listIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("public.noListsSelected"))
	}

	// Insert or update the subscriber.
	var subscriberID int
	var subscriberUUID string
	sub, hasOptin, err := a.core.InsertSubscriber(models.Subscriber{
		Name:   req.Name,
		Email:  req.Email,
		Status: models.SubscriberStatusEnabled,
	}, listIDs, nil, false, true)
	if err == nil {
		subscriberID = sub.ID
		subscriberUUID = sub.UUID
	} else if e, ok := err.(*echo.HTTPError); ok && e.Code == http.StatusConflict {
		// Subscriber already exists. Update their name and subscriptions.
		existingSub, err := a.core.GetSubscriber(0, "", req.Email)
		if err != nil {
			return err
		}
		subscriberID = existingSub.ID
		subscriberUUID = existingSub.UUID

		// Update the subscriber's name if a new name was provided.
		updatedSub := existingSub
		if req.Name != "" && req.Name != existingSub.Name {
			updatedSub.Name = req.Name
		}

		_, hasOptin, err = a.core.UpdateSubscriberWithLists(existingSub.ID, updatedSub, listIDs, nil, false, false, true)
		if err != nil {
			return err
		}
	} else {
		return err
	}

	// Get a coupon code if configured.
	var couponCode string
	// Check if coupon config has enabled field set to true
	if form.CouponConfig != nil {
		couponCode, _ = a.core.GetNextCouponCode(form.ID, req.Email)
	}

	// Record the submission.
	dataMap := models.JSON{}
	for k, v := range req.Data {
		dataMap[k] = v
	}
	submission := models.FormSubmission{
		FormID:       form.ID,
		Email:        req.Email,
		Data:         dataMap,
		PageURL:      null.String{String: req.PageURL, Valid: req.PageURL != ""},
		Referrer:     null.String{String: req.Referrer, Valid: req.Referrer != ""},
		UTMSource:    null.String{String: req.UTMSource, Valid: req.UTMSource != ""},
		UTMMedium:    null.String{String: req.UTMMedium, Valid: req.UTMMedium != ""},
		UTMCampaign:  null.String{String: req.UTMCampaign, Valid: req.UTMCampaign != ""},
		UTMTerm:      null.String{String: req.UTMTerm, Valid: req.UTMTerm != ""},
		UTMContent:   null.String{String: req.UTMContent, Valid: req.UTMContent != ""},
		DeviceType:   null.String{String: req.DeviceType, Valid: req.DeviceType != ""},
		UserAgent:    null.String{String: c.Request().UserAgent(), Valid: true},
		IPAddress:    null.String{String: c.RealIP(), Valid: true},
		CouponCode:   null.String{String: couponCode, Valid: couponCode != ""},
	}
	if subscriberID > 0 {
		submission.SubscriberID = null.Int{Int: subscriberID, Valid: true}
	}

	submissionID, err := a.core.RecordFormSubmission(submission)
	if err != nil {
		a.log.Printf("error recording form submission: %v", err)
		// Don't fail the request - the subscriber was already created.
	}

	// Update the impression as submitted if provided.
	if req.ImpressionID > 0 {
		_ = a.core.UpdateFormImpressionSubmitted(req.ImpressionID)
	}

	return c.JSON(http.StatusOK, okResp{map[string]interface{}{
		"submission_id":   submissionID,
		"has_optin":       hasOptin,
		"coupon_code":     couponCode,
		"subscriber_uuid": subscriberUUID,
	}})
}

// RecordFormImpression records a form impression for analytics.
func (a *App) RecordFormImpression(c echo.Context) error {
	// Set CORS headers for cross-origin requests
	setPublicFormsCORSHeaders(c)

	uuid := c.Param("uuid")
	if uuid == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidUUID"))
	}

	// Get the form to verify it exists and is active.
	form, err := a.core.GetForm(0, uuid)
	if err != nil {
		return err
	}

	if form.Status != models.FormStatusActive {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("forms.formNotActive"))
	}

	var req struct {
		SessionID  string `json:"session_id"`
		VisitorID  string `json:"visitor_id"`
		PageURL    string `json:"page_url"`
		DeviceType string `json:"device_type"`
		Country    string `json:"country"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	if req.SessionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "session_id is required")
	}

	id, err := a.core.RecordFormImpression(form.ID, req.SessionID, req.VisitorID, req.PageURL, req.DeviceType, req.Country)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{map[string]interface{}{
		"impression_id": id,
	}})
}

// UpdateFormImpressionClosed marks a form impression as closed.
func (a *App) UpdateFormImpressionClosed(c echo.Context) error {
	// Set CORS headers for cross-origin requests
	setPublicFormsCORSHeaders(c)

	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	if err := a.core.UpdateFormImpressionClosed(id); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// UpdateFormImpressionSubmitted marks a form impression as submitted.
func (a *App) UpdateFormImpressionSubmitted(c echo.Context) error {
	// Set CORS headers for cross-origin requests
	setPublicFormsCORSHeaders(c)

	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	if err := a.core.UpdateFormImpressionSubmitted(id); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// UpdateFormImpressionStep updates the step reached in a form impression.
func (a *App) UpdateFormImpressionStep(c echo.Context) error {
	// Set CORS headers for cross-origin requests
	setPublicFormsCORSHeaders(c)

	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	var req struct {
		Step int `json:"step"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	if err := a.core.UpdateFormImpressionStep(id, req.Step); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// GetActiveFormsForPage returns all active forms for the embed script to evaluate targeting.
func (a *App) GetActiveFormsForPage(c echo.Context) error {
	// Set CORS headers for cross-origin requests
	setPublicFormsCORSHeaders(c)

	forms, err := a.core.GetActiveFormsForTargeting()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{forms})
}

// LookupPublicSubscriber returns subscriber profile data by UUID.
// This enables GTM/analytics integrations to look up subscriber data
// similar to Klaviyo's profile lookup API.
func (a *App) LookupPublicSubscriber(c echo.Context) error {
	// Set CORS headers for cross-origin requests
	setPublicFormsCORSHeaders(c)

	uuid := c.QueryParam("uuid")
	if uuid == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "uuid parameter is required")
	}

	// Look up subscriber by UUID
	sub, err := a.core.GetSubscriber(0, uuid, "")
	if err != nil {
		// Return empty response instead of error for privacy
		return c.JSON(http.StatusOK, okResp{map[string]interface{}{
			"found": false,
		}})
	}

	// Extract profile fields from subscriber attributes
	attribs := sub.Attribs
	firstName := ""
	lastName := ""
	phone := ""
	birthday := ""
	address := ""

	if fn, ok := attribs["first_name"].(string); ok {
		firstName = fn
	} else if fn, ok := attribs["fn"].(string); ok {
		firstName = fn
	}
	if ln, ok := attribs["last_name"].(string); ok {
		lastName = ln
	} else if ln, ok := attribs["ln"].(string); ok {
		lastName = ln
	}
	if p, ok := attribs["phone"].(string); ok {
		phone = p
	}
	if bd, ok := attribs["birthday"].(string); ok {
		birthday = bd
	} else if bd, ok := attribs["Birthday"].(string); ok {
		birthday = bd
	}
	if addr, ok := attribs["address"].(string); ok {
		address = addr
	}

	return c.JSON(http.StatusOK, okResp{map[string]interface{}{
		"found":      true,
		"uuid":       sub.UUID,
		"email":      sub.Email,
		"name":       sub.Name,
		"first_name": firstName,
		"last_name":  lastName,
		"phone":      phone,
		"birthday":   birthday,
		"address":    address,
	}})
}

// parseDateRange parses from and to date parameters from the request.
func parseDateRange(c echo.Context) (time.Time, time.Time, error) {
	var (
		fromStr = c.FormValue("from")
		toStr   = c.FormValue("to")
		from    time.Time
		to      time.Time
		err     error
	)

	if fromStr != "" {
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, echo.NewHTTPError(http.StatusBadRequest, "invalid 'from' date format")
		}
	} else {
		// Default to 30 days ago.
		from = time.Now().AddDate(0, 0, -30)
	}

	if toStr != "" {
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			return time.Time{}, time.Time{}, echo.NewHTTPError(http.StatusBadRequest, "invalid 'to' date format")
		}
	} else {
		to = time.Now()
	}

	// Ensure 'to' includes the entire day.
	to = to.Add(24*time.Hour - time.Second)

	return from, to, nil
}
