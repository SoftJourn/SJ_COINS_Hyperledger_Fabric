package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"log"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type CoinChain struct {
	contractapi.Contract
}

type TransferRequest struct {
	UserId string `json:"userId"`
	Amount int    `json:"amount"`
}

type UserBalance struct {
	UserId  string `json:"userId"`
	Balance int    `json:"balance"`
}

type ExpirableTransaction struct {
  Amount    int
  TxId      string
  CreatedAt int64
}

var currencyName string

var minterKey = "minter"
var balancesKey = "balances"
var expirableBalancesKey = "expirableBalances"
var expirableTransactionsKey = "expirableTransactions"
var expirationPeriod = int64(3600)
var currencyKey = "currency"

var userAccountType = "user_"

// For TransferFrom
var txBalancesMap map[string]int
var lastTxId string

func (t *CoinChain) InitLedger(ctx contractapi.TransactionContextInterface) (string, error) {

	/* args
	0 - minter ID
	1 - Currency name
	*/

	_, args := ctx.GetStub().GetFunctionAndParameters()

	if len(args) != 2 {
		return "-1", fmt.Errorf("incorrect number of arguments. Expected 2, was %d", len(args))
	}

	currencyName = args[1]

	fmt.Println("_____ Init " + currencyName + "_____")

	currentUserId, err := getCurrentUserId(ctx)
	if err != nil {
		return currencyName, err
	}

	err = ctx.GetStub().PutState(currencyKey, []byte(currencyName))
	if err != nil {
		return currencyName, err
	}

	fmt.Println("minter ID: " + args[0])

	minterBytes := []byte(args[0])

	err = ctx.GetStub().PutState(minterKey, minterBytes)
	if err != nil {
		return currencyName, err
	}

	currentUserAccount, err := ctx.GetStub().CreateCompositeKey(userAccountType, []string{currentUserId})
	if err != nil {
		return currencyName, err
	}

	fmt.Println("currentUserAccount: " + currentUserAccount)

  // Init account balances map.
	balancesMap := t.getMap(ctx, balancesKey)
	if len(balancesMap) == 0 {
		balancesMap = map[string]int{currentUserAccount: 0}
		err = t.saveMap(ctx, balancesKey, balancesMap)
		if err != nil {
			return currencyName, err
		}
	}

	// Init account expirable balances map.
  expirableBalancesMap := t.getMap(ctx, expirableBalancesKey)
  if len(expirableBalancesMap) == 0 {
    expirableBalancesMap = map[string]int{currentUserAccount: 0}
    err = t.saveMap(ctx, expirableBalancesKey, expirableBalancesMap)
    if err != nil {
      return currencyName, err
    }
  }

  // Init expirable transaction map.
	expirableTransactionsMap := t.getExpirableTransactionMap(ctx)
  if len(expirableTransactionsMap) == 0 {
    expirableTransactionsMap = map[string][]*ExpirableTransaction{}
    err = t.saveExpirableTransactionMap(ctx, expirableTransactionsMap)
    if err != nil {
      return currencyName, err
    }
  }

	return currencyName, nil
}

func (t *CoinChain) Transfer(ctx contractapi.TransactionContextInterface, receiverAccountType string, receiver string, amount int, expirable bool) (*UserBalance, error) {

	fmt.Println("AccountType: " + receiverAccountType)
	fmt.Println("Receiver: " + receiver)
	fmt.Println("Amount: " + string(amount))

	if (amount == 0) {
		return nil, errors.New("Incorrect amount. Amount should be positive.")
	}

	currentUserId, err := getCurrentUserId(ctx)
	if (err != nil) { return nil, err }

	currentUserAccount, err := ctx.GetStub().CreateCompositeKey(userAccountType, []string{currentUserId})
	if (err != nil) { return nil, err }
	fmt.Println("CurrentUserAccount: " + currentUserAccount)

	receiverAccount, err := ctx.GetStub().CreateCompositeKey(receiverAccountType, []string{receiver})
	if (err != nil) { return nil, err }
	fmt.Println("ReceiverAccount: " + receiverAccount)

	balance, err := t.getBalance(ctx, currentUserAccount)
	if (err != nil) { return nil, err }
	if (balance.Balance < amount) {
		return nil, errors.New("Not enough coins")
	}

	t.decreaseAccountBalance(ctx, currentUserAccount, amount)
	if (expirable) {
		t.addExpirableTransaction(ctx, receiverAccount, uuid.New().String(), amount, time.Now().Unix())
	} else {
		transfers := make([]*TransferRequest, 0)
		transfers = addTransfer(transfers, receiverAccount, amount)

		err = t.changeBalance(ctx, transfers)
		if (err != nil) {
			return nil, err
		}
	}

	// Do not invoke BalanceOf method. At this time ledger is not updated yet.
	return t.getBalance(ctx, currentUserAccount)
}

