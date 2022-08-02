package use_cases

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"GoConcurrency-Bootcamp-2022/models"
)

type reader interface {
	Read() ([]models.Pokemon, error)
}

type saver interface {
	Save(context.Context, []models.Pokemon) error
}

type fetcher interface {
	FetchAbility(string) (models.Ability, error)
}

type Refresher struct {
	reader
	saver
	fetcher
}

func NewRefresher(reader reader, saver saver, fetcher fetcher) Refresher {
	return Refresher{reader, saver, fetcher}
}

func (r Refresher) Refresh(ctx context.Context) error {
	pokemonsFromFile, err := r.Read()
	if err != nil {
		return fmt.Errorf("reading pokemons from CSV file: %w", err)
	}

	// fan-in/fan-out
	sourceChan := pokeGenerator(pokemonsFromFile)
	pokemonsWithAbilitiesFanInChan, errChan := r.loadAbilitiesWithFanOutFanIn(sourceChan)

	pokemons, err := createPokemonList(pokemonsWithAbilitiesFanInChan, errChan)
	if err != nil {
		return err
	}

	if err := r.Save(ctx, pokemons); err != nil {
		return fmt.Errorf("saving pokemons in cache: %w", err)
	}

	return nil
}

func pokeGenerator(pokemons []models.Pokemon) <-chan models.Pokemon {
	pokeChan := make(chan models.Pokemon)

	go func() {
		defer close(pokeChan)

		for _, p := range pokemons {
			pokeChan <- p
		}
	}()

	return pokeChan
}

func (r Refresher) loadAbilitiesWithFanOutFanIn(source <-chan models.Pokemon) (
	<-chan models.Pokemon, <-chan PokeError) {

	errChan := make(chan PokeError)

	fanOuts := []<-chan models.Pokemon{
		r.makeFanOut(source, errChan),
		r.makeFanOut(source, errChan),
		r.makeFanOut(source, errChan),
	}
	pokemonsFanIn := make(chan models.Pokemon) // For multiplexing resulting pokemons

	wg2 := sync.WaitGroup{}
	wg2.Add(len(fanOuts))

	/* An alternative implementation of the 'fanout' without using a select statement.
	Each goroutinge triggered in each loop is equivalent for a case in the select. */
	for _, fanOut := range fanOuts {
		go func(fanOut <-chan models.Pokemon, pokemonsFanIn chan models.Pokemon) {
			defer wg2.Done()
			for p := range fanOut {
				pokemonsFanIn <- p
			}
		}(fanOut, pokemonsFanIn)
	}

	go func() {
		wg2.Wait()
		close(pokemonsFanIn)
		close(errChan)
	}()

	return pokemonsFanIn, errChan
}

func (r Refresher) makeFanOut(source <-chan models.Pokemon, errChan chan<- PokeError) <-chan models.Pokemon {
	var (
		fanOut = make(chan models.Pokemon)
		err    error
	)

	go func() {
		defer close(fanOut)
		for p := range source {
			p, err = r.fetchPokemonWithAbility(p)
			if err != nil {
				errChan <- PokeError{err}

				return // abort processing after any error
			}
			fanOut <- p

		}
	}()

	return fanOut
}

func (r Refresher) fetchPokemonWithAbility(p models.Pokemon) (models.Pokemon, error) {
	urls := strings.Split(p.FlatAbilityURLs, "|")
	var abilities []string
	for _, url := range urls {
		ability, err := r.FetchAbility(url)
		if err != nil {
			return models.Pokemon{}, fmt.Errorf("fetching ability from poke endpoint: %w", err)
		}

		for _, ee := range ability.EffectEntries {
			abilities = append(abilities, ee.Effect)
		}
	}

	p.EffectEntries = abilities
	return p, nil
}

func createPokemonList(source <-chan models.Pokemon, errChan <-chan PokeError) ([]models.Pokemon, error) {
	pokemons := []models.Pokemon{}
	for {
		select {
		case p, ok := <-source:
			if !ok {
				return pokemons, nil
			}
			pokemons = append(pokemons, p)
		case pokeErr := <-errChan:
			return nil, pokeErr.Err
		}
	}
}
