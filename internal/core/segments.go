package core

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

var (
	segmentQuerySortFields = []string{"name", "created_at", "updated_at"}
)

// SegmentQueryBuilder builds SQL WHERE clauses from segment conditions.
type SegmentQueryBuilder struct {
	params     []interface{}
	paramIndex int
}

// BuildWhereClause converts segment conditions to a SQL WHERE clause.
func (b *SegmentQueryBuilder) BuildWhereClause(conditions models.SegmentConditionGroup) (string, []interface{}, error) {
	b.params = []interface{}{}
	b.paramIndex = 1

	clause, err := b.buildGroup(conditions)
	if err != nil {
		return "", nil, err
	}

	if clause == "" {
		clause = "TRUE"
	}

	return clause, b.params, nil
}

func (b *SegmentQueryBuilder) buildGroup(group models.SegmentConditionGroup) (string, error) {
	if len(group.Conditions) == 0 {
		return "TRUE", nil
	}

	var clauses []string
	for _, cond := range group.Conditions {
		// Check if it's a nested group or a condition
		if nested, ok := cond.(map[string]interface{}); ok {
			if _, hasOperator := nested["operator"]; hasOperator {
				// Check if it has conditions array - it's a nested group
				if _, hasConditions := nested["conditions"]; hasConditions {
					var nestedGroup models.SegmentConditionGroup
					bytes, _ := json.Marshal(nested)
					if err := json.Unmarshal(bytes, &nestedGroup); err != nil {
						return "", err
					}

					clause, err := b.buildGroup(nestedGroup)
					if err != nil {
						return "", err
					}
					clauses = append(clauses, "("+clause+")")
				} else {
					// It's a condition
					clause, err := b.buildConditionFromMap(nested)
					if err != nil {
						return "", err
					}
					clauses = append(clauses, clause)
				}
			} else {
				// It's a condition
				clause, err := b.buildConditionFromMap(nested)
				if err != nil {
					return "", err
				}
				clauses = append(clauses, clause)
			}
		}
	}

	if len(clauses) == 0 {
		return "TRUE", nil
	}

	joiner := " AND "
	if strings.ToLower(group.Operator) == "or" {
		joiner = " OR "
	}

	return strings.Join(clauses, joiner), nil
}

func (b *SegmentQueryBuilder) buildConditionFromMap(m map[string]interface{}) (string, error) {
	field, _ := m["field"].(string)
	fieldType, _ := m["field_type"].(string)
	operator, _ := m["operator"].(string)
	value := m["value"]

	// Extract params if present
	var params map[string]any
	if p, ok := m["params"].(map[string]interface{}); ok {
		params = p
	}

	return b.buildCondition(field, fieldType, operator, value, params)
}

func (b *SegmentQueryBuilder) buildCondition(field, fieldType, operator string, value interface{}, params map[string]any) (string, error) {
	// Handle order-based fields specially (they require subqueries)
	if strings.HasPrefix(field, "orders.") {
		return b.buildOrderCondition(field, fieldType, operator, value)
	}

	sqlField := b.sanitizeField(field)

	switch fieldType {
	case "text":
		return b.buildTextCondition(sqlField, operator, value)
	case "number":
		return b.buildNumberCondition(sqlField, field, operator, value)
	case "date":
		return b.buildDateCondition(sqlField, operator, value)
	case "boolean":
		return b.buildBooleanCondition(sqlField, field, operator)
	case "array":
		return b.buildArrayCondition(field, operator, value)
	case "behavior":
		return b.buildBehavioralCondition(field, operator, value, params)
	default:
		return "", fmt.Errorf("unknown field type: %s", fieldType)
	}
}

func (b *SegmentQueryBuilder) sanitizeField(field string) string {
	// Handle attribs JSONB fields: attribs.key -> attribs->>'key'
	if strings.HasPrefix(field, "attribs.") {
		key := strings.TrimPrefix(field, "attribs.")
		// Support nested: attribs.user.name -> attribs->'user'->>'name'
		parts := strings.Split(key, ".")
		if len(parts) == 1 {
			return fmt.Sprintf("attribs->>'%s'", parts[0])
		}
		jsonPath := "attribs"
		for i := 0; i < len(parts)-1; i++ {
			jsonPath += fmt.Sprintf("->'%s'", parts[i])
		}
		jsonPath += fmt.Sprintf("->>'%s'", parts[len(parts)-1])
		return jsonPath
	}

	// Only allow specific fields
	allowedFields := map[string]string{
		"email":      "email",
		"name":       "name",
		"status":     "status",
		"created_at": "created_at",
		"updated_at": "updated_at",
		"uuid":       "uuid",
	}

	if mapped, ok := allowedFields[field]; ok {
		return mapped
	}

	return "NULL" // Safe fallback for unknown fields
}

