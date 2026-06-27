package repository

// Repos holds all repository instances. Add a new field here when a new repository is introduced.
type Repos struct {
	Transaction TransactionRepository
	DailyStats  DailyStatsRepository
}
