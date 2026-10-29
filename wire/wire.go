//go:build wireinject
// +build wireinject

package wire

import (
	"zedlancer/backend/bootstrap"
	"zedlancer/backend/internal/application/service"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/infrastructure/persistence"
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
	service.NewUserService,
	wire.Bind(new(usecase.UserService), new(*service.UserService)),
)

var GeneralControllersProviderSet = wire.NewSet(
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
	GeneralUserController     *general.GeneralUserController
}


type Controllers struct {
	GeneralControllers   *GeneralControllers
}



type Application struct {
	Controllers *Controllers
}

func NewApplication(controllers *Controllers) *Application {
	return &Application{
		Controllers: controllers,
	}
}

func InitializeApplication(container *bootstrap.Config) (*Application, error) {
	wire.Build(
		ProviderSet,
		NewApplication,
	)
	return &Application{}, nil
}
