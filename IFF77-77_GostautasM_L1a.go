package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"
	"text/tabwriter"
	"time"
)

const max = 200
const divider = 5      //default - 10
const modifier = 2     //default - 2
const monitorSize = 10 //default - 10
const threadsCount = 4
const file = 1

//Thing is for storing string, int and float values in single structure
//Result value is added later to store filter parameter
type Thing struct {
	Company string  `json:"company"`
	Count   int     `json:"count"`
	Price   float32 `json:"price"`
	Result  int
}

// Monitor is used by goroutines to get data
type Monitor struct {
	count     int
	array     []Thing
	sizeLimit int
	condition *sync.Cond
	mutex     *sync.Mutex
}

func initializeMonitor(size int) Monitor {
	var mutexTemp = sync.Mutex{}
	cond := sync.NewCond(&mutexTemp)
	array := make([]Thing, size)
	monitor := Monitor{count: 0, sizeLimit: size, mutex: &mutexTemp, condition: cond, array: array}
	return monitor
}

func (monitor *Monitor) take() Thing {
	var temp Thing
	monitor.mutex.Lock()
	for monitor.count == 0 {
		monitor.condition.Wait()
	}
	for i := 0; i < monitorSize; i++ {
		if monitor.array[i].Count != 0 {
			temp = monitor.array[i]
			(*monitor).array[i].Count = 0
			(*monitor).count--
			break
		}
	}

	monitor.condition.Broadcast()
	monitor.mutex.Unlock()
	return temp
}

func (monitor *Monitor) add(thing Thing) bool {

	monitor.mutex.Lock()

	for monitor.count == monitorSize {
		monitor.condition.Wait()
	}

	for i := 0; i < monitorSize; i++ {
		if monitor.array[i].Count == 0 {
			(*monitor).array[i] = thing
			(*monitor).count++
			break
		}
	}
	monitor.condition.Broadcast()
	monitor.mutex.Unlock()
	return false
}

func (monitor *Monitor) addResult(thing Thing) {
	monitor.mutex.Lock()
	var i int
	for i = int(monitor.count - 1); i >= 0 && monitor.array[i].Result > thing.Result; i-- {
		(*monitor).array[i+1] = (*monitor).array[i]
	}
	(*monitor).array[i+1] = thing
	(*monitor).count++
	monitor.mutex.Unlock()
}

func thread(limit int, wg *sync.WaitGroup, monitor *Monitor, results *Monitor, id int) {
	for int(ops) < limit {
		if monitor.count > 0 {
			var temp = monitor.take()
			primeCount := filterCondition(temp)

			fmt.Println("gija - ", id, ", kompanija - ", temp.Company, " skaicius - ", primeCount)

			if (primeCount % modifier) != 0 {
				temp.Result = primeCount
				results.addResult(temp)
			}
			atomic.AddUint64(&ops, 1)
		}
	}
	wg.Done()
}

var ops uint64

func runConcurrent(things []Thing) Monitor {
	var wg sync.WaitGroup

	monitor := initializeMonitor(monitorSize)

	results := initializeMonitor(150)
	for i := 0; i < threadsCount; i++ {
		wg.Add(1)
		go thread(len(things), &wg, &monitor, &results, i)
	}
	for _, value := range things {
		monitor.add(value)
	}

	wg.Wait()
	return results
}

func readJSON(path string) []Thing {
	file, _ := ioutil.ReadFile(path)
	var data []Thing
	_ = json.Unmarshal([]byte(file), &data)
	return data
}

func primeCount(number int) int {
	var n = 3
	var is = true
	var count int

	for n < number {
		is = true
		if n%2 == 0 {
			is = false
		}
		for i := 3; i < (n / 3); i += 2 {
			if (n % i) == 0 {
				is = false
				break
			}
		}
		if is {
			count++
		}
		n++
	}
	return count
}

func filterCondition(thing Thing) int {
	sum := 0
	for _, value := range thing.Company {
		sum += int(value)
	}
	sum += int(thing.Price)
	sum = sum * thing.Count / divider
	primeCount := primeCount(sum)
	if primeCount%modifier != 0 {
		return primeCount
	}
	return primeCount
}
func printToFile(data []Thing, things Monitor) {
	f, _ := os.Create("./IFF77_GostautasM_L1_rez.txt")
	w := tabwriter.NewWriter(f, 0, 0, 4, ' ', tabwriter.AlignRight|tabwriter.Debug)

	defer f.Close()

	fmt.Fprintln(w, "rezultatai")
	fmt.Fprintln(w, "-----------\t----------\t---------------\t-------------\t")
	fmt.Fprintln(w, "pavadinimas\tkiekis\tkaina\trezultatas\t")
	fmt.Fprintln(w, "-----------\t----------\t---------------\t-------------\t")

	for i := 0; i < int(things.count); i++ {
		s := fmt.Sprintf("%s\t%d\t%f\t%d\t", things.array[i].Company, things.array[i].Count, things.array[i].Price, things.array[i].Result)
		fmt.Fprintln(w, s)
	}
	fmt.Fprintln(w, "duomenys")
	fmt.Fprintln(w, "-----------\t----------\t---------------\t")
	fmt.Fprintln(w, "pavadinimas\tkiekis\tkaina\t")
	fmt.Fprintln(w, "-----------\t----------\t---------------\t")
	for _, value := range data {
		s := fmt.Sprintf("%s\t%d\t%f\t", value.Company, value.Count, value.Price)
		fmt.Fprintln(w, s)
	}

	w.Flush()
}

func main() {
	start := time.Now()
	var data []Thing
	data = readJSON(fmt.Sprintf("IFF77_GostautasM_L1_dat_%d.json", file))
	var results = runConcurrent(data)
	fmt.Println(time.Since(start))
	fmt.Println("all - ", len(data), "; filtered - ", results.count)

	printToFile(data, results)
}
