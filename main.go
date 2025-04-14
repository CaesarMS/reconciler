package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
)

// Transaction represents a system transaction record
type Transaction struct {
	TrxID           string    // Unique identifier for the transaction
	Amount          float64   // Transaction amount
	Type            string    // Transaction type: DEBIT or CREDIT
	TransactionTime time.Time // When the transaction occurred
}

// BankStatement represents a bank statement record
type BankStatement struct {
	BankName         string    // Name of the bank
	UniqueIdentifier string    // Unique identifier for the bank statement
	Amount           float64   // Transaction amount
	Date             time.Time // Date of the bank statement
}

// ReconciliationSummary contains the results of reconciliation between system and bank records
type ReconciliationSummary struct {
	TotalProcessed          int                        // Total number of transactions processed
	TotalMatched            int                        // Number of transactions that were matched
	TotalUnmatched          int                        // Number of transactions that couldn't be matched
	UnmatchedSystemTxns     []Transaction              // List of system transactions without matching bank statements
	UnmatchedBankStatements map[string][]BankStatement // Map of bank statements without matching system transactions, grouped by bank
	TotalDiscrepancies      float64                    // Sum of discrepancies in monetary value
}

// parseSystemTransactions reads system transactions from a CSV file and filters by date range
func parseSystemTransactions(filePath string, startDate, endDate time.Time) ([]Transaction, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	r := csv.NewReader(file)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	var transactions []Transaction
	for i, record := range records {
		if i == 0 {
			continue // Skip the header row
		}
		amount, _ := strconv.ParseFloat(record[1], 64)
		timeParsed, _ := time.Parse("2006-01-02T15:04:05", record[3])
		if timeParsed.Before(startDate) || timeParsed.After(endDate) {
			continue
		}
		transactions = append(transactions, Transaction{
			TrxID:           record[0],
			Amount:          amount,
			Type:            record[2],
			TransactionTime: timeParsed,
		})
	}
	return transactions, nil
}

// parseBankStatement reads bank statements from a CSV file and filters by date range
func parseBankStatement(filePath, bankName string, startDate, endDate time.Time) ([]BankStatement, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	r := csv.NewReader(file)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	var statements []BankStatement
	for i, record := range records {
		if i == 0 {
			continue // Skip the header row
		}
		amount, _ := strconv.ParseFloat(record[1], 64)
		dateParsed, _ := time.Parse("2006-01-02", record[2])
		if dateParsed.Before(startDate) || dateParsed.After(endDate) {
			continue
		}
		statements = append(statements, BankStatement{
			BankName:         bankName,
			UniqueIdentifier: record[0],
			Amount:           amount,
			Date:             dateParsed,
		})
	}
	return statements, nil
}

// reconcile matches system transactions with bank statements and identifies discrepancies
func reconcile(systemTxns []Transaction, bankStatements []BankStatement) ReconciliationSummary {

	// Maps to group transactions and statements by amount for easier matching
	systemMap := make(map[float64][]Transaction)
	bankMap := make(map[float64][]BankStatement)
	unmatchedSystem := []Transaction{}
	unmatchedBank := map[string][]BankStatement{}

	discrepancySum := 0.0
	var matchedCount int

	for _, txn := range systemTxns {
		if txn.Type == "DEBIT" {
			txn.Amount = -txn.Amount
		}
		systemMap[txn.Amount] = append(systemMap[txn.Amount], txn)
	}
	for _, stmt := range bankStatements {
		bankMap[stmt.Amount] = append(bankMap[stmt.Amount], stmt)
	}

	for amt, txns := range systemMap {
		stmts := bankMap[amt]
		minLen := len(txns)
		if len(stmts) < minLen {
			minLen = len(stmts)
		}
		matchedCount += minLen
		discrepancySum += math.Abs(float64(len(txns)-len(stmts))) * amt
		if len(txns) > len(stmts) {
			unmatchedSystem = append(unmatchedSystem, txns[len(stmts):]...)
		}
	}

	unmatchedBankLength := 0
	for amt, txns := range bankMap {
		sys := systemMap[amt]
		if len(txns) > len(sys) {
			unmatchedBank[txns[len(sys)].BankName] = append(unmatchedBank[txns[len(sys)].BankName], txns[len(sys):]...)
			unmatchedBankLength += len(txns[len(sys):])
		}
	}

	totalProcessed := len(systemTxns) + len(bankStatements)
	unmatchedCount := len(unmatchedSystem) + unmatchedBankLength

	return ReconciliationSummary{
		TotalProcessed:          totalProcessed,
		TotalMatched:            matchedCount,
		TotalUnmatched:          unmatchedCount,
		UnmatchedSystemTxns:     unmatchedSystem,
		UnmatchedBankStatements: unmatchedBank,
		TotalDiscrepancies:      discrepancySum,
	}
}