func (t *CoinChain) BatchTransfer(ctx contractapi.TransactionContextInterface, transferRequestsJson string) (*UserBalance, error) {
	fmt.Println("transfer requests json: " + transferRequestsJson)

	var transferRequests []TransferRequest
	err := json.Unmarshal([]byte(transferRequestsJson), &transferRequests)

	if err != nil {
		return nil, err
	}

	fmt.Println(transferRequests)

	var total = 0

	for _, tr := range transferRequests {
		total += tr.Amount
	}

	currentUserId, err := getCurrentUserId(ctx)
	if err != nil {
		return nil, err
	}

	currentUserAccount, err := ctx.GetStub().CreateCompositeKey(userAccountType, []string{currentUserId})

	if err != nil {
		return nil, err
	}

	fmt.Println("currentUserAccount " + currentUserAccount)

	balancesMap := t.getMap(ctx, balancesKey)

	currentUserBalance := balancesMap[currentUserAccount]

	fmt.Println("currentUserBalance ", currentUserBalance)

	if total > currentUserBalance {
		return nil, errors.New("not enough money")
	}

	for _, tr := range transferRequests {
		receiverAccount, err := ctx.GetStub().CreateCompositeKey(userAccountType, []string{tr.UserId})
		if err != nil {
			return nil, err
		}
		balancesMap[currentUserAccount] -= tr.Amount
		balancesMap[receiverAccount] += tr.Amount
	}

	err = t.saveMap(ctx, balancesKey, balancesMap)
	if err != nil {
		return nil, err
	}

	// Do not invoke BalanceOf method. At this time ledger is not updated yet.
	balancesResponse := new(UserBalance)
	balancesResponse.UserId = currentUserAccount
	balancesResponse.Balance = balancesMap[currentUserAccount]

	return balancesResponse, nil
}

func (t *CoinChain) Refund(ctx contractapi.TransactionContextInterface, projectId string, receiver string, amount int) (*UserBalance, error) {

	fmt.Println("receiver " + receiver)
	fmt.Println("amount " + string(amount))

	if amount == 0 {
		return nil, errors.New("incorrect amount")
	}

	currentUserId, err := getCurrentUserId(ctx)
	if err != nil {
		return nil, err
	}

	minterBytes, err := ctx.GetStub().GetState(minterKey)
	if err != nil {
		return nil, err
	}

	minterString := string(minterBytes)
	fmt.Println("minter " + minterString)

	if currentUserId != minterString {
		return nil, errors.New("no permissions")
	}

	projectAccount, err := ctx.GetStub().CreateCompositeKey("project_", []string{projectId})
	if err != nil {
		return nil, err
	}

	fmt.Println("projectAccount " + projectAccount)

	receiverAccount, err := ctx.GetStub().CreateCompositeKey("user_", []string{receiver})
	if err != nil {
		return nil, err
	}

	fmt.Println("receiverAccount " + receiverAccount)

	balancesMap := t.getMap(ctx, balancesKey)

	if balancesMap[projectAccount] < amount {
		return nil, errors.New("not enough coins")
	}

	balancesMap[projectAccount] -= amount
	balancesMap[receiverAccount] += amount

	err = t.saveMap(ctx, balancesKey, balancesMap)
	if err != nil {
		return nil, err
	}

	// Do not invoke BalanceOf method. At this time ledger is not updated yet.
	balancesResponse := new(UserBalance)
	balancesResponse.UserId = projectAccount
	balancesResponse.Balance = balancesMap[projectAccount]

	return balancesResponse, nil
}

