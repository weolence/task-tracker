package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"project-service/internal/model"
	"project-service/internal/model/dto"
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

func (controller *ProjectController) GetDashboard(ctx context.Context, userID int32) (dto.DashboardResponse, error) {
	owned, err := controller.projectRepository.GetOwnedProjects(ctx, userID)
	if err != nil {
		return dto.DashboardResponse{}, err
	}

	member, err := controller.projectRepository.GetMemberProjects(ctx, userID)
	if err != nil {
		return dto.DashboardResponse{}, err
	}

	ownedDto := make([]*dto.Project, len(owned))
	for i, p := range owned {
		ownedDto[i] = &dto.Project{
			Id:          p.ID,
			ManagerId:   p.ManagerID,
			Name:        p.Name,
			Description: p.Description,
			Status:      dto.ProjectStatus(p.Status + 1), // adjust enum
			StartDate:   p.StartDate.Format("2006-01-02"),
			EndDate:     p.EndDate,
		}
	}

	memberDto := make([]*dto.Project, len(member))
	for i, p := range member {
		memberDto[i] = &dto.Project{
			Id:          p.ID,
			ManagerId:   p.ManagerID,
			Name:        p.Name,
			Description: p.Description,
			Status:      dto.ProjectStatus(p.Status + 1),
			StartDate:   p.StartDate.Format("2006-01-02"),
			EndDate:     p.EndDate,
		}
	}

	return dto.DashboardResponse{
		OwnedProjects:  ownedDto,
		MemberProjects: memberDto,
	}, nil
}

func (controller *ProjectController) CreateProject(ctx context.Context, userID int32, project model.Project) (int, error) {
	if project.Name == "" || project.Description == "" {
		return 0, errors.New("name and description are required")
	}

	project.ManagerID = userID
	project.Status = model.ProjectStatusInWork
	return controller.projectRepository.CreateProject(ctx, project)
}

func (controller *ProjectController) GetProjectTasks(ctx context.Context, projectID int, userID int32) (dto.TasksResponse, error) {
	// Check if user is member or owner of the project
	isMember, err := controller.projectRepository.IsUserMemberOfProject(ctx, userID, projectID)
	if err != nil {
		return dto.TasksResponse{}, err
	}
	if !isMember {
		return dto.TasksResponse{}, errors.New("access denied")
	}

	tasks, err := controller.taskRepository.GetTasksByProjectAndAssignee(ctx, projectID, int(userID))
	if err != nil {
		return dto.TasksResponse{}, err
	}

	tasksDto := make([]*dto.Task, len(tasks))
	for i, t := range tasks {
		var startDate, endDate *string
		if t.StartDate != nil {
			s := t.StartDate.Format("2006-01-02")
			startDate = &s
		}
		if t.EndDate != nil {
			s := t.EndDate.Format("2006-01-02")
			endDate = &s
		}
		tasksDto[i] = &dto.Task{
			Id:          t.ID,
			ProjectId:   t.ProjectID,
			AssigneeId:  t.AssigneeID,
			Name:        t.Name,
			Description: &t.Description,
			Priority:    dto.TaskPriority(t.Priority),
			Difficulty:  dto.TaskDifficulty(t.Difficulty),
			Status:      dto.TaskStatus(t.Status),
			StartDate:   startDate,
			EndDate:     endDate,
		}
	}

	return dto.TasksResponse{Tasks: tasksDto}, nil
}

func (controller *ProjectController) UpdateTaskStatus(ctx context.Context, taskID int, newStatus model.TaskStatus, userID int32) error {
	// Optionally check if user has access to the task
	return controller.taskRepository.UpdateTaskStatus(ctx, taskID, newStatus)
}

