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

func NewAnalyticsService(app *pocketbase.PocketBase) *AnalyticsService {
	return &AnalyticsService{App: app}
}

// GetRevenueLast7Days returns daily revenue for the last 7 days
func (s *AnalyticsService) GetRevenueLast7Days() ([]RevenueStat, error) {
	stats := make([]RevenueStat, 0)

	// Lặp 7 ngày từ quá khứ đến hiện tại
	start := time.Now().AddDate(0, 0, -6)

	for i := 0; i < 7; i++ {
		date := start.AddDate(0, 0, i)
		dateStr := date.Format("2006-01-02")

		// Xác định khoảng thời gian đầu ngày và cuối ngày
		dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		dayEnd := dayStart.Add(24 * time.Hour)

		// Chuyển sang chuỗi format chuẩn của PocketBase
		startStr := dayStart.Format("2006-01-02 15:04:05")
		endStr := dayEnd.Format("2006-01-02 15:04:05")

		// [FIX] Xóa .Dao(), gọi trực tiếp s.App.FindRecordsByFilter
		invoices, err := s.App.FindRecordsByFilter(
			"invoices",
			"status = 'paid' && created >= {:start} && created < {:end}",
			"", // Không cần sort
			0,  // Không giới hạn số lượng
			0,  // Offset 0
			dbx.Params{
				"start": startStr,
				"end":   endStr,
			},
		)

		if err != nil {
			log.Printf("Error fetching invoices for %s: %v", dateStr, err)
			// Nếu lỗi, coi như doanh thu bằng 0 và tiếp tục
			stats = append(stats, RevenueStat{Date: dateStr, Amount: 0})
			continue
		}

		// Tính tổng thủ công
		var total float64
		for _, inv := range invoices {
			total += inv.GetFloat("total_amount")
		}

		stats = append(stats, RevenueStat{
			Date:   dateStr,
			Amount: total,
		})
	}

	return stats, nil
}

// GetTopTechnicians returns performance stats for verified techs
func (s *AnalyticsService) GetTopTechnicians(limit int) ([]TechPerformance, error) {
	var results []TechPerformance

	// [FIX] Xóa .DB(), dùng s.App.DB() trực tiếp (nếu App expose DB)
	// Tuy nhiên, PocketBase struct thường expose DB() qua interface
	// Nếu s.App là *pocketbase.PocketBase thì nó có phương thức DB()
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
