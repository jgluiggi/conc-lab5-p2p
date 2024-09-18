# Usando

- gRPC
- p2p

## Implementação

### Criar 2 funções:

- searchFiles
  - tem a lista de servidores, inicialmente estático. Haverá uma goroutine
   (getFileByHash) para cada servidor.
   Deve usar select pra tratar os channels (resultChan e errorChan). O servidor
   vai receber do primeiro server que colocar no canal resultChan. Deve ser
   configurado um timeout. Deve tratar quando o arquivo não existe em nenhum
   servidor.
- shareFiles
  - gera hashes de um diretório e expõe dois endpoints: das suas hashes e de
  buscar arquivo pela hash.
- extra:
    - criar serviço (ip fixo) de discovery dos peers da rede local.
    - peers devem se inscrever no discovery.
    - discovery deve retornar a lista de peers.

## Rodando

tendo tanto docker quanto docker-compose instalados, rode:

```docker compose up```

para rodar 4 contêiners que rodam a aplicação e se descobrem (alegadamente)
