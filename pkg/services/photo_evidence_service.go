package services

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// PhotoEvidenceService handles photo evidence validation and management
type PhotoEvidenceService struct {
	app core.App
}

// NewPhotoEvidenceService creates a new photo evidence service
func NewPhotoEvidenceService(app core.App) *PhotoEvidenceService {
	return &PhotoEvidenceService{app: app}
}

// PhotoEvidence represents before/after photos for a job
type PhotoEvidence struct {
	BeforeImages []string // URLs
	AfterImages  []string // URLs
	Notes        string
}

// ValidateBeforePhotos checks if before photos were uploaded when starting job
// Business rule: Before photos are optional but recommended
func (s *PhotoEvidenceService) ValidateBeforePhotos(jobReportID string) (bool, error) {
	jobReport, err := s.app.FindRecordById("job_reports", jobReportID)
	if err != nil {
		return false, err
	}

	beforeImages := jobReport.GetStringSlice("before_images")
	return len(beforeImages) > 0, nil
}

// ValidateAfterPhotos checks if mandatory after photos were uploaded
// Business rule: After photos are REQUIRED for job completion
func (s *PhotoEvidenceService) ValidateAfterPhotos(jobReportID string) error {
	jobReport, err := s.app.FindRecordById("job_reports", jobReportID)
	if err != nil {
		return err
	}

	afterImages := jobReport.GetStringSlice("after_images")
	if len(afterImages) == 0 {
		return fmt.Errorf("after_images are required to complete the job")
	}

	return nil
}

// SaveBeforePhotos saves photos taken at job start
func (s *PhotoEvidenceService) SaveBeforePhotos(jobReportID string, files []*filesystem.File) error {
	if len(files) == 0 {
		return nil // Optional, no error
	}

	jobReport, err := s.app.FindRecordById("job_reports", jobReportID)
	if err != nil {
		return fmt.Errorf("job report not found: %w", err)
	}

	// Convert to []any for PocketBase
	fileSlice := make([]any, len(files))
	for i, f := range files {
		fileSlice[i] = f
	}

	jobReport.Set("before_images", fileSlice)

	if err := s.app.Save(jobReport); err != nil {
		return fmt.Errorf("failed to save before photos: %w", err)
	}

	return nil
}

// SaveAfterPhotos saves photos taken at job completion (REQUIRED)
func (s *PhotoEvidenceService) SaveAfterPhotos(jobReportID string, files []*filesystem.File, notes string) error {
	if len(files) == 0 {
		return fmt.Errorf("after photos are mandatory for job completion")
	}

	jobReport, err := s.app.FindRecordById("job_reports", jobReportID)
	if err != nil {
		return fmt.Errorf("job report not found: %w", err)
	}

	// Convert to []any for PocketBase
	fileSlice := make([]any, len(files))
	for i, f := range files {
		fileSlice[i] = f
	}

	jobReport.Set("after_images", fileSlice)
	jobReport.Set("photo_notes", notes)

	if err := s.app.Save(jobReport); err != nil {
		return fmt.Errorf("failed to save after photos: %w", err)
	}

	return nil
}

// GetPhotoEvidence retrieves all photo evidence for a job
func (s *PhotoEvidenceService) GetPhotoEvidence(jobReportID string) (*PhotoEvidence, error) {
	jobReport, err := s.app.FindRecordById("job_reports", jobReportID)
	if err != nil {
		return nil, err
	}

	evidence := &PhotoEvidence{
		BeforeImages: jobReport.GetStringSlice("before_images"),
		AfterImages:  jobReport.GetStringSlice("after_images"),
		Notes:        jobReport.GetString("photo_notes"),
	}

	return evidence, nil
}

// CompareEvidence returns structured data for side-by-side photo comparison
// Used by admin dashboard to review quality of work
func (s *PhotoEvidenceService) CompareEvidence(jobReportID string) (map[string]interface{}, error) {
	evidence, err := s.GetPhotoEvidence(jobReportID)
	if err != nil {
		return nil, err
	}

	// Get job details for context
	jobReport, _ := s.app.FindRecordById("job_reports", jobReportID)

	comparison := map[string]interface{}{
		"job_id":        jobReportID,
		"before_images": evidence.BeforeImages,
		"after_images":  evidence.AfterImages,
		"notes":         evidence.Notes,
		"has_before":    len(evidence.BeforeImages) > 0,
		"has_after":     len(evidence.AfterImages) > 0,
		"status":        jobReport.GetString("status"),
	}

	return comparison, nil
}
