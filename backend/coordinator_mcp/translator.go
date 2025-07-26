package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// TranslationService handles multi-lingual support using Google Translation API
type TranslationService struct {
	apiKey          string
	enabled         bool
	defaultLanguage string
	translateURL    string
	detectURL       string
}

// TranslationRequest represents a translation API request
type TranslationRequest struct {
	Q      []string `json:"q"`
	Target string   `json:"target"`
	Source string   `json:"source,omitempty"`
}

// TranslationResponse represents the API response
type TranslationResponse struct {
	Data struct {
		Translations []struct {
			TranslatedText         string `json:"translatedText"`
			DetectedSourceLanguage string `json:"detectedSourceLanguage,omitempty"`
		} `json:"translations"`
	} `json:"data"`
}

// LanguageDetectionRequest for language detection
type LanguageDetectionRequest struct {
	Q []string `json:"q"`
}

// LanguageDetectionResponse from detection API
type LanguageDetectionResponse struct {
	Data struct {
		Detections [][]struct {
			Language   string  `json:"language"`
			Confidence float64 `json:"confidence"`
		} `json:"detections"`
	} `json:"data"`
}

// NewTranslationService creates a new translation service instance
func NewTranslationService() *TranslationService {
	apiKey := os.Getenv("GOOGLE_TRANSLATE_API_KEY")
	if apiKey == "" {
		// Try to use the same API key as Gemini if not separately configured
		apiKey = os.Getenv("GOOGLE_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("GEMINI_API_KEY")
		}
	}

	enabled := os.Getenv("ENABLE_TRANSLATION") == "true"
	defaultLang := os.Getenv("DEFAULT_LANGUAGE")
	if defaultLang == "" {
		defaultLang = "en"
	}

	service := &TranslationService{
		apiKey:          apiKey,
		enabled:         enabled,
		defaultLanguage: defaultLang,
		translateURL:    "https://translation.googleapis.com/language/translate/v2",
		detectURL:       "https://translation.googleapis.com/language/translate/v2/detect",
	}

	if service.enabled && service.apiKey == "" {
		log.Println("⚠️  Translation enabled but no API key found. Translation will be disabled.")
		service.enabled = false
	}

	if service.enabled {
		log.Printf("🌐 Translation service enabled (default language: %s)", defaultLang)
	} else {
		log.Println("🌐 Translation service disabled")
	}

	return service
}

// DetectLanguage detects the language of the input text
func (ts *TranslationService) DetectLanguage(text string) (string, float64, error) {
	if !ts.enabled || ts.apiKey == "" {
		return ts.defaultLanguage, 1.0, nil
	}

	reqBody := LanguageDetectionRequest{
		Q: []string{text},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return ts.defaultLanguage, 0, err
	}

	url := fmt.Sprintf("%s?key=%s", ts.detectURL, ts.apiKey)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return ts.defaultLanguage, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ts.defaultLanguage, 0, err
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Language detection failed: %s", string(body))
		return ts.defaultLanguage, 0, fmt.Errorf("detection failed: %s", string(body))
	}

	var detectResp LanguageDetectionResponse
	if err := json.Unmarshal(body, &detectResp); err != nil {
		return ts.defaultLanguage, 0, err
	}

	if len(detectResp.Data.Detections) > 0 && len(detectResp.Data.Detections[0]) > 0 {
		detection := detectResp.Data.Detections[0][0]
		return detection.Language, detection.Confidence, nil
	}

	return ts.defaultLanguage, 0, nil
}

// TranslateText translates text to the target language
func (ts *TranslationService) TranslateText(text, targetLang string) (string, error) {
	if !ts.enabled || ts.apiKey == "" || targetLang == "" || targetLang == ts.defaultLanguage {
		return text, nil
	}

	// Don't translate if text is too short or looks like a command
	if len(text) < 10 || strings.HasPrefix(text, "/") {
		return text, nil
	}

	reqBody := TranslationRequest{
		Q:      []string{text},
		Target: targetLang,
		Source: ts.defaultLanguage,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return text, err
	}

	url := fmt.Sprintf("%s?key=%s", ts.translateURL, ts.apiKey)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Translation request failed: %v", err)
		return text, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return text, err
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Translation failed: %s", string(body))
		return text, fmt.Errorf("translation failed: %s", string(body))
	}

	var transResp TranslationResponse
	if err := json.Unmarshal(body, &transResp); err != nil {
		return text, err
	}

	if len(transResp.Data.Translations) > 0 {
		return transResp.Data.Translations[0].TranslatedText, nil
	}

	return text, nil
}

// ProcessChatWithTranslation handles language detection and translation for chat
func (ts *TranslationService) ProcessChatWithTranslation(
	inputText string,
	processFunc func(string) string,
) (string, string) {
	if !ts.enabled {
		return processFunc(inputText), ""
	}

	// Detect input language
	detectedLang, confidence, err := ts.DetectLanguage(inputText)
	if err != nil {
		log.Printf("Language detection error: %v", err)
		return processFunc(inputText), ""
	}

	log.Printf("🌐 Detected language: %s (confidence: %.2f)", detectedLang, confidence)

	// Translate input to default language if needed
	processedInput := inputText
	if detectedLang != ts.defaultLanguage && confidence > 0.7 {
		translatedInput, err := ts.TranslateText(inputText, ts.defaultLanguage)
		if err == nil {
			processedInput = translatedInput
			log.Printf("📝 Translated input from %s to %s", detectedLang, ts.defaultLanguage)
		}
	}

	// Process the query (in default language)
	response := processFunc(processedInput)

	// Translate response back to user's language if needed
	if detectedLang != ts.defaultLanguage && confidence > 0.7 {
		translatedResponse, err := ts.TranslateText(response, detectedLang)
		if err == nil {
			log.Printf("📤 Translated response from %s to %s", ts.defaultLanguage, detectedLang)
			return translatedResponse, detectedLang
		}
	}

	return response, detectedLang
}

// GetSupportedLanguages returns a list of commonly supported languages
func (ts *TranslationService) GetSupportedLanguages() map[string]string {
	return map[string]string{
		"en": "English",
		"es": "Spanish",
		"fr": "French",
		"de": "German",
		"it": "Italian",
		"pt": "Portuguese",
		"ru": "Russian",
		"ja": "Japanese",
		"ko": "Korean",
		"zh": "Chinese (Simplified)",
		"zh-TW": "Chinese (Traditional)",
		"hi": "Hindi",
		"ar": "Arabic",
		"bn": "Bengali",
		"ta": "Tamil",
		"te": "Telugu",
		"mr": "Marathi",
		"gu": "Gujarati",
		"kn": "Kannada",
		"ml": "Malayalam",
		"pa": "Punjabi",
		"ur": "Urdu",
	}
}