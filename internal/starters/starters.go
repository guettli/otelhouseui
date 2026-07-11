// Package starters seeds the SQLite store with a small library of saved
// queries on first boot. In the v1 vertical slice this is one metrics query
// that exercises the full path (SQL → grid → auto time-series chart).
package starters

import (
	"context"
	"errors"

	"github.com/guettli/otelhouseview/internal/store"
)

// All is the canonical list of built-in saved queries.
var All = []store.SavedQuery{
	{
		Name: "service_request_rate_1m",
		Description: "1-minute request rate by service from otel_metrics_sum, gap-filled with WITH FILL. " +
			"Result shape (DateTime, String, Float64) triggers the auto grouped-line chart.",
		DefaultViz: "line",
		Params: []store.Param{
			{Name: "metric", Type: "String", Label: "Metric name", Widget: "text", Default: "http.server.request.duration"},
			{Name: "from", Type: "DateTime", Label: "From (UTC)", Widget: "datetime", Default: "2026-07-10 00:00:00"},
			{Name: "to", Type: "DateTime", Label: "To (UTC)", Widget: "datetime", Default: "2026-07-10 01:00:00"},
		},
		SQLTemplate: `SELECT
    toStartOfInterval(TimeUnix, INTERVAL 1 MINUTE) AS t,
    ResourceAttributes['service.name']             AS service,
    sum(Value)                                     AS v
FROM otel_metrics_sum
WHERE MetricName = {metric:String}
  AND TimeUnix BETWEEN {from:DateTime} AND {to:DateTime}
GROUP BY t, service
ORDER BY t
WITH FILL FROM toStartOfInterval({from:DateTime}, INTERVAL 1 MINUTE)
             TO toStartOfInterval({to:DateTime},   INTERVAL 1 MINUTE)
             STEP INTERVAL 1 MINUTE`,
	},
}

// Seed inserts any built-in queries that are not already present.
// It matches on `name` so it is safe to call on every boot.
func Seed(ctx context.Context, s *store.Store) error {
	for _, q := range All {
		if _, err := s.GetByName(ctx, q.Name); err == nil {
			continue
		} else if !errors.Is(err, store.ErrNotFound) {
			return err
		}
		q := q
		if err := s.Insert(ctx, &q); err != nil {
			// A concurrent boot may have inserted it first; treat as OK.
			if errors.Is(err, store.ErrDuplicateName) {
				continue
			}
			return err
		}
	}
	return nil
}
