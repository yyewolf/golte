package main

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/Duckduckgot/gtts"
	"github.com/Duckduckgot/gtts/handlers"
	"github.com/Duckduckgot/gtts/voices"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var speech = gtts.Speech{Folder: "assets/audio", Language: voices.French, Handler: &handlers.MPlayer{}}

func main() {
	Audio("Bonjour, veuillez entrez votre mot de passe.")
	Audio("Mot de passe correct.")

	for i := 0; i < 10; i++ {
		Audio(fmt.Sprintf("%d", i))
	}
}

func Audio(text string) {
	// replace special characters and spaces to create a valid filename
	filename, err := toASCII(text)
	handleError(filename, err)
	handleError(speech.CreateSpeechFile(text, filename))
}

func handleError(_ string, err error) {
	if err != nil {
		panic(fmt.Sprintf("Error generating audio: %s", err.Error()))
	}
}

func toASCII(str string) (string, error) {
	// Step 1: Decompose and remove diacritics (accents)
	t := transform.Chain(
		norm.NFD,
		runes.Remove(runes.In(unicode.Mn)), // Remove non-spacing marks
	)
	normalized, _, err := transform.String(t, str)
	if err != nil {
		return "", err
	}

	// Step 2: Remove non-ASCII and non-alphanumeric characters
	filtered := strings.Map(func(r rune) rune {
		if r > 127 {
			return -1 // remove non-ASCII
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return r
		}
		return -1 // remove symbols/punctuation
	}, normalized)

	lowercased := strings.ToLower(filtered)

	// Trim leading/trailing spaces and replace spaces with underscores
	filtered = strings.TrimSpace(lowercased)
	filtered = strings.ReplaceAll(filtered, " ", "_")

	// Ensure the filename is not empty
	if filtered == "" {
		return "", fmt.Errorf("resulting filename is empty after processing")
	}

	return filtered, nil
}
