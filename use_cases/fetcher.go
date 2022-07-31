package use_cases

import (
	"strings"
	"sync"

	"GoConcurrency-Bootcamp-2022/models"
)

type api interface {
	FetchPokemon(id int) (models.Pokemon, error)
}

type writer interface {
	Write(pokemons <-chan models.Pokemon) error
}

type Fetcher struct {
	api     api
	storage writer
}

func NewFetcher(api api, storage writer) Fetcher {
	return Fetcher{api, storage}
}

func (f Fetcher) Fetch(from, to int) error {
	return f.storage.Write(
		f.pokeGenerator(from, to),
	)
}

func (f Fetcher) pokeGenerator(from, to int) <-chan models.Pokemon {
	var (
		n        = to - from + 1
		pokemons = make(chan models.Pokemon, n)
		wg       = sync.WaitGroup{}
	)

	wg.Add(n)

	for i := from; i <= to; i++ {
		id := i
		go func() error {
			defer wg.Done()

			pokemon, err := f.api.FetchPokemon(id)
			if err != nil {
				return err
			}

			var flatAbilities []string
			for _, t := range pokemon.Abilities {
				flatAbilities = append(flatAbilities, t.Ability.URL)
			}
			pokemon.FlatAbilityURLs = strings.Join(flatAbilities, "|")
			pokemons <- pokemon
			return nil
		}()
	}

	go func() {
		wg.Wait()
		close(pokemons)
	}()

	return pokemons
}
