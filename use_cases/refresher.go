package use_cases

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"GoConcurrency-Bootcamp-2022/models"
)

type reader interface {
	Read() ([]<-chan models.Pokemon, error)
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
	csvChannels, err := r.Read()
	if err != nil {
		return fmt.Errorf("reading pokemons from CSV file: %w", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(len(csvChannels))

	afterReadFanInChan := make(chan models.Pokemon)

	for _, in := range csvChannels {
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

	// Now fan out to load abilities from 3 concurrent workers

	var (
		fanout1 = makeFanOut(afterReadFanInChan)
		fanout2 = makeFanOut(afterReadFanInChan)
		fanout3 = makeFanOut(afterReadFanInChan)
	)

	fanOuts := []<-chan models.Pokemon{fanout1, fanout2, fanout3}

	pokemonsFanIn := make(chan models.Pokemon) // For multiplexing again

	wg2 := sync.WaitGroup{}
	wg2.Add(len(fanOuts))
	for _, fanOut := range fanOuts {
		go func(fanOut <-chan models.Pokemon) error {
			defer wg2.Done()
			for pokemon := range fanOut {
				fmt.Println("Popped pokemon frm fanOut. About to load abilities")
				pokemon, err := r.loadPokemonWithAbility(pokemon)
				if err != nil {
					return fmt.Errorf("fetching ability from poke endpoint: %w", err)
				}
				pokemonsFanIn <- pokemon
				fmt.Println("pokemon sent over pokemonsFanIn")
			}
			return nil
		}(fanOut)
	}

	go func() {
		wg2.Wait()
		close(pokemonsFanIn)
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

	pokemons := []models.Pokemon{}
	for p := range pokemonsFanIn {
		pokemons = append(pokemons, p)

	}

	if err := r.Save(ctx, pokemons); err != nil {
		return fmt.Errorf("saving pokemons in cache: %w", err)
	}

	return nil
}

func makeFanOut(source <-chan models.Pokemon) <-chan models.Pokemon {
	fanOut := make(chan models.Pokemon)
	go func() {
		defer close(fanOut)
		for p := range source {
			fanOut <- p
		}
	}()
	return fanOut
}

func (r Refresher) loadPokemonWithAbility(p models.Pokemon) (models.Pokemon, error) {
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
