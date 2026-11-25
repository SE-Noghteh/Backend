package general

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"zedlancer/backend/internal/domain/enums"
)

type ProvincesResponse struct {
	Provinces []enums.Province `json:"provinces" swaggertype:"array,string"`
}

type CitiesResponse struct {
	Cities []enums.City `json:"cities" swaggertype:"array,string"`
}

// ListProvinces godoc
// @Summary List all provinces in Iran
// @Tags Locations
// @Produce json
// @Success 200 {object} ProvincesResponse
// @Router /provinces [get]
func ListProvinces(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ProvincesResponse{Provinces: enums.Provinces()})
}

// ListCities godoc
// @Summary List cities in Iran
// @Description Pass province to list its cities. Without province, returns unique city names across Iran.
// @Tags Locations
// @Produce json
// @Param province query string false "Persian province name returned by GET /v1/provinces"
// @Success 200 {object} CitiesResponse
// @Failure 400 {object} ErrorResponse
// @Router /cities [get]
func ListCities(ctx *gin.Context) {
	province := enums.Province(ctx.Query("province"))
	cities := enums.Cities()
	if province != "" {
		if !province.Valid() {
			ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid province"})
			return
		}
		cities = enums.CitiesInProvince(province)
	}
	ctx.JSON(http.StatusOK, CitiesResponse{Cities: cities})
}
