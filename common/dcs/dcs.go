package dcs

import "math"

// GetDCS: get the decentralization/consistency/scalability of the current system
// params:
// - nodesNum: the number of nodes in system
// - latency: the end-to-end latency, U. secend
// - throughput: the throughput of system, U. tps/s
func GetDCS(nodesNum int, latency float64, throughput float64) (float64, float64, float64) {
	decentralization := GetDecentralization(nodesNum)

	consistency := GetConsistency(latency)

	scalability := GetScalability(throughput)

	return decentralization, consistency, scalability
}

// GetDecentralization: get the decentralization of the current system
// params:
// - nodesNum: the number of nodes in system
func GetDecentralization(nodesNum int) float64 {
	if nodesNum > 0 {
		return 1 - 1/float64(nodesNum)
	}
	return 0.0
}

// GetConsistency: get the consistency of the current system
// params:
// - latency: the end-to-end latency, U. secend
func GetConsistency(latency float64) float64 {
	if latency > 0 {
		return math.Exp(-latency)
	}
	return 0.0
}

// GetScalability: get the scalability of the current system
// params:
// - throughput: the throughput of system, U. tps/s
func GetScalability(throughput float64) float64 {
	if throughput > 0 {
		return 1 - 1/math.Log10(throughput)
	}
	return 0.0
}
