# SQL Lessons Learned - PostgreSQL Aggregate Functions

## Problem: GROUP BY and Aggregate Function Requirements

When writing SQL queries with aggregate functions in PostgreSQL, you MUST follow this rule:

**Every column in the SELECT clause must be either:**
1. In the GROUP BY clause, OR
2. Wrapped in an aggregate function (MAX, MIN, SUM, AVG, COUNT, etc.)

## Example: The Bug We Fixed

### ❌ INCORRECT - Causes Error
```sql
WITH purchase_data AS (
    SELECT
        COUNT(*) AS total_orders,
        COALESCE(SUM(total_price), 0) AS total_revenue
    FROM purchase_attributions
    WHERE created_at >= NOW() - INTERVAL '30 days'
)
SELECT
    COALESCE(AVG(views), 0) AS avg_views,
    pd.total_orders,              -- ERROR: Not in GROUP BY or aggregate
    pd.total_revenue              -- ERROR: Not in GROUP BY or aggregate
FROM recent_campaigns
CROSS JOIN purchase_data pd;
```

**Error Message:**
```
pq: column "pd.total_orders" must appear in the GROUP BY clause or be used in an aggregate function
```

### ✅ CORRECT - Using MAX()
```sql
WITH purchase_data AS (
    SELECT
        COUNT(*) AS total_orders,
        COALESCE(SUM(total_price), 0) AS total_revenue
    FROM purchase_attributions
    WHERE created_at >= NOW() - INTERVAL '30 days'
)
SELECT
    COALESCE(AVG(views), 0) AS avg_views,
    MAX(pd.total_orders) AS total_orders,      -- Wrapped in MAX()
    MAX(pd.total_revenue) AS total_revenue     -- Wrapped in MAX()
FROM recent_campaigns
CROSS JOIN purchase_data pd;
```

**Why MAX() works here:**
- The `purchase_data` CTE returns exactly ONE row
- MAX() on a single value just returns that value
- This satisfies PostgreSQL's requirement without changing the result

## Problem 2: NULL Values from Aggregates

When aggregate functions operate on empty sets, they return NULL:

### ❌ INCORRECT - Can't Scan NULL into int
```sql
SELECT
    MAX(pd.total_orders) AS total_orders  -- Returns NULL if no rows
FROM recent_campaigns
CROSS JOIN purchase_data pd;
```

**Error when scanning into Go struct:**
```
Scan error on column index 3, name "total_orders": converting NULL to int is unsupported
```

### ✅ CORRECT - Use COALESCE
```sql
SELECT
    COALESCE(MAX(pd.total_orders), 0) AS total_orders,
    COALESCE(MAX(pd.total_revenue), 0) AS total_revenue
FROM recent_campaigns
CROSS JOIN purchase_data pd;
```

**Key Points:**
- Always use COALESCE when aggregating potentially empty data
- Especially important with CROSS JOIN where the joined table might be empty
- Default value (0) ensures type compatibility with Go struct fields

## General Rules for Safe Aggregates

1. **Always COALESCE aggregate results that might be NULL**
   ```sql
   COALESCE(SUM(amount), 0)
   COALESCE(AVG(score), 0.0)
   COALESCE(MAX(id), 0)
   ```

2. **When using CROSS JOIN with aggregates, wrap joined columns**
   ```sql
   FROM table1
   CROSS JOIN (SELECT COUNT(*) as cnt FROM table2) t2
   -- Then use MAX(t2.cnt) or add t2.cnt to GROUP BY
   ```

3. **Test with empty data sets**
   - Create test scenarios with no matching rows
   - Verify NULL handling in both SQL and application code

4. **Use CTEs for complex aggregates**
   ```sql
   WITH aggregated_data AS (
       SELECT metric1, metric2
       FROM source
       GROUP BY ...
   )
   SELECT COALESCE(MAX(metric1), 0) FROM aggregated_data
   ```

## Related PostgreSQL Documentation

- [Aggregate Functions](https://www.postgresql.org/docs/current/functions-aggregate.html)
- [GROUP BY Clause](https://www.postgresql.org/docs/current/queries-group.html)
- [COALESCE Function](https://www.postgresql.org/docs/current/functions-conditional.html#FUNCTIONS-COALESCE-NVL-IFNULL)
