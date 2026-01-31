package document

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/dslipak/pdf"
	"github.com/google/uuid"
)

type Config struct {
	OllamaURL     string
	ChromaURL     string
	ChromaAPIBase string
	DefaultModel  string
	Collection    string
}

type Handler struct {
	config Config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func NewHandler() *Handler {
	return &Handler{
		config: Config{
			OllamaURL:     getEnv("OLLAMA_URL", "http://localhost:11434"),
			ChromaURL:     getEnv("CHROMA_URL", "http://localhost:8000"),
			ChromaAPIBase: "/api/v2/tenants/default_tenant/databases/default_database/collections",
			DefaultModel:  getEnv("EMBEDDING_MODEL", "embeddinggemma:300m"),
			Collection:    getEnv("COLLECTION_NAME", "documents"),
		},
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, mw func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("/api/reset", mw(h.HandleReset))
	mux.HandleFunc("/api/upload", mw(h.HandleUpload))
	mux.HandleFunc("/api/search", mw(h.HandleSearch))
}

// Request/Response Structs
type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type EmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

type ChromaAddRequest struct {
	Documents  []string      `json:"documents"`
	Metadatas  []interface{} `json:"metadatas"`
	Ids        []string      `json:"ids"`
	Embeddings [][]float32   `json:"embeddings"`
}

type ChromaQueryRequest struct {
	QueryEmbeddings [][]float32 `json:"query_embeddings"`
	NResults        int         `json:"n_results"`
}

type ChromaQueryResponse struct {
	Ids       [][]string      `json:"ids"`
	Documents [][]string      `json:"documents"`
	Metadatas [][]interface{} `json:"metadatas"`
	Distances [][]float32     `json:"distances"`
}

// Handlers

func (h *Handler) HandleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Resetting collection: %s", h.config.Collection)

	url := fmt.Sprintf("%s%s/%s", h.config.ChromaURL, h.config.ChromaAPIBase, h.config.Collection)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create request: %v", err), http.StatusInternalServerError)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to delete collection: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf("chroma reset error: %s", string(body)), http.StatusInternalServerError)
		return
	}

	log.Printf("Collection reset successful")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "reset successful", "collection": h.config.Collection})
}

func (h *Handler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	// Get file
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get file: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	log.Printf("Received file: %s (size: %d bytes)", header.Filename, header.Size)

	// Get chunk parameters
	chunkSize := 100
	chunkStride := 80

	if cs := r.FormValue("chunkSize"); cs != "" {
		if parsed, err := strconv.Atoi(cs); err == nil && parsed > 0 {
			chunkSize = parsed
		}
	}

	if cst := r.FormValue("chunkStride"); cst != "" {
		if parsed, err := strconv.Atoi(cst); err == nil && parsed > 0 {
			chunkStride = parsed
		}
	}

	log.Printf("Processing with chunk size: %d, stride: %d", chunkSize, chunkStride)

	// Save file temporarily
	tmpFile, err := os.CreateTemp("", "upload-*.pdf")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create temp file: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, file)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to save file: %v", err), http.StatusInternalServerError)
		return
	}

	// Process PDF
	err = h.processPDF(tmpFile.Name(), header.Filename, chunkSize, chunkStride)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to process PDF: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully processed: %s", header.Filename)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "completed",
		"filename":    header.Filename,
		"chunkSize":   chunkSize,
		"chunkStride": chunkStride,
	})
}