func (b *SegmentQueryBuilder) buildTextCondition(sqlField, operator string, value interface{}) (string, error) {
	v := fmt.Sprintf("%v", value)

	switch operator {
	case "equals":
		b.params = append(b.params, v)
		return fmt.Sprintf("%s = $%d", sqlField, b.nextParam()), nil
	case "not_equals":
		b.params = append(b.params, v)
		return fmt.Sprintf("%s != $%d", sqlField, b.nextParam()), nil
	case "contains":
		b.params = append(b.params, "%"+v+"%")
		return fmt.Sprintf("%s ILIKE $%d", sqlField, b.nextParam()), nil
	case "not_contains":
		b.params = append(b.params, "%"+v+"%")
		return fmt.Sprintf("%s NOT ILIKE $%d", sqlField, b.nextParam()), nil
	case "starts_with":
		b.params = append(b.params, v+"%")
		return fmt.Sprintf("%s ILIKE $%d", sqlField, b.nextParam()), nil
	case "ends_with":
		b.params = append(b.params, "%"+v)
		return fmt.Sprintf("%s ILIKE $%d", sqlField, b.nextParam()), nil
	case "is_set":
		return fmt.Sprintf("(%s IS NOT NULL AND %s != '')", sqlField, sqlField), nil
	case "is_not_set":
		return fmt.Sprintf("(%s IS NULL OR %s = '')", sqlField, sqlField), nil
	default:
		return "", fmt.Errorf("unknown text operator: %s", operator)
	}
}

func (b *SegmentQueryBuilder) buildNumberCondition(sqlField, origField, operator string, value interface{}) (string, error) {
	// For JSONB fields, cast to numeric
	numField := sqlField
	if strings.HasPrefix(origField, "attribs.") {
		numField = fmt.Sprintf("(%s)::NUMERIC", sqlField)
	}

	switch operator {
	case "equals":
		b.params = append(b.params, value)
		return fmt.Sprintf("%s = $%d", numField, b.nextParam()), nil
	case "not_equals":
		b.params = append(b.params, value)
		return fmt.Sprintf("%s != $%d", numField, b.nextParam()), nil
	case "greater_than":
		b.params = append(b.params, value)
		return fmt.Sprintf("%s > $%d", numField, b.nextParam()), nil
	case "less_than":
		b.params = append(b.params, value)
		return fmt.Sprintf("%s < $%d", numField, b.nextParam()), nil
	case "at_least":
		b.params = append(b.params, value)
		return fmt.Sprintf("%s >= $%d", numField, b.nextParam()), nil
	case "at_most":
		b.params = append(b.params, value)
		return fmt.Sprintf("%s <= $%d", numField, b.nextParam()), nil
	default:
		return "", fmt.Errorf("unknown number operator: %s", operator)
	}
}

func (b *SegmentQueryBuilder) buildDateCondition(sqlField, operator string, value interface{}) (string, error) {
	switch operator {
	case "before":
		b.params = append(b.params, value)
		return fmt.Sprintf("%s < $%d::TIMESTAMP WITH TIME ZONE", sqlField, b.nextParam()), nil
	case "after":
		b.params = append(b.params, value)
		return fmt.Sprintf("%s > $%d::TIMESTAMP WITH TIME ZONE", sqlField, b.nextParam()), nil
	case "in_last_x_days":
		days := int(value.(float64))
		return fmt.Sprintf("%s > NOW() - INTERVAL '%d days'", sqlField, days), nil
	case "more_than_x_days_ago":
		days := int(value.(float64))
		return fmt.Sprintf("%s < NOW() - INTERVAL '%d days'", sqlField, days), nil
	case "on":
		b.params = append(b.params, value)
		return fmt.Sprintf("DATE(%s) = DATE($%d::TIMESTAMP)", sqlField, b.nextParam()), nil
	case "is_set":
		return fmt.Sprintf("%s IS NOT NULL", sqlField), nil
	case "is_not_set":
		return fmt.Sprintf("%s IS NULL", sqlField), nil

	// Enhanced date operators
	case "in_next_x_days":
		// Date is in the next X days (future)
		days := int(value.(float64))
		return fmt.Sprintf("(%s > NOW() AND %s < NOW() + INTERVAL '%d days')", sqlField, sqlField, days), nil

	case "between_dates":
		// Date is between start and end dates
		if dateRange, ok := value.(map[string]interface{}); ok {
			start, hasStart := dateRange["start"].(string)
			end, hasEnd := dateRange["end"].(string)
			if hasStart && hasEnd {
				b.params = append(b.params, start)
				startIdx := b.nextParam()
				b.params = append(b.params, end)
				endIdx := b.nextParam()
				return fmt.Sprintf("%s BETWEEN $%d::TIMESTAMP AND $%d::TIMESTAMP", sqlField, startIdx, endIdx), nil
			}
		}
		return "", fmt.Errorf("between_dates requires start and end dates")

	case "anniversary_is_today":
		// Same month and day as today (for birthdays, signup anniversaries, etc.)
		return fmt.Sprintf("(EXTRACT(MONTH FROM %s) = EXTRACT(MONTH FROM NOW()) AND EXTRACT(DAY FROM %s) = EXTRACT(DAY FROM NOW()))", sqlField, sqlField), nil

	case "day_of_week_is":
		// Day of week matches (0=Sunday, 6=Saturday)
		dayOfWeek := int(value.(float64))
		return fmt.Sprintf("EXTRACT(DOW FROM %s) = %d", sqlField, dayOfWeek), nil

	default:
		return "", fmt.Errorf("unknown date operator: %s", operator)
	}
}

