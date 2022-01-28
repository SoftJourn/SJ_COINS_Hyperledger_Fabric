package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric/common/util"
	"github.com/google/uuid"
)

type FoundationChain struct {
	contractapi.Contract
}

type AllowanceRequest struct {
	ProjectId string `json:"projectId"`
	UserId    string `json:"userId"`
	Amount    uint   `json:"amount"`
}

type Withdrawal struct {
	Id        uint      `json:"id"`
	UserId    string    `json:"userId"`
	Amount    uint      `json:"amount"`
	CreatedAt time.Time `json:"createdAt"`
	Note      string    `json:"note"`
}

type WithdrawRequest struct {
	ProjectId string `json:"projectId"`
	Recipient string `json:"recipient"`
	Amount uint      `json:"amount"`
	Note string      `json:"note"`
}

type Donation struct {
	UserId          string    `json:"userId"`
	UserAccountType string    `json:"userAccountType"`
	Currency        string    `json:"currency"`
	Amount          uint      `json:"amount"`
	CreatedAt       time.Time `json:"createdAt"`
}

type DonationRequest struct {
	ProjectId string `json:"projectId"`
	Amount    uint   `json:"amount"`
	Currency  string `json:"currency"`
}

type Foundation struct {
	Id                 string                `json:"id"`
	Name               string                `json:"name"`               // Foundation name
	CategoryId         uint                  `json:"categoryId"`
	Deadline           time.Time             `json:"deadline"`           // Contract's deadline(timestamp)
	Image              string                `json:"image"`
	Description        string                `json:"description"`
	FundingGoal        uint                  `json:"fundingGoal"`        // Amount of coins to collect
	CollectedAmount    uint                  `json:"collectedAmount"`    // Amount of coins which were collected before contract has been closed
	RemainsAmount      uint                  `json:"remainsAmount"`      // Amount of coins which were collected after contract has been closed
	MainCurrency       string                `json:"mainCurrency"`       // Currency into which should be exchanged all other currencies
	AcceptCurrencies   map[string]bool       `json:"acceptCurrencies"`   // Array of currencies which are allowed for contract
	Status             uint                  `json:"status"`
	CreatorId          string                `json:"creatorId"`          // Foundation founder ID
	AdminID            string                `json:"adminId"`            // Foundation admin ID
	CreatedAt          time.Time             `json:"createdAt"`
	CloseOnGoalReached bool                  `json:"closeOnGoalReached"` // Condition of contract closing
	FundingGoalReached bool                  `json:"fundingGoalReached"`
	WithdrawalAllowed  bool                  `json:"withdrawAllowed"`
	IsContractClosed   bool                  `json:"isContractClosed"`
	IsDonationReturned bool                  `json:"isDonationReturned"`
	DonationsMap       map[string]Donation   `json:"donationsMap"`       // Map with donation info
	DonationsMapTotal  map[string]uint       `json:"donationsMapTotal"`  // Map with donation info
	WithdrawalsMap     map[string]Withdrawal `json:"withdrawalsMap"`     // Map with withdraw info
	AllowanceMap       map[string]uint       `json:"allowanceMap"`       // Map with allowance info
}

type CreateRequest struct {
	Name               string          `json:"name"`
	Image              string          `json:"image"`
	CategoryId         uint            `json:"categoryId"`
	Description        string          `json:"description"`
	FundingGoal        uint            `json:"fundingGoal"`
	MainCurrency       string          `json:"mainCurrency"`
	AcceptCurrencies   map[string]bool `json:"acceptCurrencies"`
	Deadline           uint            `json:"deadline"`
	Status             uint            `json:"status"`
	CreatorId          string          `json:"creatorId"`
	AdminID            string          `json:"adminId"`
	CloseOnGoalReached bool            `json:"closeOnGoalReached"`
	WithdrawalAllowed  bool            `json:"withdrawAllowed"`
}

type UpdateRequest struct {
	Id                 string `json:"id"`
	Name               string `json:"name"`
	Image              string `json:"image"`
	CategoryId         uint   `json:"categoryId"`
	Description        string `json:"description"`
	FundingGoal        uint   `json:"fundingGoal"`
	Deadline           uint   `json:"deadline"`
	CloseOnGoalReached bool   `json:"closeOnGoalReached"`
}

