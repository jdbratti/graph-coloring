
## Usage:

* Download the code
* go build; ./graph-coloring -file input_example1.txt -plot 

## About the problem

This was a course assignment in which a given assembly code has to be interpreted in order to find the minimum number of registers needed to cover all variables. So, two different variables that coexist cannot be assigned to the same register. Variables are the graph vertices and the edges represent that two variables coexist. Then, the problem consists of finding the minimum number of colors needed (registers) to paint the graph without having two adjacent vertices with the same color.

The graph coloring problem is a known np-hard problem. Therefore, the algorithm might not end for a large input.

You can find more about the problem here: [geeksforgeeks](http://www.geeksforgeeks.org/graph-coloring-applications/)