func (t *CoinChain) TransferFrom(ctx contractapi.TransactionContextInterface, fromType string, from string, toType string, to string, amount uint) (*UserBalance, error) {

	fmt.Println("from " + from)
	fmt.Println("to " + to)
	fmt.Println("amount " + string(amount))

	if amount == 0 {
		return nil, errors.New("incorrect amount")
	}

	fromAccount, err := ctx.GetStub().CreateCompositeKey(fromType, []string{from})
	if err != nil {
		return nil, err
	}

	fmt.Println("fromAccount " + fromAccount)

	toAccount, err := ctx.GetStub().CreateCompositeKey(toType, []string{to})
	if err != nil {
		return nil, err
	}

	fmt.Println("toAccount " + toAccount)

	balancesMap := t.getMap(ctx, balancesKey)

	if balancesMap[fromAccount] < int(amount) {
		return nil, errors.New("not enough coins")
	}

	balancesMap[fromAccount] -= int(amount) // TODO: Such conversion isn't good. Should be changed to safe conversion.
	balancesMap[toAccount] += int(amount)

	err = t.saveMap(ctx, balancesKey, balancesMap)
	if err != nil {
		return nil, err
	}

	// Do not invoke BalanceOf method. At this time ledger is not updated yet.
	balancesResponse := new(UserBalance)
	balancesResponse.UserId = fromAccount
	balancesResponse.Balance = balancesMap[fromAccount]

	return balancesResponse, nil
}

func (t *CoinChain) BatchRefund(ctx contractapi.TransactionContextInterface, projectId string, transferRequestsJson string) (*UserBalance, error) {
	fmt.Println("refund requests json: " + transferRequestsJson)

	var transferRequests []TransferRequest
	err := json.Unmarshal([]byte(transferRequestsJson), &transferRequests)

	if err != nil {
		return nil, err
	}

	fmt.Println(transferRequests)

	var total = 0

	for _, tr := range transferRequests {
		total += tr.Amount
	}

	currentUserId, err := getCurrentUserId(ctx)
	if err != nil {
		return nil, err
	}

	minterBytes, err := ctx.GetStub().GetState(minterKey)
	if err != nil {
		return nil, err
	}

	minterString := string(minterBytes)
	fmt.Println("minter " + minterString)

	if currentUserId != minterString {
		return nil, errors.New("no permissions")
	}

	projectAccount, err := ctx.GetStub().CreateCompositeKey("project_", []string{projectId})

	if err != nil {
		return nil, err
	}

	fmt.Println("projectAccount " + projectAccount)

	balancesMap := t.getMap(ctx, balancesKey)

	currentProjectBalance := balancesMap[projectAccount]

	fmt.Println("currentProjectBalance ", currentProjectBalance)

	if total != currentProjectBalance {
		return nil, errors.New("all money must be refunded")
	}

	for _, tr := range transferRequests {
		receiverAccount, err := ctx.GetStub().CreateCompositeKey(userAccountType, []string{tr.UserId})
		if err != nil {
			return nil, err
		}
		balancesMap[projectAccount] -= tr.Amount
		balancesMap[receiverAccount] += tr.Amount
	}

	err = t.saveMap(ctx, balancesKey, balancesMap)
	if err != nil {
		return nil, err
	}

	// Do not invoke BalanceOf method. At this time ledger is not updated yet.
	balancesResponse := new(UserBalance)
	balancesResponse.UserId = projectAccount
	balancesResponse.Balance = balancesMap[projectAccount]

	return balancesResponse, nil
}

func (t *CoinChain) Mint(ctx contractapi.TransactionContextInterface, amount int) (*UserBalance, error) {

	fmt.Println("mint amount: " + string(amount))

	minterBytes, err := ctx.GetStub().GetState(minterKey)
	if err != nil {
		return nil, err
	}

	minterString := string(minterBytes)
	fmt.Println("minter " + minterString)

	currentUserId, err := getCurrentUserId(ctx)
	if err != nil {
		return nil, err
	}

	if currentUserId != minterString {
		return nil, errors.New("no permissions")
	}

	if amount == 0 {
		return nil, errors.New("incorrect amount")
	}

	currentUserAccount, err := ctx.GetStub().CreateCompositeKey(userAccountType, []string{currentUserId})
	if err != nil {
		return nil, err
	}

	fmt.Println("currentUserAccount " + currentUserAccount)

	balancesMap := t.getMap(ctx, balancesKey)

	balancesMap[currentUserAccount] += amount

	err = t.saveMap(ctx, balancesKey, balancesMap)
	if err != nil {
		return nil, err
	}

	// Do not invoke BalanceOf method. At this time ledger is not updated yet.
	balancesResponse := new(UserBalance)
	balancesResponse.UserId = currentUserAccount
	balancesResponse.Balance = balancesMap[currentUserAccount]

	return balancesResponse, nil
}