func (controller *ProjectController) IsUserManager(ctx context.Context, userID int32, projectID int) (bool, error) {
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

func (controller *ProjectController) IsUserMember(ctx context.Context, userID int32, projectID int) (bool, error) {
	return controller.projectRepository.IsUserMemberOfProject(ctx, userID, projectID)
}

func (controller *ProjectController) GetProjectMembers(ctx context.Context, projectID int, userID int32) ([]int32, error) {
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

func (controller *ProjectController) GetProjectMembersWithDetails(ctx context.Context, projectID int, userID int32) (dto.ProjectMembersResponse, error) {
	// Check if user is member or owner of the project
	isMember, err := controller.projectRepository.IsUserMemberOfProject(ctx, userID, projectID)
	if err != nil {
		return dto.ProjectMembersResponse{}, err
	}
	if !isMember {
		return dto.ProjectMembersResponse{}, errors.New("access denied")
	}

	// Get member IDs
	memberIDs, err := controller.projectRepository.GetProjectMembers(ctx, projectID)
	if err != nil {
		return dto.ProjectMembersResponse{}, err
	}

	// Deduplicate if backend returned duplicate IDs
	seen := make(map[int32]bool)
	dedupedMemberIDs := make([]int32, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		if !seen[memberID] {
			seen[memberID] = true
			dedupedMemberIDs = append(dedupedMemberIDs, memberID)
		}
	}

	// Fetch details for each member from auth-service
	members := make([]*dto.User, 0, len(dedupedMemberIDs))
	for _, memberID := range dedupedMemberIDs {
		user, err := controller.fetchUserFromAuthService(int32(memberID))
		if err != nil {
			// Log but continue if one user fetch fails
			continue
		}
		members = append(members, &dto.User{
			Id:      int32(user.ID),
			Email:   user.Email,
			Name:    user.Name,
			Surname: user.Surname,
		})
	}

	return dto.ProjectMembersResponse{Members: members}, nil
}

func (controller *ProjectController) fetchUserFromAuthService(userID int32) (model.User, error) {
	if controller.authServiceURL == "" {
		return model.User{}, errors.New("auth service URL not configured")
	}

	url := fmt.Sprintf("%s/user-info?user_id=%d", controller.authServiceURL, userID)
	resp, err := http.Get(url)
	if err != nil {
		return model.User{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return model.User{}, fmt.Errorf("failed to fetch user info from auth service: %d", resp.StatusCode)
	}

	var member model.User
	if err := json.NewDecoder(resp.Body).Decode(&member); err != nil {
		return model.User{}, err
	}

	return member, nil
}

func (controller *ProjectController) GetUserProjects(ctx context.Context, userID int32) (dto.DashboardResponse, error) {
	owned, err := controller.projectRepository.GetOwnedProjects(ctx, userID)
	if err != nil {
		return dto.DashboardResponse{}, err
	}

	member, err := controller.projectRepository.GetMemberProjects(ctx, userID)
	if err != nil {
		return dto.DashboardResponse{}, err
	}

	ownedDto := make([]*dto.Project, len(owned))
	for i, p := range owned {
		ownedDto[i] = &dto.Project{
			Id:          p.ID,
			ManagerId:   p.ManagerID,
			Name:        p.Name,
			Description: p.Description,
			Status:      dto.ProjectStatus(p.Status + 1),
			StartDate:   p.StartDate.Format("2006-01-02"),
			EndDate:     p.EndDate,
		}
	}

	memberDto := make([]*dto.Project, len(member))
	for i, p := range member {
		memberDto[i] = &dto.Project{
			Id:          p.ID,
			ManagerId:   p.ManagerID,
			Name:        p.Name,
			Description: p.Description,
			Status:      dto.ProjectStatus(p.Status + 1),
			StartDate:   p.StartDate.Format("2006-01-02"),
			EndDate:     p.EndDate,
		}
	}

	return dto.DashboardResponse{
		OwnedProjects:  ownedDto,
		MemberProjects: memberDto,
	}, nil
}

func (controller *ProjectController) GetProjectInfo(ctx context.Context, projectID int, userID int32) (*dto.Project, error) {
	// Check if user is member or owner of the project
	isMember, err := controller.projectRepository.IsUserMemberOfProject(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("access denied")
	}

	project, err := controller.projectRepository.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &dto.Project{
		Id:          project.ID,
		ManagerId:   project.ManagerID,
		Name:        project.Name,
		Description: project.Description,
		Status:      dto.ProjectStatus(project.Status + 1),
		StartDate:   project.StartDate.Format("2006-01-02"),
		EndDate:     project.EndDate,
	}, nil
}