func (h *Handler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	log.Printf("Searching for: %s", query)

	embedding, err := h.getEmbedding(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get embedding: %v", err), http.StatusInternalServerError)
		return
	}

	results, err := h.queryChroma(embedding)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to query chroma: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// Helpers

func (h *Handler) processPDF(path, filename string, chunkSize, chunkStride int) error {
	content, err := ReadPDF(path)
	if err != nil {
		return fmt.Errorf("failed to read PDF: %v", err)
	}

	log.Printf("Extracted %d characters from PDF", len(content))

	chunks := ChunkText(content, chunkSize, chunkStride)
	log.Printf("Split PDF into %d chunks (size: %d words, stride: %d words)", len(chunks), chunkSize, chunkStride)

	for i, chunk := range chunks {
		log.Printf("Processing chunk %d/%d (length: %d chars)", i+1, len(chunks), len(chunk))

		embedding, err := h.getEmbedding(chunk)
		if err != nil {
			log.Printf("WARNING: failed to get embedding for chunk %d: %v", i+1, err)
			continue
		}

		err = h.addToChroma(chunk, embedding, filename, i+1)
		if err != nil {
			log.Printf("WARNING: failed to add chunk %d to chroma: %v", i+1, err)
			continue
		}

		log.Printf("Successfully stored chunk %d/%d", i+1, len(chunks))
	}

	log.Printf("Completed processing all %d chunks", len(chunks))
	return nil
}

func (h *Handler) getEmbedding(text string) ([]float32, error) {
	reqBody, _ := json.Marshal(EmbeddingRequest{
		Model:  h.config.DefaultModel,
		Prompt: text,
	})

	resp, err := http.Post(h.config.OllamaURL+"/api/embeddings", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("http post error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var res EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return res.Embedding, nil
}

func (h *Handler) addToChroma(text string, embedding []float32, filename string, chunkNum int) error {
	colID, err := h.getOrCreateCollection(h.config.Collection)
	if err != nil {
		return fmt.Errorf("getOrCreateCollection failed: %w", err)
	}

	id := uuid.New().String()
	reqBody, _ := json.Marshal(ChromaAddRequest{
		Documents: []string{text},
		Metadatas: []interface{}{map[string]interface{}{
			"source":    "pdf",
			"filename":  filename,
			"chunk_num": chunkNum,
		}},
		Ids:        []string{id},
		Embeddings: [][]float32{embedding},
	})

	url := fmt.Sprintf("%s%s/%s/add", h.config.ChromaURL, h.config.ChromaAPIBase, colID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("http post to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chroma add returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (h *Handler) queryChroma(embedding []float32) (*ChromaQueryResponse, error) {
	colID, err := h.getOrCreateCollection(h.config.Collection)
	if err != nil {
		return nil, err
	}

	reqBody, _ := json.Marshal(ChromaQueryRequest{
		QueryEmbeddings: [][]float32{embedding},
		NResults:        5,
	})

	url := fmt.Sprintf("%s%s/%s/query", h.config.ChromaURL, h.config.ChromaAPIBase, colID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("chroma query returned status %d: %s", resp.StatusCode, string(body))
	}

	var res ChromaQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (h *Handler) getOrCreateCollection(name string) (string, error) {
	// 1. Try to get
	getURL := fmt.Sprintf("%s%s/%s", h.config.ChromaURL, h.config.ChromaAPIBase, name)
	resp, err := http.Get(getURL)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var res struct {
				ID string `json:"id"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
				return "", fmt.Errorf("failed to decode get collection response: %w", err)
			}
			return res.ID, nil
		}
	}

	// 2. Create if not found or status not OK
	createURL := fmt.Sprintf("%s%s", h.config.ChromaURL, h.config.ChromaAPIBase)
	reqBody, _ := json.Marshal(map[string]string{"name": name})
	resp, err = http.Post(createURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to POST to %s: %w", createURL, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create collection at %s returned status %d: %s", createURL, resp.StatusCode, string(body))
	}

	var res struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("failed to decode create collection response: %w", err)
	}

	if res.ID == "" {
		return "", fmt.Errorf("received empty collection ID from ChromaDB")
	}

	return res.ID, nil
}

// ReadPDF extracts plain text from a PDF file at the given path.
func ReadPDF(path string) (string, error) {
	r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		return "", err
	}
	_, err = io.Copy(&buf, b)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ChunkText splits the text into chunks of `size` words with a `stride`.
func ChunkText(text string, size int, stride int) []string {
	var chunks []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	for i := 0; i < len(words); i += stride {
		end := i + size
		if end > len(words) {
			end = len(words)
		}
		chunks = append(chunks, strings.Join(words[i:end], " "))
		if end == len(words) {
			break
		}
	}
	return chunks
}
