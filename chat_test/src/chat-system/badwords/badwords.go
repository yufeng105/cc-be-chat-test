package badwords


import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"io"
	"io/ioutil"
)

type wordMap struct {
	lenMap   map[int]int
	lenSlice []int
	words    map[string]string
}

type wordTree struct {
	wordMaxLen int
	trees      map[string]*wordMap
	file	string
}

func (t *wordTree) add(word string) {
	wSlice := strings.Split(word, "")
	length := len(wSlice)
	if length > t.wordMaxLen {
		t.wordMaxLen = length
	}
	if t.trees == nil {
		t.trees = make(map[string]*wordMap)
	}
	if t.trees[wSlice[0]] == nil {
		t.trees[wSlice[0]] = &wordMap{lenMap: make(map[int]int), words: make(map[string]string)}
	}
	tree := t.trees[wSlice[0]]
	if _, exist := tree.words[word]; !exist {
		tree.words[word] = strings.Repeat("*", length)
		if _, exist := tree.lenMap[length]; !exist {
			tree.lenMap[length] = length
			tree.lenSlice = append(tree.lenSlice, length)
			sort.Ints(tree.lenSlice)
		}
	}
}

func (t *wordTree) Add(word string){
	t.add(word)
	t.addToFile(word)
}

//save new  word to file
func (t *wordTree) addToFile(word string){
	inputFile, inputError := os.OpenFile(t.file,os.O_RDWR, 0666)
	if inputError != nil {
		fmt.Println("An error occurred on opening the inputfile\n" +
			"Does the file exist?\n" +
			"Have you got acces to it?\n")
		return // exit the function on error
	}
	defer inputFile.Close()
	inputReader := bufio.NewReader(inputFile)
	for {
		inputString, readerError := inputReader.ReadString('\n')
		inputString = strings.Trim(inputString,"\n")
		if inputString==word{
			return
		}
		if readerError == io.EOF {
			outputWriter := bufio.NewWriter(inputFile)
			outputWriter.WriteString(word+"\n")
			outputWriter.Flush()
			return
		}
	}

}
func (t *wordTree) Del(word string){
	t.del(word)
	t.delFromFile(word)
}

//delete from memory
func (t *wordTree) del(word string){
	//reload word dictionary
	Init(t.file)
	return
}

//delete from file
func (t *wordTree) delFromFile(word string){
	input, err := ioutil.ReadFile(t.file)
	if err != nil {
		fmt.Println(err)
	}

	lines := strings.Split(string(input), "\n")
	for i,v :=range lines {
		if v==word{
			lines= append(lines[:i],lines[i+1:]...)
			fmt.Println(i,v)
		}
	}

	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(t.file, []byte(output), 0644)
	if err != nil {
		fmt.Println(err)
	}
	return
}

func (t *wordTree) View() ([]string){
	input, err := ioutil.ReadFile(t.file)
	if err != nil {
		fmt.Println(err)
	}

	lines := strings.Split(string(input), "\n")
	return lines
}
type search struct {
	txt         *[]string
	txtLen      int
	tree        *wordTree
	replacement string
	matched     map[string]int
}

func (s *search) replace(start, end int) {
	for i := start; i < end; i++ {
		(*s.txt)[i] = s.replacement
	}
}

func (s *search) run() {
	s.matched = make(map[string]int)
	s.txtLen = len(*s.txt)
	for i := 0; i < s.txtLen; i++ {
		if tree, exist := s.tree.trees[(*s.txt)[i]]; exist {
			lenLen := len(tree.lenSlice)
			for j := 0; j < lenLen; j++ {
				end := i + tree.lenSlice[j]
				if end > s.txtLen {
					break
				}
				word := strings.Join((*s.txt)[i:end], "")
				if tree.words[word] != "" {
					if s.replacement != "" {
						s.replace(i, end)
					}
					sum, _ := s.matched[word]
					s.matched[word] = sum + 1
					i = end - 1
					break
				}
			}
		}
	}
}

func Init(filename string) *wordTree {
	wT := &wordTree{}
	wT.file = filename
	f, err := os.Open(filename)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		wT.add(word)
	}
	return wT
}

func Search(tree *wordTree, txt *[]string, replacement string) *map[string]int {
	s := &search{}
	s.tree = tree
	s.txt = txt
	s.replacement = replacement
	s.run()
	return &s.matched
}