func (t *CoinChain) BalanceOf(ctx contractapi.TransactionContextInterface, accountType string, accountId string) (*UserBalance, error) {

	fmt.Println("accountType " + accountType)
	fmt.Println("accountId " + accountId)

	account, err := ctx.GetStub().CreateCompositeKey(accountType, []string{accountId})
	if err != nil {
		return nil, err
	}

	fmt.Println("account " + account)

	return t.getBalance(ctx, account)
}

func (t *CoinChain) BatchBalanceOf(ctx contractapi.TransactionContextInterface, emails []string) ([]*UserBalance, error) {

	fmt.Println("userId " + strings.Join(emails, ", "))

	var balancesResponse []*UserBalance

	balancesMap := t.getMap(ctx, balancesKey)

	for _, email := range emails {
		account, err := ctx.GetStub().CreateCompositeKey(userAccountType, []string{email})
		if err != nil {
			return nil, err
		}

		fmt.Println("account " + account)

		balance := new(UserBalance)
		balance.UserId = email
		balance.Balance = balancesMap[account]
		balancesResponse = append(balancesResponse, balance)
	}

	return balancesResponse, nil
}

func (t *CoinChain) AllBalances(ctx contractapi.TransactionContextInterface) ([]*UserBalance, error) {

	balancesMap := t.getMap(ctx, balancesKey)

	keys := reflect.ValueOf(balancesMap).MapKeys()

	var balancesResponse []*UserBalance

	for i := 0; i < len(keys); i++ {
		account := keys[i].String()

		fmt.Println("account " + account)

		balance := new(UserBalance)
		balance.UserId = t.trimCompositeKey(account)
		balance.Balance = balancesMap[account]
		balancesResponse = append(balancesResponse, balance)
	}

	return balancesResponse, nil
}

func getCurrentUserId(ctx contractapi.TransactionContextInterface) (string, error) {

	var userId string

	creatorBytes, err := ctx.GetStub().GetCreator()
	if err != nil {
		return userId, err
	}

	creatorString := fmt.Sprintf("%s", creatorBytes)

	index := strings.Index(creatorString, "-----BEGIN CERTIFICATE-----")

	if index == -1 {
		index = strings.Index(creatorString, "-----BEGIN -----")
	}

	certificate := creatorString[index:]
	block, _ := pem.Decode([]byte(certificate))

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return userId, err
	}

	userId = cert.Subject.CommonName
	return userId, err
}

// Get arbitrary string to int map.
func (t *CoinChain) getMap(ctx contractapi.TransactionContextInterface, mapName string) map[string]int {

	fmt.Println("------ getMap called")

	mapBytes, err := ctx.GetStub().GetState(mapName)
	if err != nil {
		return nil
	}

	var mapObject map[string]int
	err = json.Unmarshal(mapBytes, &mapObject)
	if err != nil {
		return nil
	}

	return mapObject
}

// Get arbitrary string to array of expirable transactions map.
func (t *CoinChain) getExpirableTransactionMap(ctx contractapi.TransactionContextInterface) map[string][]*ExpirableTransaction {

	fmt.Println("------ getExpirableTransactionMap called")

	mapBytes, err := ctx.GetStub().GetState(expirableTransactionsKey)
	if err != nil { return nil }

	var mapObject map[string][]*ExpirableTransaction
	err = json.Unmarshal(mapBytes, &mapObject)
	if err != nil { return nil }

	return mapObject
}

// Save arbitrary string to int map.
func (t *CoinChain) saveMap(ctx contractapi.TransactionContextInterface, mapName string, mapObject map[string]int) error {
	fmt.Println("------ saveBalancesMap called")

	balancesMapBytes, err := json.Marshal(mapObject)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(mapName, balancesMapBytes)
	if err != nil {
		return err
	}

	return nil
}

