package server

import (
	"testing"
	s "taylor/lib/structs"
	//"fmt"
)

func assertSameNodeByName(t *testing.T, n1 *Node, n2 *Node) {
	if n1.Name != n2.Name {
		t.Log(n1.Name, " != ", n2.Name)
		t.Fail()
	}
}

func assertNodeHasJobAssigned(t *testing.T, nm NodeJobMap, n* Node, j *s.Job) {
	if nm.job.Identifier != j.Identifier {
		t.Log("Nodemap Job: ", nm.job.Identifier, " != Job: ", j.Identifier)
		t.Fail()
	}
	if nm.node.Name != n.Name {
		t.Log("Nodemap Node: ", nm.node.Name, " != Node: ", n.Name)
		t.Fail()
	}
}

func assertInt(t *testing.T, actual int, expected int) {
	if actual != expected {
		t.Log("Actual", actual, " != ", " Expected", expected)
		t.Fail()
	}
}

func TestDistributeOnCapacity(t *testing.T) {
	nodesIn := []*Node{
		&Node{
			Name: "a",
			Capacity: 3,
			JobsRunning: 2,
		},
		&Node{
			Name: "b",
			Capacity: 3,
			JobsRunning: 1,
		},
	}

	jobs := []*s.Job{
		&s.Job{
			Identifier: "0",
		},
		&s.Job{
			Identifier: "1",
		},
		&s.Job{
			Identifier: "2",
		},
		&s.Job{
			Identifier: "3",
		},
	}

	output := distribute(nodesIn, jobs)

	assertInt(t, len(output), len(jobs)-1)

	assertNodeHasJobAssigned(t, output[0], nodesIn[0], jobs[0])
	assertNodeHasJobAssigned(t, output[1], nodesIn[1], jobs[1])
	assertNodeHasJobAssigned(t, output[2], nodesIn[1], jobs[2])
}

func TestDistributeOnCapabilities(t *testing.T) {
	nodesIn := []*Node{
		&Node{
			Name: "a",
			Capacity: 3,
			JobsRunning: 0,
			Capabilities: []string{"X"},
		},
		&Node{
			Name: "b",
			Capacity: 3,
			JobsRunning: 0,
		},
		&Node{
			Name: "c",
			Capacity: 3,
			JobsRunning: 0,
			Capabilities: []string{"X", "Y"},
		},
	}

	jobs := []*s.Job{
		&s.Job{
			Identifier: "0",
		},
		&s.Job{
			Identifier: "1",
			Restrict: []string{"X"},
		},
		&s.Job{
			Identifier: "2",
		},
		&s.Job{
			Identifier: "3",
			Restrict: []string{"X", "Y"},
		},
		&s.Job{
			Identifier: "4",
			Restrict: []string{"X"},
		},
		&s.Job{
			Identifier: "5",
		},
		&s.Job{
			Identifier: "6",
			Restrict: []string{"Y"},
		},
		&s.Job{
			Identifier: "7",
			Restrict: []string{"Z"},
		},
	}

	output := distribute(nodesIn, jobs)
	assertInt(t, len(output), len(jobs) - 1)

	assertNodeHasJobAssigned(t, output[0], nodesIn[1], jobs[0])
	assertNodeHasJobAssigned(t, output[1], nodesIn[0], jobs[1])
	assertNodeHasJobAssigned(t, output[2], nodesIn[1], jobs[2])
	assertNodeHasJobAssigned(t, output[3], nodesIn[2], jobs[3])
	assertNodeHasJobAssigned(t, output[4], nodesIn[0], jobs[4])
	assertNodeHasJobAssigned(t, output[5], nodesIn[1], jobs[5])
	assertNodeHasJobAssigned(t, output[6], nodesIn[2], jobs[6])
}
