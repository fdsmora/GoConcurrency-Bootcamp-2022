package controllers

import (
	"context"
	"fmt"
	"net/http"

	"GoConcurrency-Bootcamp-2022/models"

	"github.com/gin-gonic/gin"
)

type API struct {
	fetcher
	refresher
	getter
}

func NewAPI(fetcher fetcher, refresher refresher, getter getter) API {
	return API{fetcher, refresher, getter}
}

type fetcher interface {
	Fetch(from, to int) error
}

type refresher interface {
	Refresh(context.Context) error
}

type getter interface {
	GetPokemons(context.Context) ([]models.Pokemon, error)
}

//FillCSV fill the local CSV with data from PokeAPI. By default will fetch from id 1 to 10 unless there are other information on the body
func (api API) FillCSV(c *gin.Context) {

	requestBody := struct {
		From int `json:"from"`
		To   int `json:"to"`
	}{1, 10}

	if err := c.Bind(&requestBody); err != nil {
		c.Status(http.StatusBadRequest)
		fmt.Println(err)
		return
	}

	if err := api.Fetch(requestBody.From, requestBody.To); err != nil {
		c.Status(http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	c.Status(http.StatusOK)
}

//RefreshCache feeds the csv data and save in redis
func (api API) RefreshCache(c *gin.Context) {
	if err := api.Refresh(c); err != nil {
		fmt.Println("Internal server Error:", err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
}

//GetPokemons return all pokemons in cache
func (api API) GetPokemons(c *gin.Context) {
	pokemons, err := api.getter.GetPokemons(c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, pokemons)
}
