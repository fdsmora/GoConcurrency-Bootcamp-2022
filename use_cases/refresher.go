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
	inputs, err := r.Read()
	if err != nil {
		return fmt.Errorf("reading pokemons from CSV file: %w", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(len(inputs))

	out := make(chan models.Pokemon)
	for _, in := range inputs {
		go func(in <-chan models.Pokemon) {
			for {
				pokemon, ok := <-in

				if !ok {
					wg.Done()
					break
				}

				out <- pokemon
			}
		}(in)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	// tmp
	pokemons := []models.Pokemon{}
	for p := range out {
		pokemons = append(pokemons, p)
	}
	fmt.Println("ALL pokemons read:", pokemons)

	for i, p := range pokemons {
		urls := strings.Split(p.FlatAbilityURLs, "|")
		var abilities []string
		for _, url := range urls {
			ability, err := r.FetchAbility(url)
			if err != nil {
				return fmt.Errorf("fetching ability from poke endpoint: %w", err)
			}

			for _, ee := range ability.EffectEntries {
				abilities = append(abilities, ee.Effect)
			}
		}

		pokemons[i].EffectEntries = abilities
	}
	if err := r.Save(ctx, pokemons); err != nil {
		return fmt.Errorf("saving pokemons in cache: %w", err)
	}

	return nil
}
