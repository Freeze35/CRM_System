package types

import (
	pb "testAuth/proto/protobuff"
	"time"
)

type DbServiceServer struct {
	pb.UnimplementedRegisterCompanyServer
	pb.UnimplementedLoginServer
}

type User struct {
	ID        int
	Email     string
	Phone     string
	Password  string
	CompanyID int
	CreatedAt time.Time
}
