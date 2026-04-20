package handler

import "project-service/internal/model"

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type createProjectResponse struct {
	ID int `json:"id"`
}

type ProjectMembersResponse struct {
	Members []model.MemberInfo `json:"members"`
}
