package main

import (
	"encoding/csv"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetSegments retrieves all segments with pagination.
func (a *App) GetSegments(c echo.Context) error {
	var (
		query   = c.FormValue("query")
		orderBy = c.FormValue("order_by")
		order   = c.FormValue("order")
		pg      = a.pg.NewFromURL(c.Request().URL.Query())
	)

	res, total, err := a.core.QuerySegments(query, orderBy, order, pg.Offset, pg.Limit)
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

// GetSegment retrieves a single segment by ID or UUID.
func (a *App) GetSegment(c echo.Context) error {
	var (
		id, _ = strconv.Atoi(c.Param("id"))
	)

	// If ID is 0, try to interpret as UUID
	uuid := ""
	if id == 0 {
		uuid = c.Param("id")
	}

	out, err := a.core.GetSegment(id, uuid)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateSegment handles segment creation.
func (a *App) CreateSegment(c echo.Context) error {
	var seg models.Segment
	if err := c.Bind(&seg); err != nil {
		return err
	}

	// Validate name.
	if !strHasLen(seg.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("segments.invalidName"))
	}

	out, err := a.core.CreateSegment(seg)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateSegment handles segment modification.
func (a *App) UpdateSegment(c echo.Context) error {
	var (
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	var seg models.Segment
	if err := c.Bind(&seg); err != nil {
		return err
	}

	// Validate name.
	if !strHasLen(seg.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("segments.invalidName"))
	}

	out, err := a.core.UpdateSegment(id, seg)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteSegment handles segment deletion.
func (a *App) DeleteSegment(c echo.Context) error {
	var (
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	if err := a.core.DeleteSegment(id); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// PreviewSegment returns a paginated preview of subscribers matching the segment.
func (a *App) PreviewSegment(c echo.Context) error {
	var (
		id, _ = strconv.Atoi(c.Param("id"))
		pg    = a.pg.NewFromURL(c.Request().URL.Query())
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	orderBy := c.FormValue("order_by")
	order := c.FormValue("order")

	subs, total, err := a.core.GetSegmentSubscribers(id, orderBy, order, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	out := models.PageResults{
		Results: subs,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetSegmentCount returns the subscriber count for a segment.
func (a *App) GetSegmentCount(c echo.Context) error {
	var (
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Check if cached count is requested
	useCache, _ := strconv.ParseBool(c.FormValue("cached"))

	count, err := a.core.GetSegmentCount(id, useCache)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{struct {
		Count int `json:"count"`
	}{Count: count}})
}

// PreviewSegmentConditions previews subscribers for ad-hoc conditions (without saving).
func (a *App) PreviewSegmentConditions(c echo.Context) error {
	var req struct {
		Conditions models.SegmentConditionGroup `json:"conditions"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	// Create a temporary segment with the conditions
	tempSeg := models.Segment{
		Conditions: req.Conditions,
	}

	pg := a.pg.NewFromURL(c.Request().URL.Query())
	orderBy := c.FormValue("order_by")
	order := c.FormValue("order")

	// Use GetSegmentSubscribersRaw which accepts conditions directly
	subs, total, err := a.core.GetSegmentSubscribersFromConditions(tempSeg.Conditions, orderBy, order, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	out := models.PageResults{
		Results: subs,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// ExportSegmentSubscribers exports all subscribers matching a segment's conditions as CSV.
func (a *App) ExportSegmentSubscribers(c echo.Context) error {
	var (
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Get the batched export iterator
	seg, exp, err := a.core.ExportSegmentSubscribers(id, a.cfg.DBBatchSize)
	if err != nil {
		return err
	}

	// Sanitize segment name for filename (remove special characters)
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	safeName := re.ReplaceAllString(strings.ReplaceAll(seg.Name, " ", "-"), "")
	if safeName == "" {
		safeName = "segment"
	}

	var (
		hdr = c.Response().Header()
		wr  = csv.NewWriter(c.Response())
	)

	hdr.Set(echo.HeaderContentType, echo.MIMEOctetStream)
	hdr.Set("Content-type", "text/csv")
	hdr.Set(echo.HeaderContentDisposition, "attachment; filename=\"segment-"+safeName+".csv\"")
	hdr.Set("Content-Transfer-Encoding", "binary")
	hdr.Set("Cache-Control", "no-cache")
	wr.Write([]string{"uuid", "email", "name", "attributes", "status", "created_at", "updated_at"})

loop:
	// Iterate in batches until there are no more subscribers to export
	for {
		out, err := exp()
		if err != nil {
			return err
		}
		if len(out) == 0 {
			break
		}

		for _, r := range out {
			if err = wr.Write([]string{r.UUID, r.Email, r.Name, r.Attribs, r.Status,
				r.CreatedAt.Time.String(), r.UpdatedAt.Time.String()}); err != nil {
				a.log.Printf("error streaming segment CSV export: %v", err)
				break loop
			}
		}

		// Flush CSV to stream after each batch
		wr.Flush()
	}

	return nil
}
