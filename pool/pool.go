package pool

import (
	"MTWS/core"
	"net"
)

type Job struct {
	Conn net.Conn
}

func worker(id int, jobs <-chan Job) {
	for job := range jobs {
		/*handle job worker connection*/
		core.HandleConnection(job.Conn)
	}
}

func StartWorkerPool(numWorkers int, jobs chan Job) {
	for i := 0; i < numWorkers; i++ {
		go worker(i, jobs)
	}
}
