package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/sha3"
	"io"
)

var (
	plainText = "Yassine123Yassine123Yassine12345"
	key       = "0123456789123457"
)

func main() {

	// Encrypting With GCM
	keyB := []byte(key)
	block, err := aes.NewCipher(keyB)
	if err != nil {
		fmt.Println(err)
	}
	ag, err := cipher.NewGCM(block)
	if err != nil {
		fmt.Println(err)
	}
	nonce := make([]byte, ag.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		fmt.Println(err)
	}
	fmt.Println("nonce:", nonce)
	cryptedText := ag.Seal(nonce, nonce, []byte(plainText), nil)
	fmt.Println("Encoding Text with GCM = ", cryptedText)
	fmt.Println(len(cryptedText))
	fmt.Println(base64.StdEncoding.EncodeToString(cryptedText))

	// Decrypting GCM
	nonce, cipherText := cryptedText[:ag.NonceSize()], cryptedText[ag.NonceSize():]
	fmt.Println("Nonce is ", nonce)
	fmt.Println("Ciphertext is ", cipherText)
	plainText, err := ag.Open(nil, nonce, cipherText, nil)
	fmt.Println(string(plainText))

	// Encrypting CTR
	// Create AES cipher block using the provided key
	block, err = aes.NewCipher(keyB)
	if err != nil {
		fmt.Println(err)
	}
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		fmt.Println(err)
	}
	stream := cipher.NewCTR(block, iv)
	ciphertext := make([]byte, len(plainText))
	stream.XORKeyStream(ciphertext, plainText)
	ciphertext = append(iv, ciphertext...)
	fmt.Println("Ciphertext with CTR :", ciphertext)

	// Decrypting CTR
	// Extract IV from the ciphertext
	iv = ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	stream = cipher.NewCTR(block, iv)
	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)
	fmt.Println("Decrypted plaintext:", string(plaintext))
	// Encoding base64
	data := "Gol@ng is Awesome?~"

	// Standard Base64 Encoding
	encodedData := base64.StdEncoding.EncodeToString([]byte(data))
	fmt.Println("Encoded Std Base64:", encodedData)

	// URL and filename-safe Base64 encoding
	urlSafeEncodedData := base64.URLEncoding.EncodeToString([]byte(data))
	fmt.Println("Encoded URL Base64:", urlSafeEncodedData)

	// Hex Encoding
	byteArray := []byte("Learn Go!")
	fmt.Println("byteArray: ", byteArray)
	encodedString := hex.EncodeToString(byteArray)
	fmt.Println("Encoded Hex String: ", encodedString)

	// Hashing
	s := "test"

	md5 := md5.Sum([]byte(s))
	sha1 := sha1.Sum([]byte(s))
	sha256 := sha256.Sum256([]byte(s))
	sha3 := sha3.Sum256([]byte(s))

	fmt.Printf("md5 : %x\n", md5)
	fmt.Printf("sha1 : %x\n", sha1)
	fmt.Printf("sha256 : %x\n", sha256)
	fmt.Printf("sha3 : %x\n", sha3)

}
