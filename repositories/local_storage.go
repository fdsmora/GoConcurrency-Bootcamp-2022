package repositories

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"GoConcurrency-Bootcamp-2022/models"
)

type LocalStorage struct{}

const filePath = "resources/pokemons.csv"

func (l LocalStorage) Write(pokemons []models.Pokemon) error {
	file, fErr := os.Create(filePath)
	defer file.Close()
	if fErr != nil {
		return fErr
	}

	w := csv.NewWriter(file)
	records := buildRecords(pokemons)
	if err := w.WriteAll(records); err != nil {
		return err
	}

	return nil
}

func (l LocalStorage) Read() ([]<-chan models.Pokemon, error) {
	file, fErr := os.Open(filePath)
	defer file.Close()
	if fErr != nil {
		return nil, fErr
	}

	r := csv.NewReader(file)
	records, rErr := r.ReadAll()
	if rErr != nil {
		return nil, rErr
	}

	inChannels, err := parseCSVData(records)
	if err != nil {
		return nil, err
	}

	return inChannels, nil
}

func buildRecords(pokemons []models.Pokemon) [][]string {
	headers := []string{"id", "name", "height", "weight", "flat_abilities"}
	records := [][]string{headers}

	for _, p := range pokemons {
		record := fmt.Sprintf("%d,%s,%d,%d,%s",
			p.ID,
			p.Name,
			p.Height,
			p.Weight,
			p.FlatAbilityURLs)
		records = append(records, strings.Split(record, ","))
	}

	return records
}

func parseCSVData(records [][]string) ([]<-chan models.Pokemon, error) {
	var (
		inChannels []<-chan models.Pokemon

		//wg = sync.WaitGroup{}
		totalRecords = len(records) - 1 // discount headers
	)

	if totalRecords <= 0 {
		return inChannels, nil
	}

	pokeWorker := func(records [][]string, start, end int) (<-chan models.Pokemon, error) {
		in := make(chan models.Pokemon)
		go func(in chan<- models.Pokemon) error {
			fmt.Println("about to start parsing segment from", start, "to", end)

			for i := start; i < end; i++ {
				pokemon, err := parsePokemon(records[i])
				fmt.Println("pokemon parsed")
				if err != nil {
					fmt.Printf("Error found: %s\n", err.Error())
					return err
				}
				fmt.Println("about to push pokemon into channel")
				in <- pokemon
				fmt.Println("pokemon pushed into channel")
			}
			close(in)
			fmt.Println("channel closed")
			return nil
		}(in)
		return in, nil
	}

	inChannels = []<-chan models.Pokemon{}

	// If more than 3 pokemons parse CSV in 3 segments, otherwise use 1 worker for whole CSV
	if totalRecords >= 3 {
		part := totalRecords / 3
		inCh1, err := pokeWorker(records, 1, part+1)
		if err != nil {
			return nil, err
		}
		inCh2, err := pokeWorker(records, part+1, part*2+1)
		if err != nil {
			return nil, err
		}
		inCh3, err := pokeWorker(records, part*2+1, len(records))
		if err != nil {
			return nil, err
		}
		inChannels = append(inChannels, inCh1, inCh2, inCh3)
	} else {
		inCh1, err := pokeWorker(records, 1, len(records))
		if err != nil {
			return nil, err
		}
		inChannels = append(inChannels, inCh1)
	}

	return inChannels, nil
}

func parsePokemon(record []string) (pokemon models.Pokemon, err error) {
	id, err := strconv.Atoi(record[0])
	if err != nil {
		return
	}

	height, err := strconv.Atoi(record[2])
	if err != nil {
		return
	}

	weight, err := strconv.Atoi(record[3])
	if err != nil {
		return
	}

	pokemon = models.Pokemon{
		ID:              id,
		Name:            record[1],
		Height:          height,
		Weight:          weight,
		Abilities:       nil,
		FlatAbilityURLs: record[4],
		EffectEntries:   nil,
	}

	return
}
