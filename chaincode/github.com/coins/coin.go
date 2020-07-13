package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type CoinChain struct {
	contractapi.Contract
}

type TransferRequest struct {
	UserId string `json:"userId"`
	Amount uint   `json:"amount"`
}

type UserBalance struct {
	UserId  string `json:"userId"`
	Balance uint   `json:"balance"`
}

var currencyName string

var minterKey = "minter"
var balancesKey = "balances"
var currencyKey = "currency"

var userAccountType = "user_"

//For TransferFrom
var txBalancesMap map[string]uint
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

	balancesMap := t.getMap(ctx, balancesKey)

	if len(balancesMap) == 0 {
		balancesMap = map[string]uint{currentUserAccount: 0}
		err = t.saveMap(ctx, balancesKey, balancesMap)
		if err != nil {
			return currencyName, err
		}
	}

	return currencyName, nil
}

func (t *CoinChain) Transfer(ctx contractapi.TransactionContextInterface, args []string) error {

	/* args
	0 - accountType (user_ , foundation_)
	1 - receiver ID
	2 - amount
	*/

	if len(args) != 3 {
		return errors.New("incorrect number of arguments. Expecting 3")
	}

	receiverAccountType := args[0]
	fmt.Println("accountType: " + receiverAccountType)

	receiver := args[1]
	fmt.Println("receiver " + receiver)

	fmt.Println("args[2] " + args[2])
	amount := t.parseAmountUint(args[2])
	fmt.Println("amount " + string(amount))

	if amount == 0 {
		return errors.New("incorrect amount")
	}

	currentUserId, err := getCurrentUserId(ctx)
	if err != nil {
		return err
	}

	currentUserAccount, err := ctx.GetStub().CreateCompositeKey(userAccountType, []string{currentUserId})
	if err != nil {
		return err
	}

	fmt.Println("currentUserAccount " + currentUserAccount)

	receiverAccount, err := ctx.GetStub().CreateCompositeKey(receiverAccountType, []string{receiver})
	if err != nil {
		return err
	}

	fmt.Println("receiverAccount " + receiverAccount)

	balancesMap := t.getMap(ctx, balancesKey)

	if balancesMap[currentUserAccount] < amount {
		return errors.New("not enough coins")
	}

	balancesMap[currentUserAccount] -= amount
	balancesMap[receiverAccount] += amount

	err = t.saveMap(ctx, balancesKey, balancesMap)
	if err != nil {
		return err
	}

	return nil
}

func (t *CoinChain) BatchTransfer(ctx contractapi.TransactionContextInterface, args []string) error {

	/* args
	0 - the array of the TransferRequest
	*/

	if len(args) != 1 {
		return errors.New("incorrect number of arguments. Expecting 1")
	}

	var transferRequests []TransferRequest
	err := json.Unmarshal([]byte(args[0]), &transferRequests)

	if err != nil {
		return err
	}

	fmt.Println(transferRequests)

	var total uint = 0

	for _, tr := range transferRequests {
		total += tr.Amount
	}

	currentUserId, err := getCurrentUserId(ctx)
	if err != nil {
		return err
	}

	currentUserAccount, err := ctx.GetStub().CreateCompositeKey(userAccountType, []string{currentUserId})

	if err != nil {
		return err
	}

	fmt.Println("currentUserAccount " + currentUserAccount)

	balancesMap := t.getMap(ctx, balancesKey)

	currentUserBalance := balancesMap[currentUserAccount]

	fmt.Println("currentUserBalance ", currentUserBalance)

	if total > currentUserBalance {
		return errors.New("not enough money")
	}

	for _, tr := range transferRequests {
		receiverAccount, err := ctx.GetStub().CreateCompositeKey(userAccountType, []string{tr.UserId})
		if err != nil {
			return err
		}
		balancesMap[currentUserAccount] -= tr.Amount
		balancesMap[receiverAccount] += tr.Amount
	}

	err = t.saveMap(ctx, balancesKey, balancesMap)
	if err != nil {
		return err
	}

	return nil
}

func (t *CoinChain) SetCurrency(ctx contractapi.TransactionContextInterface, args []string) error {

	//Obsolete (setColor) not sure we need this. Chaincode name is currency name

	/* args
	0 - currency name
	*/

	if len(args) != 1 {
		return errors.New("incorrect number of arguments. Expecting 1")
	}

	minterValue, err := ctx.GetStub().GetState(minterKey)
	if err != nil {
		return err
	}

	currentUserId, err := getCurrentUserId(ctx)
	if err != nil {
		return err
	}

	if reflect.DeepEqual([]byte(currentUserId), minterValue) {
		return errors.New("user has no permissions")
	}

	currency := args[0]

	err = ctx.GetStub().PutState(currencyKey, []byte(currency))
	if err != nil {
		return err
	}

	return nil
}

func (t *CoinChain) GetCurrency(ctx contractapi.TransactionContextInterface) (string, error) {

	currency, err := ctx.GetStub().GetState(currencyKey)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + currencyKey + "\"}"
		return "", errors.New(jsonResp)
	}

	return string(currency), nil
}