// Save arbitrary string to array of expirable transactions map.
func (t *CoinChain) saveExpirableTransactionMap(ctx contractapi.TransactionContextInterface, mapObject map[string][]*ExpirableTransaction) error {
	fmt.Println("------ saveExpirableTransactionMap called")

	balancesMapBytes, err := json.Marshal(mapObject)
	if err != nil { return err }

	err = ctx.GetStub().PutState(expirableTransactionsKey, balancesMapBytes)
	if err != nil { return err }

	return nil
}

func (t *CoinChain) GetTransactionBalancesMap(ctx contractapi.TransactionContextInterface) map[string]int {

	txId := ctx.GetStub().GetTxID()

	fmt.Println("lastTxId " + lastTxId)
	fmt.Println("txId " + txId)

	if txId == lastTxId {
		return txBalancesMap
	} else {
		txBalancesMap = t.getMap(ctx, balancesKey)
		lastTxId = txId
	}
	return txBalancesMap
}

func (t *CoinChain) trimCompositeKey(inputStr string) string {
	reg, err := regexp.Compile("[^a-zA-Z0-9@.!#$%&'*+-/=?^_`{|}~]+")
	if err != nil {
		log.Fatal(err)
	}

	result := reg.ReplaceAllString(inputStr, "")
	result = strings.TrimPrefix(result, userAccountType)

	return result
}

// Get account balance including burning wallet balance.
func (t *CoinChain) getBalance(ctx contractapi.TransactionContextInterface, account string) (*UserBalance, error) {
  // Permanent balance.
  balancesMap := t.getMap(ctx, balancesKey)
  balancesResponse := new(UserBalance)
  balancesResponse.UserId = account
  balancesResponse.Balance = balancesMap[account]

  // Expirable balance.
  t.flushExpiredTransactions(ctx, account)
  expirableBalancesMap := t.getMap(ctx, expirableBalancesKey)
  if (expirableBalancesMap[account] > 0) {
    balancesResponse.Balance = balancesResponse.Balance + expirableBalancesMap[account]
  }

  return balancesResponse, nil
}

// Add transaction to burning wallet and update balance map.
func (t *CoinChain) addExpirableTransaction(ctx contractapi.TransactionContextInterface, account string, txId string, amount int, createdAt int64) (error) {
  expirableTransactionsMap := t.getExpirableTransactionMap(ctx)

  expirableTransactions := expirableTransactionsMap[account]
  if (len(expirableTransactions) == 0) {
    expirableTransactions = make([]*ExpirableTransaction, 0)
  }

  transaction := new(ExpirableTransaction)
  transaction.TxId = txId
  transaction.Amount = amount
  transaction.CreatedAt = createdAt
  expirableTransactions = append(expirableTransactions, transaction)

  expirableTransactionsMap[account] = expirableTransactions
  err := t.saveExpirableTransactionMap(ctx, expirableTransactionsMap)
  if err != nil { return err }

	transfers := make([]*TransferRequest, 0)
	transfer := new(TransferRequest)
	transfer.UserId = account
	transfer.Amount = transaction.Amount
	transfers = append(transfers, transfer)
  return t.changeExpirableBalance(ctx, transfers)
}

// Burn outdated transactions and update balance map.
func (t *CoinChain) flushExpiredTransactions(ctx contractapi.TransactionContextInterface, account string) (error) {
  expirableTransactionsMap := t.getExpirableTransactionMap(ctx)
  expirableBalancesMap := t.getMap(ctx, expirableBalancesKey)

  expirableTransactions := expirableTransactionsMap[account]
  if (len(expirableTransactions) == 0) {
    return nil
  }

  indicesToRemove := make([]int, 0)
	now := time.Now().Unix()
  for i, v := range expirableTransactions {
		if (v.CreatedAt + expirationPeriod < now) {
		  expirableBalancesMap[account] = expirableBalancesMap[account] - v.Amount
		  if (expirableBalancesMap[account] < 0) {
		    return errors.New("Expirable balance of account got negative value")
		  }

		  indicesToRemove = append(indicesToRemove, i)
		} else {
		  break
		}
	}

  if (len(indicesToRemove) > 0) {
    for i := len(indicesToRemove)-1; i >= 0; i-- {
      expirableTransactions = append(expirableTransactions[:i], expirableTransactions[i+1:]...)
    }

    expirableTransactionsMap[account] = expirableTransactions
    err := t.saveExpirableTransactionMap(ctx, expirableTransactionsMap)
    if err != nil { return err }

    err = t.saveMap(ctx, expirableBalancesKey, expirableBalancesMap)
    if err != nil { return err }
  }

  return nil
}