type FoundationView struct {
	Id                 string                `json:"id"`
	Name               string                `json:"name"`
	Image              string                `json:"image"`
	CreatorId          string                `json:"creatorId"`
	AdminID            string                `json:"adminId"`
	FundingGoal        uint                  `json:"fundingGoal"`
	CollectedAmount    uint                  `json:"collectedAmount"`
	RemainsAmount      uint                  `json:"remainsAmount"`
	MainCurrency       string                `json:"mainCurrency"`
	Deadline           time.Time             `json:"deadline"`
	CloseOnGoalReached bool                  `json:"closeOnGoalReached"`
	WithdrawalAllowed  bool                  `json:"withdrawAllowed"`
	FundingGoalReached bool                  `json:"fundingGoalReached"`
	IsContractClosed   bool                  `json:"isContractClosed"`
	IsDonationReturned bool                  `json:"isDonationReturned"`
	AcceptCurrencies   map[string]bool       `json:"acceptCurrencies"`
	AllowanceMap       map[string]uint       `json:"allowanceMap"`
	Status             uint                  `json:"status"`
	CreatedAt          time.Time             `json:"createdAt"`
	CategoryId         uint                  `json:"categoryId"`
	DonationsMap       map[string]Donation   `json:"donationsMap"`
	WithdrawalsMap     map[string]Withdrawal `json:"withdrawalsMap"`
}

type UserBalance struct {
	UserId  string `json:"userId"`
	Balance int    `json:"balance"`
}

type Filter struct {
	CreatorId string `json:"creatorId"`
	Status    uint   `json:"status"`
}

const STATUS_NONE uint = 0;
const STATUS_DRAFT uint = 1
const STATUS_REWIEW uint = 2
const STATUS_ACTIVE uint = 4
const STATUS_CLOSED uint = 8
const STATUS_REJECTED uint = 16

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

