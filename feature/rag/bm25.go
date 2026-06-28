package rag

import "github.com/strings77wzq/golem/foundation/bm25"

type BM25Index = bm25.BM25Index
type ScoredDoc = bm25.ScoredDoc

var NewBM25Index = bm25.NewBM25Index
var ReciprocalRankFusion = bm25.ReciprocalRankFusion
