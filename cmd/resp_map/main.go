package main

import (
	"log"
	"net"
	"sync"
)

var store = struct {
	sync.RWMutex
	data map[string]string
}{
	data: make(map[string]string),
}



func parseRESP(reader *bufio.Reader) ([]string, error) {

	firstByte, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}


	if firstByte == '*' {

		countStr, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		countStr = strings.TrimSuffix(countStr, "\r\n")
		count, err := strconv.Atoi(countStr)
		if err != nil {
			return nil, fmt.Errorf("invalid array length: %s", countStr)
		}


		result := make([]string, count)
		for i := 0; i < count; i++ {

			typeChar, err := reader.ReadByte()
			if err != nil {
				return nil, err
			}

			if typeChar != '$' {
				return nil, fmt.Errorf("expected bulk string, got %c", typeChar)
			}


			lenStr, err := reader.ReadString('\n')
			if err != nil {
				return nil, err
			}
			lenStr = strings.TrimSuffix(lenStr, "\r\n")
			strLen, err := strconv.Atoi(lenStr)
			if err != nil {
				return nil, fmt.Errorf("invalid string length: %s", lenStr)
			}


			buf := make([]byte, strLen+2)
			_, err = io.ReadFull(reader, buf)
			if err != nil {
				return nil, err
			}


			result[i] = string(buf[:strLen])
		}

		return result, nil
	}

	return nil, fmt.Errorf("unsupported RESP type: %c", firstByte)
}


func writeRESPSimpleString(w io.Writer, str string) error {
	_, err := fmt.Fprintf(w, "+%s\r\n", str)
	return err
}


func writeRESPError(w io.Writer, err string) error {
	_, e := fmt.Fprintf(w, "-%s\r\n", err)
	return e
}


func writeRESPBulkString(w io.Writer, str string) error {
	_, err := fmt.Fprintf(w, "$%d\r\n%s\r\n", len(str), str)
	return err
}


func writeRESPNullBulkString(w io.Writer) error {
	_, err := fmt.Fprintf(w, "$-1\r\n")
	return err
}


func writeRESPArray(w io.Writer, elements []string) error {
	_, err := fmt.Fprintf(w, "*%d\r\n", len(elements))
	if err != nil {
		return err
	}

	for _, el := range elements {
		err = writeRESPBulkString(w, el)
		if err != nil {
			return err
		}
	}

	return nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		args, err := parseRESP(reader)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error parsing command: %v", err)
			}
			return
		}

		if len(args) == 0 {
			writeRESPError(conn, "ERR empty command")
			continue
		}

		cmd := strings.ToUpper(args[0])

		switch cmd {
		case "GET":
			if len(args) != 2 {
				writeRESPError(conn, "ERR wrong number of arguments for 'get' command")
				continue
			}

			key := args[1]
			store.RLock()
			value, exists := store.data[key]
			store.RUnlock()

			if exists {
				writeRESPBulkString(conn, value)
			} else {
				writeRESPNullBulkString(conn)
			}

		case "SET":
			if len(args) != 3 {
				writeRESPError(conn, "ERR wrong number of arguments for 'set' command")
				continue
			}

			key := args[1]
			value := args[2]

			store.Lock()
			store.data[key] = value
			store.Unlock()

			writeRESPSimpleString(conn, "OK")

		case "DEL":
			if len(args) < 2 {
				writeRESPError(conn, "ERR wrong number of arguments for 'del' command")
				continue
			}

			deleted := 0
			store.Lock()
			for i := 1; i < len(args); i++ {
				key := args[i]
				_, exists := store.data[key]
				if exists {
					delete(store.data, key)
					deleted++
				}
			}
			store.Unlock()


			_, err := fmt.Fprintf(conn, ":%d\r\n", deleted)
			if err != nil {
				log.Printf("Error writing response: %v", err)
				return
			}

		case "KEYS":
			if len(args) != 2 {
				writeRESPError(conn, "ERR wrong number of arguments for 'keys' command")
				continue
			}

			pattern := args[1]
			store.RLock()
			keys := make([]string, 0)


			if pattern == "*" {

				for k := range store.data {
					keys = append(keys, k)
				}
			} else {

				if _, exists := store.data[pattern]; exists {
					keys = append(keys, pattern)
				}
			}
			store.RUnlock()

			writeRESPArray(conn, keys)

		case "PING":
			writeRESPSimpleString(conn, "PONG")

		case "QUIT":
			writeRESPSimpleString(conn, "OK")
			return

		default:
			writeRESPError(conn, fmt.Sprintf("ERR unknown command '%s'", cmd))
		}
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	for {
      arg , err :=
	}
}

func main() {
	ln, err := net.Listen("tcp", ":4000")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}
		go handleConnection(conn)
	}
}
