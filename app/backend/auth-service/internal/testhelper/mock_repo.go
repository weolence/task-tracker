package testhelper

import (
	"context"
	"errors"
	"sync"

	"auth-service/internal/core/domain"
)

// MockUserRepository is an in-memory implementation of ports.UserRepository for testing.
type MockUserRepository struct {
	mu sync.Mutex

	users  map[string]*domain.User
	byID   map[int]*domain.User
	nextID int32

	CreateErr     error
	GetByEmailErr error
	GetByIDErr    error
	UpdateErr     error
	DeleteErr     error
}

func NewMockRepo() *MockUserRepository {
	return &MockUserRepository{
		users:  make(map[string]*domain.User),
		byID:   make(map[int]*domain.User),
		nextID: 1,
	}
}

func (m *MockUserRepository) CreateUser(_ context.Context, user domain.User) error {
	if m.CreateErr != nil {
		return m.CreateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	user.ID = m.nextID
	m.nextID++
	cp := user
	m.users[cp.Email] = &cp
	m.byID[int(cp.ID)] = &cp
	return nil
}

func (m *MockUserRepository) GetUserByEmail(_ context.Context, email string) (*domain.User, error) {
	if m.GetByEmailErr != nil {
		return nil, m.GetByEmailErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[email]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (m *MockUserRepository) GetUserByID(_ context.Context, userID int) (*domain.User, error) {
	if m.GetByIDErr != nil {
		return nil, m.GetByIDErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[userID]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (m *MockUserRepository) DeleteUserByEmail(_ context.Context, email string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.users[email]; ok {
		delete(m.byID, int(u.ID))
		delete(m.users, email)
	}
	return nil
}

func (m *MockUserRepository) DeleteUserByID(_ context.Context, userID int32) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.byID[int(userID)]; ok {
		delete(m.users, u.Email)
		delete(m.byID, int(userID))
	}
	return nil
}

func (m *MockUserRepository) UpdateUser(_ context.Context, user domain.User, hashedPassword *string) error {
	if m.UpdateErr != nil {
		return m.UpdateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[int(user.ID)]
	if !ok {
		return errors.New("user not found")
	}
	for k, v := range m.users {
		if v.ID == user.ID {
			delete(m.users, k)
			break
		}
	}
	u.Email = user.Email
	u.Name = user.Name
	u.Surname = user.Surname
	u.Role = user.Role
	if hashedPassword != nil {
		u.Password = *hashedPassword
	}
	m.users[user.Email] = u
	return nil
}

func (m *MockUserRepository) ChangeRole(_ context.Context, email string, newRole string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[email]
	if !ok {
		return errors.New("user not found")
	}
	u.Role = newRole
	return nil
}

// GetStoredUser returns a copy of the in-memory user (for test assertions).
func (m *MockUserRepository) GetStoredUser(email string) *domain.User {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[email]
	if !ok {
		return nil
	}
	cp := *u
	return &cp
}
