# Reconciler

## Overview

The Reconciler application is designed to reconcile system transactions with bank statements. It processes CSV files containing transaction data and bank statements, matches them, and provides a summary of matched and unmatched transactions along with any discrepancies.

## Setup

1. Clone the repository:

   ```sh
   git clone https://github.com/CaesarMS/reconciler.git
   ```

2. Install dependencies:
   ```sh
   go mod tidy
   ```

## Running the Application

To run the application, use the following command:

    go run main.go \
      --system <path-to-system-transactions-csv> \
      --bank <path-to-bank-statement-csv>:<bank-name> \
      --start <start-date> \
      --end <end-date>

### Example

To run the application with example data, use the following command:

    go run main.go \
      --system ./system_transactions.csv \
      --bank ./bank_a.csv:BankA \
      --bank ./bank_b.csv:BankB \
      --start 2025-01-01 \
      --end 2025-03-31

## Output

The application will output a reconciliation summary, which includes:

- Total number of processed transactions
- Total number of matched transactions
- Total number of unmatched transactions
- List of unmatched system transactions
- List of unmatched bank statements grouped by bank
- Total discrepancies amount

### Example Output

After running the application with the example data, you might see an output similar to the following:

    Total Processed: 15
    Matched: 5
    Unmatched: 5
    Discrepancies: 50.00
    Unmatched System Transactions:
    - TRX006 | -50.00 | 2025-03-02T12:00:00Z
    Unmatched BankA Transactions:
    - BKA006 | 100.00 | 2025-02-16
    - BKA005 | 170.00 | 2025-02-16
    Unmatched BankB Transactions:
    - BKB003 | 650.00 | 2025-03-06
    - BKB002 | -999.99 | 2025-03-05
