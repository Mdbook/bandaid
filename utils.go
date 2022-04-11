package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
)

type Colors struct {
	red     string
	green   string
	blue    string
	black   string
	yellow  string
	magenta string
	cyan    string
	white   string
	reset   string
}

func InitColors() Colors {
	if runtime.GOOS == "windows" {
		return Colors{
			reset:   "",
			black:   "",
			red:     "",
			green:   "",
			yellow:  "",
			blue:    "",
			magenta: "",
			cyan:    "",
			white:   "",
		}
	} else {
		return Colors{
			reset:   "\033[0m",
			black:   "\033[30m",
			red:     "\033[31m",
			green:   "\033[32m",
			yellow:  "\033[33m",
			blue:    "\033[34m",
			magenta: "\033[35m",
			cyan:    "\033[36m",
			white:   "\033[37m",
		}
	}

}

func ConcatenatePath(root string, file string) string {
	if root[len(root)-1:] == "/" {
		return root + file
	} else {
		return root + "/" + file
	}
}

func CopyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func GetInput() string {
	reader := bufio.NewReader(os.Stdin)
	rawAnswer, _ := reader.ReadString('\n')
	answer := trim(rawAnswer)
	return answer
}

func decrypt(ciphertext, key []byte) []byte {
	// Create the AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	// Before even testing the decryption,
	// if the text is too small, then it is incorrect
	if len(ciphertext) < aes.BlockSize {
		fmt.Println("error")
		return ciphertext
	}

	// Get the 16 byte IV
	iv := ciphertext[:aes.BlockSize]

	// Remove the IV from the ciphertext
	ciphertext = ciphertext[aes.BlockSize:]

	// Return a decrypted stream
	stream := cipher.NewCFBDecrypter(block, iv)

	// Decrypt bytes from ciphertext
	stream.XORKeyStream(ciphertext, ciphertext)

	return ciphertext
}

func encrypt(plaintext, key []byte) []byte {
	// Create the AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	// if len(plaintext) < aes.BlockSize {
	// 	return plaintext
	// }
	// Empty array of 16 + plaintext length
	// Include the IV at the beginning
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))

	// Slice of first 16 bytes
	iv := ciphertext[:aes.BlockSize]

	// Write 16 rand bytes to fill iv
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	// Return an encrypted stream
	stream := cipher.NewCFBEncrypter(block, iv)

	// Encrypt bytes from plaintext to ciphertext
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
	return ciphertext
}

func removeService(slice []Service, s int) []Service {
	isFreeing.Lock()
	defer isFreeing.Unlock()
	for _, name := range serviceNames {
		slice[s].getAttr(name).Backup = nil
		slice[s].getAttr(name).Path = ""
		slice[s].getAttr(name).Checksum = ""
		slice[s].getAttr(name).FreeBackup()
	}
	if s == len(slice) {
		return slice[:s-1]
	} else {
		return append(slice[:s], slice[s+1:]...)
	}
}
func removeSO(slice []ServiceObject, s int) []ServiceObject {
	isFreeing.Lock()
	defer isFreeing.Unlock()
	slice[s].Backup = nil
	slice[s].Checksum = ""
	slice[s].Checksum = ""
	slice[s].FreeBackup()
	if s == len(slice) {
		return slice[:s-1]
	} else {
		return append(slice[:s], slice[s+1:]...)
	}
}
func removeDirectory(slice []Directory, s int) []Directory {
	isFreeing.Lock()
	defer isFreeing.Unlock()
	for i := range slice[s].files {
		slice[s].files[i].Backup = nil
		slice[s].files[i].Checksum = ""
		slice[s].files[i].Checksum = ""
		slice[s].files[i] = nil
		slice[s].files[i].FreeBackup()
	}
	slice[s].files = nil
	if s == len(slice) {
		return slice[:s-1]
	} else {
		return append(slice[:s], slice[s+1:]...)
	}
}
func (e *Service) getAttr(field string) *ServiceObject {
	return e.locations[find(serviceNames, field)]
}

func GetPass(str string) []byte {
	str = Reverse(str)
	if len(str) >= 16 {
		str = str[:16]
	} else {
		for {
			str = "z" + str
			if len(str) == 16 {
				break
			}
		}
	}
	return []byte(str)
}

func GetConfigName(s string) string {
	s = badcaesar(s, 13)
	var filename string = s
	if strings.Contains(s, "\\") {
		filename = strings.Join(strings.Split(s, "\\"), "/")
	}
	filename = strings.Join(strings.Split(filename, "/"), "._.")
	return config.backupLocation + "/" + filename
}

func badcaesar(s string, shift int) string {
	// Shift character by specified number of places.
	valids := "ABCDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"
	str := ""
	for _, c := range strings.Split(s, "") {
		if strings.Contains(valids, c) {
			c = string(valids[strings.Index(valids, c)+shift])
		}
		str += c
	}
	return str
}

func contains(arr []string, s string) bool {
	for _, str := range arr {
		if str == s {
			return true
		}
	}
	return false
}

func containsInt(arr []int, i int) bool {
	for _, e := range arr {
		if e == i {
			return true
		}
	}
	return false
}

func find(arr []string, s string) int {
	for i, str := range arr {
		if str == s {
			return i
		}
	}
	return -1
}

func writeFile(file string, contents []byte) bool {
	f, err := os.Create(file)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.Write(contents)
	if err != nil {
		return false
	}
	return true
}

func readFile(path string) string {
	dat, _ := ioutil.ReadFile(path)
	str := string(dat)
	return str
}

func FileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func GetTail(str string, separator string) string {
	s := strings.Split(str, separator)
	return s[len(s)-1]
}

func trim(str string) string {
	return strings.TrimSuffix(strings.TrimSuffix(str, "\n"), "\r")
}

func (a *IpChairs) caret() {
	fmt.Print(colors.blue + "? " + colors.reset)
}

func caret() {
	fmt.Print(colors.green + "> " + colors.reset)
}

func Warnf(s string, params ...interface{}) {
	fmt.Printf(colors.yellow+s+colors.reset, params...)
}

func Errorf(s string, params ...interface{}) {
	fmt.Printf(colors.red+s+colors.reset, params...)
}
