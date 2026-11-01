//go:build wireinject
// +build wireinject

package wire

import (
	"zedlancer/backend/bootstrap"
	"zedlancer/backend/internal/application/service"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/infrastructure/persistence"
	"zedlancer/backend/internal/infrastructure/security"
	"zedlancer/backend/internal/infrastructure/sms"
	"zedlancer/backend/internal/presentation/controllers/v1/general"

	"github.com/google/wire"
)

var RepositoryProviderSet = wire.NewSet(
	persistence.NewRepositoryFactory,
	persistence.NewUnitOfWork,
	persistence.NewPostgresDatabase,
	wire.Bind(new(ports.RepositoryFactory), new(*persistence.RepositoryFactory)),
	wire.Bind(new(ports.UnitOfWork), new(*persistence.UnitOfWork)),
)
var ServiceProviderSet = wire.NewSet(
	security.NewBcryptPasswordHasher,
	security.ProvideJWTManager,
	service.NewJobSeekerAuthService,
	wire.Bind(new(ports.PasswordHasher), new(*security.BcryptPasswordHasher)),
	wire.Bind(new(ports.TokenManager), new(*security.JWTManager)),
	wire.Bind(new(usecase.JobSeekerAuthService), new(*service.JobSeekerAuthService)),
	service.NewJobSeekerService,
	sms.NewFerzzSender,
	wire.Bind(new(ports.OTPSender), new(*sms.FerzzSender)),
	wire.Bind(new(usecase.JobSeekerService), new(*service.JobSeekerService)),
	service.NewUserService,
	wire.Bind(new(usecase.UserService), new(*service.UserService)),
)

var GeneralControllersProviderSet = wire.NewSet(
	general.NewJobSeekerAuthController,
	general.NewJobSeekerController,
	general.NewGeneralUserController,
	wire.Struct(new(GeneralControllers), "*"),
)

var ControllersProviderSet = wire.NewSet(
	wire.Struct(new(Controllers), "*"),
)

var ProviderSet = wire.NewSet(
	ControllersProviderSet,
	GeneralControllersProviderSet,
	ServiceProviderSet,
	RepositoryProviderSet,
)

type GeneralControllers struct {
	JobSeekerAuthController *general.JobSeekerAuthController
	JobSeekerController     *general.JobSeekerController
	GeneralUserController   *general.GeneralUserController
}

type Controllers struct {
	GeneralControllers *GeneralControllers
}

type Application struct {
	Controllers          *Controllers
	JobSeekerAuthService usecase.JobSeekerAuthService
}

func NewApplication(controllers *Controllers, auth usecase.JobSeekerAuthService) *Application {
	return &Application{
		Controllers:          controllers,
		JobSeekerAuthService: auth,
	}
}

func InitializeApplication(container *bootstrap.Config) (*Application, error) {
	wire.Build(
		ProviderSet,
		NewApplication,
	)
	return &Application{}, nil
}
