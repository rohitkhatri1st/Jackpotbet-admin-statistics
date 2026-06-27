package main

import (
	"admin-stats/config"
	"admin-stats/db"
	"admin-stats/model"
	mongorepo "admin-stats/repository/mongo"
	"context"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	totalRounds = 2_000_000
	numUsers    = 700
	batchSize   = 500 // rounds per batch; each batch becomes 1000 documents (wager + payout)
)

var (
	currencies = []string{"ETH", "BTC", "USDT"}

	// Static conversion rates — good enough for seed data, no live prices needed.
	rates = map[string]decimal.Decimal{
		"ETH":  decimal.NewFromInt(3000),
		"BTC":  decimal.NewFromInt(65000),
		"USDT": decimal.NewFromInt(1),
	}

	// Realistic wager ranges per currency so the data doesn't look obviously fake.
	amountRanges = map[string][2]float64{
		"ETH":  {0.001, 10.0},
		"BTC":  {0.0001, 1.0},
		"USDT": {1.0, 10000.0},
	}
)

// roundJob tells a build worker which slice of rounds to generate.
// start/end are just index bounds — what matters is end-start (the count).
type roundJob struct {
	start   int
	end     int
	userIDs []bson.ObjectID
}

// Stage 1 — Scheduler: slices 2M rounds into batches of 500 and feeds them
// downstream one at a time. Nothing is generated here — it just decides "who
// does what chunk" and hands the work off to the builders.
func roundGenStage(total, batchSz int, userIDs []bson.ObjectID) <-chan roundJob {
	out := make(chan roundJob)
	go func() {
		defer close(out) // closing signals the next stage that no more jobs are coming
		for i := 0; i < total; i += batchSz {
			end := min(i+batchSz, total)
			out <- roundJob{start: i, end: end, userIDs: userIDs}
		}
	}()
	return out
}

// Stage 2 — Builder: the CPU-heavy stage. Multiple workers run in parallel,
// each picking up a batch of 500 rounds and generating the actual transaction
// data — random amounts, timestamps, currencies — producing 1000 documents
// (one Wager + one Payout per round) ready to be written to the database.
func buildStage(in <-chan roundJob, workers int) <-chan []model.Transaction {
	out := make(chan []model.Transaction)
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()

			// Each worker gets its own RNG — rand.Rand is not safe for concurrent use,
			// so sharing one global instance would require a lock and kill parallelism.
			rng := rand.New(rand.NewSource(rand.Int63()))

			// Compute the time window once per worker, not once per round.
			now := time.Now()
			yearAgo := now.AddDate(-1, 0, 0)
			window := now.Sub(yearAgo) // ~8760 hours as a Duration, used for random offset

			for job := range in {
				// Pre-allocate exactly the space we need: 2 docs per round.
				batch := make([]model.Transaction, 0, (job.end-job.start)*2)

				for range job.end - job.start {
					// All transactions in a round must share the same currency —
					// pick it once here and pass it to both wager and payout.
					currency := currencies[rng.Intn(len(currencies))]
					userID := job.userIDs[rng.Intn(len(job.userIDs))]
					roundID := fmt.Sprintf("round-%x", rng.Int63())

					// Wager happens at a random point in the past year.
					wagerAt := yearAgo.Add(time.Duration(rng.Int63n(int64(window))))
					// Payout must always come after the wager — offset by 1 to 59 seconds.
					payoutAt := wagerAt.Add(time.Duration(rng.Intn(59)+1) * time.Second)

					batch = append(batch,
						makeTransaction(userID, roundID, "Wager", currency, randomAmount(rng, currency), wagerAt),
						makeTransaction(userID, roundID, "Payout", currency, randomAmount(rng, currency), payoutAt),
					)
				}
				out <- batch
			}
		}()
	}
	// A separate goroutine closes out once all workers are done.
	// Can't call wg.Wait() here directly — that would block before returning out.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// Stage 3 — Writer: the I/O-heavy stage and the end of the pipeline. Multiple
// workers fire concurrent InsertMany calls to MongoDB, each writing a 1000-doc
// batch at a time. Blocks until every batch has been written, then returns so
// main() can print the final elapsed time.
func insertStage(in <-chan []model.Transaction, workers int, repo *mongorepo.TransactionRepository, ctx context.Context) {
	var (
		wg       sync.WaitGroup
		inserted atomic.Int64 // shared across workers — atomic so no mutex needed
		start    = time.Now()
	)
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for batch := range in {
				if err := repo.BulkInsertTransactions(ctx, batch); err != nil {
					log.Printf("insert error: %v", err)
					return
				}
				n := inserted.Add(int64(len(batch)))
				// Print progress every 200k docs. The modulo check detects when
				// the counter has just crossed a 200k boundary in this batch.
				if n%200_000 < int64(len(batch)) {
					fmt.Printf("  %d / %d docs (%.0fs)\n", n, totalRounds*2, time.Since(start).Seconds())
				}
			}
		}()
	}
	wg.Wait() // main() is blocked here until every insert worker finishes
}

// randomAmount picks a random wager amount within the realistic range for the
// given currency, rounded to 8 decimal places.
func randomAmount(rng *rand.Rand, currency string) decimal.Decimal {
	r := amountRanges[currency]
	f := r[0] + rng.Float64()*(r[1]-r[0])
	return decimal.NewFromFloat(f).Round(8)
}

// makeTransaction builds a single Transaction document. It converts the decimal
// amount into MongoDB's Decimal128 format and computes the USD equivalent using
// the static rate for the given currency.
func makeTransaction(userID bson.ObjectID, roundID, txType, currency string, amount decimal.Decimal, createdAt time.Time) model.Transaction {
	amountD128, _ := bson.ParseDecimal128(amount.StringFixed(8))
	usdD128, _ := bson.ParseDecimal128(amount.Mul(rates[currency]).StringFixed(2))
	return model.Transaction{
		ID:        bson.NewObjectID(),
		CreatedAt: createdAt,
		UserID:    userID,
		RoundID:   roundID,
		Type:      txType,
		Amount:    amountD128,
		Currency:  currency,
		USDAmount: usdD128,
	}
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	mongoDB := db.NewMongoDB(cfg.MongoConfig)
	ctx := context.Background()
	if err := mongoDB.Connect(ctx); err != nil {
		log.Fatalf("failed to connect to mongodb: %v", err)
	}
	defer mongoDB.Disconnect(ctx)

	repo := mongorepo.NewTransactionRepository(mongoDB.DB)

	// Generate all user IDs upfront — these are reused across every round so
	// the same 700 users appear throughout the entire dataset.
	userIDs := make([]bson.ObjectID, numUsers)
	for i := range userIDs {
		userIDs[i] = bson.NewObjectID()
	}

	// Cap workers at 20 even on beefy machines — beyond that, Mongo connection
	// overhead and channel contention outweigh the gains.
	buildWorkers := min(runtime.NumCPU(), 20)
	insertWorkers := min(runtime.NumCPU(), 20)

	fmt.Printf("seeding %d rounds (%d docs) with %d build / %d insert workers...\n",
		totalRounds, totalRounds*2, buildWorkers, insertWorkers)
	start := time.Now()

	// Wire up the three stages. Each stage returns a channel that the next reads from.
	jobs := roundGenStage(totalRounds, batchSize, userIDs)
	batches := buildStage(jobs, buildWorkers)
	insertStage(batches, insertWorkers, repo, ctx) // blocks until done

	fmt.Printf("done: %d docs in %.1fs\n", int64(totalRounds)*2, time.Since(start).Seconds())
}
