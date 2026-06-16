package rag

import (
	"math"
	"sort"
	"strings"
	"sync"
)

// BM25Index provides keyword-based search using the BM25 algorithm.
// Thread-safe for concurrent Add and Search operations.
type BM25Index struct {
	mu      sync.RWMutex
	docs    []bm25Doc
	numDocs int
	avgDL   float64
	k1      float64
	b       float64
}

type bm25Doc struct {
	id      string
	text    string
	terms   []string
	length  int
	tf      map[string]int
}

// NewBM25Index creates a new BM25 index with standard parameters.
func NewBM25Index() *BM25Index {
	return &BM25Index{
		k1: 1.5,
		b:  0.75,
	}
}

// Add adds a document to the index.
// If a document with the same ID already exists, it is replaced.
func (idx *BM25Index) Add(id, text string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	terms := tokenize(text)
	tf := make(map[string]int)
	for _, term := range terms {
		tf[term]++
	}

	doc := bm25Doc{
		id:     id,
		text:   text,
		terms:  terms,
		length: len(terms),
		tf:     tf,
	}

	// Check for existing document with same ID
	for i, existing := range idx.docs {
		if existing.id == id {
			// Replace existing document, adjust avgDL
			oldLen := float64(existing.length)
			newLen := float64(doc.length)
			idx.avgDL = (idx.avgDL*float64(idx.numDocs) - oldLen + newLen) / float64(idx.numDocs)
			idx.docs[i] = doc
			return
		}
	}

	idx.docs = append(idx.docs, doc)
	idx.numDocs++
	if idx.numDocs == 1 {
		idx.avgDL = float64(doc.length)
	} else {
		idx.avgDL = (idx.avgDL*float64(idx.numDocs-1) + float64(doc.length)) / float64(idx.numDocs)
	}
}

// ScoredDoc represents a document with its relevance score.
type ScoredDoc struct {
	ID    string
	Score float64
}

// Search finds the top-K documents matching the query.
func (idx *BM25Index) Search(query string, topK int) []ScoredDoc {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	queryTerms := tokenize(query)
	if len(queryTerms) == 0 || idx.numDocs == 0 || topK <= 0 {
		return nil
	}

	// Compute IDF for each query term
	idf := make(map[string]float64)
	for _, term := range queryTerms {
		n := 0
		for _, doc := range idx.docs {
			if _, ok := doc.tf[term]; ok {
				n++
			}
		}
		idf[term] = math.Log((float64(idx.numDocs)-float64(n)+0.5)/(float64(n)+0.5) + 1)
	}

	// Score each document
	results := make([]ScoredDoc, 0, len(idx.docs))
	for _, doc := range idx.docs {
		score := 0.0
		for _, term := range queryTerms {
			tf := float64(doc.tf[term])
			dl := float64(doc.length)
			numerator := tf * (idx.k1 + 1)
			denominator := tf + idx.k1*(1-idx.b+idx.b*dl/idx.avgDL)
			score += idf[term] * numerator / denominator
		}
		if score > 0 {
			results = append(results, ScoredDoc{ID: doc.id, Score: score})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK]
}

// tokenize splits text into lowercase terms.
func tokenize(text string) []string {
	text = strings.ToLower(text)
	var terms []string
	var current strings.Builder

	for _, r := range text {
		if isCJK(r) {
			// Flush any accumulated Latin text
			if current.Len() > 0 {
				terms = append(terms, current.String())
				current.Reset()
			}
			// CJK characters are individual tokens
			terms = append(terms, string(r))
		} else if isTokenDelimiter(r) {
			// Flush accumulated text
			if current.Len() > 0 {
				terms = append(terms, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(r)
		}
	}

	// Flush remaining
	if current.Len() > 0 {
		terms = append(terms, current.String())
	}

	return terms
}

// isCJK checks if a rune is a CJK character (Chinese, Japanese, Korean).
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
		(r >= 0x20000 && r <= 0x2A6DF) || // CJK Extension B
		(r >= 0x2A700 && r <= 0x2B73F) || // CJK Extension C
		(r >= 0x2B740 && r <= 0x2B81F) || // CJK Extension D
		(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility Ideographs
		(r >= 0x2F800 && r <= 0x2FA1F) // CJK Compatibility Supplement
}

// isTokenDelimiter checks if a rune is a token delimiter.
func isTokenDelimiter(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' ||
		r == '.' || r == ',' || r == ';' || r == ':' ||
		r == '!' || r == '?' || r == '"' || r == '\'' ||
		r == '(' || r == ')' || r == '[' || r == ']' ||
		r == '{' || r == '}' || r == '-' || r == '_'
}

// ReciprocalRankFusion merges multiple ranked lists using RRF.
// k is the constant (default 60).
func ReciprocalRankFusion(rankLists [][]ScoredDoc, k int) []ScoredDoc {
	if k == 0 {
		k = 60
	}

	scores := make(map[string]float64)
	docs := make(map[string]ScoredDoc)

	for _, list := range rankLists {
		for rank, doc := range list {
			score := 1.0 / float64(k+rank+1)
			scores[doc.ID] += score
			docs[doc.ID] = doc
		}
	}

	// Convert to slice and sort
	results := make([]ScoredDoc, 0, len(scores))
	for id, score := range scores {
		results = append(results, ScoredDoc{ID: id, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}
