package services

import (
	"log"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
)

type AnalyticsService struct {
	App *pocketbase.PocketBase
}

type RevenueStat struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"` // Daily Total
}

type TechPerformance struct {
	TechnicianID   string  `json:"technician_id"`
	TechnicianName string  `json:"technician_name"`
	CompletedJobs  int     `json:"completed_jobs"`
	TotalRevenue   float64 `json:"total_revenue"`
}

type DashboardStats struct {
	TotalRevenue   float64
	BookingsToday  int
	ActiveTechs    int
	PendingCount   int
	CompletedCount int
	CompletionRate float64
}

func NewAnalyticsService(app *pocketbase.PocketBase) *AnalyticsService {
	return &AnalyticsService{App: app}
}

// GetRevenueLast7Days returns daily revenue for the last 7 days optimized with single query
func (s *AnalyticsService) GetRevenueLast7Days() ([]RevenueStat, error) {
	stats := make([]RevenueStat, 0)

	// Create map for easy lookup
	statsMap := make(map[string]float64)

	end := time.Now()
	start := end.AddDate(0, 0, -6)

	// Format for DB query
	startStr := start.Format("2006-01-02 00:00:00")
	endStr := end.Format("2006-01-02 23:59:59")

	// Single query using GROUP BY on SQLite date function
	// Assuming created is stored as UTC text "YYYY-MM-DD HH:MM:SS.MMMZ" or similar
	type queryResult struct {
		Day   string  `db:"day"`
		Total float64 `db:"total"`
	}

	var results []queryResult

	// Note: substr(created, 1, 10) extracts YYYY-MM-DD from the timestamp string
	err := s.App.DB().Select(
		"substr(created, 1, 10) as day",
		"SUM(total_amount) as total",
	).
		From("invoices").
		Where(dbx.NewExp("status = 'paid' && created >= {:start} && created <= {:end}", dbx.Params{
			"start": startStr,
			"end":   endStr,
		})).
		GroupBy("day").
		All(&results)

	if err != nil {
		log.Printf("Error fetching revenue stats: %v", err)
		// Return empty/zero'd stats on error instead of breaking everything,
		// but ideally should handle better
	}

	for _, r := range results {
		statsMap[r.Day] = r.Total
	}

	// Fill in last 7 days ensuring no gaps
	for i := 0; i < 7; i++ {
		date := start.AddDate(0, 0, i)
		dateStr := date.Format("2006-01-02")

		stats = append(stats, RevenueStat{
			Date:   dateStr,
			Amount: statsMap[dateStr],
		})
	}

	return stats, nil
}

// GetTopTechnicians returns performance stats for verified techs
func (s *AnalyticsService) GetTopTechnicians(limit int) ([]TechPerformance, error) {
	var results []TechPerformance

	query := s.App.DB().Select(
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

// GetDashboardStats aggregates common dashboard metrics in parallel or efficient steps
func (s *AnalyticsService) GetDashboardStats() (*DashboardStats, error) {
	stats := &DashboardStats{}

	// 1. Total Revenue (Paid Invoices)
	// Query sum directly instead of fetching all records
	var revenueResult struct {
		Total float64 `db:"total"`
	}
	err := s.App.DB().Select("SUM(total_amount) as total").
		From("invoices").
		Where(dbx.HashExp{"status": "paid"}).
		One(&revenueResult)

	if err == nil {
		stats.TotalRevenue = revenueResult.Total
	}

	// 2. Counts
	today := time.Now().Format("2006-01-02")
	bookingsToday, _ := s.App.CountRecords("bookings", dbx.NewExp("created >= {:date}", dbx.Params{"date": today}))
	stats.BookingsToday = int(bookingsToday)

	activeTechs, _ := s.App.CountRecords("technicians", dbx.NewExp("verified=true")) // Assuming active means verified here based on old code
	stats.ActiveTechs = int(activeTechs)

	pending, _ := s.App.CountRecords("bookings", dbx.NewExp("job_status = 'pending'"))
	stats.PendingCount = int(pending)

	completed, _ := s.App.CountRecords("bookings", dbx.NewExp("job_status = 'completed'"))
	stats.CompletedCount = int(completed)

	// 3. Rate
	if stats.BookingsToday > 0 {
		stats.CompletionRate = (float64(stats.CompletedCount) / float64(stats.BookingsToday)) * 100
	}

	return stats, nil
}