func (b *SegmentQueryBuilder) buildBooleanCondition(sqlField, origField, operator string) (string, error) {
	// For JSONB fields
	if strings.HasPrefix(origField, "attribs.") {
		switch operator {
		case "is_true":
			return fmt.Sprintf("(%s)::BOOLEAN = TRUE", sqlField), nil
		case "is_false":
			return fmt.Sprintf("(%s)::BOOLEAN = FALSE", sqlField), nil
		}
	}
	return "", fmt.Errorf("boolean operator not supported for field: %s", origField)
}

func (b *SegmentQueryBuilder) buildArrayCondition(origField, operator string, value interface{}) (string, error) {
	// Array conditions work on JSONB array fields
	// attribs.tags -> attribs->'tags'
	if !strings.HasPrefix(origField, "attribs.") {
		return "", fmt.Errorf("array conditions only supported for attribs fields")
	}

	key := strings.TrimPrefix(origField, "attribs.")
	jsonField := fmt.Sprintf("attribs->'%s'", key)

	switch operator {
	case "includes_any":
		vals, ok := value.([]interface{})
		if !ok {
			return "", fmt.Errorf("includes_any requires an array value")
		}
		placeholders := make([]string, len(vals))
		for i, v := range vals {
			b.params = append(b.params, fmt.Sprintf("%v", v))
			placeholders[i] = fmt.Sprintf("$%d", b.nextParam())
		}
		return fmt.Sprintf("%s ?| ARRAY[%s]", jsonField, strings.Join(placeholders, ",")), nil
	case "includes_all":
		vals, ok := value.([]interface{})
		if !ok {
			return "", fmt.Errorf("includes_all requires an array value")
		}
		placeholders := make([]string, len(vals))
		for i, v := range vals {
			b.params = append(b.params, fmt.Sprintf("%v", v))
			placeholders[i] = fmt.Sprintf("$%d", b.nextParam())
		}
		return fmt.Sprintf("%s ?& ARRAY[%s]", jsonField, strings.Join(placeholders, ",")), nil
	case "is_empty":
		return fmt.Sprintf("(jsonb_array_length(%s) = 0 OR %s IS NULL)", jsonField, jsonField), nil
	case "has_at_least":
		count := int(value.(float64))
		return fmt.Sprintf("jsonb_array_length(%s) >= %d", jsonField, count), nil
	default:
		return "", fmt.Errorf("unknown array operator: %s", operator)
	}
}

// buildBehavioralCondition builds SQL for behavioral conditions (email opens, link clicks, etc.)
func (b *SegmentQueryBuilder) buildBehavioralCondition(field, operator string, value interface{}, params map[string]any) (string, error) {
	switch field {
	case "behavior.email_opens":
		return b.buildEmailOpensCondition(operator, value, params)
	case "behavior.link_clicks":
		return b.buildLinkClicksCondition(operator, value, params)
	case "behavior.opened_campaign":
		return b.buildOpenedCampaignCondition(operator, value)
	case "behavior.clicked_campaign":
		return b.buildClickedCampaignCondition(operator, value)
	default:
		return "", fmt.Errorf("unknown behavioral field: %s", field)
	}
}

