package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric/common/util"
)

type FoundationChain struct {
	contractapi.Contract
}

type WithdrawDetails struct {
	Amount uint      `json:"amount"`
	Id     uint      `json:"id"`
	Time   time.Time `json:"time"`
	Note   string    `json:"note"`
}

type Donation struct {
	UserId          string `json:"userId"`
	UserAccountType string `json:"userAccountType"`
	Currency        string `json:"currency"`
	Amount          uint   `json:"amount"`
}

type Foundation struct {
	Name               string                  `json:"name"`               //Foundation name
	CreatorId          string                  `json:"creatorId"`          //Foundation founder ID
	AdminID            string                  `json:"adminId"`            //Foundation admin ID
	FundingGoal        uint                    `json:"fundingGoal"`        //Amount of coins to collect
	CollectedAmount    uint                    `json:"collectedAmount"`    //Amount of coins which were collected before contract has been closed
	ContractRemains    uint                    `json:"contractRemains"`    //Amount of coins which were collected after contract has been closed
	MainCurrency       string                  `json:"mainCurrency"`       //Currency into which should be exchanged all other currencies
	Deadline           time.Time               `json:"deadline"`           //Contract's deadline(timestamp)
	CloseOnGoalReached bool                    `json:"closeOnGoalReached"` //Condition of contract closing
	AcceptCurrencies   map[string]bool         `json:"acceptCurrencies"`   //Array of currencies which are allowed for contract
	DonationsMapOld    map[string]uint         `json:"donationsMapOld"`    //Map with donation info
	DonationsMap       map[int]Donation        `json:"donationsMap"`       //Map with donation info
	WithdrawDetailsMap map[int]WithdrawDetails `json:"withdrawDetailsMap"` //Map with withdraw info
	WithdrawalAllowed  bool                    `json: withdrawAllowed`
	FundingGoalReached bool                    `json:"fundingGoalReached"`
	IsContractClosed   bool                    `json:"isContractClosed"`
	IsDonationReturned bool                    `json:"isDonationReturned"`
	AllowanceMap       map[string]uint         `json:"allowanceMap"` //Map with allowance info
}

var channelName string = "mychannel"
var foundationAccountType string = "foundation_"
var userAccountType string = "user_"
var foundationsKey string = "foundations"

func main() {
	chaincode, err := contractapi.NewChaincode(new(FoundationChain))

	if err != nil {
		fmt.Printf("Error create fabcar chaincode: %s", err.Error())
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting fabcar chaincode: %s", err.Error())
	}
}

func (t *FoundationChain) InitLedger(ctx contractapi.TransactionContextInterface) error {

	fmt.Println("######### " + foundationsKey + " Init ########")

	mapBytes, err := ctx.GetStub().GetState(foundationsKey)
	if err != nil {
		fmt.Println("Init get foundations error: ", err)
	}

	fmt.Println("Init foundations map %s: ", mapBytes)
	if len(mapBytes) == 0 {
		foundationsMap := make(map[string]Foundation)
		err = saveFoundations(ctx.GetStub(), foundationsMap)
		if err != nil {
			return errors.New(err.Error())
		}
	}

	return nil
}

