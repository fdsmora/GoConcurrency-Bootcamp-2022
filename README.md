# GoConcurrency-Bootcamp-2022 challenge by Fausto Salazar
  
## Patterns implemented

| Endpoint                   | Pattern applied |
|----------------------------|-----------------|
| `(POST) /api/provide`      | generator       |
| `(PUT) /api/refresh-cache` | fan-out/fan-in  |
 
## Execution times comparison table 

| Endpoint                   | Number of records | Exec time before solution | Exec time after solution | Notes                                                    |
|----------------------------|-------------------|---------------------------|--------------------------|----------------------------------------------------------|
| `(POST) /api/provide`      | 100               | 25.423223773s             | 964.983809ms             | 1 goroutine for each Poke API request                    |
| `(PUT) /api/refresh-cache` | 40                | 16.48707516s              | 5.998939774s             | 3 fan-out workers that fetch abilities from the Poke API |