// buildEmailOpensCondition builds SQL for email open count conditions.
// Uses campaign_views table with optional time range filtering.
func (b *SegmentQueryBuilder) buildEmailOpensCondition(operator string, value interface{}, params map[string]any) (string, error) {
	// Get time range from params (0 = all time)
	timeRangeDays := 0
	if params != nil {
		if tr, ok := params["time_range_days"].(float64); ok {
			timeRangeDays = int(tr)
		}
	}

	// Build time filter clause
	timeFilter := ""
	if timeRangeDays > 0 {
		timeFilter = fmt.Sprintf(" AND created_at > NOW() - INTERVAL '%d days'", timeRangeDays)
	}

	switch operator {
	case "zero":
		// Never opened any email
		return fmt.Sprintf(`subscribers.id NOT IN (
			SELECT DISTINCT subscriber_id FROM campaign_views
			WHERE subscriber_id IS NOT NULL%s
		)`, timeFilter), nil

	case "greater_than_zero":
		// Opened at least one email
		return fmt.Sprintf(`subscribers.id IN (
			SELECT DISTINCT subscriber_id FROM campaign_views
			WHERE subscriber_id IS NOT NULL%s
		)`, timeFilter), nil

	case "at_least":
		count := 1
		if v, ok := value.(float64); ok {
			count = int(v)
		}
		return fmt.Sprintf(`subscribers.id IN (
			SELECT subscriber_id FROM campaign_views
			WHERE subscriber_id IS NOT NULL%s
			GROUP BY subscriber_id
			HAVING COUNT(*) >= %d
		)`, timeFilter, count), nil

	case "at_most":
		count := 0
		if v, ok := value.(float64); ok {
			count = int(v)
		}
		// This includes subscribers who never opened (0 opens) and those with <= count opens
		return fmt.Sprintf(`(subscribers.id NOT IN (
			SELECT DISTINCT subscriber_id FROM campaign_views
			WHERE subscriber_id IS NOT NULL%s
		) OR subscribers.id IN (
			SELECT subscriber_id FROM campaign_views
			WHERE subscriber_id IS NOT NULL%s
			GROUP BY subscriber_id
			HAVING COUNT(*) <= %d
		))`, timeFilter, timeFilter, count), nil

	case "equals":
		count := 0
		if v, ok := value.(float64); ok {
			count = int(v)
		}
		if count == 0 {
			// Equals zero means never opened
			return fmt.Sprintf(`subscribers.id NOT IN (
				SELECT DISTINCT subscriber_id FROM campaign_views
				WHERE subscriber_id IS NOT NULL%s
			)`, timeFilter), nil
		}
		return fmt.Sprintf(`subscribers.id IN (
			SELECT subscriber_id FROM campaign_views
			WHERE subscriber_id IS NOT NULL%s
			GROUP BY subscriber_id
			HAVING COUNT(*) = %d
		)`, timeFilter, count), nil

	default:
		return "", fmt.Errorf("unknown email opens operator: %s", operator)
	}
}

// buildLinkClicksCondition builds SQL for link click count conditions.
// Uses link_clicks table with optional time range filtering.
func (b *SegmentQueryBuilder) buildLinkClicksCondition(operator string, value interface{}, params map[string]any) (string, error) {
	// Get time range from params (0 = all time)
	timeRangeDays := 0
	if params != nil {
		if tr, ok := params["time_range_days"].(float64); ok {
			timeRangeDays = int(tr)
		}
	}

	// Build time filter clause
	timeFilter := ""
	if timeRangeDays > 0 {
		timeFilter = fmt.Sprintf(" AND created_at > NOW() - INTERVAL '%d days'", timeRangeDays)
	}

	switch operator {
	case "zero":
		// Never clicked any link
		return fmt.Sprintf(`subscribers.id NOT IN (
			SELECT DISTINCT subscriber_id FROM link_clicks
			WHERE subscriber_id IS NOT NULL%s
		)`, timeFilter), nil

	case "greater_than_zero":
		// Clicked at least one link
		return fmt.Sprintf(`subscribers.id IN (
			SELECT DISTINCT subscriber_id FROM link_clicks
			WHERE subscriber_id IS NOT NULL%s
		)`, timeFilter), nil

	case "at_least":
		count := 1
		if v, ok := value.(float64); ok {
			count = int(v)
		}
		return fmt.Sprintf(`subscribers.id IN (
			SELECT subscriber_id FROM link_clicks
			WHERE subscriber_id IS NOT NULL%s
			GROUP BY subscriber_id
			HAVING COUNT(*) >= %d
		)`, timeFilter, count), nil

	case "at_most":
		count := 0
		if v, ok := value.(float64); ok {
			count = int(v)
		}
		return fmt.Sprintf(`(subscribers.id NOT IN (
			SELECT DISTINCT subscriber_id FROM link_clicks
			WHERE subscriber_id IS NOT NULL%s
		) OR subscribers.id IN (
			SELECT subscriber_id FROM link_clicks
			WHERE subscriber_id IS NOT NULL%s
			GROUP BY subscriber_id
			HAVING COUNT(*) <= %d
		))`, timeFilter, timeFilter, count), nil

	case "equals":
		count := 0
		if v, ok := value.(float64); ok {
			count = int(v)
		}
		if count == 0 {
			return fmt.Sprintf(`subscribers.id NOT IN (
				SELECT DISTINCT subscriber_id FROM link_clicks
				WHERE subscriber_id IS NOT NULL%s
			)`, timeFilter), nil
		}
		return fmt.Sprintf(`subscribers.id IN (
			SELECT subscriber_id FROM link_clicks
			WHERE subscriber_id IS NOT NULL%s
			GROUP BY subscriber_id
			HAVING COUNT(*) = %d
		)`, timeFilter, count), nil

	default:
		return "", fmt.Errorf("unknown link clicks operator: %s", operator)
	}
}

