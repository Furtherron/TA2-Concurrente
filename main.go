package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"encoding/csv"
	"fmt"
	"math"
	"sort"
	"sync"
)

var tpl *template.Template

func init(){
	tpl = template.Must((template.ParseGlob("templates/*")))
}


type Parametros struct {
	KNearest     string
	GRUPO        string 
	EDAD         string
	SEXO         string
	DOSIS        string
	UBIGEO       string
	Eleccion     string
	RESULTADO    sortedClassVotes	
	 
}
func foo(w http.ResponseWriter, req *http.Request){
	k := req.FormValue("KNearest")
	g := req.FormValue("GRUPO")
	e := req.FormValue("EDAD")
	s := req.FormValue("SEXO")
	d := req.FormValue("DOSIS")
	u := req.FormValue("UBIGEO")

	kint , _ := strconv.Atoi(k)
	gf ,_ := strconv.ParseFloat(g, 64)
	ef ,_ := strconv.ParseFloat(e, 64)
	sf ,_ := strconv.ParseFloat(s, 64)
	df ,_ := strconv.ParseFloat(d, 64)
	uf ,_ := strconv.ParseFloat(u, 64)

	
	
    
	

	m := make(chan sortedClassVotes)
	wg := sync.WaitGroup{}
	wg.Add(1)

	
	go Data(kint,gf,ef,sf,df,uf,m) 

	wg.Done()

	fin := <-m

	resulta := fin

	

	err := tpl.ExecuteTemplate(w,"index.gohtml",Parametros{k,g,e,s,d,u,resulta[0].key,resulta})

	if err != nil{
		http.Error(w,err.Error(),500)
		log.Fatalln(err)
	}

	close(m)

	

	


	
}
func main(){
	http.Handle("/favicon.ico", http.NotFoundHandler())
	http.HandleFunc("/",foo)
	http.ListenAndServe(":8080", nil)
	<-end
}


type Vacunacion struct {
	GRUPO_RIESGO float64 
	EDAD         float64 
	SEXO         float64 
	DOSIS        float64 
	UBIGEO       float64 
	FABRICANTE   string  
}

// Lectura de dataset
func readCSVFromUrl(url string) ([][]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	reader := csv.NewReader(resp.Body)
	reader.Comma = ','

	data, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	return data, nil
}
//Conversion de la estructura
func parseVacunacion(record []string) Vacunacion {
	var vacuna Vacunacion

	vacuna.GRUPO_RIESGO, _ = strconv.ParseFloat(record[0], 64)
	vacuna.EDAD, _ = strconv.ParseFloat(record[1], 64)
	vacuna.SEXO, _ = strconv.ParseFloat(record[2], 64)
	vacuna.DOSIS, _ = strconv.ParseFloat(record[3], 64)
	vacuna.UBIGEO, _ = strconv.ParseFloat(record[4], 64)
	vacuna.FABRICANTE = record[5]

	return vacuna
}

type classVote struct {
	key   string
	value int
}

type sortedClassVotes []classVote

func (scv sortedClassVotes) Len() int           { return len(scv) }
func (scv sortedClassVotes) Less(i, j int) bool { return scv[i].value < scv[j].value }
func (scv sortedClassVotes) Swap(i, j int)      { scv[i], scv[j] = scv[j], scv[i] }

func getResponse(neighbors []Vacunacion) sortedClassVotes {
	classVotes := make(map[string]int)

	for x := range neighbors {
		response := neighbors[x].FABRICANTE
		if contains(classVotes, response) {
			classVotes[response] += 1
		} else {
			classVotes[response] = 1
		}
	}

	scv := make(sortedClassVotes, len(classVotes))
	i := 0
	for k, v := range classVotes {
		scv[i] = classVote{k, v}
		i++
	}

	sort.Sort(sort.Reverse(scv))
	return scv
}

type distancePair struct {
	record   Vacunacion
	distance float64
}

type distancePairs []distancePair

func (slice distancePairs) Len() int           { return len(slice) }
func (slice distancePairs) Less(i, j int) bool { return slice[i].distance < slice[j].distance }
func (slice distancePairs) Swap(i, j int)      { slice[i], slice[j] = slice[j], slice[i] }

func getNeighbors(trainingSet []Vacunacion, testRecord Vacunacion, k int, c chan []Vacunacion) []Vacunacion {
	var distances distancePairs
	for i := range trainingSet {
		dist := Manhattan(testRecord, trainingSet[i])
		distances = append(distances, distancePair{trainingSet[i], dist})
	}

	sort.Sort(distances)

	var neighbors []Vacunacion

	for x := 0; x < k; x++ {
		neighbors = append(neighbors, distances[x].record)
	}

	c <- neighbors

	return neighbors

}

func Manhattan(instanceOne Vacunacion, instanceTwo Vacunacion) float64 {
	var distance float64

	distance += math.Abs(instanceOne.GRUPO_RIESGO - instanceTwo.GRUPO_RIESGO) +
	math.Abs(instanceOne.EDAD - instanceTwo.EDAD) + 
	math.Abs(instanceOne.SEXO - instanceTwo.SEXO) +
	math.Abs(instanceOne.DOSIS - instanceTwo.DOSIS)+
	math.Abs(instanceOne.UBIGEO - instanceTwo.UBIGEO)

	return distance
}

func errHandle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func contains(votesMap map[string]int, name string) bool {
	for s, _ := range votesMap {
		if s == name {
			return true
		}
	}

	return false
}

var end chan bool

func Data(k int, Grupo float64, Edad float64, Sexo float64, Dosis float64, Ubigeo float64,m chan sortedClassVotes ) {
	url := "https://raw.githubusercontent.com/Furtherron/TA2-Concurrente/main/Vacunacion.csv"

	var recordSet []Vacunacion

	data, err := readCSVFromUrl(url)

	if err != nil {
		panic(err)
	}

	for idx, row := range data {
		if idx == 0 {
			continue
		}

		recordSet = append(recordSet, parseVacunacion(row))

	}
	var testSet []Vacunacion
	var trainSet []Vacunacion
	for i := range recordSet {

		trainSet = append(trainSet, recordSet[i])

	}


	
	end = make(chan bool)

	testSet = append(testSet, Vacunacion{Grupo,Edad,Sexo,Dosis,Ubigeo,""})

	c := make(chan []Vacunacion)

	go getNeighbors(trainSet, testSet[0], k, c)

	neighbors := <-c
	result := getResponse(neighbors)
	

	fmt.Printf("Actual: %s\n", result[0].key)


	fmt.Println(result)

	m <- result




	

}