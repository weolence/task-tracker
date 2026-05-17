package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	userv1 "auth-service/api/proto/userv1"
	projectv1 "project-service/api/proto/projectv1"
	"project-service/internal/core/domain"
	"project-service/internal/core/ports"

	"google.golang.org/protobuf/encoding/protojson"
)

type ProjectUseCase struct {
	projectRepo    ports.ProjectRepository
	taskRepo       ports.TaskRepository
	authServiceURL string
}

func NewProjectUseCase(projectRepo ports.ProjectRepository, taskRepo ports.TaskRepository, authServiceURL string) *ProjectUseCase {
	return &ProjectUseCase{
		projectRepo:    projectRepo,
		taskRepo:       taskRepo,
		authServiceURL: authServiceURL,
	}
}

func (uc *ProjectUseCase) GetDashboard(ctx context.Context, userID int32) (projectv1.DashboardResponse, error) {
	owned, err := uc.projectRepo.GetOwnedProjects(ctx, userID)
	if err != nil {
		return projectv1.DashboardResponse{}, err
	}

	member, err := uc.projectRepo.GetMemberProjects(ctx, userID)
	if err != nil {
		return projectv1.DashboardResponse{}, err
	}

	ownedDto := make([]*projectv1.Project, len(owned))
	for i, p := range owned {
		ownedDto[i] = &projectv1.Project{
			Id:          p.ID,
			ManagerId:   p.ManagerID,
			Name:        p.Name,
			Description: p.Description,
			Status:      projectv1.ProjectStatus(p.Status + 1),
			StartDate:   p.StartDate.Format("2006-01-02"),
			EndDate:     p.EndDate,
		}
	}

	memberDto := make([]*projectv1.Project, len(member))
	for i, p := range member {
		memberDto[i] = &projectv1.Project{
			Id:          p.ID,
			ManagerId:   p.ManagerID,
			Name:        p.Name,
			Description: p.Description,
			Status:      projectv1.ProjectStatus(p.Status + 1),
			StartDate:   p.StartDate.Format("2006-01-02"),
			EndDate:     p.EndDate,
		}
	}

	return projectv1.DashboardResponse{
		OwnedProjects:  ownedDto,
		MemberProjects: memberDto,
	}, nil
}

func (uc *ProjectUseCase) CreateProject(ctx context.Context, userID int32, project domain.Project) (int, error) {
	if project.Name == "" || project.Description == "" {
		return 0, errors.New("name and description are required")
	}

	project.ManagerID = userID
	project.Status = domain.ProjectStatusInWork

	projectID, err := uc.projectRepo.CreateProject(ctx, project)
	if err != nil {
		return 0, err
	}

	return projectID, nil
}

func (uc *ProjectUseCase) GetProjectTasks(ctx context.Context, projectID int, userID int32) (projectv1.TasksResponse, error) {
	isMember, err := uc.projectRepo.IsUserMemberOfProject(ctx, userID, projectID)
	if err != nil {
		return projectv1.TasksResponse{}, err
	}
	if !isMember {
		return projectv1.TasksResponse{}, errors.New("access denied")
	}

	tasks, err := uc.taskRepo.GetTasksByProjectAndAssignee(ctx, projectID, int(userID))
	if err != nil {
		return projectv1.TasksResponse{}, err
	}

	return projectv1.TasksResponse{Tasks: tasksToDTO(tasks)}, nil
}

func (uc *ProjectUseCase) UpdateTaskStatus(ctx context.Context, taskID int, newStatus domain.TaskStatus, userID int32) error {
	return uc.taskRepo.UpdateTaskStatus(ctx, taskID, newStatus)
}

func (uc *ProjectUseCase) IsUserManager(ctx context.Context, userID int32, projectID int) (bool, error) {
	project, err := uc.projectRepo.GetProjectByID(ctx, projectID)
	if err != nil {
		return false, err
	}
	if project == nil {
		return false, errors.New("project not found")
	}
	return project.ManagerID == userID, nil
}

