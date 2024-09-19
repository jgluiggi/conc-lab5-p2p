package main

import (
	"context"
	"crypto/sha256" // Biblioteca para criar hashes SHA-256
	"encoding/hex"  // Biblioteca para converter bytes em uma string hexadecimal
	"flag"          // Biblioteca para analisar argumentos da linha de comando
	"fmt"           // Biblioteca de formatação de strings
	"io"            // Biblioteca de entrada/saída
	"log"           // Biblioteca de logging
	"net"           // Biblioteca de rede para conexões TCP/IP
	"os"            // Biblioteca para operações com o sistema operacional, como ler arquivos e argumentos
	"strings"       // Biblioteca para manipulação de strings
	"sync"          // Biblioteca de sincronização para concorrência
	"time"          // Biblioteca de tempo

	pb "github.com/jgluiggi/conc-lab5-p2p/helloworld" // Importa o pacote gerado pelo protocolo gRPC
	"google.golang.org/grpc"                          // Pacote gRPC para comunicação
	"google.golang.org/grpc/credentials/insecure"     // gRPC sem criptografia (somente para teste)
	"google.golang.org/grpc/peer"                     // Utilizado para obter informações sobre o cliente que se conecta ao servidor
)

// Definição do servidor que implementa a interface do gRPC
type server struct {
	pb.UnimplementedGreeterServer // Necessário para implementar o servidor do gRPC
}

// Método SayHello, recebe um nome e retorna o caminho de arquivo associado ao hash, se existir
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	hash := in.GetName() // Obtém o nome (hash) enviado pelo cliente
	res := ""
	// Procura o hash recebido na lista de arquivos locais
	for _, it := range hashes {
		if it.Hash == hash {
			res = it.FilePath // Se o hash for encontrado, salva o caminho do arquivo
		}
	}

	// Obtém as informações do cliente (endereço IP, porta, etc.)
	p, _ := peer.FromContext(ctx)
	log.Printf("RECEIVED: %v FROM %v", in.GetName(), p.Addr.String())     // Log do nome recebido e endereço do cliente
	return &pb.HelloReply{Message: res + " " + p.LocalAddr.String()}, nil // Retorna a resposta contendo o caminho do arquivo e o endereço local do servidor
}

// Estrutura para armazenar o caminho do arquivo e o hash gerado
type LocalFile struct {
	FilePath string
	Hash     string
}

const (
	dirPath = "./tmp/dataset" // Caminho do diretório onde os arquivos locais estão armazenados
)

var (
	hashes   []LocalFile // Lista de arquivos locais com seus hashes
	machines []string    // Lista de endereços IP de outras máquinas encontradas
)

func main() {
	args := os.Args[1:] // Argumentos da linha de comando, exceto o nome do programa

	// Verifica se há argumentos
	if len(args) >= 1 {
		// Se o argumento for "server", inicia o servidor e o discovery (descoberta de outras máquinas)
		if len(args) == 1 && args[0] == "server" {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				createServer() // Cria o servidor
			}()
			discovery() // Executa o processo de descoberta
			for i := range machines {
				log.Printf(machines[i]) // Exibe as máquinas encontradas
			}
			wg.Wait()
		} else if len(args) == 2 && args[0] == "search" {
			search(args[1]) // Executa a função de busca para o hash fornecido
		} else if len(args) == 1 && args[0] == "discovery" {
			discovery() // Somente executa a descoberta de máquinas
		} else {
			log.Fatalf("Comando inválido") // Comando não reconhecido
		}

	} else {
		log.Fatalf("Comando inválido") // Nenhum comando foi fornecido
	}
}

// Função para criar o servidor gRPC
func createServer() {
	hashes = generateHashes() // Gera os hashes dos arquivos locais

	// Exibe o caminho dos arquivos e seus hashes
	for _, it := range hashes {
		log.Printf(it.FilePath + " " + it.Hash + "\n")
	}

	lis, err := net.Listen("tcp", ":50051") // Escuta na porta 50051
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()                            // Cria um novo servidor gRPC
	pb.RegisterGreeterServer(s, &server{})           // Registra o serviço Greeter no servidor
	log.Printf("server listening at %v", lis.Addr()) // Log do endereço em que o servidor está escutando
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err) // Inicia o servidor gRPC
	}
}

