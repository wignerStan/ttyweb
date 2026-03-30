package service

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"ttyweb/db"
)

// TaskStats holds aggregate counts of tasks by status.
type TaskStats struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	Failed     int `json:"failed"`
}

// DailyCount holds the number of tasks completed on a specific day.
type DailyCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// StatsService provides aggregate task statistics.
type StatsService struct {
	db *gorm.DB
}

// NewStatsService creates a new StatsService backed by the given database.
func NewStatsService(gormDB *gorm.DB) *StatsService {
	return &StatsService{db: gormDB}
}

// dailyRow is used for scanning raw SQL results from GetDailyCompletions.
type dailyRow struct {
	Day   string
	Count int64
}

// GetTaskStats returns aggregate task counts: total, completed, in_progress, and failed.
func (s *StatsService) GetTaskStats() (TaskStats, error) {
	var total int64
	if err := s.db.Model(&db.TaskSegment{}).Count(&total).Error; err != nil {
		return TaskStats{}, fmt.Errorf("failed to count total segments: %w", err)
	}

	type statusCount struct {
		TaskStatus string
		Count      int64
	}
	var counts []statusCount
	if err := s.db.Model(&db.TaskSegment{}).
		Select("task_status, COUNT(*) as count").
		Group("task_status").
		Find(&counts).Error; err != nil {
		return TaskStats{}, fmt.Errorf("failed to count segments by status: %w", err)
	}

	stats := TaskStats{Total: int(total)}
	for _, c := range counts {
		switch c.TaskStatus {
		case StatusCompleted:
			stats.Completed = int(c.Count)
		case StatusInProgress:
			stats.InProgress = int(c.Count)
		case "failed":
			stats.Failed = int(c.Count)
		}
	}

	return stats, nil
}

// GetDailyCompletions returns the number of completed tasks per day for the
// last `days` days, ordered by day descending. Only tasks with status
// "completed" and a non-nil completed_at are included.
func (s *StatsService) GetDailyCompletions(days int) ([]DailyCount, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days+1).Format("2006-01-02")

	var rows []dailyRow
	if err := s.db.Model(&db.TaskSegment{}).
		Select("DATE(completed_at) as day, COUNT(*) as count").
		Where("task_status = ? AND DATE(completed_at) >= ?", "completed", cutoffDate).
		Group("DATE(completed_at)").
		Order("day DESC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to get daily completions: %w", err)
	}

	result := make([]DailyCount, len(rows))
	for i, r := range rows {
		result[i] = DailyCount{
			Date:  r.Day,
			Count: int(r.Count),
		}
	}

	return result, nil
}

// GetStatusBreakdown returns a map of task_status to count for all segments.
func (s *StatsService) GetStatusBreakdown() (map[string]int, error) {
	type statusCount struct {
		TaskStatus string
		Count      int64
	}
	var counts []statusCount
	if err := s.db.Model(&db.TaskSegment{}).
		Select("task_status, COUNT(*) as count").
		Group("task_status").
		Find(&counts).Error; err != nil {
		return nil, fmt.Errorf("failed to get status breakdown: %w", err)
	}

	breakdown := make(map[string]int, len(counts))
	for _, c := range counts {
		breakdown[c.TaskStatus] = int(c.Count)
	}

	return breakdown, nil
}
