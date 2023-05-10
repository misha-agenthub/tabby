// Package main is the entry point of the program
package main

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// IgnoreMap is a map containing regular expressions used for ignoring files
type IgnoreMap map[string]*regexp.Regexp

var ignore IgnoreMap

// file_is_saved checks if the given file is already saved
func file_is_saved(file string) bool {
	return strings.Index(file, string(os.PathSeparator)) != -1
}

// name_is_ignored checks if the given name is to be ignored based on IgnoreMap
func name_is_ignored(name string) bool {
	for _, re := range ignore {
		if nil == re {
			continue
		}
		if re.Match([]byte(name)) {
			return true
		}
	}
	return false
}

// get_stack_set_add_file adds the given file to the stack
func get_stack_set_add_file(file string, m map[string]int, l []string, s *int) {
	if !file_is_saved(file) {
		return
	}
	_, found := m[file]
	if !found {
		m[file] = 1
		l[*s] = file
		*s++
	}
}

// get_stack_set returns the set of files contained in stack + cur_file
func get_stack_set() (map[string]int, []string, int) {
	m := make(map[string]int)
	list := make([]string, STACK_SIZE)
	list_size := 0
	get_stack_set_add_file(cur_file, m, list, &list_size)
	for {
		file := file_stack_pop()
		if "" == file {
			break
		}
		get_stack_set_add_file(file, m, list, &list_size)
	}
	return m, list, list_size
}

// get_file_info returns information about the given file
func get_file_info(file string) string {
	rec := file_map[file]
	be_str := strconv.Itoa(rec.sel_be)
	en_str := strconv.Itoa(rec.sel_en)
	return file + ":" + be_str + ":" + en_str + "\n"
}

// session_save saves the current session
func session_save() {
	file_save_current()
	file, _ := os.OpenFile(os.Getenv("HOME")+"/.tabby", os.O_CREATE|os.O_WRONLY, 0644)
	if nil == file {
		tabby_log("unable to save session")
		return
	}
	file.Truncate(0)
	stack_set, list, list_size := get_stack_set()
	// Dump all the files not contained in file_stack.
	for k, _ := range file_map {
		_, found := stack_set[k]
		if (false == found) && file_is_saved(k) {
			file.WriteString(get_file_info(k))
		}
	}
	// Dump files from stack in the right order. Last file should be last in the
	// list of files in .tabby file.
	for y := list_size - 1; y >= 0; y-- {
		file.WriteString(get_file_info(list[y]))
	}
	file.Close()
}

// session_open_and_read_file opens and reads the given file in the session
func session_open_and_read_file(name string) bool {
	read_ok, buf := open_file_read_to_buf(name, false)
	if false == read_ok {
		return false
	}
	if add_file_record(name, buf, true) {
		file_stack_push(name)
		return true
	}
	return false
}

// session_restore restores the previous session
func session_restore() {
	reader, file := take_reader_from_file(os.Getenv("HOME") + "/.tabby")
	defer file.Close()
	var str string
	for next_string_from_reader(reader, &str) {
		split_str := strings.SplitN(str, ":", 3)
		if session_open_and_read_file(split_str[0]) {
			be, _ := strconv.Atoi(split_str[1])
			en, _ := strconv.Atoi(split_str[2])
			file_map[split_str[0]].sel_be = be
			file_map[split_str[0]].sel_en = en
		}
	}
	ignore = make(IgnoreMap)
	reader, file = take_reader_from_file(os.Getenv("HOME") + "/.tabbyignore")
	for next_string_from_reader(reader, &str) {
		ignore[str], _ = regexp.Compile(str)
	}
}

// take_reader_from_file gets a reader for the given file
func take_reader_from_file(name string) (*bufio.Reader, *os.File) {
	file, _ := os.OpenFile(name, os.O_CREATE|os.O_RDONLY, 0644)
	if nil == file {
		tabby_log("unable to Open file for reading: " + name)
		return nil, nil
	}
	return bufio.NewReader(file), file
}

// next_string_from_reader reads the next string from the given reader and stores it in s
func next_string_from_reader(reader *bufio.Reader, s *string) bool {
	str, err := reader.ReadString('\n')
	if nil != err {
		return false
	}
	*s = str[:len(str)-1]
	return true
}