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
	go shareFiles(lfCh)
	for lf := range lfCh {
		log.Printf(lf.FilePath)
	}
}

func shareFiles(lfCh chan LocalFile) {
	dir, err := os.Open(dirPath)
	if err != nil {
		log.Fatalf("Erro ao abrir diretório: %s", err)
		return
	}
	defer dir.Close()

	files, err := dir.Readdir(-1)
	if err != nil {
		log.Fatalf("Erro ao abrir diretório: %s", err)
		return
	}

	for _, f := range files {
		if !f.IsDir() {
			filePath := dirPath + "/" + f.Name()
			hash := generateHash(filePath)
			lfCh <- LocalFile{FilePath: filePath, Hash: hash}
		}
	}
	close(lfCh)
}

func generateHash(filePath string) int {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Erro ao abrir arquivo: %v", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Erro ao abrir arquivo: %s", err)
		return 0
	}

	// TODO calcular hash

	return len(string(content))
}