func (uc *ProjectUseCase) IsUserMember(ctx context.Context, userID int32, projectID int) (bool, error) {
	return uc.projectRepo.IsUserMemberOfProject(ctx, userID, projectID)
}

func (uc *ProjectUseCase) GetProjectMembers(ctx context.Context, projectID int, userID int32) ([]int32, error) {
	isMember, err := uc.projectRepo.IsUserMemberOfProject(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("access denied")
	}

	return uc.projectRepo.GetProjectMembers(ctx, projectID)
}

func (uc *ProjectUseCase) GetProjectMembersWithDetails(ctx context.Context, projectID int, userID int32) (projectv1.ProjectMembersResponse, error) {
	isMember, err := uc.projectRepo.IsUserMemberOfProject(ctx, userID, projectID)
	if err != nil {
		return projectv1.ProjectMembersResponse{}, err
	}
	if !isMember {
		return projectv1.ProjectMembersResponse{}, errors.New("access denied")
	}

	memberIDs, err := uc.projectRepo.GetProjectMembers(ctx, projectID)
	if err != nil {
		return projectv1.ProjectMembersResponse{}, err
	}

	seen := make(map[int32]bool)
	dedupedMemberIDs := make([]int32, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		if !seen[memberID] {
			seen[memberID] = true
			dedupedMemberIDs = append(dedupedMemberIDs, memberID)
		}
	}

	members := make([]*userv1.User, 0, len(dedupedMemberIDs))
	for _, memberID := range dedupedMemberIDs {
		user, err := uc.fetchUserFromAuthService(int32(memberID))
		if err != nil {
			continue
		}
		members = append(members, &userv1.User{
			Id:      int32(user.ID),
			Email:   user.Email,
			Name:    user.Name,
			Surname: user.Surname,
		})
	}

	return projectv1.ProjectMembersResponse{Members: members}, nil
}

func (uc *ProjectUseCase) AddProjectMember(ctx context.Context, projectID int, managerID int32, email string) (*userv1.User, error) {
	if email == "" {
		return nil, errors.New("email is required")
	}

	isManager, err := uc.IsUserManager(ctx, managerID, projectID)
	if err != nil {
		return nil, err
	}
	if !isManager {
		return nil, errors.New("access denied")
	}

	project, err := uc.projectRepo.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, errors.New("project not found")
	}

	user, err := uc.fetchUserFromAuthServiceByEmail(email)
	if err != nil {
		return nil, err
	}
	if int32(user.ID) == project.ManagerID {
		return nil, errors.New("this user is the project manager and is already a member")
	}

	if err := uc.projectRepo.AddProjectMember(ctx, projectID, int32(user.ID)); err != nil {
		return nil, err
	}

	return &userv1.User{
		Id:      int32(user.ID),
		Email:   user.Email,
		Name:    user.Name,
		Surname: user.Surname,
		Role:    user.Role,
	}, nil
}

func (uc *ProjectUseCase) TransferProjectManager(ctx context.Context, projectID int, currentManagerID int32, newManagerID int32) error {
	if newManagerID == 0 {
		return errors.New("new_manager_id is required")
	}

	isManager, err := uc.IsUserManager(ctx, currentManagerID, projectID)
	if err != nil {
		return err
	}
	if !isManager {
		return errors.New("access denied")
	}

	if newManagerID == currentManagerID {
		return errors.New("manager cannot transfer project to themselves")
	}

	isMember, err := uc.projectRepo.IsUserMemberOfProject(ctx, newManagerID, projectID)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("selected user is not a project member")
	}

	return uc.projectRepo.TransferProjectManager(ctx, projectID, currentManagerID, newManagerID)
}