// buildOpenedCampaignCondition builds SQL for "opened specific campaign" conditions.
func (b *SegmentQueryBuilder) buildOpenedCampaignCondition(operator string, value interface{}) (string, error) {
	campaignID := 0
	if v, ok := value.(float64); ok {
		campaignID = int(v)
	}

	if campaignID == 0 {
		return "", fmt.Errorf("opened_campaign requires a valid campaign_id")
	}

	switch operator {
	case "is_true":
		return fmt.Sprintf(`subscribers.id IN (
			SELECT subscriber_id FROM campaign_views
			WHERE campaign_id = %d AND subscriber_id IS NOT NULL
		)`, campaignID), nil

	case "is_false":
		return fmt.Sprintf(`subscribers.id NOT IN (
			SELECT subscriber_id FROM campaign_views
			WHERE campaign_id = %d AND subscriber_id IS NOT NULL
		)`, campaignID), nil

	default:
		return "", fmt.Errorf("unknown opened_campaign operator: %s", operator)
	}
}

// buildClickedCampaignCondition builds SQL for "clicked link in specific campaign" conditions.
func (b *SegmentQueryBuilder) buildClickedCampaignCondition(operator string, value interface{}) (string, error) {
	campaignID := 0
	if v, ok := value.(float64); ok {
		campaignID = int(v)
	}

	if campaignID == 0 {
		return "", fmt.Errorf("clicked_campaign requires a valid campaign_id")
	}

	switch operator {
	case "is_true":
		return fmt.Sprintf(`subscribers.id IN (
			SELECT subscriber_id FROM link_clicks
			WHERE campaign_id = %d AND subscriber_id IS NOT NULL
		)`, campaignID), nil

	case "is_false":
		return fmt.Sprintf(`subscribers.id NOT IN (
			SELECT subscriber_id FROM link_clicks
			WHERE campaign_id = %d AND subscriber_id IS NOT NULL
		)`, campaignID), nil

	default:
		return "", fmt.Errorf("unknown clicked_campaign operator: %s", operator)
	}
}

// buildOrderCondition builds SQL for order history conditions.
// These query the shopify_orders and shopify_order_line_items tables.
func (b *SegmentQueryBuilder) buildOrderCondition(field, fieldType, operator string, value interface{}) (string, error) {
	switch field {
	case "orders.product_title":
		// Search by product title in order line items (text search)
		return b.buildProductTitleCondition(operator, value)
	case "orders.product_type":
		// Search by product type in order line items (text search)
		return b.buildProductTypeCondition(operator, value)
	case "orders.total_value":
		// Total order value across all orders (number comparison)
		return b.buildTotalOrderValueCondition(operator, value)
	case "orders.order_count":
		// Count of orders in database (number comparison)
		return b.buildOrderCountCondition(operator, value)
	default:
		return "", fmt.Errorf("unknown order field: %s", field)
	}
}

// buildProductTitleCondition builds SQL to find subscribers who purchased products with matching titles.
func (b *SegmentQueryBuilder) buildProductTitleCondition(operator string, value interface{}) (string, error) {
	v := fmt.Sprintf("%v", value)

	switch operator {
	case "contains":
		b.params = append(b.params, "%"+v+"%")
		return fmt.Sprintf(`subscribers.id IN (
			SELECT so.subscriber_id
			FROM shopify_orders so
			JOIN shopify_order_line_items li ON li.order_id = so.id
			WHERE li.product_title ILIKE $%d
		)`, b.nextParam()), nil
	case "equals":
		b.params = append(b.params, v)
		return fmt.Sprintf(`subscribers.id IN (
			SELECT so.subscriber_id
			FROM shopify_orders so
			JOIN shopify_order_line_items li ON li.order_id = so.id
			WHERE li.product_title = $%d
		)`, b.nextParam()), nil
	case "not_contains":
		b.params = append(b.params, "%"+v+"%")
		return fmt.Sprintf(`subscribers.id NOT IN (
			SELECT so.subscriber_id
			FROM shopify_orders so
			JOIN shopify_order_line_items li ON li.order_id = so.id
			WHERE li.product_title ILIKE $%d
		)`, b.nextParam()), nil
	case "starts_with":
		b.params = append(b.params, v+"%")
		return fmt.Sprintf(`subscribers.id IN (
			SELECT so.subscriber_id
			FROM shopify_orders so
			JOIN shopify_order_line_items li ON li.order_id = so.id
			WHERE li.product_title ILIKE $%d
		)`, b.nextParam()), nil
	case "is_set":
		// Has purchased any product
		return `subscribers.id IN (
			SELECT so.subscriber_id
			FROM shopify_orders so
			JOIN shopify_order_line_items li ON li.order_id = so.id
			WHERE li.product_title IS NOT NULL
		)`, nil
	case "is_not_set":
		// Has not purchased any product (no orders or no line items)
		return `subscribers.id NOT IN (
			SELECT so.subscriber_id
			FROM shopify_orders so
			JOIN shopify_order_line_items li ON li.order_id = so.id
			WHERE li.product_title IS NOT NULL
		)`, nil
	default:
		return "", fmt.Errorf("unknown product_title operator: %s", operator)
	}
}

