package router

import (
	"GoConcurrency-Bootcamp-2022/controllers"
	"GoConcurrency-Bootcamp-2022/repositories"
	"GoConcurrency-Bootcamp-2022/use_cases"

	"github.com/gin-gonic/gin"
)

func Init() *gin.Engine {
	r := gin.Default()

	api := repositories.PokeAPI{}
	storage := repositories.LocalStorage{}
	cache := repositories.NewCache()

	refresher := use_cases.NewRefresher(storage, cache, api)
	svc := use_cases.NewFetcher(api, storage)

	ctrl := controllers.NewAPI(svc, refresher, cache)

	r.POST("/api/provide", ctrl.FillCSV)
	r.PUT("/api/refresh-cache", ctrl.RefreshCache)
	r.GET("/api/pokemons", ctrl.GetPokemons)

	return r
}
