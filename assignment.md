# GoLang/MongoDB Developer Interview Exercise: Admin Statistics API

**Objective:**

Develop a GoLang REST API that demonstrates proficiency in modern backend development and advanced MongoDB queries. Create an admin API that aggregates useful data from a large MongoDB collection of user transactions.

## Technology Stack

- Language: GoLang
- Database: MongoDB (Can optionally use Redis in addition to MongoDB for caching)
- Mongo Package: Official MongoDB package (<https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo>)
- API/Router Package: Your choice of the native `net/http` package, `gorilla/mux`, or `gin-gonic/gin`
- Validation Package: `go-playground/validator`
- Deployment: Provide a working local development setup

**Setup:**
Create a utility script to fill a mongo "transactions" collection with randomly generated sample data using the schema provided below. The transaction collection stores user wagers & payouts from casino game rounds.

- Each round should have a minimum of one "Wager" transaction and one "Payout" transaction.
- Insert >=2,000,000 game rounds worth of transactions with >=500 unique user IDs.
- Each game round should have a randomly generated `createdAt` time within the past year, and the "Payout" transaction should always have a later `createdAt` than the corresponding "Wager" transaction.
- All transactions with a matching `roundId` should have a matching `currency` field.

``` Golang
type Transaction struct {
    ID        primitive.ObjectID   `bson:"_id"`
    CreatedAt time.Time            `bson:"createdAt"`
    UserID    primitive.ObjectID   `bson:"userId"`
    RoundID   string               `bson:"roundId"`
    Type      string               `bson:"type"`      // Either "Wager" or "Payout"
    Amount    primitive.Decimal128 `bson:"amount"`    // Should always be >= 0
    Currency  string               `bson:"currency"`  // Either "ETH", "BTC", or "USDT"
    USDAmount primitive.Decimal128 `bson:"usdAmount"` // The USD value of the `amount` and `currency` in this transaction, can use static conversion rates for this example.
}
```

### API Requirements

- Authentication Middleware: Validate a static "Authorization" header on all routes
- Timeframe Selection: Every route should take from and to dates to narrow down the timeframe of the data were querying.
- Queries: Utilize mongo aggregation queries to condense the raw transactions into useful data.
- Caching: Use any form of caching you deem fit to provide max efficiency when querying large amounts of transactions.
- Input Validation: Validate all user input into the API

### API Routes

- GET /gross_gaming_rev?from=<from_date>&to=<to_date> - Calculate the Gross Gaming Revenue(GGR) in each crypto currency and USD over the specified timeframe.
- GET /daily_wager_volume?from=<from_date>&to=<to_date> - Calculate the daily wager volume in each crypto currency and USD over the specified timeframe.
- GET /user/<user_id>/wager_percentile?from=<from_date>&to=<to_date> - Calculate what percentile the user is from their total USD wagered amount over the specified timeframe. For example: If a user is the 10th highest wagerer out of 500 users, then they are in the top 2%

### Evaluation Criteria

- Code Quality: Clean, maintainable code following modern best practices
- Technical Decisions: Ability to explain and justify implementation choices
- Query Optimization: Ability to understand and implement effecient Mongo queries.
