package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"time"

	pb "github.com/jgluiggi/conc-lab5-p2p/helloworld"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// createServer is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(_ context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

type LocalFile struct {
	FilePath string
	Hash     string
}

const (
	dirPath = "./tmp/dataset"
)

func main() {
	args := os.Args[1:]

	if len(args) >= 1 {

		if len(args) == 1 && args[0] == "server" {
			createServer()
		} else if len(args) == 2 && args[0] == "search" {
			search(args[1])
		} else if len(args) == 1 && args[1] == "discovery" {
			discovery()
		} else {
			log.Fatalf("Comando inválido")
		}

	} else {
		log.Fatalf("Comando inválido")
	}
}

func createServer() {
	lfCh := make(chan LocalFile, 10)
	go generateHashes(lfCh)
	for lf := range lfCh {
		log.Printf(lf.FilePath)
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func search(hash string) {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	log.Printf("conn.GetState(): %v\n", conn.GetState())

	defer conn.Close()

	c := pb.NewGreeterClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: *flag.String("name", "world", "Name to greet")})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.GetMessage())
}

func discovery() {

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

	hash := string(len(string(content)))

	lfCh <- LocalFile{FilePath: filePath, Hash: hash}
	finishedCh <- true
}