// Decrese account balance starting from oldest expirable transactions down to permanent balance.
func (t *CoinChain) decreaseAccountBalance(ctx contractapi.TransactionContextInterface, account string, amount int) (*UserBalance, error) {
	balance, err := t.getBalance(ctx, account)
	if (err != nil) { return nil, err }
	if (amount < 1) { return balance, nil }
	if (balance.Balance < amount) {
		return nil, errors.New("Balance amount is less than needed")
	}

	expirableTransactionsMap := t.getExpirableTransactionMap(ctx)
	expirableTransactions := expirableTransactionsMap[account]
	if (len(expirableTransactions) > 0) {
		transactionIdsToRemove := make([]int, 0)
		decreasedAmount := 0

		for i, v := range expirableTransactions {
			if (v.Amount < amount) {
				amount -= v.Amount
				decreasedAmount += v.Amount
				transactionIdsToRemove = append(transactionIdsToRemove, i)
			} else {
				expirableTransactions[i].Amount = v.Amount - amount
				decreasedAmount += amount
				amount = 0

				if (expirableTransactions[i].Amount < 1) {
					transactionIdsToRemove = append(transactionIdsToRemove, i)
				}
			}
		}

		if (decreasedAmount > 0) {
			transfers := make([]*TransferRequest, 0)
			transfer := new(TransferRequest)
			transfer.UserId = account
			transfer.Amount = decreasedAmount
			transfers = append(transfers, transfer)

			err = t.changeExpirableBalance(ctx, transfers)
			if (err != nil) { return nil, err }
		}

		for i := len(transactionIdsToRemove)-1; i >= 0; i-- {
      expirableTransactions = append(expirableTransactions[:i], expirableTransactions[i+1:]...)
    }

		expirableTransactionsMap[account] = expirableTransactions
    err = t.saveExpirableTransactionMap(ctx, expirableTransactionsMap)
    if (err != nil) { return nil, err }
	}

	if (amount > 0) {
		transfers := make([]*TransferRequest, 0)
		transfers = t.addTransfer(transfers, account, -1 * amount)
		t.changeBalance(ctx, transfers)
	}

	return t.getBalance(ctx, account)
}

// Wrapper for permanent balance value change.
func (t *CoinChain) changeBalance(ctx contractapi.TransactionContextInterface, transfers []*TransferRequest) (error) {
	return t.changeBalanceInternal(ctx, balancesKey, transfers, "Permanent")
}

// Wrapper for expirable balance value change.
func (t *CoinChain) changeExpirableBalance(ctx contractapi.TransactionContextInterface, transfers []*TransferRequest) (error) {
	return t.changeBalanceInternal(ctx, expirableBalancesKey, transfers, "Expirable")
}

// Actual logic of balance change.
func (t *CoinChain) changeBalanceInternal(ctx contractapi.TransactionContextInterface, key string, transfers []*TransferRequest, balanceType string) (error) {
	balanceMap := t.getMap(ctx, key)

	for _, transfer := range transfers {
		if (balanceMap[transfer.UserId] > 0) {
			balanceMap[transfer.UserId] = balanceMap[transfer.UserId] + transfer.Amount
		} else {
			balanceMap[transfer.UserId] = transfer.Amount
		}

		if (balanceMap[transfer.UserId] < 0) {
			return errors.New(balanceType + " balance of account '" + transfer.UserId + "' is below zero.")
		}
	}

	return t.saveMap(ctx, key, balanceMap)
}

func (t *CoinChain) addTransfer(transfers []*TransferRequest, account string, amount int) ([]*TransferRequest) {
	transfer := new(TransferRequest)
	transfer.UserId = account
	transfer.Amount = amount
	transfers = append(transfers, transfer)
	return transfers
}

func main() {

	chaincode, err := contractapi.NewChaincode(new(CoinChain))

	if err != nil {
		fmt.Printf("Error create fabcar chaincode: %s", err.Error())
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting fabcar chaincode: %s", err.Error())
	}
}
