package ports

import (
	"context"

	"project-service/internal/core/domain"
)

type ProjectRepository interface {
	CreateProject(ctx context.Context, project domain.Project) (int, error)
	GetOwnedProjects(ctx context.Context, userID int32) ([]domain.Project, error)
	GetMemberProjects(ctx context.Context, userID int32) ([]domain.Project, error)
	IsUserMemberOfProject(ctx context.Context, userID int32, projectID int) (bool, error)
	GetProjectByID(ctx context.Context, projectID int) (*domain.Project, error)
	GetProjectByName(ctx context.Context, name string) (*domain.Project, error)
	GetProjectMembers(ctx context.Context, projectID int) ([]int32, error)
	AddProjectMember(ctx context.Context, projectID int, userID int32) error
	TransferProjectManager(ctx context.Context, projectID int, currentManagerID int32, newManagerID int32) error
	RemoveProjectMember(ctx context.Context, projectID int, userID int32) error
	UpdateProject(ctx context.Context, project domain.Project) error
	DeleteProject(ctx context.Context, projectID int32) error
}
