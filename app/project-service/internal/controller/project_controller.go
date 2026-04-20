package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"project-service/internal/model"
	"project-service/internal/repository"
)

type ProjectController struct {
	projectRepository repository.ProjectRepository
	taskRepository    repository.TaskRepository
	authServiceURL    string
}

func NewProjectController(projectRepository repository.ProjectRepository, taskRepository repository.TaskRepository, authServiceURL string) *ProjectController {
	return &ProjectController{
		projectRepository: projectRepository,
		taskRepository:    taskRepository,
		authServiceURL:    authServiceURL,
	}
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

func (controller *ProjectController) GetProjectTasks(ctx context.Context, projectID int, userID int) ([]model.Task, error) {
	// Check if user is member or owner of the project
	isMember, err := controller.projectRepository.IsUserMemberOfProject(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("access denied")
	}

	return controller.taskRepository.GetTasksByProjectAndAssignee(ctx, projectID, userID)
}

func (controller *ProjectController) UpdateTaskStatus(ctx context.Context, taskID int, newStatus model.TaskStatus, userID int) error {
	// Optionally check if user has access to the task
	return controller.taskRepository.UpdateTaskStatus(ctx, taskID, newStatus)
}

func (controller *ProjectController) IsUserManager(ctx context.Context, userID int, projectID int) (bool, error) {
	// Get project to check if user is manager
	project, err := controller.projectRepository.GetProjectByID(ctx, projectID)
	if err != nil {
		return false, err
	}
	if project == nil {
		return false, errors.New("project not found")
	}
	return project.ManagerID == userID, nil
}

func (controller *ProjectController) IsUserMember(ctx context.Context, userID int, projectID int) (bool, error) {
	return controller.projectRepository.IsUserMemberOfProject(ctx, userID, projectID)
}

func (controller *ProjectController) GetProjectMembers(ctx context.Context, projectID int, userID int) ([]int, error) {
	// Check if user is member or owner of the project
	isMember, err := controller.projectRepository.IsUserMemberOfProject(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("access denied")
	}

	return controller.projectRepository.GetProjectMembers(ctx, projectID)
}

func (controller *ProjectController) GetProjectMembersWithDetails(ctx context.Context, projectID int, userID int) ([]model.MemberInfo, error) {
	// Check if user is member or owner of the project
	isMember, err := controller.projectRepository.IsUserMemberOfProject(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("access denied")
	}

	// Get member IDs
	memberIDs, err := controller.projectRepository.GetProjectMembers(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Deduplicate if backend returned duplicate IDs
	seen := make(map[int]bool)
	dedupedMemberIDs := make([]int, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		if !seen[memberID] {
			seen[memberID] = true
			dedupedMemberIDs = append(dedupedMemberIDs, memberID)
		}
	}

	// Fetch details for each member from auth-service
	members := make([]model.MemberInfo, 0, len(dedupedMemberIDs))
	for _, memberID := range dedupedMemberIDs {
		member, err := controller.fetchUserFromAuthService(memberID)
		if err != nil {
			// Log but continue if one user fetch fails
			continue
		}
		members = append(members, member)
	}

	return members, nil
}

func (controller *ProjectController) fetchUserFromAuthService(userID int) (model.MemberInfo, error) {
	if controller.authServiceURL == "" {
		return model.MemberInfo{}, errors.New("auth service URL not configured")
	}

	url := fmt.Sprintf("%s/user-info?user_id=%d", controller.authServiceURL, userID)
	resp, err := http.Get(url)
	if err != nil {
		return model.MemberInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return model.MemberInfo{}, fmt.Errorf("failed to fetch user info from auth service: %d", resp.StatusCode)
	}

	var member model.MemberInfo
	if err := json.NewDecoder(resp.Body).Decode(&member); err != nil {
		return model.MemberInfo{}, err
	}

	return member, nil
}

func (controller *ProjectController) GetUserProjects(ctx context.Context, userID int) (model.DashboardResponse, error) {
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

func (controller *ProjectController) GetProjectInfo(ctx context.Context, projectID int, userID int) (*model.Project, error) {
	// Check if user is member or owner of the project
	isMember, err := controller.projectRepository.IsUserMemberOfProject(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("access denied")
	}

	return controller.projectRepository.GetProjectByID(ctx, projectID)
}
