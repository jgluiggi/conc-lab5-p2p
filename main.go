package main

import (
    "log"
    "os"
	//"google.golang.org/grpc"
)

const (
    dir = "/tmp/dataset"
)

func main() {
    shareFiles()
}

func searchFiles() {
}

func shareFiles() {
    d, err := os.Open(dir)
    if err != nil {
        log.Fatalf("Erro ao abrir diretório: ", err)
        return
    }
    defer d.Close()

    files, err := d.Readdir(-1)
    if err != nil {
        log.Fatalf("Erro ao abrir diretório: ", err)
        return
    }

    for _, file := range files {
        if file.IsDir() {
            log.Printf("Diretório: %s\n", file.Name())
        } else {
            log.Printf("Arquivo: %s\n", file.Name())
        }
    }
}