// Função para buscar o hash em outras máquinas
func search(hash string) {
	ips := machines // Usa a lista de máquinas encontradas no discovery

	var wg sync.WaitGroup
	// Para cada IP, cria uma nova goroutine para tentar encontrar o arquivo
	for _, i := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()

			machine := ip + ":50051"                                                                       // Porta padrão usada pelo servidor
			conn, err := grpc.NewClient(machine, grpc.WithTransportCredentials(insecure.NewCredentials())) // Conecta-se ao servidor gRPC da máquina
			if err != nil {
				log.Printf("did not connect: %v", err)
				return
			}
			defer conn.Close()

			c := pb.NewGreeterClient(conn) // Cria um cliente gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			r, err := c.SayHello(ctx, &pb.HelloRequest{Name: *flag.String(ip, hash, "Name to greet")}) // Envia o hash e recebe a resposta
			if err != nil {
				log.Fatalf("could not greet: %v", err)
			}
			message := r.GetMessage()
			if strings.Contains(message, "file") {
				log.Printf("RECEIVED FILE:\t%s", message) // Se encontrou o arquivo, loga o caminho
			} else {
				log.Printf("RECEIVED NOTHING:\t%s", message) // Caso contrário, indica que nada foi encontrado
			}
		}(i)
	}
	wg.Wait()
}

// Função para descobrir outras máquinas na rede local
func discovery() {
	ipChan := make(chan string) // Canal para comunicar IPs encontrados
	ips := scanSubnet(ipChan)   // Escaneia a sub-rede para encontrar máquinas
	machines = ips              // Armazena os IPs encontrados na variável global
}

// Função para gerar hashes SHA-256 dos arquivos no diretório local
func generateHashes() []LocalFile {
	dir, err := os.Open(dirPath) // Abre o diretório com os arquivos
	if err != nil {
		log.Fatalf("Erro ao abrir diretório: %s", err)
	}
	defer dir.Close()

	files, err := dir.Readdir(-1) // Lê todos os arquivos no diretório
	if err != nil {
		log.Fatalf("Erro ao abrir diretório: %s", err)
	}

	lfCh := make(chan LocalFile, 10) // Canal para receber os hashes gerados
	for _, f := range files {
		if !f.IsDir() {
			filePath := dirPath + "/" + f.Name()
			go generateHash(lfCh, filePath) // Gera o hash de cada arquivo
		}
	}

	hashes := make([]LocalFile, len(files))
	for i := range files {
		hashes[i] = <-lfCh // Coleta os hashes gerados
	}
	close(lfCh)

	return hashes
}

// Função auxiliar para gerar o hash de um arquivo
func generateHash(lfCh chan LocalFile, filePath string) {
	file, err := os.Open(filePath) // Abre o arquivo
	if err != nil {
		log.Fatalf("Erro ao abrir arquivo: %v", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file) // Lê o conteúdo do arquivo
	if err != nil {
		log.Fatalf("Erro ao abrir arquivo: %s", err)
	}

	hash := sha256.New() // Cria um novo hash SHA-256
	hash.Write(content)
	encoded := hex.EncodeToString(hash.Sum(nil)) // Converte o hash para string hexadecimal

	lfCh <- LocalFile{FilePath: filePath, Hash: encoded} // Envia o resultado pelo canal
}

// Função para escanear a sub-rede e encontrar máquinas com a porta 50051 aberta
func scanSubnet(ipChan chan string) []string {
	interfaces, err := net.InterfaceAddrs() // Obtém os endereços de interface de rede
	if err != nil {
		log.Fatalf("Erro ao receber interfaces %v", err)
		return nil
	}

	var ipv4Interfaces []net.Addr
	for _, i := range interfaces {
		if addr, ok := i.(*net.IPNet); ok && addr.IP.To4() != nil {
			ipv4Interfaces = append(ipv4Interfaces, i) // Filtra apenas interfaces IPv4
		}
	}

	var wg sync.WaitGroup
	for _, i := range ipv4Interfaces {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch v := i.(type) {
			case *net.IPNet:
				if v.IP.To4() != nil {
					ip := v.IP.To4()
					mask := v.Mask
					subnet := ip.Mask(mask) // Calcula a sub-rede

					for i := 0; i <= 255-1; i++ { // Varre os endereços da sub-rede
						ip := subnet.To4()
						ip[3] = byte(i)
						host := ip.String()
						if isPortOpen(host, 50051) { // Verifica se a porta 50051 está aberta
							ipChan <- host
						}
					}
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(ipChan) // Fecha o canal após a conclusão
	}()
	var ips []string
	for ip := range ipChan {
		ips = append(ips, ip) // Coleta os IPs encontrados
	}
	return ips
}

// Função para verificar se a porta está aberta
func isPortOpen(host string, port int) bool {
	timeout := time.Millisecond * 10 // Tempo limite curto para tentar a conexão
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