// buildProductTypeCondition builds SQL to find subscribers who purchased products with matching types.
func (b *SegmentQueryBuilder) buildProductTypeCondition(operator string, value interface{}) (string, error) {
	v := fmt.Sprintf("%v", value)

	switch operator {
	case "contains":
		b.params = append(b.params, "%"+v+"%")
		return fmt.Sprintf(`subscribers.id IN (
			SELECT so.subscriber_id
			FROM shopify_orders so
			JOIN shopify_order_line_items li ON li.order_id = so.id
			WHERE li.product_type ILIKE $%d
		)`, b.nextParam()), nil
	case "equals":
		b.params = append(b.params, v)
		return fmt.Sprintf(`subscribers.id IN (
			SELECT so.subscriber_id
			FROM shopify_orders so
			JOIN shopify_order_line_items li ON li.order_id = so.id
			WHERE li.product_type = $%d
		)`, b.nextParam()), nil
	case "not_contains":
		b.params = append(b.params, "%"+v+"%")
		return fmt.Sprintf(`subscribers.id NOT IN (
			SELECT so.subscriber_id
			FROM shopify_orders so
			JOIN shopify_order_line_items li ON li.order_id = so.id
			WHERE li.product_type ILIKE $%d
		)`, b.nextParam()), nil
	default:
		return "", fmt.Errorf("unknown product_type operator: %s", operator)
	}
}

// buildTotalOrderValueCondition builds SQL for total order value comparisons.
func (b *SegmentQueryBuilder) buildTotalOrderValueCondition(operator string, value interface{}) (string, error) {
	switch operator {
	case "equals":
		b.params = append(b.params, value)
		return fmt.Sprintf(`subscribers.id IN (
			SELECT subscriber_id
			FROM shopify_orders
			GROUP BY subscriber_id
			HAVING COALESCE(SUM(total_price), 0) = $%d
		)`, b.nextParam()), nil
	case "not_equals":
		b.params = append(b.params, value)
		return fmt.Sprintf(`subscribers.id NOT IN (
			SELECT subscriber_id
			FROM shopify_orders
			GROUP BY subscriber_id
			HAVING COALESCE(SUM(total_price), 0) = $%d
		)`, b.nextParam()), nil
	case "greater_than":
		b.params = append(b.params, value)
		return fmt.Sprintf(`subscribers.id IN (
			SELECT subscriber_id
			FROM shopify_orders
			GROUP BY subscriber_id
			HAVING COALESCE(SUM(total_price), 0) > $%d
		)`, b.nextParam()), nil
	case "less_than":
		b.params = append(b.params, value)
		return fmt.Sprintf(`subscribers.id IN (
			SELECT subscriber_id
			FROM shopify_orders
			GROUP BY subscriber_id
			HAVING COALESCE(SUM(total_price), 0) < $%d
		)`, b.nextParam()), nil
	case "at_least":
		b.params = append(b.params, value)
		return fmt.Sprintf(`subscribers.id IN (
			SELECT subscriber_id
			FROM shopify_orders
			GROUP BY subscriber_id
			HAVING COALESCE(SUM(total_price), 0) >= $%d
		)`, b.nextParam()), nil
	case "at_most":
		b.params = append(b.params, value)
		return fmt.Sprintf(`subscribers.id IN (
			SELECT subscriber_id
			FROM shopify_orders
			GROUP BY subscriber_id
			HAVING COALESCE(SUM(total_price), 0) <= $%d
		)`, b.nextParam()), nil
	default:
		return "", fmt.Errorf("unknown total_value operator: %s", operator)
	}
}

// buildOrderCountCondition builds SQL for order count comparisons.
func (b *SegmentQueryBuilder) buildOrderCountCondition(operator string, value interface{}) (string, error) {
	switch operator {
	case "equals":
		count := 0
		if v, ok := value.(float64); ok {
			count = int(v)
		}
		if count == 0 {
			// No orders
			return `subscribers.id NOT IN (
				SELECT DISTINCT subscriber_id FROM shopify_orders
			)`, nil
		}
		return fmt.Sprintf(`subscribers.id IN (
			SELECT subscriber_id
			FROM shopify_orders
			GROUP BY subscriber_id
			HAVING COUNT(*) = %d
		)`, count), nil
	case "not_equals":
		count := 0
		if v, ok := value.(float64); ok {
			count = int(v)
		}
		return fmt.Sprintf(`subscribers.id NOT IN (
			SELECT subscriber_id
			FROM shopify_orders
			GROUP BY subscriber_id
			HAVING COUNT(*) = %d
		)`, count), nil
	case "greater_than":
		count := 0
		if v, ok := value.(float64); ok {
			count = int(v)
		}
		return fmt.Sprintf(`subscribers.id IN (
			SELECT subscriber_id
			FROM shopify_orders
			GROUP BY subscriber_id
			HAVING COUNT(*) > %d
		)`, count), nil
	case "less_than":
		count := 0
		if v, ok := value.(float64); ok {
			count = int(v)
		}
		return fmt.Sprintf(`subscribers.id IN (
			SELECT subscriber_id
			FROM shopify_orders
			GROUP BY subscriber_id
			HAVING COUNT(*) < %d
		)`, count), nil
	case "at_least":
		count := 1
		if v, ok := value.(float64); ok {
			count = int(v)
		}
		return fmt.Sprintf(`subscribers.id IN (
			SELECT subscriber_id
			FROM shopify_orders
			GROUP BY subscriber_id
			HAVING COUNT(*) >= %d
		)`, count), nil
	case "at_most":
		count := 0
		if v, ok := value.(float64); ok {
			count = int(v)
		}
		return fmt.Sprintf(`(subscribers.id NOT IN (
			SELECT DISTINCT subscriber_id FROM shopify_orders
		) OR subscribers.id IN (
			SELECT subscriber_id
			FROM shopify_orders
			GROUP BY subscriber_id
			HAVING COUNT(*) <= %d
		))`, count), nil
	default:
		return "", fmt.Errorf("unknown order_count operator: %s", operator)
	}
}

