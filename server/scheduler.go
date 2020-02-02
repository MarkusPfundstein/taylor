package server

import (
	"time"
	"fmt"
	"os"
	"sync"
	"math/rand"
	"math"

	"taylor/server/database"
	"taylor/lib/structs"
	"taylor/lib/tcp"
)

type Scheduler struct {
	sleepMs		time.Duration
	tcpServer	*TcpServer
	store		*database.Store
	config		Config
}

func freeNodes(nodes []*Node) ([]*Node, uint) {
	sumCap := uint(0)
	freeNodes := make([]*Node, 0)
	for _, node := range nodes {
		if node.Capacity > node.JobsRunning {
			freeNodes = append(freeNodes, node)
			sumCap = sumCap + (node.Capacity - node.JobsRunning)
		}
	}
	return freeNodes, sumCap
}

type NodeProxyJobMap struct {
	node *NodeProxy
	job  *structs.Job 
}

type NodeJobMap struct {
	node *Node
	job  *structs.Job 
}

// A proxy for the Node struct that only has the important values
// necessary to calculate the distribution
type NodeProxy struct {
	Capacity	uint
	JobsRunning	uint
	Name		string
	// original index of the Node in the nodes slice.
	ogIdx		int
}

func removeNode(slice []*NodeProxy, s int) []*NodeProxy {
    return append(slice[:s], slice[s+1:]...)
}

func (node *NodeProxy) HasFreeCapacity() bool {
	return (node.Capacity - node.JobsRunning) > 0
}

func distribute(jobs []*structs.Job, nodesIn []*Node) []NodeJobMap{

	res := make([]NodeProxyJobMap, 0)

	nodes := make([]*NodeProxy, len(nodesIn))
	for i, v := range(nodesIn) {
		// We need a proxy because we will do some increment
		nodes[i]= &NodeProxy{
			Capacity:    	v.Capacity,
			JobsRunning: 	v.JobsRunning,
			Name: 		v.Name,
			ogIdx: 		i,
		}
	}

	// pick some connected node at random to start round robin on
	counter := rand.Intn(len(nodes))
	for i := 0; i < len(jobs); i++ {
		// round robin
		idx := counter % len(nodes)
		counter++
		node := nodes[idx]
		if node.HasFreeCapacity() {
			// we can schedule
			res = append(res, NodeProxyJobMap{
				node: node,
				job: jobs[i],
			})
			node.JobsRunning++
			// check if we still dont have any capacity. if so, remove preliminary
			if node.HasFreeCapacity() == false {
				nodes = removeNode(nodes, idx)
			}
		} else {
			nodes = removeNode(nodes, idx)
			// couldnt schedule try again
			i--
		}
		// this should never happen
		if len(nodes) == 0 {
			break
		}
	}

	// get original Nodes out. Copy over from NodeProxyJobMap to NodeJobMap
	out := make([]NodeJobMap, len(res))
	for i, v := range res {
		out[i] = NodeJobMap{
			node: nodesIn[v.node.ogIdx],
			job:  v.job,
		}
	}

	return out
}

func (s *Scheduler) schedule () {
	for {
		time.Sleep(s.sleepMs * time.Millisecond)
		
		// get all nodes and check if they have free capacity
		freeNodes, cap := freeNodes(s.tcpServer.Nodes())
		if len(freeNodes) == 0 {
			continue
		}

		jobs, err := s.store.JobsWithStatus(structs.JOB_STATUS_WAITING, cap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error %v\n", err)
			// what shall we do?
			panic(err)
		}
		if len(jobs) == 0 {
			continue
		}

		fmt.Printf("free capacity %d over %d nodes\n", cap, len(freeNodes))
		fmt.Printf("Got %d jobs to distribute\n", len(jobs))

		distributed := distribute(jobs, freeNodes)

		// informative message if my algorithm suxxxx
		if len(distributed) != int(math.Min(float64(len(jobs)), float64(cap))) {
			fmt.Printf("MISTAKE DURING SCHEDULING (scheduled: %d, needed: %d)\n", len(distributed), cap)
		}

		// try to schedule all jobs on corresponding nodes
		var wg sync.WaitGroup
		for _, v := range distributed {
			fmt.Printf("%s -> %s\n", v.job.Identifier, v.node.Name)

			wg.Add(1)
			go func (njm NodeJobMap) {
				defer wg.Done()

				payload := &tcp.MsgNewJobOffer{
					MsgBase: tcp.MsgBase{
						Command: tcp.MSG_NEW_JOB_OFFER,
						NodeName: s.config.Name,
					},
					Job: *njm.job,
				}

				s.tcpServer.Unicast(njm.node, payload)
			}(v)
		}

		wg.Wait()
	}
}

func StartScheduler(config Config, store *database.Store, server *TcpServer) {
	scheduler := Scheduler{
		sleepMs: 1000,
		store: store,
		tcpServer: server,
		config: config,
	}

	go scheduler.schedule()
}
