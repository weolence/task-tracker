package testhelper

import (
	"context"
	"sync"

	"project-service/internal/core/domain"
)

// MockProjectRepository implements ports.ProjectRepository for testing.
type MockProjectRepository struct {
	mu             sync.RWMutex
	projects       map[int]*domain.Project
	projectMembers map[int][]int32
	lastID         int

	GetOwnedErr     error
	GetMemberErr    error
	CreateErr       error
	UpdateErr       error
	DeleteErr       error
	GetByIDErr      error
	GetMembersErr   error
	AddMemberErr    error
	RemoveMemberErr error
	IsMemberErr     error
}

func NewMockProjectRepository() *MockProjectRepository {
	return &MockProjectRepository{
		projects:       make(map[int]*domain.Project),
		projectMembers: make(map[int][]int32),
	}
}

func (m *MockProjectRepository) CreateProject(ctx context.Context, project domain.Project) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.CreateErr != nil {
		return 0, m.CreateErr
	}

	m.lastID++
	project.ID = int32(m.lastID)
	m.projects[m.lastID] = &project
	m.projectMembers[m.lastID] = []int32{project.ManagerID}
	return m.lastID, nil
}

func (m *MockProjectRepository) GetOwnedProjects(ctx context.Context, managerID int32) ([]domain.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetOwnedErr != nil {
		return nil, m.GetOwnedErr
	}

	var result []domain.Project
	for _, p := range m.projects {
		if p.ManagerID == managerID {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *MockProjectRepository) GetMemberProjects(ctx context.Context, userID int32) ([]domain.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetMemberErr != nil {
		return nil, m.GetMemberErr
	}

	seen := make(map[int]bool)
	var result []domain.Project
	for projectID, members := range m.projectMembers {
		for _, memberID := range members {
			if memberID == userID && !seen[projectID] {
				if p, ok := m.projects[projectID]; ok {
					result = append(result, *p)
					seen[projectID] = true
				}
			}
		}
	}
	return result, nil
}

func (m *MockProjectRepository) IsUserMemberOfProject(ctx context.Context, userID int32, projectID int) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.IsMemberErr != nil {
		return false, m.IsMemberErr
	}

	for _, id := range m.projectMembers[projectID] {
		if id == userID {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockProjectRepository) GetProjectByID(ctx context.Context, projectID int) (*domain.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetByIDErr != nil {
		return nil, m.GetByIDErr
	}

	p, ok := m.projects[projectID]
	if !ok {
		return nil, domain.ErrProjectNotFound
	}
	return p, nil
}

func (m *MockProjectRepository) GetProjectByName(ctx context.Context, name string) (*domain.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.projects {
		if p.Name == name {
			return p, nil
		}
	}
	return nil, domain.ErrProjectNotFound
}

func (m *MockProjectRepository) GetProjectMembers(ctx context.Context, projectID int) ([]int32, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetMembersErr != nil {
		return nil, m.GetMembersErr
	}

	members, ok := m.projectMembers[projectID]
	if !ok {
		return nil, domain.ErrProjectNotFound
	}
	return members, nil
}

func (m *MockProjectRepository) AddProjectMember(ctx context.Context, projectID int, userID int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.AddMemberErr != nil {
		return m.AddMemberErr
	}

	if _, ok := m.projects[projectID]; !ok {
		return domain.ErrProjectNotFound
	}

	for _, id := range m.projectMembers[projectID] {
		if id == userID {
			return domain.ErrUserAlreadyMember
		}
	}
	m.projectMembers[projectID] = append(m.projectMembers[projectID], userID)
	return nil
}

func (m *MockProjectRepository) TransferProjectManager(ctx context.Context, projectID int, currentManagerID int32, newManagerID int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.projects[projectID]
	if !ok {
		return domain.ErrProjectNotFound
	}
	p.ManagerID = newManagerID
	return nil
}

func (m *MockProjectRepository) RemoveProjectMember(ctx context.Context, projectID int, userID int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.RemoveMemberErr != nil {
		return m.RemoveMemberErr
	}

	if _, ok := m.projects[projectID]; !ok {
		return domain.ErrProjectNotFound
	}

	members := m.projectMembers[projectID]
	for i, id := range members {
		if id == userID {
			m.projectMembers[projectID] = append(members[:i], members[i+1:]...)
			return nil
		}
	}
	return domain.ErrMemberNotFound
}

func (m *MockProjectRepository) UpdateProject(ctx context.Context, project domain.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.UpdateErr != nil {
		return m.UpdateErr
	}

	if _, ok := m.projects[int(project.ID)]; !ok {
		return domain.ErrProjectNotFound
	}
	m.projects[int(project.ID)] = &project
	return nil
}

func (m *MockProjectRepository) UpdateProjectMeta(ctx context.Context, projectID int, name, description string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.projects[projectID]
	if !ok {
		return domain.ErrProjectNotFound
	}
	p.Name = name
	p.Description = description
	return nil
}

func (m *MockProjectRepository) UpdateProjectStatus(ctx context.Context, projectID int, status domain.ProjectStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.projects[projectID]
	if !ok {
		return domain.ErrProjectNotFound
	}
	p.Status = status
	return nil
}

func (m *MockProjectRepository) DeleteProject(ctx context.Context, projectID int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.DeleteErr != nil {
		return m.DeleteErr
	}

	if _, ok := m.projects[int(projectID)]; !ok {
		return domain.ErrProjectNotFound
	}
	delete(m.projects, int(projectID))
	delete(m.projectMembers, int(projectID))
	return nil
}

// MockTaskRepository implements ports.TaskRepository for testing.
type MockTaskRepository struct {
	mu     sync.RWMutex
	tasks  map[int]*domain.Task
	lastID int
	LastID int // exported for test setup access

	CreateErr   error
	UpdateErr   error
	DeleteErr   error
	GetByIDErr  error
	AssignErr   error
	UnassignErr error
}

func NewMockTaskRepository() *MockTaskRepository {
	return &MockTaskRepository{
		tasks: make(map[int]*domain.Task),
	}
}

func (m *MockTaskRepository) CreateTask(ctx context.Context, task domain.Task) (int32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.CreateErr != nil {
		return 0, m.CreateErr
	}

	m.lastID++
	task.ID = int32(m.lastID)
	m.tasks[m.lastID] = &task
	m.LastID = m.lastID
	return int32(m.lastID), nil
}

func (m *MockTaskRepository) GetTaskByID(ctx context.Context, taskID int) (*domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetByIDErr != nil {
		return nil, m.GetByIDErr
	}

	task, ok := m.tasks[taskID]
	if !ok {
		return nil, domain.ErrTaskNotFound
	}
	return task, nil
}

func (m *MockTaskRepository) UpdateTask(ctx context.Context, task domain.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.UpdateErr != nil {
		return m.UpdateErr
	}

	if _, ok := m.tasks[int(task.ID)]; !ok {
		return domain.ErrTaskNotFound
	}
	m.tasks[int(task.ID)] = &task
	return nil
}

func (m *MockTaskRepository) DeleteTask(ctx context.Context, taskID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.DeleteErr != nil {
		return m.DeleteErr
	}

	if _, ok := m.tasks[taskID]; !ok {
		return domain.ErrTaskNotFound
	}
	delete(m.tasks, taskID)
	return nil
}

func (m *MockTaskRepository) GetTasksByProjectAndAssignee(ctx context.Context, projectID int, assigneeID int) ([]domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []domain.Task
	for _, task := range m.tasks {
		if task.ProjectID == int32(projectID) && task.AssigneeID != nil && *task.AssigneeID == int32(assigneeID) {
			result = append(result, *task)
		}
	}
	return result, nil
}

func (m *MockTaskRepository) GetAllTasksByProject(ctx context.Context, projectID int) ([]domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []domain.Task
	for _, task := range m.tasks {
		if task.ProjectID == int32(projectID) {
			result = append(result, *task)
		}
	}
	return result, nil
}

func (m *MockTaskRepository) GetClosedTasksByProject(ctx context.Context, projectID int) ([]domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []domain.Task
	for _, task := range m.tasks {
		if task.ProjectID == int32(projectID) && task.Status == domain.TaskStatusClosed {
			result = append(result, *task)
		}
	}
	return result, nil
}

func (m *MockTaskRepository) GetTaskByProjectAndName(ctx context.Context, projectID int32, name string) (*domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, task := range m.tasks {
		if task.ProjectID == projectID && task.Name == name {
			return task, nil
		}
	}
	return nil, domain.ErrTaskNotFound
}

func (m *MockTaskRepository) AssignTask(ctx context.Context, taskID int, assigneeID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.AssignErr != nil {
		return m.AssignErr
	}

	task, ok := m.tasks[taskID]
	if !ok {
		return domain.ErrTaskNotFound
	}
	id := int32(assigneeID)
	task.AssigneeID = &id
	return nil
}

func (m *MockTaskRepository) UnassignTask(ctx context.Context, taskID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.UnassignErr != nil {
		return m.UnassignErr
	}

	task, ok := m.tasks[taskID]
	if !ok {
		return domain.ErrTaskNotFound
	}
	task.AssigneeID = nil
	return nil
}

func (m *MockTaskRepository) UpdateTaskStatus(ctx context.Context, taskID int, status domain.TaskStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return domain.ErrTaskNotFound
	}
	task.Status = status
	return nil
}

func (m *MockTaskRepository) CloseTask(ctx context.Context, taskID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return domain.ErrTaskNotFound
	}
	task.Status = domain.TaskStatusClosed
	return nil
}

func (m *MockTaskRepository) UnassignTasksByMemberAndProject(ctx context.Context, projectID int, userID int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, task := range m.tasks {
		if task.ProjectID == int32(projectID) && task.AssigneeID != nil && *task.AssigneeID == userID {
			task.AssigneeID = nil
		}
	}
	return nil
}

// MockCommentRepository implements ports.CommentRepository for testing.
type MockCommentRepository struct {
	mu       sync.RWMutex
	comments map[int32]*domain.Comment
	lastID   int32

	CreateErr   error
	UpdateErr   error
	DeleteErr   error
	GetByIDErr  error
	ByTaskErr   error
	ByAuthorErr error
}

func NewMockCommentRepository() *MockCommentRepository {
	return &MockCommentRepository{
		comments: make(map[int32]*domain.Comment),
	}
}

func (m *MockCommentRepository) CreateComment(ctx context.Context, comment domain.Comment) (int32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.CreateErr != nil {
		return 0, m.CreateErr
	}

	m.lastID++
	comment.ID = m.lastID
	m.comments[m.lastID] = &comment
	return m.lastID, nil
}

// GetCommentByID returns nil, nil when the comment does not exist (not an error).
func (m *MockCommentRepository) GetCommentByID(ctx context.Context, commentID int32) (*domain.Comment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetByIDErr != nil {
		return nil, m.GetByIDErr
	}

	comment, ok := m.comments[commentID]
	if !ok {
		return nil, nil
	}
	return comment, nil
}

func (m *MockCommentRepository) UpdateComment(ctx context.Context, comment domain.Comment) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.UpdateErr != nil {
		return m.UpdateErr
	}

	if _, ok := m.comments[comment.ID]; !ok {
		return domain.ErrProjectNotFound // reuse a sentinel; repo layer would have its own
	}
	m.comments[comment.ID] = &comment
	return nil
}

func (m *MockCommentRepository) DeleteComment(ctx context.Context, commentID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.DeleteErr != nil {
		return m.DeleteErr
	}

	id := int32(commentID)
	if _, ok := m.comments[id]; !ok {
		return domain.ErrMemberNotFound // any sentinel error works here
	}
	delete(m.comments, id)
	return nil
}

func (m *MockCommentRepository) GetCommentsByTaskID(ctx context.Context, taskID int) ([]domain.Comment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ByTaskErr != nil {
		return nil, m.ByTaskErr
	}

	var result []domain.Comment
	for _, c := range m.comments {
		if c.TaskID == int32(taskID) {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (m *MockCommentRepository) GetCommentsByAuthorID(ctx context.Context, authorID int) ([]domain.Comment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ByAuthorErr != nil {
		return nil, m.ByAuthorErr
	}

	var result []domain.Comment
	for _, c := range m.comments {
		if c.AuthorID == int32(authorID) {
			result = append(result, *c)
		}
	}
	return result, nil
}
