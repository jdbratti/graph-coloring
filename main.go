package main

import (
	"bufio"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/skratchdot/open-golang/open"
)

// Stores the file path for the input file
var filePath = flag.String("file", "", "the file used as input")

// Stores if it needs to plot the graph
var plotGraph = flag.Bool("plot", false, "if it should plot the graph")

// Number of available colors
var numberOfColors = 12

// The name of all available colors
var colors = [12]string{"none", "red", "blue", "green", "yellow", "purple", "black", "pink", "orange", "cyan", "white", "brown"}

// The html color's code
var htmlColors = [12]string{"#909090", "#EA0D0D", "#0D67EA", "#29C529", "#E5F30C", "#980CF3", "#000000", "#F510E3", "#F5A210", "#10F5C8", "#FFFFFF", "#6E2C00"}

// Graph structures
var adjacencyList [][]int
var vertices []string

// Stores an index color for every vertex
var result []int

// Number of registers available
var registers int

func parseFile() {
	// Opens the file
	f, err := os.Open(*filePath)
	if err != nil {
		panic("Error opening the file!")
	}

	// Creates a file reader
	r := bufio.NewReader(f)
	defer f.Close()

	// Reads the first line
	l, _, err := r.ReadLine()
	if err != nil {
		panic("Error reading the first line!")
	}
	firstLine := string(l)

	// Breaks the first line and gets the number of registers (without spaces or tabs)
	registers, err = strconv.Atoi(strings.TrimSpace(strings.Split(firstLine, ":")[1]))
	if err != nil {
		panic("Error parsing the number of registers!")
	}

	// Maintains record of variables usage, such as first and last access
	type variableUsage struct {
		index int
		first int
		last  int
	}

	// Map of all different variables found in the assembly code
	allVariables := make(map[string]variableUsage, 0)
	vertices = make([]string, 0)

	count := 0
	for {
		// Reads the file line by line
		l, _, err = r.ReadLine()
		if err != nil {
			break
		}

		line := string(l)

		// Skip blank lines
		if strings.TrimSpace(line) != "" {
			// Formats the line
			op := formatInstruction(line)
			// Breaks instruction by commas
			instruction := strings.Split(op, ",")

			// For each parameter in the instruction [1:]
			for _, param := range instruction[1:] {
				// If the parameter has up to two letters, it's not a memory address
				if len(param) <= 2 {
					// If it is a known variable
					if value, ok := allVariables[param]; ok {
						value.last = count
						allVariables[param] = value
					} else {
						// First time the variable appears in the code
						v := variableUsage{
							index: len(vertices),
							first: count,
							last:  count,
						}

						// Append the variable in the vertices slice.
						vertices = append(vertices, param)

						// Insert a map reference to the variable
						allVariables[param] = v
					}
				}
			}

			count++
		}
	}

	adjacencyList = make([][]int, len(vertices))
	result = make([]int, len(vertices))

	// For each variable in the map, search for time intersections
	for _, each := range allVariables {
		adjacencyList[each.index] = make([]int, 0)
		for _, other := range allVariables {
			if each.index != other.index {
				if (each.first >= other.first && each.first <= other.last) ||
					(each.last >= other.first && each.last <= other.last) {
					// Add the relation between both variables in the graph
					adjacencyList[each.index] = append(adjacencyList[each.index], other.index)
				}
			}
		}

		// The color for all vertices start at none
		result[each.index] = 0
	}
}

// Formats a given instruction and returns it as: instruction,param1,param2
// Result example: LOAD,A,FA10
func formatInstruction(line string) string {
	str := strings.Fields(line)

	result := str[1] + ","
	for _, s := range str[2:] {
		result += s
	}

	return result
}

// Color the graph. Receives the initial vertex's index.
func colorGraph(vertex int) bool {
	// Iterates over all colors
	for i := 1; i < numberOfColors; i++ {
		var branch bool

		// If the color can be used for the given vertex
		if isPossible(vertex, i) {
			result[vertex] = i

			// If there are more vertices to be colored
			if vertex+1 < len(result) {
				branch = colorGraph(vertex + 1)
			} else {
				return true
			}
		}

		if branch {
			return true
		}
	}

	return false
}

// Check if a given color can be used in a certain vertex.
func isPossible(vertex, color int) bool {
	for _, adj := range adjacencyList[vertex] {
		if result[adj] == color {
			return false
		}
	}

	return true
}

