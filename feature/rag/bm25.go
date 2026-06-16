package rag

import (
	"math"
	"sort"
	"strings"
)

// BM25Index provides keyword-based search using the BM25 algorithm.
type BM25Index struct {
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
func (idx *BM25Index) Add(id, text string) {
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

	idx.docs = append(idx.docs, doc)
	idx.numDocs++
	idx.avgDL = float64((idx.avgDL*float64(idx.numDocs-1) + float64(doc.length)) / float64(idx.numDocs))
}

// ScoredDoc represents a document with its relevance score.
type ScoredDoc struct {
	ID    string
	Score float64
}

// Search finds the top-K documents matching the query.
func (idx *BM25Index) Search(query string, topK int) []ScoredDoc {
	queryTerms := tokenize(query)
	if len(queryTerms) == 0 || idx.numDocs == 0 {
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
	// Simple whitespace + punctuation split
	terms := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' ||
			r == '.' || r == ',' || r == ';' || r == ':' ||
			r == '!' || r == '?' || r == '"' || r == '\'' ||
			r == '(' || r == ')' || r == '[' || r == ']' ||
			r == '{' || r == '}' || r == '-' || r == '_'
	})
	return terms
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
