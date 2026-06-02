package experimental

import "sort"

// ClusterFloats groups sorted values into runs where each value is within tol
// of the previous one. It is the building block for detecting vertical lanes
// (columns) from a bag of word x-positions, and rows from y-positions:
//
//	xs := words.Lefts()
//	lanes := experimental.ClusterFloats(xs, 8) // 8pt column tolerance
//
// The input is not mutated; output groups are in ascending order.
func ClusterFloats(values []float64, tol float64) [][]float64 {
	if len(values) == 0 {
		return nil
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)

	clusters := [][]float64{{sorted[0]}}
	for _, v := range sorted[1:] {
		last := clusters[len(clusters)-1]
		if v-last[len(last)-1] <= tol {
			clusters[len(clusters)-1] = append(last, v)
		} else {
			clusters = append(clusters, []float64{v})
		}
	}
	return clusters
}

// Mean returns the arithmetic mean of values (0 for an empty slice). Handy for
// turning a position cluster into a single lane center.
func Mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// Lefts returns the left edge (x0) of every word — feed straight into
// ClusterFloats to find column lanes.
func (ws Words) Lefts() []float64 {
	out := make([]float64, len(ws))
	for i, w := range ws {
		out[i] = w.Rect.X0
	}
	return out
}

// Tops returns the top edge (y0) of every word.
func (ws Words) Tops() []float64 {
	out := make([]float64, len(ws))
	for i, w := range ws {
		out[i] = w.Rect.Y0
	}
	return out
}