func (uc *ProjectUseCase) GetUserProjects(ctx context.Context, userID int32) (projectv1.DashboardResponse, error) {
	owned, err := uc.projectRepo.GetOwnedProjects(ctx, userID)
	if err != nil {
		return projectv1.DashboardResponse{}, err
	}

	member, err := uc.projectRepo.GetMemberProjects(ctx, userID)
	if err != nil {
		return projectv1.DashboardResponse{}, err
	}

	ownedDto := make([]*projectv1.Project, len(owned))
	for i, p := range owned {
		ownedDto[i] = &projectv1.Project{
			Id:          p.ID,
			ManagerId:   p.ManagerID,
			Name:        p.Name,
			Description: p.Description,
			Status:      projectv1.ProjectStatus(p.Status + 1),
			StartDate:   p.StartDate.Format("2006-01-02"),
			EndDate:     p.EndDate,
		}
	}

	memberDto := make([]*projectv1.Project, len(member))
	for i, p := range member {
		memberDto[i] = &projectv1.Project{
			Id:          p.ID,
			ManagerId:   p.ManagerID,
			Name:        p.Name,
			Description: p.Description,
			Status:      projectv1.ProjectStatus(p.Status + 1),
			StartDate:   p.StartDate.Format("2006-01-02"),
			EndDate:     p.EndDate,
		}
	}

	return projectv1.DashboardResponse{
		OwnedProjects:  ownedDto,
		MemberProjects: memberDto,
	}, nil
}

func (uc *ProjectUseCase) GetProjectInfo(ctx context.Context, projectID int, userID int32) (*projectv1.Project, error) {
	isMember, err := uc.projectRepo.IsUserMemberOfProject(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("access denied")
	}

	project, err := uc.projectRepo.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &projectv1.Project{
		Id:          project.ID,
		ManagerId:   project.ManagerID,
		Name:        project.Name,
		Description: project.Description,
		Status:      projectv1.ProjectStatus(project.Status + 1),
		StartDate:   project.StartDate.Format("2006-01-02"),
		EndDate:     project.EndDate,
	}, nil
}

func (uc *ProjectUseCase) RemoveProjectMember(ctx context.Context, projectID int, requesterID int32, targetID int32) error {
	if targetID == 0 {
		return errors.New("member_id is required")
	}

	isManager, err := uc.IsUserManager(ctx, requesterID, projectID)
	if err != nil {
		return err
	}
	if !isManager {
		return errors.New("access denied")
	}

	project, err := uc.projectRepo.GetProjectByID(ctx, projectID)
	if err != nil {
		return err
	}
	if project == nil {
		return errors.New("project not found")
	}
	if project.ManagerID == targetID {
		return errors.New("cannot remove the project manager")
	}

	isMember, err := uc.projectRepo.IsUserMemberOfProject(ctx, targetID, projectID)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("user is not a project member")
	}

	if err := uc.taskRepo.UnassignTasksByMemberAndProject(ctx, projectID, targetID); err != nil {
		return err
	}

	return uc.projectRepo.RemoveProjectMember(ctx, projectID, targetID)
}

func (uc *ProjectUseCase) LeaveProject(ctx context.Context, projectID int, userID int32) error {
	project, err := uc.projectRepo.GetProjectByID(ctx, projectID)
	if err != nil {
		return err
	}
	if project == nil {
		return errors.New("project not found")
	}
	if project.ManagerID == userID {
		return errors.New("manager cannot leave the project; transfer management first")
	}

	isMember, err := uc.projectRepo.IsUserMemberOfProject(ctx, userID, projectID)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("you are not a member of this project")
	}

	if err := uc.taskRepo.UnassignTasksByMemberAndProject(ctx, projectID, userID); err != nil {
		return err
	}

	return uc.projectRepo.RemoveProjectMember(ctx, projectID, userID)
}