// Rendered List of templates rendered for the webserver
var Rendered map[string]*template.Template

func main() {
	flag.Parse()
	if strings.TrimSpace(*filePath) == "" {
		log.Fatal("Missing input file! Use: --help")
	}

	parseFile()
	ok := colorGraph(0)

	// If it was able to find a solution
	if ok {
		fmt.Println("It was possible to find a solution!")

		memory := make(map[int]int, numberOfColors)
		max := -1
		for _, v := range result {
			if v > max {
				max = v
			}

			memory[v]++
		}

		fmt.Println("The number of needed registers is:", max)
		fmt.Println("The number of available registers is:", registers)

		// Sorts the map of most used colors
		sortedMem := sortedKeys(memory)

		// Prints the most used colors and it's vertices
		for _, mem := range sortedMem[:min(len(sortedMem), registers)] {
			for vindex, vcolor := range result {
				if vcolor == mem {
					fmt.Println(vertices[vindex], "gonna be colored with", colors[mem])
				}
			}
		}

		// Prints the colored vertices with the least used colors
		for _, mem := range sortedMem[min(len(sortedMem), registers):] {
			for vindex, vcolor := range result {
				if vcolor == mem {
					fmt.Println(vertices[vindex], "stays in memory :'(")
					result[vindex] = 0
				}
			}
		}

	} else {
		fmt.Println("It was not possible to find a solution with", numberOfColors, "colors")
	}

	// If it has to plot the graph
	if *plotGraph {
		Rendered = make(map[string]*template.Template, 0)

		// Renders the template
		t, err := template.ParseFiles("graph.html")
		if err != nil {
			panic("Error parsing the template graph.html")
		}

		Rendered["graph"] = t

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/graph", http.StatusFound)
		})

		http.HandleFunc("/graph", Graph)

		// Opens the default browser
		err = open.Start("http://127.0.0.1:50000/graph")
		if err != nil {
			fmt.Println("Error opening the default browser!")
		}

		// Opens the webserver
		err = http.ListenAndServe(":50000", nil)
		if err != nil {
			fmt.Println("Error starting the webserver")
		}
	}
}

// Vertex's info required by the sigma.js library
type vertex struct {
	VertexID    int
	VertexLabel string
	VertexColor string
}

// Edge's info required by the sigma.js library
type edge struct {
	EdgeID     int
	EdgeSource int
	EdgeTarget int
}

// All the html info
type info struct {
	Date     string
	Github   string
	Vertices []vertex
	Edges    []edge
}

// Graph Function that handles the http request from the main page and shows the graph
func Graph(w http.ResponseWriter, r *http.Request) {
	screenInfo := info{
		Date:   "June, 2017",
		Github: "github.com/jdbratti",
	}

	// Creates the vertices
	for vindex, vcolor := range result {
		v := vertex{
			VertexID:    vindex,
			VertexLabel: vertices[vindex],
			VertexColor: htmlColors[vcolor],
		}

		screenInfo.Vertices = append(screenInfo.Vertices, v)
	}

	// Creates the edges
	edgeID := 0
	for i, list := range adjacencyList {
		for _, j := range list {
			e := edge{
				EdgeID:     edgeID,
				EdgeSource: i,
				EdgeTarget: j,
			}

			screenInfo.Edges = append(screenInfo.Edges, e)
			edgeID++
		}
	}

	// Executes the template
	err := Rendered["graph"].Execute(w, screenInfo)
	if err != nil {
		panic("Error executing the graph.html template")
	}
}

// Returns the minimum between two values
func min(a, b int) int {
	if a <= b {
		return a
	}

	return b
}

// Code to sort a map by value. Thanks to: gist.github.com/ikbear/4038654
type sortedMap struct {
	m map[int]int
	s []int
}

func (sm *sortedMap) Len() int {
	return len(sm.m)
}

func (sm *sortedMap) Less(i, j int) bool {
	return sm.m[sm.s[i]] > sm.m[sm.s[j]]
}

func (sm *sortedMap) Swap(i, j int) {
	sm.s[i], sm.s[j] = sm.s[j], sm.s[i]
}

func sortedKeys(m map[int]int) []int {
	sm := new(sortedMap)
	sm.m = m
	sm.s = make([]int, len(m))
	i := 0

	for key := range m {
		sm.s[i] = key
		i++
	}

	sort.Sort(sm)
	return sm.s
}
