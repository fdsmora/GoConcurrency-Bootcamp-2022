package use_cases

import (
	"fmt"
	"strings"
	"sync"

	"GoConcurrency-Bootcamp-2022/models"
)

type api interface {
	FetchPokemon(id int) (models.Pokemon, error)
}

type writer interface {
	Write(pokemons []models.Pokemon) error
}

type Fetcher struct {
	api     api
	storage writer
}

type PokeError struct {
	Err error
}

func NewFetcher(api api, storage writer) Fetcher {
	return Fetcher{api, storage}
}

func (f Fetcher) Fetch(from, to int) error {
	if from > to {
		return fmt.Errorf("'to' must be greater than or equal to 'from'")
	}

	var (
		n        = to - from + 1
		pokemons = make([]models.Pokemon, n)
		i        int
		errCh    = make(chan PokeError)
	)

	pokeGenerator := f.pokeGenerator(from, to, errCh)

	for i = 0; i < n; i++ {
		select {
		case p, ok := <-pokeGenerator:
			if !ok {
				break
			}
			pokemons[i] = p
		case pokeErr, ok := <-errCh:
			if !ok {
				break
			}
			return fmt.Errorf("error fetching pokemon from endpoint: %w", pokeErr.Err)
		}
	}

	return f.storage.Write(pokemons)
}

func (f Fetcher) pokeGenerator(from, to int, errCh chan<- PokeError) <-chan models.Pokemon {
	var (
		n        = to - from + 1
		pokemons = make(chan models.Pokemon, n)
		wg       = sync.WaitGroup{}
	)

	wg.Add(n)

	for i := from; i <= to; i++ {
		go func(id int) {
			defer wg.Done()

			pokemon, err := f.api.FetchPokemon(id)
			if err != nil {
				errCh <- PokeError{err}
				return
			}

			var flatAbilities []string
			for _, t := range pokemon.Abilities {
				flatAbilities = append(flatAbilities, t.Ability.URL)
			}
			pokemon.FlatAbilityURLs = strings.Join(flatAbilities, "|")
			pokemons <- pokemon
		}(i)
	}

	go func() {
		wg.Wait()
		close(pokemons)
		errCh <- PokeError{}
	}()
	return pokemons
}
