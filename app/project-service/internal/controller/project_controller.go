package controller

import (
	"context"
	"errors"
	"project-service/internal/model"
	"project-service/internal/repository"
)

type ProjectController struct {
	projectRepository repository.ProjectRepository
}

func NewProjectController(projectRepository repository.ProjectRepository) *ProjectController {
	return &ProjectController{projectRepository: projectRepository}
}

func (controller *ProjectController) GetDashboard(ctx context.Context, userID int) (model.DashboardResponse, error) {
	owned, err := controller.projectRepository.GetOwnedProjects(ctx, userID)
	if err != nil {
		return model.DashboardResponse{}, err
	}

	member, err := controller.projectRepository.GetMemberProjects(ctx, userID)
	if err != nil {
		return model.DashboardResponse{}, err
	}

	return model.DashboardResponse{
		OwnedProjects:  owned,
		MemberProjects: member,
	}, nil
}

func (controller *ProjectController) CreateProject(ctx context.Context, userID int, project model.Project) (int, error) {
	if project.Name == "" || project.Description == "" {
		return 0, errors.New("name and description are required")
	}

	project.ManagerID = userID
	project.Status = model.ProjectStatusInWork
	return controller.projectRepository.CreateProject(ctx, project)
}
