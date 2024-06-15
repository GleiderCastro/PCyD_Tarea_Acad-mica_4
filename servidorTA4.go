package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

// Struc de los vectores
type Vector struct {
	Data []float64
}

type Request struct {
	Data          [][]float64
	K             int
	MaxIterations int
}

type Response struct {
	Centroids   [][]float64
	Assignments []int
}

// La función squaredDistance sirve para calcular la distancia euclidiana entre los vectores
func squaredDistance(p1, p2 *Vector) float64 {
	var sum float64
	for i := 0; i < len(p1.Data); i++ {
		sum += (p1.Data[i] - p2.Data[i]) * (p1.Data[i] - p2.Data[i])
	}
	return sum
}

// La función initializeCentroids sirve para inicializar los centroides de manera aleatoria
func initializeCentroids(data [][]float64, k int) [][]float64 {
	nSamples := len(data)
	centroids := make([][]float64, k)
	for i := 0; i < k; i++ {
		idx := rand.Intn(nSamples)
		centroids[i] = make([]float64, len(data[0]))
		copy(centroids[i], data[idx])
	}
	return centroids
}

// La función mean sirve para calcular la media de los vectores
func mean(vectors []*Vector) *Vector {
	n := len(vectors)
	meanVector := make([]float64, len(vectors[0].Data))
	for _, v := range vectors {
		for i := range v.Data {
			meanVector[i] += v.Data[i] / float64(n)
		}
	}
	return &Vector{meanVector}
}

// La función kMeans contiene el algoritmo de K-means
func kMeans(data [][]float64, k int, maxIterations int) ([][]float64, []int) {
	nSamples := len(data)
	centroids := initializeCentroids(data, k)
	assignments := make([]int, nSamples)
	vectors := make([]*Vector, nSamples)
	for i := range data {
		vectors[i] = &Vector{data[i]}
	}

	for iter := 0; iter < maxIterations; iter++ {
		assignChan := make(chan struct{ index, assignment int }, nSamples)
		updateChan := make(chan struct {
			index    int
			centroid []float64
		}, k)
		doneChan := make(chan bool)

		// Asignar puntos a los centroides más cercanos en paralelo
		go func() {
			var wg sync.WaitGroup
			wg.Add(nSamples)
			for i := 0; i < nSamples; i++ {
				go func(i int) {
					defer wg.Done()
					minDist := squaredDistance(vectors[i], &Vector{centroids[0]})
					assignment := 0
					for j := 1; j < k; j++ {
						dist := squaredDistance(vectors[i], &Vector{centroids[j]})
						if dist < minDist {
							minDist = dist
							assignment = j
						}
					}
					assignChan <- struct{ index, assignment int }{i, assignment}
				}(i)
			}
			wg.Wait()
			close(assignChan)
		}()

		go func() {
			for assignment := range assignChan {
				assignments[assignment.index] = assignment.assignment
			}
			doneChan <- true
		}()

		<-doneChan

		// Actualizar centroides en paralelo
		clusters := make([][]*Vector, k)
		for i := range clusters {
			clusters[i] = make([]*Vector, 0)
		}
		for i, idx := range assignments {
			clusters[idx] = append(clusters[idx], vectors[i])
		}

		go func() {
			var wg sync.WaitGroup
			wg.Add(k)
			for j := 0; j < k; j++ {
				go func(j int) {
					defer wg.Done()
					centroid := mean(clusters[j]).Data
					updateChan <- struct {
						index    int
						centroid []float64
					}{j, centroid}
				}(j)
			}
			wg.Wait()
			close(updateChan)
		}()

		go func() {
			for update := range updateChan {
				centroids[update.index] = update.centroid
			}
			doneChan <- true
		}()

		<-doneChan
	}

	return centroids, assignments
}

// Manejador de conexiones entrantes
func handleConnection(conn net.Conn) {
	defer conn.Close()

	var req Request
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&req); err != nil {
		fmt.Println("Error decoding request:", err)
		return
	}

	centroids, assignments := kMeans(req.Data, req.K, req.MaxIterations)

	resp := Response{Centroids: centroids, Assignments: assignments}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(&resp); err != nil {
		fmt.Println("Error encoding response:", err)
		return
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	listener, err := net.Listen("tcp", ":12345")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server listening on port 12345")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}