func (uc *ProjectUseCase) GetProjectForAdmin(ctx context.Context, projectID *int32, name *string) (*projectv1.Project, error) {
	var (
		project *domain.Project
		err     error
	)

	switch {
	case projectID != nil && *projectID > 0:
		project, err = uc.projectRepo.GetProjectByID(ctx, int(*projectID))
	case name != nil && *name != "":
		project, err = uc.projectRepo.GetProjectByName(ctx, *name)
	default:
		return nil, errors.New("project_id or name is required")
	}
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, errors.New("project not found")
	}

	return &projectv1.Project{
		Id:          project.ID,
		ManagerId:   project.ManagerID,
		Name:        project.Name,
		Description: project.Description,
		Status:      projectv1.ProjectStatus(project.Status + 1),
		StartDate:   project.StartDate.Format("2006-01-02"),
		EndDate:     project.EndDate,
	}, nil
}

func (uc *ProjectUseCase) UpdateProjectForAdmin(ctx context.Context, project *projectv1.Project) error {
	if project == nil || project.Id == 0 {
		return errors.New("project id is required")
	}

	startDate, err := time.Parse("2006-01-02", project.StartDate)
	if err != nil {
		return err
	}

	return uc.projectRepo.UpdateProject(ctx, domain.Project{
		ID:          project.Id,
		ManagerID:   project.ManagerId,
		Name:        project.Name,
		Description: project.Description,
		Status:      domain.ProjectStatus(project.Status - 1),
		StartDate:   startDate,
		EndDate:     project.EndDate,
	})
}

func (uc *ProjectUseCase) DeleteProject(ctx context.Context, projectID int, managerID int32) error {
	isManager, err := uc.IsUserManager(ctx, managerID, projectID)
	if err != nil {
		return err
	}
	if !isManager {
		return errors.New("access denied")
	}
	return uc.projectRepo.DeleteProject(ctx, int32(projectID))
}

func (uc *ProjectUseCase) DeleteProjectForAdmin(ctx context.Context, projectID int32) error {
	if projectID == 0 {
		return errors.New("project id is required")
	}
	return uc.projectRepo.DeleteProject(ctx, projectID)
}

func (uc *ProjectUseCase) fetchUserFromAuthService(userID int32) (domain.User, error) {
	req := &userv1.GetUserRequest{UserId: &userID}
	return uc.fetchUserByRequest(req)
}

func (uc *ProjectUseCase) fetchUserFromAuthServiceByEmail(email string) (domain.User, error) {
	req := &userv1.GetUserRequest{Email: &email}
	return uc.fetchUserByRequest(req)
}

func (uc *ProjectUseCase) fetchUserByRequest(request *userv1.GetUserRequest) (domain.User, error) {
	if uc.authServiceURL == "" {
		return domain.User{}, errors.New("auth service URL not configured")
	}

	reqBody, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(request)
	if err != nil {
		return domain.User{}, err
	}

	resp, err := http.Post(uc.authServiceURL+"/user-info", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return domain.User{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain.User{}, fmt.Errorf("failed to fetch user info from auth service: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.User{}, err
	}

	var member userv1.User
	if err := protojson.Unmarshal(body, &member); err != nil {
		return domain.User{}, err
	}

	return domain.User{
		ID:      int(member.Id),
		Email:   member.Email,
		Name:    member.Name,
		Surname: member.Surname,
		Role:    member.Role,
	}, nil
}

func tasksToDTO(tasks []domain.Task) []*projectv1.Task {
	result := make([]*projectv1.Task, len(tasks))
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
		result[i] = &projectv1.Task{
			Id:          t.ID,
			ProjectId:   t.ProjectID,
			AssigneeId:  t.AssigneeID,
			Name:        t.Name,
			Description: &t.Description,
			Priority:    projectv1.TaskPriority(t.Priority),
			Difficulty:  projectv1.TaskDifficulty(t.Difficulty),
			Status:      projectv1.TaskStatus(t.Status),
			StartDate:   startDate,
			EndDate:     endDate,
		}
	}
	return result
}
