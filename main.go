package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"io"
	"log"
	"fmt"
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

type server struct {
	pb.UnimplementedGreeterServer
}

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
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				createServer()
			}()
			discovery()
			for i := range machines {
				log.Printf(machines[i])
			}
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
}

func search(hash string) {
	ips := machines

	var wg sync.WaitGroup
	for _, i := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()

			machine := ip + ":50051"
			conn, err := grpc.NewClient(machine, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v", err)
				return
			}
			defer conn.Close()

			c := pb.NewGreeterClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			r, err := c.SayHello(ctx, &pb.HelloRequest{Name: *flag.String(ip, hash, "Name to greet")})
			if err != nil {
				log.Fatalf("could not greet: %v", err)
			}
			message := r.GetMessage()
			if strings.Contains(message, "file") {
				log.Printf("RECEIVED FILE:\t%s", message)
			} else {
				log.Printf("RECEIVED NOTHING:\t%s", message)
			}
		}(i)
	}
	wg.Wait()
}

func discovery() {
    ipChan := make(chan string)
    ips := scanSubnet(ipChan)
	machines = ips
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

func scanSubnet(ipChan chan string) []string {
	interfaces, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalf("Erro ao receber interfaces %v", err)
		return nil
	}

    var ipv4Interfaces []net.Addr
    for _, i := range interfaces {
        if addr, ok := i.(*net.IPNet); ok && addr.IP.To4() != nil {
            ipv4Interfaces = append(ipv4Interfaces, i)
        }
    }

    var wg sync.WaitGroup
	for _, i := range ipv4Interfaces {
        wg.Add(1)
        go func () {
            defer wg.Done()
            switch v := i.(type) {
            case *net.IPNet:
                if v.IP.To4() != nil {
                    ip := v.IP.To4()
                    mask := v.Mask
                    subnet := ip.Mask(mask)

                    for i := 0; i <= 255-1; i++ {
                        ip := subnet.To4()
                        ip[3] = byte(i)
                        host := ip.String()
                        if isPortOpen(host, 50051) {
                            ipChan <- host
                        }                     
                    }
                }
            }
        }()
	}
    go func(){
        wg.Wait()
        close(ipChan)
    }()
    var ips []string
    for ip := range ipChan {
        ips = append(ips, ip)
    }
    return ips
}

func isPortOpen(host string, port int) bool {
	timeout := time.Millisecond * 10
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
