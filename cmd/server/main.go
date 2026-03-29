package main

import (
	"MTWS/pool"
	"log"
	"net"
)

func main() {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	/* make job pool here.*/
	jobs := make(chan pool.Job)
	pool.StartWorkerPool(10, jobs)

	log.Println("Server running on port:8080!")

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Accept error : ", err)
			continue
		}

		jobs <- pool.Job{Conn: conn}
	}
}
