package main

import (
	"net"
	"flag"
	"io"
	"log"
	"strconv"
	"gosdcc/sdlib"
	"strings"
	"sync"
	"reflect"
)
type Sala struct {
	connections []net.Conn
	name	string
}

var (
	port *string
	salas map[string] Sala
	sessao map[net.Conn] string
	mutex sync.RWMutex

)

func obterListaDeSalas() string {
	
	var lista string

	mutex.RLock()
	for _,nomeSala := range reflect.ValueOf(salas).MapKeys() {
		lista += (nomeSala.Interface().(string) + "\n")
	}
	mutex.RUnlock()

	return lista
}

func obterSala(id string) *Sala {

	mutex.RLock()
	sala, ok := salas[id]
	mutex.RUnlock()

	if true == ok {
		return &sala
	} else {
		return nil
	}
}

func removeSala(nomeSala string) {
	
	mutex.Lock()
	delete(salas, nomeSala)
	mutex.Unlock()

}

func removeConn(conn net.Conn) {

	mutex.RLock()
	var sala Sala = salas[sessao[conn]]
	mutex.RUnlock()

	var i int

	for i = range sala.connections {

		if sala.connections[i] == conn {
			break
		}
	}
	
	if len(sala.connections) != 0 {
		sala.connections = append(sala.connections[:i], sala.connections[i+1:]...)
		mutex.Lock()
		salas[sessao[conn]] = sala
		mutex.Unlock()
	} 
	
	if len(sala.connections) == 0{
		removeSala(sala.name)
	}

	mutex.Lock()
	delete(sessao, conn)
	mutex.Unlock()
}

func irASala(nomeSala string, conn net.Conn) {

	mutex.RLock()
	sala, ok := salas[nomeSala]
	mutex.RUnlock()

	if true == ok {
		removeConn(conn)
		mutex.Lock()
		sessao[conn] = nomeSala
		conexoes := sala.connections
		sala.connections = append(conexoes, conn)
		salas[nomeSala] = sala
		mutex.Unlock()

		err := sdlib.WriteMsg(conn, "Em nova sala: "+nomeSala)
		

		if err != nil {
			log.Println(err)
		}

	} else {
		sala := Sala{name: nomeSala, connections: make([]net.Conn,0)}
		removeConn(conn)

		mutex.Lock()
		sessao[conn] = nomeSala
		sala.connections = append(sala.connections, conn)
		salas[nomeSala] = sala
		mutex.Unlock()
		
		err := sdlib.WriteMsg(conn, "Em nova sala: "+nomeSala)
		

		if err != nil {
			log.Println(err)
		}
	}

}


func broadcast(conn net.Conn, msg string) {

	mutex.RLock()
	var sala Sala = salas[sessao[conn]]
	mutex.RUnlock()

	for i := range sala.connections {
		if sala.connections[i] != conn {
		
			err := sdlib.WriteMsg(sala.connections[i], msg)
			
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func handleConn(conn net.Conn) {

	defer conn.Close()
	
	for {
		msg, err := sdlib.ReadMsg(conn)

		if err != nil {

			if err == io.EOF {

				removeConn(conn)
				conn.Close()
				
				return
			}

			log.Println(err)
			
			return
		}

		if sdlib.IsCommand(msg) {
			if msg == "/quit" {
				removeConn(conn)
				break
			} else if msg == "/list" {
				sdlib.WriteMsg(conn, obterListaDeSalas())
			} else if  strings.HasPrefix(msg, "/join") {
				nomeSala := strings.Fields(msg)[1]
				irASala(nomeSala, conn)
			}

		} else {
			broadcast(conn, msg)
		}
	}
}

func init() {
	
	port = flag.String("port", "6667", "porta da aplicação.")
	flag.Parse()

	if _, err := strconv.ParseUint(*port, 10, 16); nil != err {
		log.Fatal("Porta inválida.")
	}

	salaDefault := Sala{connections: make([]net.Conn,0), name: "default"}
	salas = make(map[string]Sala)
	salas["default"] = salaDefault
	sessao = make(map[net.Conn]string)

}

func main() {

	listener, err := net.Listen("tcp", "localhost:"+*port)
	
	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()

	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Print(err)
			continue
		}


		mutex.Lock()
		sessao[conn] = "default"
		sala := salas["default"]
		conexoes := sala.connections
		sala.connections = append(conexoes, conn)
		salas["default"] = sala
		mutex.Unlock()

		go handleConn(conn) // uma para cada cliente
	}
}
