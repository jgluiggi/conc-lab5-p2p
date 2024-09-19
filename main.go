package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	pb "github.com/jgluiggi/conc-lab5-p2p/helloworld"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"
)

// createServer is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	hash := in.GetName()
	res := ""
	for _, it := range hashes {
		if it.Hash == hash {
			res = it.FilePath
		}
	}

	p, _ := peer.FromContext(ctx)
	log.Printf("RECEIVED: %v FROM %v", in.GetName(), p.Addr.String())
	return &pb.HelloReply{Message: res + " " + p.LocalAddr.String()}, nil
}

type LocalFile struct {
	FilePath string
	Hash     string
}

const (
	dirPath = "./tmp/dataset"
)

var (
	hashes   []LocalFile
	machines []string
)

func main() {
	args := os.Args[1:]

	if len(args) >= 1 {

		if len(args) == 1 && args[0] == "server" {
			discovery()
			for i := range machines {
				log.Printf(machines[i])
			}
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				createServer()
			}()
			wg.Wait()
		} else if len(args) == 2 && args[0] == "search" {
			search(args[1])
		} else if len(args) == 1 && args[0] == "discovery" {
			discovery()
		} else {
			log.Fatalf("Comando inválido")
		}

	} else {
		log.Fatalf("Comando inválido")
	}
}

func createServer() {
	hashes = generateHashes()

	for _, it := range hashes {
		log.Printf(it.FilePath + " " + it.Hash + "\n")
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

	// TODO CRIAR ENDPOINTS

	// o server deve receber nome de arquivo e retorna se existe ou não
	// o server deve listar seus arquivos
}

func search(hash string) {
	var ips []string
	file, err := os.Open("cache.txt")
	if err == nil {
		defer file.Close()
		byteValue, _ := io.ReadAll(file)
		ips = strings.Split(string(byteValue), "\n")
	} else {
		discovery()
	}

	for _, i := range ips {
		machine := i + ":50051"
		conn, err := grpc.NewClient(machine, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}

		//log.Printf("conn.GetState(): %v\n", conn.GetState())

		defer conn.Close()

		c := pb.NewGreeterClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		r, err := c.SayHello(ctx, &pb.HelloRequest{Name: *flag.String(i, hash, "Name to greet")})
		if err != nil {
			log.Fatalf("could not greet: %v", err)
		}
		message := r.GetMessage()
		if strings.Contains(message, "file") {
			log.Printf("RECEIVED FILE:\t%s", message)
		} else {
			log.Printf("RECEIVED NOTHING:\t%s", message)
		}
	}

	// TODO CRIAR CHAMADAS

	// o cliente deve chamar o server
	// o cliente deve perguntar se arquivo existe a partir da hash
}

func discovery() {
	ips, err := getSubnetMachines()
	if err != nil {
		log.Fatal(err)
	}
	machines = ips

	f, _ := os.Create("cache.txt")
	defer f.Close()
	for _, ip := range ips {
		f.WriteString(ip + "\n")
	}
}

func generateHashes() []LocalFile {
	dir, err := os.Open(dirPath)
	if err != nil {
		log.Fatalf("Erro ao abrir diretório: %s", err)
	}
	defer dir.Close()

	files, err := dir.Readdir(-1)
	if err != nil {
		log.Fatalf("Erro ao abrir diretório: %s", err)
	}

	lfCh := make(chan LocalFile, 10)
	for _, f := range files {
		if !f.IsDir() {
			filePath := dirPath + "/" + f.Name()
			go generateHash(lfCh, filePath)
		}
	}

	hashes := make([]LocalFile, len(files))
	for i := range files {
		hashes[i] = <-lfCh
	}
	close(lfCh)

	return hashes
}

func generateHash(lfCh chan LocalFile, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Erro ao abrir arquivo: %v", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Erro ao abrir arquivo: %s", err)
	}

	hash := sha256.New()
	hash.Write(content)
	encoded := hex.EncodeToString(hash.Sum(nil))

	lfCh <- LocalFile{FilePath: filePath, Hash: encoded}
}

func getSubnetMachines() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var ips []string
	for _, i := range interfaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}

		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil {
				if ip.To4() != nil {
					ips = append(ips, ip.String())
				}
			}
		}
	}

	return ips, nil
}