func (t *FoundationChain) CreateFoundation(ctx contractapi.TransactionContextInterface, foundationJson string) error {

	request := new(CreateRequest)
	err := json.Unmarshal([]byte(foundationJson), &request)

	if err != nil {
		return err
	}

	stub := ctx.GetStub()

	foundations, err := getFoundationsMap(stub)
	if err != nil {
		return errors.New(err.Error())
	}

	currentUserId, err := getCurrentUserId(stub)
	if err != nil {
		return errors.New(err.Error())
	}

	// idHash := md5.Sum([]byte(request.Name + strconv.Itoa(rand.Intn(99999)) + time.Now().Format(time.RFC3339)))

	foundation := Foundation{}
	foundation.Id = uuid.New().String()
	foundation.Name = request.Name
	foundation.CategoryId = request.CategoryId
	foundation.Description = request.Description
	foundation.Image = request.Image
	foundation.Status = request.Status
	foundation.FundingGoal = request.FundingGoal
	foundation.Deadline = time.Unix(int64(request.Deadline), 0)
	foundation.CloseOnGoalReached = request.CloseOnGoalReached
	foundation.MainCurrency = request.MainCurrency
	foundation.AcceptCurrencies = request.AcceptCurrencies
	foundation.CreatorId = currentUserId
	foundation.AdminID = request.AdminID
	foundation.CreatedAt = time.Now()
	foundation.DonationsMapTotal = make(map[string]uint)
	foundation.DonationsMap = make(map[string]Donation)
	foundation.WithdrawalsMap = make(map[string]Withdrawal)
	foundation.WithdrawalAllowed = request.WithdrawalAllowed
	foundation.AllowanceMap = make(map[string]uint)

	err = updateAllowance(&foundation, foundation.CreatorId, foundation.FundingGoal)
	if err != nil {
		return errors.New(err.Error())
	}

	foundations[foundation.Id] = foundation
	err = saveFoundations(stub, foundations)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func (t *FoundationChain) UpdateFoundation(ctx contractapi.TransactionContextInterface, requestJson string) error {

	request := new(UpdateRequest)
	err := json.Unmarshal([]byte(requestJson), &request)

	if err != nil {
		return err
	}

	stub := ctx.GetStub()

	foundations, err := getFoundationsMap(stub)
	if err != nil {
		return errors.New(err.Error())
	}

	foundation, exist := foundations[request.Id]
	if !exist {
		return errors.New("Foundation doesn't exists")
	}

	currentUserId, err := getCurrentUserId(stub)
	if err != nil {
		return errors.New(err.Error())
	}

	if foundation.Status != STATUS_DRAFT {
		return errors.New("Foundation is not in draft status")
	}

	foundation.Name = request.Name
	foundation.CategoryId = request.CategoryId
	foundation.Description = request.Description

	if request.image {
		foundation.Image = request.Image
	}

	foundation.FundingGoal = request.FundingGoal
	foundation.Deadline = time.Unix(int64(request.Deadline), 0)
	foundation.CloseOnGoalReached = request.CloseOnGoalReached

	foundations[foundation.Id] = foundation
	err = saveFoundations(stub, foundations)

	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func (t *FoundationChain) GetFoundations(ctx contractapi.TransactionContextInterface, filter Filter) ([]*FoundationView, error) {

	stub := ctx.GetStub()

	foundations, err := getFoundationsMap(stub)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	currentUserId, err := getCurrentUserId(stub)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	var statusMask uint = STATUS_ACTIVE | STATUS_CLOSED;
	if (len(filter.CreatorId) > 0 && filter.CreatorId == currentUserId) || (filter.Status != STATUS_NONE && (filter.Status & statusMask) == filter.Status) {
		statusMask = filter.Status
	}

	views := []*FoundationView{}
	for _, foundation := range(foundations) {
		if (len(filter.CreatorId) > 0 && foundation.CreatorId != filter.CreatorId) {
			continue
		}

		if (statusMask != STATUS_NONE && (foundation.Status & statusMask) == 0) {
			continue
		}

		views = append(views, getViewFromFoundation(&foundation))
	}

	if (len(views) > 0) {
		sort.SliceStable(views, func(i, j int) bool {
			return views[i].CreatedAt.Before(views[j].CreatedAt)
		})
	}

	return views, nil
}

func (t *FoundationChain) GetFoundation(ctx contractapi.TransactionContextInterface, id string) (*Foundation, error) {

	foundations, err := getFoundationsMap(ctx.GetStub())
	if err != nil {
		return nil, errors.New(err.Error())
	}

	foundation, exist := foundations[id]
	if !exist {
		return nil, errors.New("Foundation does not exist.")
	}

	return &foundation, nil
}

func (t *FoundationChain) Donate(ctx contractapi.TransactionContextInterface, donationRequestJson string) (error) {

	stub := ctx.GetStub()

	request := new(DonationRequest)
	err := json.Unmarshal([]byte(donationRequestJson), &request)
	if err != nil {
		return err
	}

	foundations, err := getFoundationsMap(stub)
	if err != nil {
		return errors.New(err.Error())
	}

	foundation, exist := foundations[request.ProjectId]
	if !exist {
		return errors.New("Foundation does not exist")
	}

	if foundation.IsContractClosed {
		return errors.New("Foundation is closed")
	}

	fmt.Println("Currency (chaincode) Name: ", request.Currency)

	fmt.Println("acceptCurrencies ", foundation.AcceptCurrencies)
	if !foundation.AcceptCurrencies[request.Currency] {
		return errors.New("Can not accept currency " + request.Currency)
	}

	fmt.Println("amount: ", request.Amount)

	if request.Amount == 0 {
		return errors.New("Error. Amount must be > 0")
	}

	fmt.Println("Invoke Transfer method on: ", request.Currency)
	queryArgs := util.ToChaincodeArgs("transfer", foundationAccountType, foundation.Id, fmt.Sprint(request.Amount))
	response := stub.InvokeChaincode(request.Currency, queryArgs, channelName)
	fmt.Println("Transfer Response status: ", response.Status)

	if response.Status == shim.OK {

		currentUserId, err := getCurrentUserId(stub)
		if err != nil {
			return errors.New(err.Error())
		}

		donation := Donation{
			UserId:          currentUserId,
			UserAccountType: userAccountType,
			Currency:        request.Currency,
			Amount:          request.Amount,
			CreatedAt:       time.Now(),
		}

		foundation.DonationsMap[strconv.Itoa(len(foundation.DonationsMap)+1)] = donation

		donationKey, err := stub.CreateCompositeKey(request.Currency, []string{userAccountType, currentUserId})
		if err != nil {
			return errors.New(err.Error())
		}

		foundation.DonationsMapTotal[donationKey] += request.Amount
		foundation.CollectedAmount += request.Amount
		fmt.Println(foundation.Id, " - foundation.CollectedAmount ", foundation.CollectedAmount)

		checkGoalReached(&foundation)

		foundations[foundation.Id] = foundation
		err = saveFoundations(stub, foundations)
		if err != nil {
			return errors.New(err.Error())
		}

		return nil
	}

	return errors.New(response.Message)
}

func (t *FoundationChain) CloseFoundation(ctx contractapi.TransactionContextInterface, id string) (uint64, error) {

	stub := ctx.GetStub()

	fmt.Println("Foundation name: ", id)

	foundations, err := getFoundationsMap(stub)
	if err != nil {
		return 0, errors.New(err.Error())
	}

	foundation, ok := foundations[id]
	if !ok {
		return 0, errors.New("Foundation does not exist.")
	}

	currentUserId, err := getCurrentUserId(stub)
	if err != nil {
		return 0, errors.New(err.Error())
	}

	if currentUserId != foundation.AdminID {
		return 0, errors.New("Failed. Only admin can close foundation.")
	}

	checkGoalReached(&foundation)

	if foundation.IsContractClosed {
		return 0, errors.New("Failed. Foundation is already closed.")
	}

	// TODO Define Return donations flow
	if foundation.FundingGoalReached {
		foundation.RemainsAmount = foundation.CollectedAmount
		fmt.Println(foundation.Name, " - Contract Remains: ", foundation.RemainsAmount)
	}

	if !foundation.FundingGoalReached {
		if !foundation.IsDonationReturned {

			// Old map
			for k, v := range foundation.DonationsMapTotal {
				if v > 0 {
					currency, parts, err := stub.SplitCompositeKey(k)
					fmt.Println("Key : ", k)
					fmt.Println("currency: ", currency)
					fmt.Println("parts: ", parts)
					fmt.Println("amount value v: ", v)

					if err != nil {
						return 0, errors.New(err.Error())
					}

					/* transferFrom args
					0 - sender account type (user_ , foundation_)
					1 - sender ID
					2 - receiver account type (user_ , foundation_)
					3 - receiver ID
					4 - amount
					*/

					fmt.Println("Invoke transferFrom method on: ", currency)
					queryArgs := util.ToChaincodeArgs("transferFrom", foundationAccountType, foundation.Id, userAccountType, parts[1], strconv.FormatUint(uint64(v), 10))
					response := stub.InvokeChaincode(currency, queryArgs, channelName)
					fmt.Println("Response status: ", response.Status)

					if response.Status != shim.OK {
						return 0, errors.New(response.Message)
					}
					//foundation.DonationsMapTotal[k] = 0;
				}
			}
			foundation.IsDonationReturned = true
		}
	} else {
		foundation.RemainsAmount = foundation.CollectedAmount
		fmt.Println(foundation.Name, " - Contract Remains: ", foundation.RemainsAmount)
	}

	foundation.IsContractClosed = true
	foundations[foundation.Id] = foundation
	err = saveFoundations(stub, foundations)
	if err != nil {
		return 0, errors.New(err.Error())
	}

	return uint64(foundation.RemainsAmount), nil
}

func (t *FoundationChain) Withdraw(ctx contractapi.TransactionContextInterface, withdrawRequestJson string) error {

	stub := ctx.GetStub()

	request := new(WithdrawRequest)
	err := json.Unmarshal([]byte(withdrawRequestJson), &request)
	if err != nil {
		return err
	}

	foundations, err := getFoundationsMap(stub)
	if err != nil {
		return errors.New(err.Error())
	}

	foundation, ok := foundations[request.ProjectId]
	if !ok {
		return errors.New("Foundation does not exist.")
	}

	// amount := t.parseAmountUint(amountString)
	fmt.Println("amount: ", request.Amount)
	fmt.Println("note: ", request.Note)
	fmt.Println("RemainsAmount: ", foundation.RemainsAmount)

	currentUserId, err := getCurrentUserId(stub)
	if err != nil {
		return errors.New(err.Error())
	}

	if !foundation.WithdrawalAllowed || foundation.AllowanceMap[currentUserId] < request.Amount {
		return errors.New("withdrawal not allowed")
	}

	if !foundation.IsContractClosed {
		return errors.New("contract is not closed")
	}

	if request.Amount > foundation.RemainsAmount {
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
	queryArgs := util.ToChaincodeArgs("transferFrom", foundationAccountType, foundation.Id, userAccountType, request.Recipient, strconv.FormatUint(uint64(request.Amount), 10))
	response := stub.InvokeChaincode(foundation.MainCurrency, queryArgs, channelName)
	fmt.Println("Response status: ", response.Status)

	if response.Status != shim.OK {
		return errors.New(response.Message)
	}

	foundation.RemainsAmount -= request.Amount
	foundation.AllowanceMap[currentUserId] -= request.Amount

	newDetail := Withdrawal{
		Id: uint(len(foundation.WithdrawalsMap) + 1),
		UserId: currentUserId,
		Amount: request.Amount,
		CreatedAt: time.Now(),
		Note: request.Note,
	}
	foundation.WithdrawalsMap[strconv.Itoa(len(foundation.WithdrawalsMap)+1)] = newDetail
	fmt.Println("detailsMap: ", foundation.WithdrawalsMap)

	foundations[foundation.Id] = foundation
	err = saveFoundations(stub, foundations)
	if err != nil {
		return errors.New(err.Error())
	}

	fmt.Println("---- withdraw successful")
	return nil
}

func (t *FoundationChain) SetAllowance(ctx contractapi.TransactionContextInterface, allowanceRequestJson string) error {

	stub := ctx.GetStub()

	request := new(AllowanceRequest)
	err := json.Unmarshal([]byte(allowanceRequestJson), &request)
	if err != nil {
		return err
	}

	foundations, err := getFoundationsMap(stub)
	if err != nil {
		return errors.New(err.Error())
	}

	foundation, exist := foundations[request.ProjectId]
	if !exist {
		return errors.New("Foundation does not exist.")
	}

	currentUserId, err := getCurrentUserId(stub)
	if err != nil {
		return errors.New(err.Error())
	}

	if currentUserId == foundation.AdminID && foundation.WithdrawalAllowed {
		updateAllowance(&foundation, request.UserId, request.Amount)
		foundations[foundation.Id] = foundation
		saveFoundations(stub, foundations)
		return nil
	}

	return errors.New("Failed to set allowance")
}

// func (t *FoundationChain) SetStatus(ctx contractapi.TransactionContextInterface, projectId string, status uint) error {

// 	stub := ctx.GetStub()

// 	foundations, err := getFoundationsMap(stub)
// 	if err != nil {
// 		return errors.New(err.Error())
// 	}

// 	foundation, exist := foundations[projectId]
// 	if !exist {
// 		return errors.New("Foundation does not exist.")
// 	}

// 	currentUserId, err := getCurrentUserId(stub)
// 	if err != nil {
// 		return errors.New(err.Error())
// 	}

// 	if status < STATUS_DRAFT || status > STATUS_REJECTED {
// 		return errors.New("Invalid status value")
// 	}

// 	if currentUserId != foundation.AdminID
// 			|| !(foundation.Status == STATUS_DRAFT && status == STATUS_REWIEW
// 					&& foundation.CreatorId == currentUserId)
// 	{
// 		return errors.New("Permission denied")
// 	}

// 	foundation.Status = status
// 	foundations[foundation.Name] = foundation
// 	saveFoundations(stub, foundations)
	
// 	return nil
// }

////////// Internal functions ////////////

func parseAmountUint(amount string) uint {
	amount32, err := strconv.ParseUint(amount, 10, 32)
	if err != nil {
		return 0
	}
	return uint(amount32)
}

func checkGoalReached(foundation *Foundation) bool {

	if foundation.CollectedAmount >= foundation.FundingGoal {
		foundation.FundingGoalReached = true
	}

	if foundation.CloseOnGoalReached && (foundation.FundingGoalReached || time.Now().After(foundation.Deadline)) {
		foundation.RemainsAmount = foundation.CollectedAmount
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

func getFoundationsMap(stub shim.ChaincodeStubInterface) (map[string]Foundation, error) {

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

func getViewFromFoundation(foundation *Foundation) *FoundationView {
	view := new(FoundationView)
	view.Id = foundation.Id
	view.Name = foundation.Name
	view.Image = foundation.Image
	view.CreatorId = foundation.CreatorId
	view.AdminID = foundation.AdminID
	view.FundingGoal = foundation.FundingGoal
	view.CollectedAmount = foundation.CollectedAmount
	view.RemainsAmount = foundation.RemainsAmount
	view.MainCurrency = foundation.MainCurrency
	view.Deadline = foundation.Deadline
	view.CloseOnGoalReached = foundation.CloseOnGoalReached
	view.AcceptCurrencies = foundation.AcceptCurrencies
	view.WithdrawalAllowed = foundation.WithdrawalAllowed
	view.FundingGoalReached = foundation.FundingGoalReached
	view.IsContractClosed = foundation.IsContractClosed
	view.IsDonationReturned = foundation.IsDonationReturned
	view.AllowanceMap = foundation.AllowanceMap
	view.CategoryId = foundation.CategoryId
	view.Status = foundation.Status
	view.DonationsMap = foundation.DonationsMap
	view.WithdrawalsMap = foundation.WithdrawalsMap
	view.CreatedAt = foundation.CreatedAt

	return view;
}

func updateAllowance(foundation *Foundation, userId string, amount uint) error {
	if (amount == 0) {
		delete(foundation.AllowanceMap, userId)
	} else {
		foundation.AllowanceMap[userId] = amount
	}
	return nil
}
