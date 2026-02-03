package service

import (
	"hvac-system/internal/core"
	"time"
)

type AnalyticsService struct {
	repo core.AnalyticsRepository
}

func NewAnalyticsService(repo core.AnalyticsRepository) core.AnalyticsService {
	return &AnalyticsService{repo: repo}
}

func (s *AnalyticsService) GetRevenueLast7Days() ([]core.RevenueStat, error) {
	stats := make([]core.RevenueStat, 0)
	statsMap := make(map[string]float64)

	end := time.Now()
	start := end.AddDate(0, 0, -6)

	startStr := start.Format("2006-01-02 00:00:00")
	endStr := end.Format("2006-01-02 23:59:59")

	results, err := s.repo.GetDailyRevenue(startStr, endStr)
	if err == nil {
		for _, r := range results {
			statsMap[r.Date] = r.Amount
		}
	}

	// Fill in last 7 days ensuring no gaps
	for i := 0; i < 7; i++ {
		date := start.AddDate(0, 0, i)
		dateStr := date.Format("2006-01-02")

		stats = append(stats, core.RevenueStat{
			Date:   dateStr,
			Amount: statsMap[dateStr],
		})
	}

	return stats, nil
}

func (s *AnalyticsService) GetTopTechnicians(limit int) ([]core.TechPerformance, error) {
	return s.repo.GetTopTechnicians(limit)
}

func (s *AnalyticsService) GetDashboardStats() (*core.DashboardStats, error) {
	stats := &core.DashboardStats{}

	// 1. Total Revenue
	totalRev, _ := s.repo.GetTotalRevenue() // Ignore error, default 0
	stats.TotalRevenue = totalRev

	// 2. Counts
	today := time.Now().Format("2006-01-02")
	bookingsToday, _ := s.repo.CountBookings("created >= '" + today + " 00:00:00'")
	stats.BookingsToday = int(bookingsToday)

	activeTechs, _ := s.repo.CountTechnicians("verified=true")
	stats.ActiveTechs = int(activeTechs)

	pendingCount, _ := s.repo.CountBookings("job_status = 'pending'")
	stats.PendingCount = int(pendingCount)

	completedCount, _ := s.repo.CountBookings("job_status = 'completed'")
	stats.CompletedCount = int(completedCount)

	// 3. Rate (Same-day completion rate)
	if stats.BookingsToday > 0 {
		// Count bookings created today that are also completed
		completedToday, _ := s.repo.CountBookings("created >= '" + today + " 00:00:00' && job_status = 'completed'")
		stats.CompletionRate = (float64(completedToday) / float64(stats.BookingsToday)) * 100
	} else {
		stats.CompletionRate = 0
	}

	return stats, nil
}
