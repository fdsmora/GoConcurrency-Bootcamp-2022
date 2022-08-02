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
	wg := sync.WaitGroup{}
	wg.Add(len(pokemons))
	for _, p := range pokemons {
		go func(p models.Pokemon) {
			pokeChan <- p
			wg.Done()
		}(p)
	}
	go func() {
		wg.Wait()
		close(pokeChan)
	}()

	return pokeChan
}

func doFanIn(inputChans []<-chan models.Pokemon) <-chan models.Pokemon {
	afterReadFanInChan := make(chan models.Pokemon)

	wg := sync.WaitGroup{}
	wg.Add(len(inputChans))

	for _, in := range inputChans {
		go func(in <-chan models.Pokemon) {
			for {
				pokemon, ok := <-in

				if !ok {
					wg.Done()
					break
				}

				afterReadFanInChan <- pokemon
			}
		}(in)
	}
	go func() {
		wg.Wait()
		close(afterReadFanInChan)
	}()

	return afterReadFanInChan
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

	for _, fanOut := range fanOuts {
		go func(fanOut <-chan models.Pokemon) {
			defer wg2.Done()
			for p := range fanOut {
				pokemonsFanIn <- p
			}
		}(fanOut)
	}

	go func() {
		wg2.Wait()
		close(pokemonsFanIn)
		close(errChan)
	}()

	// Tried first this approach but I didn't find a way to exit the for loop, causing a goroutine leak
	/* 	go func() error {
		for {
			select {
			case pokemon := <-fanout1:
				pokemon, err := r.loadPokemonWithAbility(pokemon)
				if err != nil {
					return fmt.Errorf("fetching ability from poke endpoint: %w", err)
				}
				pokemonsFanIn <- pokemon
			case pokemon := <-fanout2:
				pokemon, err = r.loadPokemonWithAbility(pokemon)
				if err != nil {
					return fmt.Errorf("fetching ability from poke endpoint: %w", err)
				}
				pokemonsFanIn <- pokemon
			case pokemon := <-fanout3:
				pokemon, err := r.loadPokemonWithAbility(pokemon)
				if err != nil {
					return fmt.Errorf("fetching ability from poke endpoint: %w", err)
				}
				pokemonsFanIn <- pokemon
			}
		}
	}() */

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
