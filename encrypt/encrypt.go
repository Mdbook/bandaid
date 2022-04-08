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
	"strings"
)

func decrypt(ciphertext, key []byte) []byte {
	// Create the AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	// Before even testing the decryption,
	// if the text is too small, then it is incorrect
	if len(ciphertext) < aes.BlockSize {
		panic("Text is too short")
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

func readline() string {
	bio := bufio.NewReader(os.Stdin)
	line, _, err := bio.ReadLine()
	if err != nil {
		fmt.Println(err)
	}
	return string(line)
}

func writeToFile(data, file string) {
	ioutil.WriteFile(file, []byte(data), 777)
}
func trim(str string) string {
	return strings.TrimSuffix(strings.TrimSuffix(str, "\n"), "\r")
}
func readFromFile(file string) []byte {
	data, _ := ioutil.ReadFile(file)
	return data
}

func main() {
	key := GetPass(trim(os.Args[3]))
	// fmt.Println(trim(os.Args[3]))
	switch os.Args[1] {
	case "encrypt":
		fmt.Println(string(encrypt(readFromFile(os.Args[2]), key)))
	case "decrypt":
		fmt.Println(string(decrypt(readFromFile(os.Args[2]), key)))
	}
	// x := encrypt([]byte(plain), key)
	// fmt.Println("Plaintext: " + plain)
	// fmt.Println("Encrypted: " + string(x))
	// fmt.Println("Decrypted: " + string(decrypt(x, key)))
}

func GetPass(str string) []byte {
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
