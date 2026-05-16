package grpcclients

import (
	authsvcv1 "auth-service/api/proto/authsvcv1"
	projectsvcv1 "project-service/api/proto/projectsvcv1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Clients struct {
	Auth    authsvcv1.AuthServiceClient
	Project projectsvcv1.ProjectServiceClient

	authConn    *grpc.ClientConn
	projectConn *grpc.ClientConn
}

func New(authAddr, projectAddr string) (*Clients, error) {
	authConn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	projectConn, err := grpc.NewClient(projectAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		authConn.Close()
		return nil, err
	}

	return &Clients{
		Auth:        authsvcv1.NewAuthServiceClient(authConn),
		Project:     projectsvcv1.NewProjectServiceClient(projectConn),
		authConn:    authConn,
		projectConn: projectConn,
	}, nil
}

func (c *Clients) Close() {
	c.authConn.Close()
	c.projectConn.Close()
}