func (t *CoinChain) getMap(ctx contractapi.TransactionContextInterface, mapName string) map[string]uint {

	fmt.Println("------ getMap called")

	mapBytes, err := ctx.GetStub().GetState(mapName)
	if err != nil {
		return nil
	}

	var mapObject map[string]uint
	err = json.Unmarshal(mapBytes, &mapObject)
	if err != nil {
		return nil
	}

	return mapObject
}

func (t *CoinChain) saveMap(ctx contractapi.TransactionContextInterface, mapName string, mapObject map[string]uint) error {
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

func (t *CoinChain) Mint(ctx contractapi.TransactionContextInterface, args []string) error {

	/* args
	0 - amount
	*/

	if len(args) != 1 {
		return errors.New("incorrect number of arguments. Expecting 1")
	}

	minterBytes, err := ctx.GetStub().GetState(minterKey)
	if err != nil {
		return err
	}

	minterString := string(minterBytes)
	fmt.Println("minter " + minterString)

	currentUserId, err := getCurrentUserId(ctx)
	if err != nil {
		return err
	}

	if currentUserId != minterString {
		return errors.New("no permissions")
	}

	amount := t.parseAmountUint(args[0])
	if amount == 0 {
		return errors.New("incorrect amount")
	}

	currentUserAccount, err := ctx.GetStub().CreateCompositeKey(userAccountType, []string{currentUserId})
	if err != nil {
		return err
	}

	fmt.Println("currentUserAccount " + currentUserAccount)

	balancesMap := t.getMap(ctx, balancesKey)

	balancesMap[currentUserAccount] += amount

	err = t.saveMap(ctx, balancesKey, balancesMap)
	if err != nil {
		return err
	}

	return nil
}

func (t *CoinChain) Distribute(ctx contractapi.TransactionContextInterface, args []string) error {

	/* args
	0.. n-1 - accounts
	n - amount
	*/

	if len(args) < 3 {
		return errors.New("incorrect number of arguments. Expecting at least 3")
	}

	amount := t.parseAmountUint(args[len(args)-1])
	if amount == 0 {
		return errors.New("incorrect amount")
	}

	accounts := args[:len(args)-1]

	fmt.Println("accounts: " + strings.Join(accounts, ", "))
	fmt.Println("amount " + string(amount))

	currentUserId, err := getCurrentUserId(ctx)
	if err != nil {
		return err
	}

	currentUserAccount, err := ctx.GetStub().CreateCompositeKey(userAccountType, []string{currentUserId})
	if err != nil {
		return err
	}
	fmt.Println("currentUserAccount " + currentUserAccount)

	balancesMap := t.getMap(ctx, balancesKey)

	if balancesMap[currentUserAccount] < amount {
		return errors.New("not enough coins")
	}

	mean := amount / uint(len(accounts))
	fmt.Println("mean " + string(mean))

	if mean == 0 {
		return err
	}

	var i uint = 0

	fmt.Println("uint(len(accounts)) " + string(len(accounts)))

	for i < uint(len(accounts)) {
		fmt.Println("i " + string(i))

		receiverAccount, err := ctx.GetStub().CreateCompositeKey(userAccountType, []string{accounts[i]})
		if err != nil {
			return err
		}
		fmt.Println("receiverAccount " + receiverAccount)

		balancesMap[currentUserAccount] -= mean
		fmt.Println("balancesMap[currentUserAccount} " + string(balancesMap[currentUserAccount]))
		fmt.Println("receiverAccount " + receiverAccount)
		balancesMap[receiverAccount] += mean
		fmt.Println("balancesMap[receiverAccount] " + string(balancesMap[receiverAccount]))
		i += 1
	}

	err = t.saveMap(ctx, balancesKey, balancesMap)
	if err != nil {
		return err
	}

	return nil
}

func (t *CoinChain) BalanceOf(ctx contractapi.TransactionContextInterface, args []string) (uint, error) {

	/* args
	0 - user ID
	*/

	if len(args) != 1 {
		return 0, errors.New("incorrect number of arguments. Expecting 1")
	}

	account, err := ctx.GetStub().CreateCompositeKey(userAccountType, []string{args[0]})
	if err != nil {
		return 0, err
	}

	fmt.Println("account " + account)

	balancesMap := t.getMap(ctx, balancesKey)

	return balancesMap[account], nil
}

func (t *CoinChain) BatchBalanceOf(ctx contractapi.TransactionContextInterface, args []string) ([]*UserBalance, error) {

	/* args
	0 - the array of the user emails
	*/

	if len(args) != 1 {
		return nil, errors.New("incorrect number of arguments. Expecting 1")
	}

	var emails []string

	var balancesResponse []*UserBalance

	err := json.Unmarshal([]byte(args[0]), &emails)

	if err != nil {
		return nil, err
	}

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

func (t *CoinChain) GetTransactionBalancesMap(ctx contractapi.TransactionContextInterface) map[string]uint {

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

func (t *CoinChain) parseAmountUint(amount string) uint {
	amount32, err := strconv.ParseUint(amount, 10, 32)
	if err != nil {
		return 0
	}
	return uint(amount32)
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
