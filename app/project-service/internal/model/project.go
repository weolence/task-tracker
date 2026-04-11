package model

import (
	"encoding/json"
	"time"
)

type ProjectStatus int

const (
	ProjectStatusInWork ProjectStatus = iota
	ProjectStatusEnded
)

type Project struct {
	ID          int           `json:"id"`
	ManagerID   int           `json:"manager_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Status      ProjectStatus `json:"status"`
	StartDate   time.Time     `json:"start_date"`
	EndDate     *string       `json:"end_date"`
}

func (p Project) MarshalJSON() ([]byte, error) {
	type Alias Project
	return json.Marshal(&struct {
		StartDate string `json:"start_date"`
		*Alias
	}{
		StartDate: p.StartDate.Format("2006-01-02"),
		Alias:     (*Alias)(&p),
	})
}

type DashboardResponse struct {
	OwnedProjects  []Project `json:"owned_projects"`
	MemberProjects []Project `json:"member_projects"`
}