// parsePathAndBankName extracts the file path and bank name from a combined string
// Format: "path/to/file:BankName"
func parsePathAndBankName(pathWithBank string) (string, string) {
	colonIdx := strings.Index(pathWithBank, ":")
	if colonIdx == -1 {
		return pathWithBank, "UnknownBank"
	}
	return pathWithBank[:colonIdx], pathWithBank[colonIdx+1:]
}

func main() {
	app := &cli.App{
		Name:  "reconciliation-service",
		Usage: "Reconcile system transactions with bank statements",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "system", Usage: "Path to system transactions CSV", Required: true},
			&cli.StringSliceFlag{Name: "bank", Usage: "Paths to bank statements CSV(s) with optional bank name separated by ':'", Required: true},
			&cli.StringFlag{Name: "start", Usage: "Start date (YYYY-MM-DD)", Required: true},
			&cli.StringFlag{Name: "end", Usage: "End date (YYYY-MM-DD)", Required: true},
		},
		Action: func(ctx *cli.Context) error {
			systemPath := ctx.String("system")
			bankFiles := ctx.StringSlice("bank")
			start, _ := time.Parse("2006-01-02", ctx.String("start"))
			end, _ := time.Parse("2006-01-02", ctx.String("end"))

			systemTxns, err := parseSystemTransactions(systemPath, start, end)
			if err != nil {
				return err
			}

			var bankStatements []BankStatement
			var wg sync.WaitGroup
			var mu sync.Mutex

			for _, pathWithBank := range bankFiles {
				wg.Add(1)
				go func(p string) {
					defer wg.Done()
					path, bank := parsePathAndBankName(p)
					stmts, err := parseBankStatement(path, bank, start, end)
					if err != nil {
						log.Printf("Error reading %s: %v", path, err)
						return
					}
					mu.Lock()
					bankStatements = append(bankStatements, stmts...)
					mu.Unlock()
				}(pathWithBank)
			}
			wg.Wait()

			summary := reconcile(systemTxns, bankStatements)
			fmt.Printf("Total Processed: %d\n", summary.TotalProcessed)
			fmt.Printf("Matched: %d\n", summary.TotalMatched)
			fmt.Printf("Unmatched: %d\n", summary.TotalUnmatched)
			fmt.Printf("Discrepancies: %.2f\n", summary.TotalDiscrepancies)
			fmt.Println("Unmatched System Transactions:")
			for _, txn := range summary.UnmatchedSystemTxns {
				fmt.Printf("- %s | %.2f | %s\n", txn.TrxID, txn.Amount, txn.TransactionTime.Format(time.RFC3339))
			}
			for bank, stmts := range summary.UnmatchedBankStatements {
				fmt.Printf("Unmatched %s Transactions:\n", bank)
				for _, stmt := range stmts {
					fmt.Printf("- %s | %.2f | %s\n", stmt.UniqueIdentifier, stmt.Amount, stmt.Date.Format("2006-01-02"))
				}
			}
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
