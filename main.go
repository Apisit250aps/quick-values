package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func getTokenFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Cannot determine home directory:", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".qv")
}

func loadTokens() map[string]string {
	path := getTokenFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}
	var tokens map[string]string
	_ = json.Unmarshal(data, &tokens)
	return tokens
}

func saveTokens(tokens map[string]string) {
	path := getTokenFilePath()
	data, _ := json.MarshalIndent(tokens, "", "  ")
	os.WriteFile(path, data, 0600)
}

func setToken(key, value string) {
	tokens := loadTokens()
	tokens[key] = value
	saveTokens(tokens)
	fmt.Println("Set", key, "successfully.")
}

func getToken(key string) {
	tokens := loadTokens()
	if val, ok := tokens[key]; ok {
		fmt.Println(key+":", val)
		copyToClipboard(val)
		fmt.Println("(Copied to clipboard)")
	} else {
		fmt.Println("Token not found for key:", key)
	}
}

func copyToClipboard(text string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "echo "+text+"|clip")
	case "darwin":
		cmd = exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
	case "linux":
		cmd = exec.Command("sh", "-c", "echo '"+text+"' | xclip -selection clipboard || echo '"+text+"' | xsel --clipboard")
	default:
		fmt.Println("(Clipboard not supported on this OS)")
		return
	}

	_ = cmd.Run()
}

func deleteToken(key string) {
	tokens := loadTokens()
	if _, ok := tokens[key]; ok {
		delete(tokens, key)
		saveTokens(tokens)
		fmt.Println("Deleted", key, "successfully.")
	} else {
		fmt.Println("Key not found:", key)
	}
}

func listKeys() {
	tokens := loadTokens()
	if len(tokens) == 0 {
		fmt.Println("No tokens saved.")
		return
	}
	fmt.Println("Available keys:")
	keys := make([]string, 0, len(tokens))
	for k := range tokens {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Println("-", k)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: qv [set|get|del|list] ...")
		return
	}

	cmd := os.Args[1]
	switch cmd {
	case "set":
		if len(os.Args) < 4 {
			fmt.Println("Usage: qv set <key> <value>")
			return
		}
		setToken(os.Args[2], os.Args[3])
	case "get":
		if len(os.Args) < 3 {
			fmt.Println("Usage: qv get <key>")
			return
		}
		getToken(os.Args[2])
	case "del":
		if len(os.Args) < 3 {
			fmt.Println("Usage: qv del <key>")
			return
		}
		deleteToken(os.Args[2])
	case "list":
		listKeys()
	default:
		fmt.Println("Unknown command:", cmd)
	}
}
