package compton

import (
  "math"
  "sort"
  "log"
)

type Pair struct {
  People string
  Amount float64
}

type ByAmount []Pair

func (a ByAmount) Len() int           { return len(a) }
func (a ByAmount) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAmount) Less(i, j int) bool { return a[i].Amount < a[j].Amount }


func findOptimalArrangment(balances map[string]float64) map[string][]Pair {
  
  reimbursments := make(map[string][]Pair)
  
  giver := make([]Pair, 0, len(balances))
  receiver := make([]Pair, 0, len(balances))
  
  for p, diff := range balances {
    if math.Abs(diff) >= 0.01 {
      pair := Pair{p, diff}
      if diff < 0 {
        pair.Amount *= -1
        giver = append(giver, pair)
        reimbursments[p] = make([]Pair, 0, 10)
      } else {
        receiver = append(receiver, pair)
      }
    }
  }
  
  sort.Sort(ByAmount(receiver))
  sort.Sort(ByAmount(giver))
  
  for len(giver) > 0 {
    log.Println(giver, receiver)
    if giver[0].Amount <= receiver[0].Amount {
      reimbursments[giver[0].People] = append(reimbursments[giver[0].People], Pair{receiver[0].People, giver[0].Amount})
      receiver[0].Amount -= giver[0].Amount
      giver = giver[1:]
      if receiver[0].Amount < 0.01 {
        receiver = receiver[1:]
      }
    } else {
      reimbursments[giver[0].People] = append(reimbursments[giver[0].People], Pair{receiver[0].People, receiver[0].Amount})
      giver[0].Amount -= receiver[0].Amount
      receiver = receiver[1:]
      if giver[0].Amount < 0.01 {
        giver = giver[1:]
      }
    }
  }
  
  return reimbursments
}