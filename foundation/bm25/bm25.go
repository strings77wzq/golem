// Package bm25 provides keyword-based search using the BM25 algorithm
// and Reciprocal Rank Fusion for merging ranked lists.
//
// BM25 is a probabilistic information retrieval function that extends
// TF-IDF with term frequency saturation and document length normalization.
// RRF merges multiple ranked lists by rank position, avoiding the need
// to normalize scores across different scoring functions.
package bm25

import (
	"fmt"
	"math"
	"sort"
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
	id     string
	text   string
	terms  []string
	length int
	tf     map[string]int
}

// NewBM25Index creates a new BM25 index with standard parameters (k1=1.5, b=0.75).
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

	for i, existing := range idx.docs {
		if existing.id == id {
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

// Remove removes a document from the index by ID.
// Returns true if the document was found and removed.
func (idx *BM25Index) Remove(id string) bool {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	for i, doc := range idx.docs {
		if doc.id == id {
			idx.docs = append(idx.docs[:i], idx.docs[i+1:]...)
			idx.numDocs--
			if idx.numDocs == 0 {
				idx.avgDL = 0
			} else {
				idx.avgDL = (idx.avgDL*float64(idx.numDocs+1) - float64(doc.length)) / float64(idx.numDocs)
			}
			return true
		}
	}
	return false
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

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK]
}

func (d ScoredDoc) String() string {
	return fmt.Sprintf("%s (score=%.4f)", d.ID, d.Score)
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

	results := make([]ScoredDoc, 0, len(scores))
	for id, score := range scores {
		results = append(results, ScoredDoc{ID: id, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}
