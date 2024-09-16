package main

import (
	"io"
	"log"
	"os"
	//"google.golang.org/grpc"
)

type LocalFile struct {
	FilePath string
	Hash     int
}

const (
	dirPath = "./tmp/dataset"
)

func main() {
	lfCh := make(chan LocalFile, 10)
	go generateHashes(lfCh)
	for lf := range lfCh {
		log.Printf(lf.FilePath)
	}

	// GRPC ENDPOINTS
}

func generateHashes(lfCh chan LocalFile) {
	dir, err := os.Open(dirPath)
	if err != nil {
		log.Fatalf("Erro ao abrir diretório: %s", err)
	}
	defer dir.Close()

	files, err := dir.Readdir(-1)
	if err != nil {
		log.Fatalf("Erro ao abrir diretório: %s", err)
	}

	finishedCh := make(chan bool, 10)
	for _, f := range files {
		if !f.IsDir() {
			filePath := dirPath + "/" + f.Name()
			go generateHash(lfCh, filePath, finishedCh)
		}
	}

	for range files {
		<-finishedCh
	}
	close(lfCh)
}

func generateHash(lfCh chan LocalFile, filePath string, finishedCh chan bool) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Erro ao abrir arquivo: %v", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Erro ao abrir arquivo: %s", err)
	}

	// TODO calcular hash

	hash := len(string(content))

	lfCh <- LocalFile{FilePath: filePath, Hash: hash}
	finishedCh <- true
}