func (b *SegmentQueryBuilder) nextParam() int {
	idx := b.paramIndex
	b.paramIndex++
	return idx
}

// CreateSegment creates a new segment.
func (c *Core) CreateSegment(seg models.Segment) (models.Segment, error) {
	uu, err := uuid.NewV4()
	if err != nil {
		c.log.Printf("error generating UUID: %v", err)
		return models.Segment{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}

	// Marshal conditions to JSON
	conditionsJSON, err := json.Marshal(seg.Conditions)
	if err != nil {
		c.log.Printf("error marshaling segment conditions: %v", err)
		return models.Segment{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.invalidData"))
	}

	// Validate conditions by building WHERE clause
	builder := &SegmentQueryBuilder{}
	if _, _, err := builder.BuildWhereClause(seg.Conditions); err != nil {
		return models.Segment{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("segments.invalidConditions", "error", err.Error()))
	}

	var newID int
	if err := c.q.CreateSegment.Get(&newID, uu.String(), seg.Name, seg.Description,
		conditionsJSON, pq.StringArray(normalizeTags(seg.Tags))); err != nil {
		c.log.Printf("error creating segment: %v", err)
		return models.Segment{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.segment}", "error", pqErrMsg(err)))
	}

	return c.GetSegment(newID, "")
}

// GetSegment retrieves a segment by ID or UUID.
func (c *Core) GetSegment(id int, uuid string) (models.Segment, error) {
	var uu interface{}
	if uuid != "" {
		uu = uuid
	}

	var out models.Segment
	if err := c.q.GetSegment.Get(&out, id, uu); err != nil {
		if err == sql.ErrNoRows {
			return models.Segment{}, echo.NewHTTPError(http.StatusBadRequest,
				c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.segment}"))
		}
		c.log.Printf("error fetching segment: %v", err)
		return models.Segment{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.segment}", "error", pqErrMsg(err)))
	}

	if out.Tags == nil {
		out.Tags = []string{}
	}

	return out, nil
}

// QuerySegments retrieves paginated segments.
func (c *Core) QuerySegments(searchStr, orderBy, order string, offset, limit int) (models.Segments, int, error) {
	queryStr, stmt := makeSearchQuery(searchStr, orderBy, order, c.q.QuerySegments, segmentQuerySortFields)

	var out models.Segments
	if err := c.db.Select(&out, stmt, 0, "", queryStr, offset, limit); err != nil {
		c.log.Printf("error fetching segments: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.segments}", "error", pqErrMsg(err)))
	}

	total := 0
	if len(out) > 0 {
		total = out[0].Total

		// Replace null tags.
		for i := range out {
			if out[i].Tags == nil {
				out[i].Tags = []string{}
			}
		}
	}

	return out, total, nil
}

// UpdateSegment updates a segment.
func (c *Core) UpdateSegment(id int, seg models.Segment) (models.Segment, error) {
	// Marshal conditions to JSON
	conditionsJSON, err := json.Marshal(seg.Conditions)
	if err != nil {
		c.log.Printf("error marshaling segment conditions: %v", err)
		return models.Segment{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.invalidData"))
	}

	// Validate conditions by building WHERE clause
	builder := &SegmentQueryBuilder{}
	if _, _, err := builder.BuildWhereClause(seg.Conditions); err != nil {
		return models.Segment{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("segments.invalidConditions", "error", err.Error()))
	}

	res, err := c.q.UpdateSegment.Exec(id, seg.Name, seg.Description, conditionsJSON, pq.StringArray(normalizeTags(seg.Tags)))
	if err != nil {
		c.log.Printf("error updating segment: %v", err)
		return models.Segment{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.segment}", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return models.Segment{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.segment}"))
	}

	return c.GetSegment(id, "")
}

// DeleteSegment deletes a segment.
func (c *Core) DeleteSegment(id int) error {
	if _, err := c.q.DeleteSegment.Exec(id); err != nil {
		c.log.Printf("error deleting segment: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.segment}", "error", pqErrMsg(err)))
	}

	return nil
}

