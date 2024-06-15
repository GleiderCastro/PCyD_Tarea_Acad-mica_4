package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
)

type Request struct {
	Data          [][]float64
	K             int
	MaxIterations int
}

type Response struct {
	Centroids   [][]float64
	Assignments []int
}

// Leer dataset desde archivo CSV
func readDataset(filename string) ([][]float64, []string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, err
	}

	data := make([][]float64, len(records)-1)
	ids := make([]string, len(records)-1)
	for i := 1; i < len(records); i++ {
		data[i-1] = make([]float64, len(records[i])-1)
		ids[i-1] = records[i][0]
		for j := 1; j < len(records[i]); j++ {
			data[i-1][j-1], err = strconv.ParseFloat(records[i][j], 64)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	return data, ids, nil
}

// Guardar resultados en archivo CSV
func saveResults(filename string, ids []string, data [][]float64, assignments []int) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"ID", "Frecuencia", "GastoT", "DiasSinCompra", "VariedadDeProductos", "Cluster"}
	writer.Write(header)

	for i := range ids {
		record := make([]string, len(data[i])+2)
		record[0] = ids[i]
		for j := range data[i] {
			record[j+1] = fmt.Sprintf("%.2f", data[i][j])
		}
		record[len(data[i])+1] = strconv.Itoa(assignments[i])
		writer.Write(record)
	}

	return nil
}

func main() {
	inputFilename := "dataset.csv"
	outputFilename := "resultados.csv"
	k := 3
	maxIterations := 100

	data, ids, err := readDataset(inputFilename)
	if err != nil {
		fmt.Println("Error al leer el dataset:", err)
		return
	}

	conn, err := net.Dial("tcp", "localhost:12345")
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	req := Request{Data: data, K: k, MaxIterations: maxIterations}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(&req); err != nil {
		fmt.Println("Error encoding request:", err)
		return
	}

	var resp Response
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		fmt.Println("Error decoding response:", err)
		return
	}

	err = saveResults(outputFilename, ids, data, resp.Assignments)
	if err != nil {
		fmt.Println("Error al guardar los resultados:", err)
		return
	}

	fmt.Println("Resultados guardados correctamente en", outputFilename)
	fmt.Println("Centroides:")
	for _, c := range resp.Centroids {
		fmt.Println(c)
	}
}
