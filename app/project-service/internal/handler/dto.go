package handler

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type createProjectResponse struct {
	ID int `json:"id"`
}