// GetSegmentSubscribers retrieves subscribers matching segment conditions.
func (c *Core) GetSegmentSubscribers(segmentID int, orderBy, order string, offset, limit int) (models.Subscribers, int, error) {
	seg, err := c.GetSegment(segmentID, "")
	if err != nil {
		return nil, 0, err
	}

	// Build WHERE clause from conditions
	builder := &SegmentQueryBuilder{}
	whereClause, params, err := builder.BuildWhereClause(seg.Conditions)
	if err != nil {
		return nil, 0, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("segments.invalidConditions", "error", err.Error()))
	}

	// Count query - no status filter; users can add status conditions explicitly
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM subscribers WHERE (%s)", whereClause)
	var total int
	if err := c.db.Get(&total, countQuery, params...); err != nil {
		c.log.Printf("error counting segment subscribers: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	if total == 0 {
		return models.Subscribers{}, 0, nil
	}

	// Validate sort field
	sortField := "subscribers.id"
	if orderBy != "" {
		for _, f := range subQuerySortFields {
			if f == orderBy {
				sortField = "subscribers." + orderBy
				break
			}
		}
	}

	sortOrder := "DESC"
	if order == SortAsc {
		sortOrder = "ASC"
	}

	// Fetch subscribers - no status filter; users can add status conditions explicitly
	selectQuery := fmt.Sprintf(`SELECT subscribers.* FROM subscribers
		WHERE (%s)
		ORDER BY %s %s
		OFFSET $%d LIMIT $%d`,
		whereClause, sortField, sortOrder, len(params)+1, len(params)+2)

	params = append(params, offset, limit)

	var out models.Subscribers
	if err := c.db.Select(&out, selectQuery, params...); err != nil {
		c.log.Printf("error fetching segment subscribers: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	// Load lists
	if err := out.LoadLists(c.q.GetSubscriberListsLazy); err != nil {
		c.log.Printf("error loading subscriber lists: %v", err)
	}

	return out, total, nil
}

// GetSegmentSubscribersFromConditions retrieves subscribers matching raw conditions (for ad-hoc previews).
func (c *Core) GetSegmentSubscribersFromConditions(conditions models.SegmentConditionGroup, orderBy, order string, offset, limit int) (models.Subscribers, int, error) {
	// Build WHERE clause from conditions
	builder := &SegmentQueryBuilder{}
	whereClause, params, err := builder.BuildWhereClause(conditions)
	if err != nil {
		return nil, 0, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("segments.invalidConditions", "error", err.Error()))
	}

	// Count query - no status filter; users can add status conditions explicitly
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM subscribers WHERE (%s)", whereClause)
	var total int
	if err := c.db.Get(&total, countQuery, params...); err != nil {
		c.log.Printf("error counting segment subscribers: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	if total == 0 {
		return models.Subscribers{}, 0, nil
	}

	// Validate sort field
	sortField := "subscribers.id"
	if orderBy != "" {
		for _, f := range subQuerySortFields {
			if f == orderBy {
				sortField = "subscribers." + orderBy
				break
			}
		}
	}

	sortOrder := "DESC"
	if order == SortAsc {
		sortOrder = "ASC"
	}

	// Fetch subscribers - no status filter; users can add status conditions explicitly
	selectQuery := fmt.Sprintf(`SELECT subscribers.* FROM subscribers
		WHERE (%s)
		ORDER BY %s %s
		OFFSET $%d LIMIT $%d`,
		whereClause, sortField, sortOrder, len(params)+1, len(params)+2)

	params = append(params, offset, limit)

	var out models.Subscribers
	if err := c.db.Select(&out, selectQuery, params...); err != nil {
		c.log.Printf("error fetching segment subscribers: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	// Load lists
	if err := out.LoadLists(c.q.GetSubscriberListsLazy); err != nil {
		c.log.Printf("error loading subscriber lists: %v", err)
	}

	return out, total, nil
}

// GetSegmentCount returns the subscriber count for a segment (with optional caching).
func (c *Core) GetSegmentCount(segmentID int, useCache bool) (int, error) {
	seg, err := c.GetSegment(segmentID, "")
	if err != nil {
		return 0, err
	}

	// Return cached count if valid and requested
	if useCache && seg.CachedCount.Valid && seg.CachedAt.Valid {
		// Consider cache valid for 1 hour
		if time.Since(seg.CachedAt.Time) < time.Hour {
			return seg.CachedCount.Int, nil
		}
	}

	// Build and execute count query
	builder := &SegmentQueryBuilder{}
	whereClause, params, err := builder.BuildWhereClause(seg.Conditions)
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("segments.invalidConditions", "error", err.Error()))
	}

	// No status filter; users can add status conditions explicitly
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM subscribers WHERE (%s)", whereClause)
	var count int
	if err := c.db.Get(&count, countQuery, params...); err != nil {
		c.log.Printf("error counting segment subscribers: %v", err)
		return 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.segment}", "error", pqErrMsg(err)))
	}

	// Update cache
	if _, err := c.q.UpdateSegmentCache.Exec(segmentID, count); err != nil {
		c.log.Printf("error updating segment cache: %v", err)
	}

	return count, nil
}
