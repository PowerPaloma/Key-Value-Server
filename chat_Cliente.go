package main

import (
	"flag"
	"net"
	"strconv"
	"log"
	"io"
	"os"
	"strings"
	"gosdcc/sdlib"
	"bufio"
	"fmt"
)

var (
	address, port, userName *string
)

func chatHandleConnection(c net.Conn, fromControl <-chan string, toControl chan<- string){

	ch := make(chan string)
	eCh := make(chan error)

	// Inicia uma goroutine para ler dados da conexão
	go func(ch chan string, eCh chan error) {
		for {
			// tenta ler dados
			data, err := sdlib.ReadMsg(c)
			if err != nil {
				// envia erro encontrado
				eCh<- err
				return
			}
			// envia dados se lidos
			ch<- data
		}
	}(ch, eCh)

	for{
		select {
			case text := <-fromControl:
				err := sdlib.WriteMsg(c, text)
				if err != nil {
					eCh <- err
				}
			case text := <-ch:
				toControl <- text
			case err := <-eCh:
				
				if err == io.EOF { // conexão fechada (o server parou)
					c.Close()
					os.Exit(0)
				}
				
				if err != nil {
					log.Fatal(err)
				}
		}
	}
}

func readShell(toControl chan<- string) {
    //pega algo que esta sendo digit pelo user 

	reader := bufio.NewReader(os.Stdin)
	// cria o buffer "reader" e o que foi digit sera lido
	fmt.Println("Shell")
	fmt.Println("---------------------")

	for {
		text, _ := reader.ReadString('\n')
		// ler a string do buffer ate o /n
		text = strings.Replace(text, "\n", "", -1)
		// retira o /n
		toControl <- text
		//envia para o controle a  msg
	}

	close(toControl)

}

func writeShell(fromControl <-chan string){

	for text := range fromControl {

		fmt.Println(text)

	}
}

func chatControl(fromNet <-chan string, toNet chan<- string){

    //fromNet - dados vindos da rede
    //TONet - dados enviados para rede

	fromShell := make(chan string, 1000)
	toShell := make(chan string, 1000)

	go readShell(fromShell)
	go writeShell(toShell)

	for {
		select {

			case text := <-fromShell:
				if sdlib.IsCommand(text) { // verifica se a msg é de comando 
					toNet <- text
				} else {
					toNet <- "["+*userName+"] " + text
				}
			case text := <-fromNet:
				toShell <- text

		}

	}

	close(toShell)

}


func init() {
    // Todo programa em go tem essa func - Primeira func a ser chamada dps do main

	address = flag.String("address", "localhost", "endereço do servidor.")
	// flag sempre retorna ponteiros 
	// flag - 1Param - em qual var vai guardar
	//        2Param - o default, caso ele n digite nada 
	//        3Param - Help   
	port = flag.String("port", "6667", "porta da aplicação.")
	userName = flag.String("user", "anônimo", "nome do usuário")
	flag.Parse()
	//pega o que foi digit pelo usuario e disponibiliza para uso no resto do programa

	if serverIp := net.ParseIP(*address); (nil == serverIp) && ("localhost" != *address) {
	//ParseIp - Valida o ip digitado 
		log.Fatal("Endereço IP inválido.")
	}

	if _, err := strconv.ParseUint(*port, 10, 16); nil != err {
	// o underline eh para ignorar o valor, já que só o segundo retorno é relevante
	// ParseUint - valida a porta
		log.Fatal("Porta inválida.")
	}

	if len(*userName) == 0 {
		log.Fatal("Nome inválido.")
	}

}

func main() {

	conn, err := net.Dial("tcp", *address+":"+*port)
	// Dial - Informa que a conexão é tcp 

	controlToNet := make(chan string, 1000)
	netToControl := make(chan string, 1000)

	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()
	//será realizada qnd o codigo abaixo terminar

	go chatHandleConnection(conn, controlToNet, netToControl)  
	// "go" deixa paralelo - Consome menos recurso que uma thread, mas faz a mesma coisa. Sufuciente para classificar o socket como não bloqueante
	chatControl(netToControl, controlToNet) // principal 
	

}
