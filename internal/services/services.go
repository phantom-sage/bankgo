package services

import (
	"github.com/phantom-sage/bankgo/internal/repository"
	"github.com/rs/zerolog"
)

// Services holds all business logic services
type Services struct {
	UserService     UserService
	AccountService  AccountService
	TransferService TransferService
}

// NewServices creates a new services instance with all business logic services
func NewServices(repos *repository.Repositories, repo *repository.Repository, logger zerolog.Logger) *Services {
	return &Services{
		UserService:     NewUserService(repos.UserRepo, logger),
		AccountService:  NewAccountService(repos.AccountRepo, repos.TransferRepo, logger),
		TransferService: NewTransferService(repo, repos.AccountRepo, repos.TransferRepo, logger),
	}
}