func (t *FoundationChain) CreateFoundation(ctx contractapi.TransactionContextInterface, args []string) error {

	/* args
	0 - foundation Name
	1 - admin account
	2 - foundation creator
	3 - Goal
	4 - Deadline Minutes
	5 - Close on reached goal
	6 - withdraw allowed
	7 - Currency
	... n - accept currencies
	*/

	stub := ctx.GetStub()

	if len(args) < 9 {
		return errors.New("Incorrect number of arguments. Expecting at least 9")
	}

	foundations, err := getFoundations(stub)
	if err != nil {
		return errors.New(err.Error())
	}

	_, exist := foundations[args[0]]
	if exist {
		return errors.New("Foundation already exists")
	}

	foundation := Foundation{}
	foundation.Name = args[0]
	fmt.Println("foundationName: ", foundation.Name)

	foundation.AdminID = args[1]
	fmt.Println("admin ID: ", foundation.AdminID)

	foundation.CreatorId = args[2]
	fmt.Println("creator ID: ", foundation.CreatorId)

	fundingGoalArg, err := strconv.ParseUint(args[3], 10, 32)
	if err != nil {
		return errors.New(err.Error())
	}
	foundation.FundingGoal = uint(fundingGoalArg)
	fmt.Println("funding Goal: ", foundation.FundingGoal)

	minutesInt, err := strconv.ParseInt(args[4], 10, 32)
	if err != nil {
		return errors.New(err.Error())
	}
	duration := time.Minute * time.Duration(minutesInt)
	currentTime := time.Now()
	foundation.Deadline = currentTime.Add(duration)
	fmt.Println("deadline: ", foundation.Deadline.Format(time.RFC3339))

	closeOnGoal, err := strconv.ParseBool(args[5])
	if err != nil {
		return errors.New(err.Error())
	}

	foundation.CloseOnGoalReached = closeOnGoal
	fmt.Println("closeOnGoalReached: ", foundation.CloseOnGoalReached)

	withdrawalAllowed, err := strconv.ParseBool(args[6])
	if err != nil {
		return errors.New(err.Error())
	}

	foundation.WithdrawalAllowed = withdrawalAllowed

	foundation.MainCurrency = args[7]
	fmt.Println("Main currency: ", foundation.MainCurrency)

	currencies := args[8:]
	fmt.Println("currencies: ", currencies)

	foundation.AcceptCurrencies = make(map[string]bool)
	for _, v := range currencies {
		foundation.AcceptCurrencies[v] = true
	}
	fmt.Println("Accept Currencies: ", foundation.AcceptCurrencies)

	foundation.DonationsMapOld = make(map[string]uint)
	foundation.DonationsMap = make(map[int]Donation)
	foundation.WithdrawDetailsMap = make(map[int]WithdrawDetails)
	foundation.AllowanceMap = make(map[string]uint)
	foundations[foundation.Name] = foundation
	err = saveFoundations(stub, foundations)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func (t *FoundationChain) Donate(ctx contractapi.TransactionContextInterface, args []string) ([]byte, error) {

	/* args
	0 - currency name (docker container name - coin)
	1 - amount
	2 - foundation name
	*/

	stub := ctx.GetStub()

	if len(args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 3")
	}

	foundations, err := getFoundations(stub)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	foundation, exist := foundations[args[2]]
	if !exist {
		return nil, errors.New("Foundation does not exist")
	}

	if foundation.IsContractClosed {
		return nil, errors.New("Foundation is closed")
	}

	currency := args[0]
	fmt.Println("Currency (chaincode) Name: ", currency)

	fmt.Println("acceptCurrencies ", foundation.AcceptCurrencies)
	if !foundation.AcceptCurrencies[currency] {
		return nil, errors.New("Can not accept currency " + currency)
	}

	amount := t.parseAmountUint(args[1])
	fmt.Println("amount: ", amount)

	if amount == 0 {
		return nil, errors.New("Error. Amount must be > 0")
	}

	fmt.Println("Invoke Transfer method on: ", currency)
	queryArgs := util.ToChaincodeArgs("transfer", foundationAccountType, foundation.Name, args[1])
	response := stub.InvokeChaincode(currency, queryArgs, channelName)
	fmt.Println("Transfer Response status: ", response.Status)

	if response.Status == shim.OK {

		currentUserId, err := getCurrentUserId(stub)
		if err != nil {
			return nil, errors.New(err.Error())
		}

		donation := Donation{
			UserId:          currentUserId,
			UserAccountType: userAccountType,
			Currency:        currency,
			Amount:          amount,
		}

		foundation.DonationsMap[len(foundation.DonationsMap)+1] = donation

		donationKey, err := stub.CreateCompositeKey(currency, []string{userAccountType, currentUserId})
		if err != nil {
			return nil, errors.New(err.Error())
		}

		foundation.DonationsMapOld[donationKey] += amount
		foundation.CollectedAmount += amount
		fmt.Println(foundation.Name, " - foundation.CollectedAmount ", foundation.CollectedAmount)

		checkGoalReached(&foundation)

		foundations[foundation.Name] = foundation
		err = saveFoundations(stub, foundations)
		if err != nil {
			return nil, errors.New(err.Error())
		}

		return []byte(strconv.FormatUint(uint64(foundation.CollectedAmount), 10)), nil
	}

	return nil, errors.New(response.Message)
}

func (t *FoundationChain) CloseFoundation(ctx contractapi.TransactionContextInterface, args []string) ([]byte, error) {

	/* args
	0 - foundation name
	*/

	stub := ctx.GetStub()

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	fmt.Println("Foundation name: ", args[0])

	foundations, err := getFoundations(stub)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	foundation, ok := foundations[args[0]]
	if !ok {
		return nil, errors.New("Foundation does not exist.")
	}

	checkGoalReached(&foundation)

	if foundation.IsContractClosed {
		return nil, errors.New("Failed. Foundation is already closed.")
	}

	currentUserId, err := getCurrentUserId(stub)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	if currentUserId != foundation.AdminID {
		return nil, errors.New("Failed. Only admin can close foundation.")
	}

	// TODO Define Return donations flow
	if foundation.FundingGoalReached {
		foundation.ContractRemains = foundation.CollectedAmount
		fmt.Println(foundation.Name, " - Contract Remains: ", foundation.ContractRemains)
	}

	if !foundation.FundingGoalReached {
		if !foundation.IsDonationReturned {

			// Old map
			for k, v := range foundation.DonationsMapOld {
				if v > 0 {
					currency, parts, err := stub.SplitCompositeKey(k)
					fmt.Println("Key : ", k)
					fmt.Println("currency: ", currency)
					fmt.Println("parts: ", parts)
					fmt.Println("amount value v: ", v)

					if err != nil {
						return nil, errors.New(err.Error())
					}

					/* transferFrom args
					0 - sender account type (user_ , foundation_)
					1 - sender ID
					2 - receiver account type (user_ , foundation_)
					3 - receiver ID
					4 - amount
					*/

					fmt.Println("Invoke transferFrom method on: ", currency)
					queryArgs := util.ToChaincodeArgs("transferFrom", foundationAccountType, foundation.Name, userAccountType, parts[1], strconv.FormatUint(uint64(v), 10))
					response := stub.InvokeChaincode(currency, queryArgs, channelName)
					fmt.Println("Response status: ", response.Status)

					if response.Status != shim.OK {
						return nil, errors.New(response.Message)
					}
					//foundation.DonationsMapOld[k] = 0;
				}
			}
			foundation.IsDonationReturned = true
		}
	} else {
		foundation.ContractRemains = foundation.CollectedAmount
		fmt.Println(foundation.Name, " - Contract Remains: ", foundation.ContractRemains)
	}

	foundation.IsContractClosed = true
	foundations[foundation.Name] = foundation
	err = saveFoundations(stub, foundations)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	return []byte(strconv.FormatUint(uint64(foundation.ContractRemains), 10)), nil
}

func (t *FoundationChain) Withdraw(ctx contractapi.TransactionContextInterface, args []string) error {

	/* args
	0 - foundation name
	1 - receiverId (user_ type)
	2 - amount
	3 - note
	*/

	stub := ctx.GetStub()

	if len(args) != 4 {
		return errors.New("Incorrect number of arguments. Expecting 4")
	}

	foundationName := args[0]
	receiverId := args[1]
	amountString := args[2]
	note := args[3]

	foundations, err := getFoundations(stub)
	if err != nil {
		return errors.New(err.Error())
	}

	foundation, ok := foundations[foundationName]
	if !ok {
		return errors.New("Foundation does not exist.")
	}

	amount := t.parseAmountUint(amountString)
	fmt.Println("amount: ", amount)
	fmt.Println("note: ", note)
	fmt.Println("contractRemains: ", foundation.ContractRemains)

	currentUserId, err := getCurrentUserId(stub)
	if err != nil {
		return errors.New(err.Error())
	}

	if !foundation.WithdrawalAllowed || foundation.AllowanceMap[currentUserId] < amount {
		return errors.New("withdrawal not allowed")
	}

	if !foundation.IsContractClosed {
		return errors.New("contract is not closed")
	}

	if amount > foundation.ContractRemains {
		return errors.New("not enough funds")
	}

	/* transferFrom args
	0 - sender account type (user_ , foundation_)
	1 - sender ID
	2 - receiver account type (user_ , foundation_)
	3 - receiver ID
	4 - amount
	*/

	fmt.Println("Invoke transferFrom method on: ", foundation.MainCurrency)
	queryArgs := util.ToChaincodeArgs("transferFrom", foundationAccountType, foundation.Name, userAccountType, receiverId, strconv.FormatUint(uint64(amount), 10))
	response := stub.InvokeChaincode(foundation.MainCurrency, queryArgs, channelName)
	fmt.Println("Response status: ", response.Status)

	if response.Status != shim.OK {
		return errors.New(response.Message)
	}

	foundation.ContractRemains -= amount

	newDetail := WithdrawDetails{Time: time.Now(), Amount: amount, Note: note, Id: uint(len(foundation.WithdrawDetailsMap) + 1)}
	foundation.WithdrawDetailsMap[len(foundation.WithdrawDetailsMap)+1] = newDetail
	fmt.Println("detailsMap: ", foundation.WithdrawDetailsMap)

	foundations[foundation.Name] = foundation
	err = saveFoundations(stub, foundations)
	if err != nil {
		return errors.New(err.Error())
	}

	fmt.Println("---- withdraw successful")
	return nil
}

// Get foundations call handler. Just get a list of foundations' names.
func (t *FoundationChain) GetFoundations(ctx contractapi.TransactionContextInterface, args []string) ([]byte, error) {

	foundations, err := getFoundations(ctx.GetStub())
	if err != nil {
		return nil, errors.New(err.Error())
	}

	keys := make([]string, 0, len(foundations))
	for k := range foundations {
		keys = append(keys, k)
	}

	bytes, err := json.Marshal(keys)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	return bytes, nil
}

// Get foundation by name call handler.
func (t *FoundationChain) getFoundationByName(ctx contractapi.TransactionContextInterface, args []string) ([]byte, error) {

	/* args
	0 - foundation name
	*/

	foundations, err := getFoundations(ctx.GetStub())
	if err != nil {
		return nil, errors.New(err.Error())
	}

	foundation, exist := foundations[args[0]]
	if !exist {
		return nil, errors.New("Foundation does not exist.")
	}

	bytes, err := json.Marshal(foundation)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	return bytes, nil
}

// Set amount of allowed withdraw for user.
func (t *FoundationChain) SetAllowance(ctx contractapi.TransactionContextInterface, args []string) error {

	/* args
	0 - foundation name
	1 - user ID
	2 - amount
	*/

	stub := ctx.GetStub()

	if len(args) != 3 {
		return errors.New("Incorrect number of arguments. Expecting 3")
	}

	foundationName := args[0]
	userId := args[1]
	amountString := args[2]
	amount := t.parseAmountUint(amountString)

	foundations, err := getFoundations(stub)
	if err != nil {
		return errors.New(err.Error())
	}

	foundation, exist := foundations[foundationName]
	if !exist {
		return errors.New("Foundation does not exist.")
	}

	currentUserId, err := getCurrentUserId(stub)
	if err != nil {
		return errors.New(err.Error())
	}

	if currentUserId == foundation.AdminID && foundation.WithdrawalAllowed || currentUserId == foundation.Name {
		//userAccount, err := stub.CreateCompositeKey(userType, []string{userId})
		//if err != nil {
		//	return shim.Error(err.Error())
		//}

		//foundation.AllowanceMap[userAccount] = amount

		foundation.AllowanceMap[userId] = amount
		foundations[foundation.Name] = foundation
		saveFoundations(stub, foundations)
		return nil
	}

	return errors.New("Failed to set allowance")
}

func (t *FoundationChain) parseAmountUint(amount string) uint {
	amount32, err := strconv.ParseUint(amount, 10, 32)
	if err != nil {
		return 0
	}
	return uint(amount32)
}

// // TODO: What it should do?
// func (t *FoundationChain) receiveApproval(stub shim.ChaincodeStubInterface, args []string) pb.Response {
// 	return shim.Success(nil)
// }

func (t *FoundationChain) TestChaincodeInvoke(ctx contractapi.TransactionContextInterface) error {

	queryArgs := util.ToChaincodeArgs("balanceOf", "Jim")
	response := ctx.GetStub().InvokeChaincode("coin", queryArgs, channelName)
	fmt.Println("Transfer Response status: ", response.Status)

	return nil
}

// Check goal is reached.
func checkGoalReached(foundation *Foundation) bool {

	if foundation.CollectedAmount >= foundation.FundingGoal {
		foundation.FundingGoalReached = true
	}

	if foundation.CloseOnGoalReached && (foundation.FundingGoalReached || time.Now().After(foundation.Deadline)) {
		foundation.ContractRemains = foundation.CollectedAmount
		foundation.IsContractClosed = true
	}

	fmt.Println(foundation.Name, " - FundingGoalReached: ", foundation.FundingGoalReached)
	fmt.Println(foundation.Name, " -   isContractClosed: ", foundation.IsContractClosed)

	return foundation.FundingGoalReached
}

func getCurrentUserId(stub shim.ChaincodeStubInterface) (string, error) {

	var userId string

	creatorBytes, err := stub.GetCreator()
	if err != nil {
		return userId, err
	}

	creatorString := fmt.Sprintf("%s", creatorBytes)
	index := strings.Index(creatorString, "-----BEGIN CERTIFICATE-----")
	certificate := creatorString[index:]
	block, _ := pem.Decode([]byte(certificate))

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return userId, err
	}

	userId = cert.Subject.CommonName
	fmt.Println("---- Current User ID: %v ", userId)
	return userId, err
}

func getFoundations(stub shim.ChaincodeStubInterface) (map[string]Foundation, error) {

	fmt.Println("------ getFoundations called")
	mapBytes, err := stub.GetState(foundationsKey)
	if err != nil {
		return nil, err
	}

	var mapObject map[string]Foundation
	err = json.Unmarshal(mapBytes, &mapObject)
	if err != nil {
		return nil, err
	}
	fmt.Println("received Foundations map %s", mapObject)
	return mapObject, nil
}

func saveFoundations(stub shim.ChaincodeStubInterface, mapObject map[string]Foundation) error {
	fmt.Println("------ saveFoundations called")

	mapBytes, err := json.Marshal(mapObject)
	if err != nil {
		return err
	}
	err = stub.PutState(foundationsKey, mapBytes)
	if err != nil {
		return err
	}
	fmt.Println("saved ", mapObject)
	return nil
}
