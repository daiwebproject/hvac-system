package repository

import (
	"hvac-system/internal/core"
	"log"

	"github.com/pocketbase/dbx"
	pbCore "github.com/pocketbase/pocketbase/core"
)

type PBAnalyticsRepo struct {
	app pbCore.App
}

func NewAnalyticsRepo(app pbCore.App) core.AnalyticsRepository {
	return &PBAnalyticsRepo{app: app}
}

func (r *PBAnalyticsRepo) GetDailyRevenue(start, end string) ([]core.RevenueStat, error) {
	// Single query using GROUP BY on SQLite date function
	type queryResult struct {
		Day   string  `db:"day"`
		Total float64 `db:"total"`
	}

	var results []queryResult

	err := r.app.DB().Select(
		"substr(created, 1, 10) as day",
		"SUM(total_amount) as total",
	).
		From("invoices").
		Where(dbx.NewExp("status = 'paid' AND created >= {:start} AND created <= {:end}", dbx.Params{
			"start": start,
			"end":   end,
		})).
		GroupBy("day").
		All(&results)

	if err != nil {
		log.Printf("Error fetching revenue stats: %v", err)
		return nil, err
	}

	stats := make([]core.RevenueStat, len(results))
	for i, res := range results {
		stats[i] = core.RevenueStat{
			Date:   res.Day,
			Amount: res.Total,
		}
	}

	return stats, nil
}

func (r *PBAnalyticsRepo) GetTopTechnicians(limit int) ([]core.TechPerformance, error) {
	var results []core.TechPerformance

	query := r.app.DB().Select(
		"t.id as technician_id",
		"t.name as technician_name",
		"COUNT(b.id) as completed_jobs",
	).
		From("bookings b").
		LeftJoin("technicians t", dbx.NewExp("b.technician_id = t.id")).
		Where(dbx.HashExp{"b.job_status": "completed"}).
		GroupBy("t.id", "t.name").
		OrderBy("completed_jobs DESC").
		Limit(int64(limit))

	err := query.All(&results)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *PBAnalyticsRepo) GetTotalRevenue() (float64, error) {
	var result struct {
		Total float64 `db:"total"`
	}
	err := r.app.DB().Select("SUM(total_amount) as total").
		From("invoices").
		Where(dbx.HashExp{"status": "paid"}).
		One(&result)

	if err != nil {
		return 0, err // Return 0 if no records or error
	}
	return result.Total, nil
}

func (r *PBAnalyticsRepo) CountBookings(filter string) (int64, error) {
	return r.app.CountRecords("bookings", dbx.NewExp(filter))
}

func (r *PBAnalyticsRepo) CountTechnicians(filter string) (int64, error) {
	return r.app.CountRecords("technicians", dbx.NewExp(filter))
}
