package request

import "time"

type Guidance struct {
	ExpectedVersion       uint      `json:"expectedVersion" binding:"required"`
	Guidance              string    `json:"guidance" binding:"required,max=4000"`
	ResponsibleAssigneeID uint      `json:"responsibleAssigneeId" binding:"required"`
	DueAt                 time.Time `json:"dueAt" binding:"required"`
}

type Intervene struct {
	ExpectedVersion       uint      `json:"expectedVersion" binding:"required"`
	Result                string    `json:"result" binding:"required,max=4000"`
	ResponsibleAssigneeID uint      `json:"responsibleAssigneeId" binding:"required"`
	DueAt                 time.Time `json:"dueAt" binding:"required"`
}

type ReviseDailySummary struct {
	ExpectedVersion uint   `json:"expectedVersion" binding:"required"`
	Reason          string `json:"reason" binding:"required,max=1000"`
}
